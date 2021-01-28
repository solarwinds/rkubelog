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
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	syslog "github.com/RackSec/srslog"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const papertrailRootCAsURL = "https://papertrailapp.com/tools/papertrail-bundle.pem"
const papertrailRootCAFile = "/etc/ssl/certs/papertrail-bundle.pem"

type syslogProto string

const (
	// UDP represents UDP protocol
	UDP syslogProto = "udp"

	// TCP represents TCP protocol
	TCP syslogProto = "tcp"

	// TLS represents TLS protocol
	TLS syslogProto = "tcp+tls"

	// like time.RFC3339Nano but with a limit of 6 digits in the SECFRAC part
	rfc5424time = "2006-01-02T15:04:05.999999Z07:00"
)

type papertrailShipper interface {
	Write(packet *SyslogPacket) error
	Close() error
}

// A SyslogPacket represents an RFC5425 syslog message
type SyslogPacket struct {
	Severity syslog.Priority
	Hostname string
	Tag      string
	Time     time.Time
	Message  string
}

// SrslogShipper represents srslog shipper
type SrslogShipper struct {
	writer    *syslog.Writer
	protocol  syslogProto
	hostport  string
	tlsConfig *tls.Config

	tag string
}

// Dial attempts to dial a syslog connection
func (s *SrslogShipper) Dial() (err error) {
	if s == nil {
		err = fmt.Errorf("syslog shipper is nil")
		logrus.Error(err)
		return err
	}

	logrus.Debugf("dialing syslog over protocol: %s, host: %s", s.protocol, s.hostport)

	switch {
	case s.protocol == UDP, s.protocol == TCP:
		s.writer, err = syslog.Dial(string(s.protocol), s.hostport, syslog.LOG_NOTICE, s.tag)
	default:
		s.writer, err = syslog.DialWithTLSConfig(string(s.protocol), s.hostport, syslog.LOG_NOTICE, s.tag, s.tlsConfig)
	}

	return err
}

// Write writes packets on the wire
func (s *SrslogShipper) Write(packet *SyslogPacket) (err error) {
	if s == nil {
		err = fmt.Errorf("syslog shipper is nil")
		logrus.Error(err)
		return err
	}

	// s.writer.SetHostname(packet.Hostname)
	s.writer.SetFormatter(s.Formatter)

	switch {
	case s.protocol == UDP:
		defer func() {
			_ = s.writer.Close()
		}()
		fallthrough
	case s.writer == nil:
		if err = s.Dial(); err != nil {
			return err
		}
	}

	ts := packet.Time

	if ts.IsZero() {
		ts = time.Now()
	}

	// msg := fmt.Sprintf("<%d> %s %s %s %s - - - %s", packet.Severity, ts.Format(rfc5424time), packet.Hostname, s.tag, packet.Tag, packet.Message)
	var msg string

	switch s.protocol {
	case TCP, TLS:
		msg = fmt.Sprintf("%s %s %s - %s", packet.Hostname, s.tag, packet.Tag, packet.Message)
	default:
		msg = fmt.Sprintf("%s %s - %s", packet.Hostname, packet.Tag, packet.Message)
	}

	_, err = s.writer.WriteWithPriority(packet.Severity, []byte(msg))
	return err
}

// Formatter is actually a dummy placeholder
func (s *SrslogShipper) Formatter(p syslog.Priority, hostname, tag, content string) string {
	return content
}

// Close will close the syslog writer
func (s *SrslogShipper) Close() error {
	if s != nil && s.writer != nil {
		_ = s.writer.Close()
	}

	return nil
}

// NewPapertailShipper creates an instance of the papertrail shipper with the given protocol, host, port
func NewPapertailShipper(paperTrailProtocol, paperTrailHost string, paperTrailPort int, tag string) (*SrslogShipper, error) {
	var tlsConfig *tls.Config

	paperTrailHost = strings.TrimSpace(paperTrailHost)
	paperTrailProtocol = strings.TrimSpace(strings.ToLower(paperTrailProtocol))

	// if !validateProtocol(paperTrailProtocol) {
	// 	return nil, errors.New("given protocol is not valid, supported protocols are udp, tcp, tls (for tls over tcp)")
	// }
	selectedProtocol := validateProtocol(paperTrailProtocol)

	if paperTrailHost == "" {
		return nil, errors.New("given papertrail host is not valid")
	}

	if paperTrailPort <= 0 {
		return nil, errors.New("given papertrail host port is not valid")
	}

	// add the papertrail root CA if necessary
	if strings.Contains(paperTrailProtocol, "tls") && strings.HasSuffix(strings.ToLower(paperTrailHost), "papertrailapp.com") {
		logrus.Infof("retrieving root CAs for Papertrail")
		rootCAs, err := getRootCAs()
		if err != nil {
			return nil, err
		}
		logrus.Debugf("retrieved root CA: %+v", rootCAs)

		tlsConfig = &tls.Config{
			RootCAs: rootCAs,
		}
	}

	raddr := net.JoinHostPort(paperTrailHost, strconv.Itoa(paperTrailPort))
	logrus.Infof("Connecting to %s over %s", raddr, paperTrailProtocol)

	srs := &SrslogShipper{
		protocol:  selectedProtocol,
		hostport:  raddr,
		tlsConfig: tlsConfig,
		tag:       tag,
	}

	return srs, srs.Dial()
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

func validateProtocol(protocol string) syslogProto {
	switch protocol {
	case "tls":
		return TLS
	case "tcp":
		return TCP
	default:
		return UDP
	}
}
