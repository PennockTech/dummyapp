// Copyright © 2017,2018 Pennock Tech, LLC.
// All rights reserved, except as granted under license.
// Licensed per file LICENSE.txt

// +build heroku

package stats

import (
	"go.pennock.tech/hmetrics"

	"go.pennock.tech/dummyapp/internal/logging"
)

// This posts metrics to Heroku, per
// <https://devcenter.heroku.com/articles/language-runtime-metrics-go>
//
// The metrics are posted to whatever URL is in HEROKU_METRICS_URL in environ.

func herokuStart(logger logging.Logger) (func(), error) {
	msg, cancel, err := hmetrics.Spawn(func(err error) {
		logger.WithError(err).Error("heroku error callback")
	})
	if err != nil {
		logger.WithError(err).Error(msg)
		return nil, err
	}
	logger.Info(msg)
	return cancel, nil
}
