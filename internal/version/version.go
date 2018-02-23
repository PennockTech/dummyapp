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

// VersionString is overridden at link-time, if using the Makefile.
// If you see x.y.z output from the version command then that should mean
// 'x.y.z'; if you see 'x.y.z-barebuild' then it's a hint that the version
// number is merely whatever was in source, not authoritatively stamped in
// later; this could thus be a build from any of a range of commits.
var VersionString = "0.0.2-barebuild"

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
