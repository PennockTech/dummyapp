// Copyright © 2018 Pennock Tech, LLC.
// All rights reserved, except as granted under license.
// Licensed per file LICENSE.txt

// This is a mostly-lightweight webserver as a dummy app; it imports an
// external package or two mostly to have an excuse to use dep, for the
// dummy demonstration.
//
// Real code might use better structured work, or even a real middleware
// setup.  This is in one file.  It could be made much simpler, but I
// wanted proper logging of startup, respawn loop delay protection,
// and something which can serve from a filesystem area, to highlight
// other parts of the build system.

package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	"go.pennock.tech/dummyapp/internal/logging"
	"go.pennock.tech/dummyapp/internal/stats"
	"go.pennock.tech/dummyapp/internal/version"
)

const defaultPortSpec = ":8080"

var options struct {
	portspec    string
	showVersion bool
}

func init() {
	flag.StringVar(&options.portspec, "port", defaultPortSpec, "port to listen on for HTTP requests")
	flag.BoolVar(&options.showVersion, "version", false, "show version and exit")
}

type dummyAppFirstLevelPage struct {
	name         string
	function     http.HandlerFunc
	handler      http.Handler
	skipIndex    bool
	skipRegister bool
	onlyExistIf  func(logger logging.Logger) bool
}

var firstLevelPages map[string]dummyAppFirstLevelPage

func commonAddFirstLevelPage(name string) {
	if firstLevelPages == nil {
		firstLevelPages = make(map[string]dummyAppFirstLevelPage, 10)
	}
	if _, ok := firstLevelPages[name]; ok {
		time.Sleep(time.Second)
		panic("duplicate page '" + name + "' registered")
	}
}

// can have a Handler variant too, I'm just dealing only in Funcs for this dummy app
func addFirstLevelPageFunc(name string, f http.HandlerFunc) {
	commonAddFirstLevelPage(name)
	firstLevelPages[name] = dummyAppFirstLevelPage{name: name, function: f}
}

func addUnindexedFirstLevelPageFunc(name string, f http.HandlerFunc) {
	commonAddFirstLevelPage(name)
	firstLevelPages[name] = dummyAppFirstLevelPage{name: name, function: f, skipIndex: true}
}

func addFirstLevelPageItem(item dummyAppFirstLevelPage) {
	commonAddFirstLevelPage(item.name)
	firstLevelPages[item.name] = item
}

type dummyappReqContextKey int

const (
	dummyappLoggerKey dummyappReqContextKey = iota
)

func loggerFromContext(ctx context.Context) logging.Logger {
	l := ctx.Value(dummyappLoggerKey)
	if l == nil {
		return nil
	}
	return l.(logging.Logger)
}

var lastRequestID uint64 // atomic bump; note this is not great for clustered operations

// LogWrapHandler adds logging to received HTTP requests, logging before and after the
// handling and providing the logger in the context to requests.
func LogWrapHandler(h http.Handler, logger logging.Logger, name string) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		requestID := atomic.AddUint64(&lastRequestID, 1)
		logger = logger.WithField("request", requestID).WithField("page", name)
		logger.
			WithField("method", req.Method).
			WithField("url", req.URL).
			WithField("host", req.Host).
			WithField("remote", req.RemoteAddr).
			Info("received") // can decorate with body size, etc etc
		defer logger.Info("responded")
		h.ServeHTTP(w, req.WithContext(context.WithValue(req.Context(), dummyappLoggerKey, logger)))
	}
}

func send404(w http.ResponseWriter, req *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	w.Header().Set("Content-Type", "text/plain")
	io.WriteString(w, "page not found")
	l := loggerFromContext(req.Context())
	if l != nil {
		l.WithField("URL", req.URL).Info("sent 404")
	}
}

func rootHandle(w http.ResponseWriter, req *http.Request) {
	// All paths for valid sub-trees must have been explicitly registered
	if req.URL.Path != "/" {
		send404(w, req)
		return
	}

	type page struct {
		name string
		hf   http.HandlerFunc
	}

	pageNames := make([]string, 0, len(firstLevelPages))
	for k := range firstLevelPages {
		if firstLevelPages[k].skipIndex {
			continue
		}
		if firstLevelPages[k].onlyExistIf != nil && !firstLevelPages[k].onlyExistIf(nil) {
			continue
		}
		n := strings.Replace(strings.TrimRight(firstLevelPages[k].name, "/"), "/", " ", -1)
		pageNames = append(pageNames, n)
	}
	sort.Strings(pageNames)

	io.WriteString(w, "<html><head><title>Dummy App</title></head><body><h1>Dummy App</h1>\n<ul>\n")
	for _, n := range pageNames {
		fmt.Fprintf(w, " <li><a href=\"%s\">%s</a></li>\n", n, n)
	}
	io.WriteString(w, "</ul>\n</body></html>\n")
}

