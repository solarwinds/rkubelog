// Copyright 2021 Solarwinds Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package papertrailgo

import (
	"crypto/x509"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/papertrail/remote_syslog2/syslog"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const papertrailRootCAsURL = "https://papertrailapp.com/tools/papertrail-bundle.pem"
const papertrailRootCAFile = "/etc/ssl/certs/papertrail-bundle.pem"

type papertrailShipper interface {
	Write(packet syslog.Packet)
	Close() error
}

// NewPapertailShipper creates an instance of the papertrail shipper with the given protocol, host, port
func NewPapertailShipper(paperTrailProtocol, paperTrailHost string, paperTrailPort int) (*syslog.Logger, error) {
	var rootCAs *x509.CertPool
	var err error

	paperTrailHost = strings.TrimSpace(paperTrailHost)
	paperTrailProtocol = strings.TrimSpace(strings.ToLower(paperTrailProtocol))

	if !validateProtocol(paperTrailProtocol) {
		return nil, errors.New("given protocol is not valid, supported protocols are udp, tcp, tls (for tls over tcp)")
	}
	if paperTrailHost == "" {
		return nil, errors.New("given papertrail host is not valid")
	}
	if paperTrailPort <= 0 {
		return nil, errors.New("given papertrail host port is not valid")
	}

	// add the papertrail root CA if necessary
	if strings.Contains(strings.TrimSpace(paperTrailProtocol), "tls") && strings.HasSuffix(strings.ToLower(paperTrailHost), "papertrailapp.com") {
		logrus.Infof("retrieving root CAs for Papertrail")
		rootCAs, err = getRootCAs()
		if err != nil {
			return nil, err
		}
		logrus.Debugf("retrieved root CA: %+v", rootCAs)
	}
	raddr := net.JoinHostPort(paperTrailHost, strconv.Itoa(paperTrailPort))
	logrus.Infof("Connecting to %s over %s", raddr, paperTrailProtocol)
	sysL, err := syslog.Dial(
		paperTrailHost,
		paperTrailProtocol,
		raddr, rootCAs,
		30*time.Second, // these 3 values borrowed from https://github.com/papertrail/remote_syslog2/blob/master/config.go
		30*time.Second,
		99990,
	)
	if err != nil {
		logrus.Errorf("Initial connection to server failed: %v - connection will be retried", err)
		return nil, err
	}
	go func(sl *syslog.Logger) {
		for err = range sl.Errors {
			logrus.Errorf("syslog error: %v", err)
		}
	}(sysL)
	return sysL, nil
}

func getCerts() ([]byte, error) {
	resp, err := http.Get(papertrailRootCAsURL)
	if err != nil {
		err = errors.Wrap(err, "unable to fetch root CAs")
		logrus.Error(err)
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		err = errors.Wrap(err, "unable to parse root CAs")
		logrus.Error(err)
		return nil, err
	}
	return body, nil
}

func getRootCAs() (*x509.CertPool, error) {
	pool := *x509.NewCertPool()
	var certs []byte
	var err error
	if _, err = os.Stat(papertrailRootCAFile); err == nil {
		certs, err = ioutil.ReadFile(papertrailRootCAFile)
		if err != nil {
			logrus.Warnf("unable to read papertrail CA file from disk: %s, proceeding to fetch the cert...", papertrailRootCAFile)
		} else {
			logrus.Infof("using the papertrail root CA file from the local filesystem")
		}
	}
	if certs == nil {
		logrus.Infof("unable to find a valid papertrail root CA file in the local filesystem, attempting to fetch from %s", papertrailRootCAsURL)
		certs, err = getCerts()
		if err != nil {
			return nil, err
		}
	}
	if ok := pool.AppendCertsFromPEM(certs); !ok {
		err := errors.New("unable to append certs")
		logrus.Error(err)
		return nil, err
	}
	return &pool, nil
}

func validateProtocol(protocol string) bool {
	switch protocol {
	case "tls", "tcp", "udp":
		return true
	default:
		return false
	}
}
