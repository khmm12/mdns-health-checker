package worker

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/khmm12/mdns-health-checker/internal/common/logging"
	"github.com/khmm12/mdns-health-checker/internal/common/tracing"
)

type Task interface {
	Execute(ctx context.Context) error
}

type Worker struct {
	logger *slog.Logger

	interval time.Duration
	task     Task

	ctx    context.Context
	cancel context.CancelFunc

	mu sync.Mutex
}

func NewWorker(logger *slog.Logger, interval time.Duration, task Task) *Worker {
	return &Worker{
		logger:   logger,
		interval: interval,
		task:     task,
	}
}

func (w *Worker) Start() error {
	locked := w.mu.TryLock()
	if !locked {
		return fmt.Errorf("worker is already running")
	}

	defer w.mu.Unlock()

	w.ctx, w.cancel = context.WithCancel(context.Background())
	defer w.cancel()

	ticker := newTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-w.ctx.Done():
			return nil
		case <-ticker.C:
			err := w.run(w.ctx)
			if err != nil && !errors.Is(err, context.Canceled) {
				w.logger.ErrorContext(w.ctx, "Failed to execute task", logging.Error(err))
			}
		}
	}
}

func (w *Worker) Shutdown(_ context.Context) error {
	if w.cancel != nil {
		w.cancel()
	}

	return nil
}

func (w *Worker) run(ctx context.Context) error {
	return w.task.Execute(tracing.WithTraceID(ctx))
}

func newTicker(repeat time.Duration) *time.Ticker {
	ticker := time.NewTicker(repeat)
	oc := ticker.C
	nc := make(chan time.Time, 1)
	go func() {
		nc <- time.Now()
		for tm := range oc {
			nc <- tm
		}
	}()
	ticker.C = nc
	return ticker
}
