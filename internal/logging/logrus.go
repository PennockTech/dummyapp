// Copyright © 2016,2018 Pennock Tech, LLC.
// All rights reserved, except as granted under license.
// Licensed per file LICENSE.txt

// +build logrus !zerolog

package logging

import (
	"flag"
	stdlog "log"
	"log/syslog"
	"os"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	logrus_syslog "github.com/sirupsen/logrus/hooks/syslog"

	"go.pennock.tech/dummyapp/internal/version"
)

var logOpts struct {
	level        string
	json         bool
	syslogRemote string
	syslogProto  string
	syslogTag    string
	noLocal      bool
}

func init() {
	flag.StringVar(&logOpts.level, "log.level", "info", "logging level")
	flag.BoolVar(&logOpts.json, "log.json", false, "format logs into JSON")
	flag.BoolVar(&logOpts.noLocal, "log.no-local", false, "inhibit stdio logging, only use any log hooks (syslog)")
	flag.StringVar(&logOpts.syslogRemote, "log.syslog.address", "", "host:port to send logs to via syslog")
	// We can add more variants, such as "rfcFOO", if needed:
	flag.StringVar(&logOpts.syslogProto, "log.syslog.proto", "udp", "protocol to use; [udp, tcp]")
	flag.StringVar(&logOpts.syslogTag, "log.syslog.tag", version.Program, "tag for syslog messages")
}

// Enabled is a predicate stating if logging is enabled.
// It has to be usable after flags init but _before_ implSetup, being called by
// Setup() in core.go to decide if we should be called.
func Enabled() bool {
	return !logOpts.noLocal || logOpts.syslogRemote != ""
}

// Used by Setup() to log which we are:
const implPackage = "logrus"

// -------------------------8< wrap logrus type >8-------------------------

type wrapLogrus struct {
	*logrus.Entry
}

// WithField adds a k/v pair to the accumulated logging details.
func (w wrapLogrus) WithField(key string, value interface{}) Logger {
	return wrapLogrus{w.Entry.WithField(key, value)}
}

// WithError adds an error to the accumulated logging details.
func (w wrapLogrus) WithError(err error) Logger {
	return wrapLogrus{w.Entry.WithError(err)}
}

// Debug logs at debug level.
// It maps our reduced signature to the fuller signature of logrus.
func (w wrapLogrus) Debug(message string) { w.Entry.Debug(message) }

// Info logs at info level.
// It maps our reduced signature to the fuller signature of logrus.
func (w wrapLogrus) Info(message string) { w.Entry.Info(message) }

// Warning logs at warning level.
// It maps our reduced signature to the fuller signature of logrus.
func (w wrapLogrus) Warning(message string) { w.Entry.Warning(message) }

// Error logs at error level.
// It maps our reduced signature to the fuller signature of logrus.
func (w wrapLogrus) Error(message string) { w.Entry.Error(message) }

// IsDisabled is always false for a logrus logger.
func (w wrapLogrus) IsDisabled() bool { return false }

// -------------------------8< wrap logrus type >8-------------------------

// implSetup sets up logging and is called by Setup (usually)
//
// Setup should be changed to add whatever remote logging you want;
// <https://github.com/sirupsen/logrus> lists a variety of supported hooks for
// remote logging, whether into corporate log services, cloud log services,
// chat services, email, error/exception aggregation services or whatever else.
//
// You can also use a remote service as the `.Out` field, if it's configured to
// provide an io.Writer interface instead of being set as a hook.
//
// Tune to taste in this file and it should just work.
//
// If logging can't be set-up, please assume that this is fatal and abort; we
// don't run without an audit trail going where it is supposed to go.
// If a network setup fails initial setup when called and returns an error,
// then so be it: we're a finger service, not critical plumbing infrastructure
// which must come up so that other things can come up.  Don't add complexity.
// (If it turns out that complexity is needed for one flaky setup, then and only
// then add it.)
//
// Recommend a sleep before Fatal so that if we keep dying, we don't die in a
// fast loop and chew system resources.
func implSetup() Logger {
	l := logrus.New()
	lvl, err := logrus.ParseLevel(logOpts.level)
	if err != nil {
		time.Sleep(time.Second)
		l.WithError(err).Fatal("unable to parse logging level")
	}
	l.SetLevel(lvl)

	// other plugins available include "logstash", in case that's of interest
	// in your environment.
	if logOpts.json {
		l.Formatter = &logrus.JSONFormatter{}
	}

	// nb: looks like logrus_syslog as a hook is not filtering out ANSI color
	// escape sequences.  So probably best to just use with JSON.  Or tell me
	// what I'm doing wrong with logging setup.
	if logOpts.syslogRemote != "" {
		switch strings.ToLower(logOpts.syslogProto) {
		case "tcp", "udp":
			logOpts.syslogProto = strings.ToLower(logOpts.syslogProto)
		default:
			time.Sleep(time.Second)
			l.WithField("protocol", logOpts.syslogProto).Fatal("unknown syslog protocol")
		}
		hook, err := logrus_syslog.NewSyslogHook(
			logOpts.syslogProto,
			logOpts.syslogRemote,
			syslog.LOG_DAEMON|syslog.LOG_INFO,
			logOpts.syslogTag)
		if err != nil {
			time.Sleep(time.Second)
			l.WithError(err).Fatal("unable to setup remote syslog")
		} else {
			l.Hooks.Add(hook)
		}
	}

	if logOpts.noLocal {
		f, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
		if err != nil {
			l.WithError(err).Error("unable to open system sink device")
		} else {
			l.Out = f
		}
	}

	stdlog.SetFlags(0)
	stdlog.SetOutput(l.WithField("via", "stdlog").Writer())
	return wrapLogrus{logrus.NewEntry(l)}
}
