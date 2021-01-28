// Copyright 2019 SolarWinds Worldwide, LLC.
// SPDX-License-Identifier: Apache-2.0

// Package logshipper contains all the log shippers
package logshipper

import "github.com/boz/kail"

// LogShipper is the interface which all log shipper types have to adhere to
type LogShipper interface {
	Log(kail.Event) error
	Close() error
}
