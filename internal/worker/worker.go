package worker

import (
	"context"
	"time"

	cfgpkg "github.com/eryalito/multi-wordpress-file-manager/pkg/config"
)

// WorkFunc is the function executed by the worker each cycle.
// It receives the current config snapshot at execution time.
type WorkFunc func(ctx context.Context, cfg *cfgpkg.Config) error

// Worker runs a function on a fixed interval and can be externally triggered.
// Triggers are coalesced so that at most one pending run is queued.
type Worker struct {
	fn       WorkFunc
	getCfg   func() *cfgpkg.Config
	interval time.Duration
	reqCh    chan struct{}
	logf     func(string, ...any)
}

// New creates a new Worker.
func New(fn WorkFunc, getCfg func() *cfgpkg.Config, interval time.Duration, logf func(string, ...any)) *Worker {
	if interval <= 0 {
		interval = 3 * time.Minute
	}
	if logf == nil {
		logf = func(string, ...any) {}
	}
	return &Worker{
		fn:       fn,
		getCfg:   getCfg,
		interval: interval,
		reqCh:    make(chan struct{}, 1),
		logf:     logf,
	}
}

// Start runs the worker loop until ctx is canceled.
func (w *Worker) Start(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	// initial run
	w.tryTrigger()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.tryTrigger()
		case <-w.reqCh:
			// Execute the work synchronously; additional triggers are coalesced
			// because reqCh is size 1.
			if err := w.fn(ctx, w.getCfg()); err != nil {
				w.logf("worker run error: %v", err)
			}
		}
	}
}

// Trigger requests an immediate run (coalesced).
func (w *Worker) Trigger() { w.tryTrigger() }

func (w *Worker) tryTrigger() {
	select {
	case w.reqCh <- struct{}{}:
	default:
		// already a run pending or in progress; coalesce
	}
}
