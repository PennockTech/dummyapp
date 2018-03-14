// Copyright © 2018 Pennock Tech, LLC.
// All rights reserved, except as granted under license.
// Licensed per file LICENSE.txt

package logging

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
}
