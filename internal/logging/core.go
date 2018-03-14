// Copyright © 2018 Pennock Tech, LLC.
// All rights reserved, except as granted under license.
// Licensed per file LICENSE.txt

package logging

import (
	"log"
	"os"
)

// Logger is the interface API which the rest of our code should use for logging.
// Originally this was a type alias for logrus.FieldLogger, but we now enumerate
// for ourselves exactly what we do rely upon, so that we have a smaller surface
// and can drop something else in.  As we now do, under certain build tags.
type Logger interface {
	WithField(key string, value interface{}) Logger
	WithError(err error) Logger

	// while logrus uses `args ...interface{}`, we push hard enough for
	// structured logging that not only do we not use any of the Foof()
	// variants, but we also only ever call these with one item, a string
	// message for the humans.  EVERYTHING else in our usage must be properly
	// structured, under some key.
	Debug(message string)
	Info(message string)
	Warning(message string)
	Error(message string)

	// IsDisabled is a simple one-liner which real implementations should use
	// to return false, unless they support the concept of disabling.
	IsDisabled() bool
}

// Setup is used to setup logging.
func Setup() Logger {
	if Enabled() {
		return implSetup()
	}
	return newNilLoggerDisablingLog()
}

// NilLogger returns a typed nil which satisfies Logger but does nothing.
func NilLogger() Logger {
	var nl *nilLogger
	return nl
}

func newNilLoggerDisablingLog() Logger {
	f, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err == nil {
		log.SetFlags(0)
		log.SetOutput(f)
	}
	return NilLogger()
}

type nilLogger struct{}

// WithField on *nilLogger just returns the *nilLogger.
func (n *nilLogger) WithField(key string, value interface{}) Logger { return n }

// WithError on *nilLogger just returns the *nilLogger.
func (n *nilLogger) WithError(err error) Logger { return n }

// Debug on *nilLogger does nothing.
func (n *nilLogger) Debug(message string) {}

// Info on *nilLogger does nothing.
func (n *nilLogger) Info(message string) {}

// Warning on *nilLogger does nothing.
func (n *nilLogger) Warning(message string) {}

// Error on *nilLogger does nothing.
func (n *nilLogger) Error(message string) {}

// IsDisabled on *nilLogger always returns true
func (n *nilLogger) IsDisabled() bool { return true }
