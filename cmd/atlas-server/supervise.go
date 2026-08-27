package main

import (
	"context"
	"errors"
	"log/slog"
	"time"
)

// maxBackoff caps the delay between restarts of a supervised loop.
const maxBackoff = 30 * time.Second

// supervise runs fn in a loop: recovers a panic, logs it, and restarts
// after a backoff (capped at maxBackoff, resetting after a clean run).
// Applies to rootcause.Watcher.Run and (later) discovery.RunAll so a panic
// in one background subsystem doesn't silently freeze it while ingest and
// query keep serving stale state.
func supervise(ctx context.Context, name string, fn func(ctx context.Context) error) {
	backoff := time.Second

	for {
		if ctx.Err() != nil {
			return
		}

		err := runOnce(ctx, name, fn)

		if ctx.Err() != nil {
			return
		}
		if err == nil {
			backoff = time.Second
			continue
		}

		slog.ErrorContext(ctx, "supervised loop exited, restarting", "loop", name, "error", err, "backoff", backoff)
		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}
		if backoff *= 2; backoff > maxBackoff {
			backoff = maxBackoff
		}
	}
}

func runOnce(ctx context.Context, name string, fn func(ctx context.Context) error) (err error) {
	defer func() {
		if p := recover(); p != nil {
			slog.ErrorContext(ctx, "supervised loop panicked", "loop", name, "panic", p)
			err = errors.New("panic recovered in supervised loop")
		}
	}()
	return fn(ctx)
}