func init() { addUnindexedFirstLevelPageFunc("favicon.ico", send404) }

func parseFlagsSanely() {
	envPort := os.Getenv("PORT")
	if envPort != "" {
		options.portspec = envPort
	}
	flag.Parse()
	if options.portspec == "" {
		options.portspec = defaultPortSpec
	}
	if !strings.Contains(options.portspec, ":") {
		options.portspec = ":" + options.portspec
	}
}

func registerHandlersOnDefault(logger logging.Logger) {
	for i := range firstLevelPages {
		if firstLevelPages[i].skipRegister {
			continue
		}
		if firstLevelPages[i].onlyExistIf != nil && !firstLevelPages[i].onlyExistIf(logger) {
			continue
		}
		n := firstLevelPages[i].name
		f := firstLevelPages[i].function
		h := firstLevelPages[i].handler
		if f != nil && h != nil {
			panic("given both function and handler for http setup of " + n)
		}
		if f == nil && h == nil {
			panic("missing both function and handler for http setup of " + n)
		}
		if h == nil {
			h = http.HandlerFunc(f)
		}
		if logger != nil {
			h = LogWrapHandler(h, logger, n)
			logger.Debugf("registering handler for page /%s", n)
		}
		http.Handle("/"+n, h)
	}
	h := http.HandlerFunc(rootHandle)
	if logger != nil {
		h = LogWrapHandler(h, logger, "/")
	}
	http.Handle("/", h)
}

func setupWebserver(logger logging.Logger) func() error {
	registerHandlersOnDefault(logger)

	server := &http.Server{
		Addr: options.portspec,
	}
	listener, err := net.Listen("tcp", server.Addr)
	if err != nil {
		if logger != nil {
			logger.
				WithField("listen", server.Addr).
				WithError(err).
				Error("listening failed")
		} else {
			fmt.Fprintf(os.Stderr, "%s\n", err)
		}
		return nil
	}

	return func() error {
		if logger != nil {
			logger.
				WithField("listen", server.Addr).
				WithField("bound", listener.Addr().String()).
				Info("accepting connections")
		}
		return server.Serve(listener)
	}
}

func realMain() int {
	parseFlagsSanely()

	if options.showVersion {
		version.Version(os.Stdout)
		// skip the safety checks on rapid respawning
		os.Exit(0)
	}

	if !logging.Enabled() {
		statsManager, err := stats.Start(nil)
		if err != nil {
			defer statsManager.Stop()
		}
		err = setupWebserver(nil)()
		if err != nil {
			fmt.Fprintf(os.Stderr, "webserver failed: %s\n", err)
			return 1
		}
		return 0
	}

	logger := logging.Setup()
	masterThreadLogger := logger.
		WithField("uid", os.Getuid()).
		WithField("gid", os.Getgid()).
		WithField("pid", os.Getpid())

	startupLogCtx := masterThreadLogger.WithField("version", version.CurrentVersion())
	if os.Getenv(version.ENV_LOCATION) != "" {
		startupLogCtx = startupLogCtx.WithField("location", os.Getenv(version.ENV_LOCATION))
	}
	startupLogCtx.Info("starting")

	statsManager, err := stats.Start(logger.WithField("component", "stats"))
	if err != nil {
		logger.WithError(err).Error("failed to start stats manager")
	}
	defer statsManager.Stop()

	_ = setupPoetry(logger) // we don't care if it succeeds or not, let it log

	serve := setupWebserver(logger)
	if serve == nil {
		return 1
	}

	// Maybe show using waitgroups and shutdown channels, but that's not really
	// the point of this demo and I've already done too much on this side,
	// complicating things.
	//
	// Note that we allow the stupidity of running without logging, and short-circuit
	// that above for convenience.  So rework that if expanding this.

	err = serve()
	if err != nil {
		masterThreadLogger.Errorf("web server exited with error: %s", err)
		return 1
	}
	return 0
}

func main() {
	// Avoid busy-loop respawning if there's a fatal error on startup
	// Won't handle panic not guarded by sleep
	start := time.Now()
	rv := realMain()
	duration := time.Now().Sub(start)
	if duration < 3*time.Second {
		time.Sleep(2 * time.Second)
	}
	os.Exit(rv)
}
