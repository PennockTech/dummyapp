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

	"github.com/felixge/httpsnoop"

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

// note that zerolog also has its own facility for registering with the context
// and TODO we should consider how to integrate that too, for other
// zerolog-using libs, without requiring extra functionality of other libs.
// Probably a new interface method with a default implementation?
func loggerFromContext(ctx context.Context) logging.Logger {
	l := ctx.Value(dummyappLoggerKey)
	if l == nil {
		return logging.NilLogger()
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
			WithField("url_path", req.URL.Path).
			WithField("url_query", req.URL.RawQuery).
			WithField("host", req.Host).
			WithField("remote", req.RemoteAddr).
			Info("received") // can decorate with body size, etc etc
		defer func() {
			if x := recover(); x != nil {
				logger.WithField("panic", x).Error("run-time panic")
			}
		}()
		req = req.WithContext(context.WithValue(req.Context(), dummyappLoggerKey, logger))
		m := httpsnoop.CaptureMetrics(h, w, req)
		// Writing non-JSON logs, `m.Duration` is string-formatted so we get
		// a pretty value with a suffix, probably µs.  With JSON, we just get
		// the integer value, which is in ns, and is not obviously so.
		// Coerce to get a string-of-floating-point.
		dur := fmt.Sprintf("%.2f", float64(m.Duration)/float64(time.Microsecond))
		logger.WithField("code", m.Code).WithField("duration_us", dur).WithField("length", m.Written).Info("responded")
	}
}

func send404(w http.ResponseWriter, req *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	w.Header().Set("Content-Type", "text/plain")
	io.WriteString(w, "page not found")
	l := loggerFromContext(req.Context())
	if !l.IsDisabled() {
		l.WithField("http_error", 404).WithField("URL", req.URL).Info("sent 404")
	}
}

func rootHandle(w http.ResponseWriter, req *http.Request) {
	// All paths for valid sub-trees must have been explicitly registered
	if req.URL.Path != "/" {
		send404(w, req)
		return
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
		// missing the IsDisabled is harmless aside from some extra cycles on each call
		if !logger.IsDisabled() {
			h = LogWrapHandler(h, logger, n)
			logger.WithField("page", "/"+n).Debug("registering page handler")
		}
		http.Handle("/"+n, h)
	}
	h := http.HandlerFunc(rootHandle)
	if !logger.IsDisabled() {
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
		if logger.IsDisabled() {
			fmt.Fprintf(os.Stderr, "%s\n", err)
		} else {
			logger.
				WithField("listen", server.Addr).
				WithError(err).
				Error("listening failed")
		}
		return nil
	}

	return func() error {
		logger.
			WithField("listen", server.Addr).
			WithField("bound", listener.Addr().String()).
			Info("accepting connections")
		return server.Serve(listener)
	}
}

func realMain() int {
	parseFlagsSanely()

	if options.showVersion {
		version.PrintTo(os.Stdout)
		// skip the safety checks on rapid respawning
		os.Exit(0)
	}

	logger := logging.Setup()
	masterThreadLogger := logger.
		WithField("uid", os.Getuid()).
		WithField("gid", os.Getgid()).
		WithField("pid", os.Getpid())

	startupLogCtx := masterThreadLogger
	for _, pair := range version.LogPairs() {
		startupLogCtx = startupLogCtx.WithField(pair.Key, pair.Value)
	}
	startupLogCtx.Info("starting")

	statsManager, err := stats.Start(logger.WithField("component", "stats"))
	if err != nil {
		logger.WithError(err).Error("failed to start stats manager")
	} else {
		defer statsManager.Stop()
	}

	_ = setupPoetry(logger) // we don't care if it succeeds or not, let it log

	serve := setupWebserver(logger)
	if serve == nil {
		return 1
	}

	demonstrateStdlibLogger()

	// Maybe show using waitgroups and shutdown channels, but that's not really
	// the point of this demo and I've already done too much on this side,
	// complicating things.
	//
	// Note that we allow the stupidity of running without logging, and short-circuit
	// that above for convenience.  So rework that if expanding this.

	err = serve()
	if err != nil {
		masterThreadLogger.WithError(err).Error("web server error exited")
		return 1
	}
	return 0
}

func main() {
	// Avoid busy-loop respawning if there's a fatal error on startup
	// Won't handle panic not guarded by sleep
	start := time.Now()
	rv := realMain()
	duration := time.Since(start)
	if duration < 3*time.Second {
		time.Sleep(2 * time.Second)
	}
	os.Exit(rv)
}
