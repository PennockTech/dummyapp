// Copyright © 2017,2018 Pennock Tech, LLC.
// All rights reserved, except as granted under license.
// Licensed per file LICENSE.txt

// +build !heroku

package stats

import (
	"go.pennock.tech/dummyapp/internal/logging"
)

func herokuStart(logger logging.Logger) (func(), error) {
	logger.Info("built without Heroku stats support")
	return nil, nil
}
