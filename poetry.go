// Copyright © 2018 Pennock Tech, LLC.
// All rights reserved, except as granted under license.
// Licensed per file LICENSE.txt

package main

import (
	"errors"
	"flag"
	"io"
	"net/http"
	"os"
	"strings"
	"syscall"

	"go.pennock.tech/dummyapp/internal/logging"
)

// envPoetryDir is the name of the environment variable which provides a
// default location for looking for poetry; there's no point making it
// exported, since we reference it at init time.  defaultPoetryDir is the
// default in the absence of the environment variable, similarly referenced
// at init time.
const (
	envPoetryDir     = "POETRY_DIR"
	defaultPoetryDir = "poetry"

	// oops_REGISTER_POEM_POETRY is a hack, so that I can leave in the code
	// showing the structure of what I was trying to do, far too late at night,
	// but disable it at compile time.  I should revisit this and clean it up.
	//
	// This is not golint-approved naming, but what I'm doing here is wrong and
	// I want it to stand out as compile-time disabling of code.
	oops_REGISTER_POEM_POETRY = false
)

var (
	// ErrNoPoemGiven indicates poetry handler not given a poem name when one
	// was expected.
	ErrNoPoemGiven = errors.New("poetry: no poem name given")
)

var poetryOptions struct {
	dir string
}

func init() {
	defaultDir := os.Getenv(envPoetryDir)
	var suffix string
	if defaultDir == "" {
		defaultDir = defaultPoetryDir
	} else {
		suffix = " (environ overrides '" + defaultPoetryDir + "')"
	}
	flag.StringVar(&poetryOptions.dir, "poetry.dir", defaultDir, "poetry serving directory"+suffix)
}

type poetryDir string

func (dir poetryDir) Open(name string) (http.File, error) {
	// We can't use loggerFromContext because we don't get the req,
	// only a literal filename.
	//
	// We register with StripPrefix, so that a leading /poem is removed; after which,
	// we're called with `/` for the root page, and if we return a directory,
	// then called again with `/index.html`.
	// Note that ${poetry}/ might contain sub-directories with dot files.
	// Thus rather than look for a leading dot, we just look for /. to prohibit.
	//
	// However, the indexing is not solved.  I can't be bothered to figure out
	// correct portable (across path-sep variances) changes in safely joining
	// the path to the FS root, replicating the unexported functionality of the
	// stdlib, and this was supposed to be a small demo.  So for now, rather
	// than just handle correctness at the top-level and not for sub-dirs,
	// we'll just blindly accept that we're returning plainly formatted lists
	// which contains entries which might be rejected.

	if oops_REGISTER_POEM_POETRY {
		// We return our own index, at /poetry/, to handle stripping out leading-dots.
		// So error if someone tries to access the FS root.
		if name == "" || name == "/" {
			return nil, ErrNoPoemGiven
		}
	}
	if strings.Contains(name, "/.") {
		return nil, os.ErrPermission
	}
	return http.Dir(dir).Open(name)
}

func poetryHandleFunc(w http.ResponseWriter, req *http.Request) {
	io.WriteString(w, "<html><head><title>Dummy App: Poetry</title></head><body><h1>Poetry</h1>\n")
	io.WriteString(w, "</body></html>\n")
}

// setupPoetry should be called by the main go-routine after options have been
// parsed.  We init various data-structions based upon the values of the options.
func setupPoetry(logger logging.Logger) bool {
	err := setupPoetryNolog()
	if logger == nil {
		if err != nil {
			return false
		}
		return true
	}
	logger = logger.WithField("directory", poetryOptions.dir)
	if err != nil {
		logger.WithError(err).Warning("skipping poetry setup")
		return false
	}
	logger.Info("serving poetry")
	return true
}

func setupPoetryNolog() error {
	fi, err := os.Stat(poetryOptions.dir)
	if err != nil {
		return err
	}
	if !fi.IsDir() {
		return syscall.ENOTDIR
	}

	if oops_REGISTER_POEM_POETRY {
		addFirstLevelPageItem(dummyAppFirstLevelPage{
			name:      "poem/",
			handler:   http.StripPrefix("/poem", http.FileServer(poetryDir(poetryOptions.dir))),
			skipIndex: true,
		})

		addFirstLevelPageItem(dummyAppFirstLevelPage{
			name:     "poetry",
			function: poetryHandleFunc,
		})
	} else {
		addFirstLevelPageItem(dummyAppFirstLevelPage{
			name:    "poetry/",
			handler: http.StripPrefix("/poetry", http.FileServer(poetryDir(poetryOptions.dir))),
		})
	}
	return nil
}
