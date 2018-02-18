// Copyright © 2017,2018 Pennock Tech, LLC.
// All rights reserved, except as granted under license.
// Licensed per file LICENSE.txt

// +build heroku

package stats

import (
	"context"

	"github.com/heroku/x/hmetrics"

	"go.pennock.tech/dummyapp/internal/logging"
)

func herokuStart(logger logging.Logger) (func(), error) {
	ctx, cancel := context.WithCancel(context.Background())

	var errHandler func(err error) error

	if logger != nil {
		errHandler = func(err error) error {
			logger.WithError(err).Error("heroku error callback")
			return nil // returning non-nil terminates Heroku error logging
		}
	} else {
		errHandler = func(_ error) error { return nil }
	}

	if err := hmetrics.Report(ctx, errHandler); err != nil {
		cancel()
		return nil, err
	}

	if logger != nil {
		logger.Info("heroku metrics exporter started")
	}

	return cancel, nil
}
