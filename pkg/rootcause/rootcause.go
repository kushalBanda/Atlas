package rootcause

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"atlas/pkg/storage"
)

// Watcher drives the trace-close loop: two triggers converge on the same
// score-and-write step. Primary: root-span arrival (parent_span_id IS
// NULL) — the tree is structurally complete, since a root span's end_time
// can't be written until everything it waited on has finished, so a
// long-running child (e.g. a 60s LLM call) can never cause an early close
// via this path. Fallback: idle past closeTimeout with no root span yet —
// the crashed-hop case.
type Watcher struct {
	store        storage.Store
	closeTimeout time.Duration
	threshold    float64
}

// NewWatcher returns a Watcher scoring closed traces against threshold
// (fraction of trace duration, e.g. 0.30) and falling back to closeTimeout
// idle when no root span ever arrives.
func NewWatcher(store storage.Store, closeTimeout time.Duration, threshold float64) *Watcher {
	return &Watcher{store: store, closeTimeout: closeTimeout, threshold: threshold}
}

// Run ticks every tick until ctx is canceled, driving both close triggers
// each tick. Errors from a single tick are logged, not returned — a
// transient storage error on one tick must not stop the loop (the caller's
// supervise() wrapper is the last line of defense for a panic, not for
// ordinary tick errors).
func (w *Watcher) Run(ctx context.Context, tick time.Duration) error {
	ticker := time.NewTicker(tick)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := w.tickOnce(ctx); err != nil {
				slog.ErrorContext(ctx, "rootcause watcher tick failed", "error", err)
			}
		}
	}
}

func (w *Watcher) tickOnce(ctx context.Context) error {
	var errs []error

	rootArrived, err := w.store.ListRootArrivedTraces(ctx)
	if err != nil {
		errs = append(errs, fmt.Errorf("listing root-arrived traces: %w", err))
	}
	for _, traceID := range rootArrived {
		if err := w.closeAndScore(ctx, traceID); err != nil {
			errs = append(errs, err)
		}
	}

	stale, err := w.store.ListStaleOpenTraces(ctx, time.Now().UTC().Add(-w.closeTimeout))
	if err != nil {
		errs = append(errs, fmt.Errorf("listing stale open traces: %w", err))
	}
	for _, traceID := range stale {
		if err := w.closeAndScore(ctx, traceID); err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

// closeAndScore is shared by both close triggers: fetch the trace's spans,
// score them, and write the verdict via a targeted update.
func (w *Watcher) closeAndScore(ctx context.Context, traceID string) error {
	spans, err := w.store.GetTraceSpans(ctx, traceID)
	if err != nil {
		return fmt.Errorf("getting spans for trace %s: %w", traceID, err)
	}
	if len(spans) == 0 {
		return nil
	}

	verdict, err := Score(spans, w.threshold)
	if err != nil {
		return fmt.Errorf("scoring trace %s: %w", traceID, err)
	}

	if err := w.store.MarkTraceClosed(ctx, traceID, storage.CloseVerdict{
		SpanID:      verdict.SpanID,
		Reason:      verdict.Reason,
		SelfTimePct: verdict.SelfTimePct,
	}); err != nil {
		return fmt.Errorf("marking trace %s closed: %w", traceID, err)
	}
	return nil
}
