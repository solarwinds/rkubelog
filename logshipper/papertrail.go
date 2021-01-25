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

// PapertrailShipper type represents a papertrail log shipper
type PapertrailShipper struct {
	papertrailShipperInst papertrailgo.LoggerInterface
}

// CreatePapertrailShipper creates a PapertrailShipper with the given metadata
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

// Log ships the kail event to papertrail asynchronously
func (l *PapertrailShipper) Log(ev kail.Event) error {
	if l.papertrailShipperInst != nil && ev != nil && len(ev.Log()) > 0 {
		payload := &papertrailgo.Payload{
			Hostname: fmt.Sprintf("%s/%s", ev.Source().Namespace(), ev.Source().Container()),
			Tag:      fmt.Sprintf("rkubelog/node(%s)/pod(%s)", ev.Source().Node(), ev.Source().Name()),
			Log:      string(ev.Log()),
		}
		return l.papertrailShipperInst.Log(payload)
	}
	return nil
}

// Close closes the papertrailshipper
func (l *PapertrailShipper) Close() error {
	if l.papertrailShipperInst != nil {
		return l.papertrailShipperInst.Close()
	}
	return nil
}
