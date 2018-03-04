// Copyright © 2017,2018 Pennock Tech, LLC.
// All rights reserved, except as granted under license.
// Licensed per file LICENSE.txt

/*
Package version provides a home for metadata about the program, how it's
running, what it's name is, and so forth.  It should not depend upon any
non-stdlib packages, so that it's safe for import anywhere in the project.
*/
package version

import (
	"fmt"
	"io"
	"os"
	"runtime"
)

// Program is our idea of the persistent name of the program, for use where
// we don't want to use argv[0].  It's exported so that this doesn't have
// to be repeated.  Customize this if copying this code for use elsewhere!
const Program = "dummyapp"

// VersionString is potentially set at link-time.
// If you see x.y.z output from the version command then that should mean
// 'x.y.z'; if you see 'x.y.z-barebuild' then it's a hint that the version
// number is merely whatever was in source, not authoritatively stamped in
// later; this could thus be a build from any of a range of commits.
var versionString = "0.0.4-barebuild"

// BuildTime is potentially set at link-time.
// It should be used to record the timestamp (date+time) of the build.
var BuildTime string

const envLocation = "LOCATION"

// CurrentVersion provides the single-string derivation of what we consider
// our current version to be.  It might be derived from multiple pieces of
// information, whatever we think makes sense.  It probably fits on one line.
func CurrentVersion() string {
	return versionString
}

// PrintTo is used to write multiple whole lines of version information to
// an io.Writer.  Contrast with LogPairs.
func PrintTo(w io.Writer) {
	fmt.Fprintf(w, "%s: Version %s\n", Program, CurrentVersion())
	fmt.Fprintf(w, "%s: Golang: Runtime: %s\n", Program, runtime.Version())
	if BuildTime != "" {
		fmt.Fprintf(w, "%s: Build-time: %s\n", Program, BuildTime)
	}
	if t := os.Getenv(envLocation); t != "" {
		fmt.Fprintf(w, "%s: current location (per Environ): %q\n", Program, t)
	}
}

// A LogPair is a simple Key,Value pair of strings, suitable for using in
// structured logging as one item.
type LogPair struct {
	Key, Value string
}

// LogPairs returns a slice of items suitable for logging.  It's roughly
// equivalent to PrintTo, but it's the caller's responsibility to iterate
// the slice and add each pair to the logging call.
func LogPairs() []LogPair {
	pairs := make([]LogPair, 0, 4)
	pairs = append(pairs, LogPair{"version", CurrentVersion()})
	pairs = append(pairs, LogPair{"golang_runtime", runtime.Version()})
	if BuildTime != "" {
		pairs = append(pairs, LogPair{"build_time", BuildTime})
	}
	if t := os.Getenv(envLocation); t != "" {
		pairs = append(pairs, LogPair{"location", t})
	}
	return pairs
}
