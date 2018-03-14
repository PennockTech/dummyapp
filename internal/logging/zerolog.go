// Copyright © 2018 Pennock Tech, LLC.
// All rights reserved, except as granted under license.
// Licensed per file LICENSE.txt

// +build zerolog,!logrus

package logging

import (
	"flag"
	"fmt"
	"io"
	stdlog "log"
	"log/syslog"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"

	"go.pennock.tech/dummyapp/internal/version"
)

var logOpts struct {
	level        string
	json         bool
	syslogLocal  bool
	syslogRemote string
	syslogProto  string
	syslogTag    string
	noLocal      bool
}

var enabledAtomic uint32

func init() {
	flag.StringVar(&logOpts.level, "log.level", "info", "logging level")
	flag.BoolVar(&logOpts.json, "log.json", false, "format logs into JSON")
	flag.BoolVar(&logOpts.noLocal, "log.no-local", false, "inhibit stdio logging, only use any log hooks (syslog)")
	flag.BoolVar(&logOpts.syslogLocal, "log.syslog.local", false, "log to local syslog")
	flag.StringVar(&logOpts.syslogRemote, "log.syslog.address", "", "host:port to send logs to via syslog")
	// We can add more variants, such as "rfcFOO", if needed:
	flag.StringVar(&logOpts.syslogProto, "log.syslog.proto", "udp", "protocol to use; [udp, tcp]")
	flag.StringVar(&logOpts.syslogTag, "log.syslog.tag", version.Program, "tag for syslog messages")
}

// Enabled is a predicate stating if logging is enabled.
// It has to be usable after flags init but _before_ implSetup, being called by
// Setup() in core.go to decide if we should be called.
func Enabled() bool {
	return !logOpts.noLocal || logOpts.syslogLocal || logOpts.syslogRemote != ""
}

// ------------------------8< wrap zerolog type >8-------------------------

type wrapZerolog struct {
	zerolog.Logger
}

// These methods are not the most efficient for use in chaining; if we switch
// to zerolog then rework the internal logging API to have a lower impedance
// mismatch.

// WithField adds a k/v pair to the accumulated logging details.
func (w wrapZerolog) WithField(key string, value interface{}) Logger {
	return wrapZerolog{w.Logger.With().Interface(key, value).Logger()}
}

// WithError adds an error to the accumulated logging details.
func (w wrapZerolog) WithError(err error) Logger {
	return wrapZerolog{w.Logger.With().Err(err).Logger()}
}

func (w wrapZerolog) Debug(message string) {
	w.Logger.Debug().Msg(message)
}
func (w wrapZerolog) Info(message string) {
	w.Logger.Info().Msg(message)
}
func (w wrapZerolog) Warning(message string) {
	w.Logger.Warn().Msg(message)
}
func (w wrapZerolog) Error(message string) {
	w.Logger.Error().Msg(message)
}

// IsDisabled is always false for a zerolog logger.
// Note that our level can be set to disabled, but in that case we return a
// nilLogger.
func (w wrapZerolog) IsDisabled() bool { return false }

// ------------------------8< wrap zerolog type >8-------------------------

func ourFatalf(spec string, args ...interface{}) {
	time.Sleep(time.Second)
	fmt.Fprintf(os.Stderr, spec, args...)
	os.Exit(1)
}

// implSetup sets up logging and is called by Setup (usually).
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
	// divert anything using stdlib log to use a logger which notes this
	// FIXME: add this to other logger implementations too
	setupStdlogAndDone := func(l zerolog.Logger) Logger {
		stdlog.SetFlags(0)
		stdlog.SetOutput(l.With().Str("via", "stdlog").Logger())
		return wrapZerolog{l}
	}

	var lvl zerolog.Level
	switch strings.ToLower(logOpts.level) {
	case "debug":
		lvl = zerolog.DebugLevel
	case "info", "":
		lvl = zerolog.InfoLevel
	case "warn", "warning":
		lvl = zerolog.WarnLevel
	case "error", "err":
		lvl = zerolog.ErrorLevel
	case "fatal":
		lvl = zerolog.FatalLevel
	case "panic":
		lvl = zerolog.PanicLevel
	case "none", "disable", "disabled":
		lvl = zerolog.Disabled
	// zerolog.NoLevel has no applicability here
	default:
		ourFatalf("unable to parse logging level, %q unrecognized\n", logOpts.level)
	}
	if lvl == zerolog.Disabled {
		return newNilLoggerDisablingLog()
	}

	var expectedNormalOutput io.Writer = os.Stderr

	l := zerolog.New(expectedNormalOutput).With().Timestamp().Logger().Level(lvl)

	// compatibility with existing logs from logrus
	zerolog.MessageFieldName = "msg"
	// conversely, we should seriously consider whether or not to explicitly
	// set zerolog.TimeFieldFormat to the empty string and just use epochseconds

	// We currently log all durations as strings explicitly because of issues with
	// logrus handling.  This should ease removing that, if we cut-over:
	zerolog.DurationFieldUnit = time.Microsecond

	// We should consider testing if stdout is a tty and auto-setting JSON in
	// that case.  But for now, stick to existing behavior.
	// TODO: reconsider this.
	if !logOpts.json {
		// we make no attempt to match layout etc of logrus: if you use the
		// console logger, then you get what you get.  If consistency matters,
		// then you should be using JSON-stream logging.
		expectedNormalOutput = zerolog.ConsoleWriter{Out: os.Stderr}
		l = l.Output(expectedNormalOutput)
	}

	if logOpts.syslogLocal || logOpts.syslogRemote != "" {
		var (
			w       *syslog.Writer
			err     error
			failMsg string
		)
		if logOpts.syslogRemote != "" {
			failMsg = "unable to dial remote syslog"
			w, err = syslog.Dial(logOpts.syslogProto, logOpts.syslogRemote, syslog.LOG_INFO, logOpts.syslogTag)
		} else {
			failMsg = "unable to setup local syslog"
			w, err = syslog.New(syslog.LOG_INFO, logOpts.syslogTag)
		}
		if err != nil {
			l.Error().Err(err).Msg(failMsg)
			return setupStdlogAndDone(l)
		}
		if logOpts.noLocal {
			l = l.Output(zerolog.SyslogLevelWriter(w))
		} else {
			l = l.Output(zerolog.MultiLevelWriter(
				expectedNormalOutput,
				zerolog.SyslogLevelWriter(w),
			))
		}
	}
	return setupStdlogAndDone(l)
}
