// Copyright 2019 SolarWinds Worldwide, LLC.
// SPDX-License-Identifier: Apache-2.0

package logshipper

import (
	"context"
	"fmt"
	"time"

	"github.com/boz/kail"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	papertrailgo "github.com/solarwinds/papertrail-go"
)

type PapertrailShipper struct {
	papertrailShipperInst papertrailgo.LoggerInterface
}

func CreatePapertrailShipper(ctx context.Context, paperTrailProtocol, paperTrailHost string, paperTrailPort int,
	dbLocation string, retention time.Duration,
	workerCount int, maxDiskUsage float64) (*PapertrailShipper, error) {
	lg, err := papertrailgo.NewPapertrailLogger(ctx, paperTrailProtocol, paperTrailHost, paperTrailPort, dbLocation, retention, workerCount, maxDiskUsage)
	if err != nil {
		err = errors.Wrap(err, "unable to create a papertrail logger")
		logrus.Error(err)
		return nil, err
	}
	return &PapertrailShipper{
		papertrailShipperInst: lg,
	}, nil
}

func (l *PapertrailShipper) Log(ev kail.Event) error {
	if l.papertrailShipperInst != nil && ev != nil && len(ev.Log()) > 0 {
		payload := &papertrailgo.Payload{
			Hostname: ev.Source().Name(),
			Tag:      fmt.Sprintf("node(%s)/namespace(%s)/container(%s)", ev.Source().Node(), ev.Source().Namespace(), ev.Source().Container()),
			Log:      string(ev.Log()),
		}
		return l.papertrailShipperInst.Log(payload)
	}
	return nil
}

func (l *PapertrailShipper) Close() error {
	if l.papertrailShipperInst != nil {
		return l.papertrailShipperInst.Close()
	}
	return nil
}
