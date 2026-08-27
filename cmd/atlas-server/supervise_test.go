package main

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestSupervise_RecoversPanicAndRestarts(t *testing.T) {
	t.Parallel()
	var calls atomic.Int32

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		supervise(ctx, "panics-once", func(_ context.Context) error {
			n := calls.Add(1)
			if n == 1 {
				panic("boom")
			}
			cancel()
			return nil
		})
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("supervise did not return after cancel")
	}

	if calls.Load() < 2 {
		t.Fatalf("expected fn to run at least twice (panic then restart), got %d", calls.Load())
	}
}

func TestSupervise_StopsOnContextCancel(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		supervise(ctx, "immediate-cancel", func(_ context.Context) error {
			return errors.New("transient failure")
		})
		close(done)
	}()

	cancel()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("supervise did not stop after context cancel")
	}
}
