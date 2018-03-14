package stats

import (
	"go.pennock.tech/dummyapp/internal/logging"
)

// Handle is our controller object for statistics/metrics
// logging/export/collection-for-something-to-retrieve.
type Handle struct {
	cancelHeroku func()

	// We should have other options in here, including the classic Go expvar
	// handling

	// For testing cancellation, I sometimes insert calls to cancel
	// temporarily, but don't want to mess with the deferred call or change it
	// to a closure, so just handle repeat calls.
	stopped bool
}

// Start begins handling statistics/metrics.
func Start(logger logging.Logger) (*Handle, error) {
	cancel, err := herokuStart(logger.WithField("stats", "heroku"))
	if err != nil {
		return nil, err
	}

	handle := &Handle{
		cancelHeroku: cancel,
	}

	return handle, nil
}

// Stop stops handling statistics/metrics; once stopped, the handle
// can't be restarted and you'll need to create a new one.
func (h *Handle) Stop() {
	if h == nil {
		return
	}
	if h.stopped {
		return
	}

	if h.cancelHeroku != nil {
		h.cancelHeroku()
	}

	h.stopped = true
}
