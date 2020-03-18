// Package logshipper contains all the log shippers
package logshipper

import "github.com/boz/kail"

type LogShipper interface {
	Log(kail.Event) error
	Close() error
}
