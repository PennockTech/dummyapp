package stats

import (
	"go.pennock.tech/dummyapp/internal/logging"
)

type Handle struct {
	cancelHeroku func()

	// We should have other options in here, including the classic Go expvar
	// handling
}

func Start(logger logging.Logger) (*Handle, error) {
	cancel, err := herokuStart(logger)
	if err != nil {
		return nil, err
	}

	handle := &Handle{
		cancelHeroku: cancel,
	}

	return handle, nil
}

func (h *Handle) Stop() {
	if h == nil {
		return
	}

	if h.cancelHeroku != nil {
		h.cancelHeroku()
	}
}
