// Copyright © 2017,2018 Pennock Tech, LLC.
// All rights reserved, except as granted under license.
// Licensed per file LICENSE.txt

package version

import (
	"fmt"
	"io"
	"runtime"
)

const Program = "dummyapp"

// overridden at link-time, if using Makefile
var VersionString = "0.0.1"

// Pull the version derivation from whatever variables go into the makeup out
// into a function so that we can log it at startup.
func CurrentVersion() string {
	return VersionString
}

func Version(w io.Writer) {
	fmt.Fprintf(w, "%s: Version %s\n", Program, CurrentVersion())
	fmt.Fprintf(w, "%s: Golang: Runtime: %s\n", Program, runtime.Version())
}

const ENV_LOCATION = "LOCATION"
