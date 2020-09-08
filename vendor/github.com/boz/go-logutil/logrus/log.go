package logrus

import (
	"fmt"

	logutil "github.com/boz/go-logutil"
	lr "github.com/sirupsen/logrus"
)

const (
	ComponentFieldName = "cmp"
	traceFieldName     = "func"
)

type log struct {
	parent lr.FieldLogger
}

func New(parent lr.FieldLogger) logutil.Log {
	return &log{parent}
}

func (l *log) WithComponent(name string) logutil.Log {
	return &log{l.parent.WithField(ComponentFieldName, name)}
}

func (l *log) Trace(format string, args ...interface{}) string {
	name := fmt.Sprintf(format, args...)
	l.parent.WithField(traceFieldName, name).Debug("ENTER")
	return name
}

func (l *log) Un(name string) {
	l.parent.WithField(traceFieldName, name).Debug("LEAVE")
}

func (l *log) Err(err error, fmt string, args ...interface{}) error {
	l.parent.WithError(err).Errorf(fmt, args...)
	return err
}

func (l *log) ErrWarn(err error, fmt string, args ...interface{}) error {
	l.parent.WithError(err).Warningf(fmt, args...)
	return err
}

func (l *log) ErrFatal(err error, fmt string, args ...interface{}) error {
	l.parent.WithError(err).Fatalf(fmt, args...)
	return err
}

func (l *log) Debugf(fmt string, args ...interface{}) {
	l.parent.Debugf(fmt, args...)
}

func (l *log) Infof(fmt string, args ...interface{}) {
	l.parent.Infof(fmt, args...)
}

func (l *log) Warnf(fmt string, args ...interface{}) {
	l.parent.Warningf(fmt, args...)
}

func (l *log) Errorf(fmt string, args ...interface{}) {
	l.parent.Errorf(fmt, args...)
}

func (l *log) Fatalf(fmt string, args ...interface{}) {
	l.parent.Fatalf(fmt, args...)
}
