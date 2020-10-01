// Copyright 2019 SolarWinds Worldwide, LLC.
// SPDX-License-Identifier: Apache-2.0

package logshipper

import (
	"github.com/boz/kail"
	loggly "github.com/segmentio/go-loggly"
)

type LogglyShipper struct {
	logglyClient *loggly.Client
}

func CreateLogglyShipper(token string) *LogglyShipper {
	return &LogglyShipper{
		logglyClient: loggly.New(token, "rkubelog"),
	}
}

func (l *LogglyShipper) Log(ev kail.Event) error {
	if l.logglyClient != nil && ev != nil && len(ev.Log()) > 0 {
		return l.logglyClient.Send(map[string]interface{}{
			"rkubelog": map[string]interface{}{
				"message":   string(ev.Log()),
				"node":      ev.Source().Node(),
				"pod":       ev.Source().Name(),
				"namespace": ev.Source().Namespace(),
				"container": ev.Source().Container(),
			},
		})
	}
	return nil
}

func (l *LogglyShipper) Close() error {
	return nil
}
