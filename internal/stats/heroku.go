// Copyright © 2017,2018 Pennock Tech, LLC.
// All rights reserved, except as granted under license.
// Licensed per file LICENSE.txt

// +build heroku

package stats

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/heroku/x/hmetrics"

	"go.pennock.tech/dummyapp/internal/logging"
)

// This posts metrics to Heroku, per
// <https://devcenter.heroku.com/articles/language-runtime-metrics-go>
//
// The metrics are posted to whatever URL we provide; Heroku provide the URL in
// HEROKU_METRICS_URL in environ and capture that value in
// hmetrics.DefaultEndpoint at init() time.
//
// This code is a blend of the advanced example and what their own onload
// handler does, plus our logging error handler thrown in.  Plus we want to
// avoid the loop trying to start on a supposedly fatal error if just not
// enabled, so we check that too.  Why do we retry "fatal"?  Because they don't
// document all the causes of Fatal and have already shown a willingness to
// make breaking API changes, so we can't assume that it's just the variable
// being unset.  Meanwhile I'm not killing our process because we can't export
// improved visibility metrics.  So, we just deem "fatal" to be "jump
// immediately to maximum backoff before trying again" instead of using
// exponential backoff.

const (
	maxFailureBackoff        = 10 * time.Minute
	resetFailureBackoffAfter = 5 * time.Minute
	resetFailureBackoffTo    = time.Second
)

// ErrHerokuMetricsNotEnabled just means "don't worry about it" and is non-fatal.
var ErrHerokuMetricsNotEnabled = errors.New("heroku metrics not enabled ('$HEROKU_METRICS_URL' empty?)")

func herokuStart(logger logging.Logger) (func(), error) {
	// hmetrics.Report no longer knows about the environment variable trigger,
	// that's a package variable which the onload package references, and we do
	// too, where the package init grabs an environment variable to get the
	// value.  We don't want to do any of this if the variable is unset and
	// should exit more cleanly if so.  Which the onload package no longer does.
	if hmetrics.DefaultEndpoint == "" {
		return nil, ErrHerokuMetricsNotEnabled
	}

	ctx, cancel := context.WithCancel(context.Background())

	// See above re our fatal handling.
	type fataler interface {
		Fatal() bool
	}

	errHandler := func(err error) error {
		logger.WithError(err).Error("heroku error callback")
		return nil // returning non-nil terminates Heroku error logging
	}

	// WARNING WARNING: commit aab18502 2017-10-04 of github.com/heroku/x
	// changed the hmetrics package fundamentally, such that `Report()` changed
	// from "do sanity checks, return err if need be, else spawn a go-routine
	// and return nil" to "do all the work in this go-routine, such that we
	// become blocking".  They also changed the function signature, but that's
	// a lesser issue.

	go func() {
		spawnerLogger := logger.WithField("where", "spawner")
		var backoff time.Duration

		for backoff = resetFailureBackoffTo; ; backoff *= 2 {
			if isDeadContext(ctx) {
				spawnerLogger.
					WithField("context_done_reason", ctx.Err()).
					Info("context cancelled, exiting")
				return
			}

			startLatest := time.Now()
			err := hmetrics.Report(ctx, hmetrics.DefaultEndpoint, errHandler)
			duration := time.Since(startLatest)
			durMs := fmt.Sprintf("%.2f", float64(duration)/float64(time.Microsecond))

			if duration >= resetFailureBackoffAfter {
				backoff = resetFailureBackoffTo
			}
			if backoff > maxFailureBackoff {
				backoff = maxFailureBackoff
			}

			thisLog := spawnerLogger.WithField("duration_us", durMs)
			if err != nil {
				thisLog = thisLog.WithError(err)
			}

			messageBase := "heroku metrics Report exited"
			messageState := ""
			messageAgain := "backing off before trying again"
			dead := false

			if f, ok := err.(fataler); ok && f.Fatal() {
				backoff = maxFailureBackoff
				messageState = " fatally"
				messageAgain = "(we'll try again much later)"
				thisLog = thisLog.WithField("reporter_fatal", true)
			}
			if isDeadContext(ctx) {
				dead = true
				messageAgain = "context cancelled, exiting"
				thisLog = thisLog.WithField("context_done_reason", ctx.Err())
			}

			message := messageBase + messageState + "; " + messageAgain
			if dead {
				thisLog.Error(message)
				return
			}
			thisLog.WithField("sleep_s", int64(backoff/time.Second)).Error(message)

			timer := time.NewTimer(backoff)
			select {
			case <-ctx.Done():
				spawnerLogger.
					WithField("context_done_reason", ctx.Err()).
					Info("context cancelled while in delay backoff, exiting")
				// nb: Leaks the channel, unless raced and already exited.
				if !timer.Stop() {
					<-timer.C
				}
				return
			case <-timer.C:
			}
		}
	}()

	logger.Info("heroku metrics exporter keepalive-loop started")

	return cancel, nil
}

func isDeadContext(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
		return false
	}
}
