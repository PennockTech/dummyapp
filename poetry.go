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
	"sync/atomic"

	"go.pennock.tech/dummyapp/internal/logging"
)

const ENV_POETRY_DIR = "POETRY_DIR"
const DEFAULT_POETRY_DIR = "poetry"

const REGISTER_POEM_POETRY = false

var (
	// ErrNoPoemGiven indicates poetry handler not given a poem name when one
	// was expected.
	ErrNoPoemGiven = errors.New("poetry: no poem name given")
)

var poetryOptions struct {
	dir string
}

func init() {
	defaultDir := os.Getenv(ENV_POETRY_DIR)
	var suffix string
	if defaultDir == "" {
		defaultDir = DEFAULT_POETRY_DIR
	} else {
		suffix = " (environ overrides '" + DEFAULT_POETRY_DIR + "')"
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

	if REGISTER_POEM_POETRY {
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

var poetryComplainButOnce uint32

func init() {
	guardExistence := func(logger logging.Logger) bool {
		// A scenario where nil logger makes sense, given that we're also
		// called at per-request time for root page.  Plus we want to
		// inhibit double logging, since called for two handlers.
		// Don't want sync.Once, we take a parameter (beginning to regret
		// passing the logging in, but we can't just return an error and
		// get JSON fields populated with the poetry dir.
		// I need to rethink this, missing something obvious if ending
		// up this clumsy.
		if logger != nil {
			if atomic.AddUint32(&poetryComplainButOnce, 1) != 1 {
				logger = nil
			} else {
				logger = logger.WithField("directory", poetryOptions.dir)
			}
		}
		fi, err := os.Stat(poetryOptions.dir)
		if err != nil {
			if logger != nil {
				logger.WithError(err).Warning("skipping poetry setup")
			}
			return false
		}
		if !fi.IsDir() {
			if logger != nil {
				logger.Warning("not a directory, skipping poetry setup")
			}
			return false
		}
		if logger != nil {
			logger.Info("serving poetry")
		}
		return true
	}

	if REGISTER_POEM_POETRY {
		addFirstLevelPageItem(dummyAppFirstLevelPage{
			name:        "poem/",
			handler:     http.StripPrefix("/poem", http.FileServer(poetryDir(poetryOptions.dir))),
			skipIndex:   true,
			onlyExistIf: guardExistence,
		})

		addFirstLevelPageItem(dummyAppFirstLevelPage{
			name:        "poetry",
			function:    poetryHandleFunc,
			onlyExistIf: guardExistence,
		})
	} else {
		addFirstLevelPageItem(dummyAppFirstLevelPage{
			name:        "poetry/",
			handler:     http.StripPrefix("/poetry", http.FileServer(poetryDir(poetryOptions.dir))),
			onlyExistIf: guardExistence,
		})
	}
}
