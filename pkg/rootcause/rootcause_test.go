package rootcause

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"atlas/pkg/storage"
)

// fakeStore is a minimal storage.Store stub giving each test full control
// over what the two close triggers see, isolating Watcher's orchestration
// logic from the real DuckDB close-trigger semantics (covered separately
// in pkg/storage/duckdb_test.go).
type fakeStore struct {
	storage.Store
	rootArrived    []string
	staleOpen      []string
	spansByTrace   map[string][]storage.Span
	closedVerdicts map[string]storage.CloseVerdict
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		spansByTrace:   make(map[string][]storage.Span),
		closedVerdicts: make(map[string]storage.CloseVerdict),
	}
}

func (f *fakeStore) ListRootArrivedTraces(_ context.Context) ([]string, error) {
	return f.rootArrived, nil
}

func (f *fakeStore) ListStaleOpenTraces(_ context.Context, _ time.Time) ([]string, error) {
	return f.staleOpen, nil
}

func (f *fakeStore) GetTraceSpans(_ context.Context, traceID string) ([]storage.Span, error) {
	return f.spansByTrace[traceID], nil
}

func (f *fakeStore) MarkTraceClosed(_ context.Context, traceID string, verdict storage.CloseVerdict) error {
	f.closedVerdicts[traceID] = verdict
	return nil
}

func TestWatcher_ClosesOnRootSpanArrival(t *testing.T) {
	t.Parallel()
	store := newFakeStore()
	store.rootArrived = []string{"t1"}
	store.spansByTrace["t1"] = []storage.Span{
		span("root", "", 0, 100, "error"),
	}

	w := NewWatcher(store, 30*time.Second, 0.30)
	require.NoError(t, w.tickOnce(context.Background()))

	verdict, closed := store.closedVerdicts["t1"]
	require.True(t, closed)
	require.Equal(t, "root", verdict.SpanID)
}

func TestWatcher_ClosesStaleTraceAfterTimeout_NoRootSpan(t *testing.T) {
	t.Parallel()
	store := newFakeStore()
	store.staleOpen = []string{"t2"}
	store.spansByTrace["t2"] = []storage.Span{
		span("orphan-child", "missing-parent", 0, 10, "ok"),
	}

	w := NewWatcher(store, 30*time.Second, 0.30)
	require.NoError(t, w.tickOnce(context.Background()))

	_, closed := store.closedVerdicts["t2"]
	require.True(t, closed)
}

func TestWatcher_LongRunningChildSpan_DoesNotCloseTraceEarly(t *testing.T) {
	t.Parallel()
	// Regression guard: a trace with an in-flight long-running child (e.g.
	// a 60s LLM call) and no root span yet must not be reported by either
	// trigger — that's storage's job (see
	// TestListStaleOpenTraces_ExcludesRecentlyActive and
	// TestListRootArrivedTraces_OnlyReturnsTracesWithRootSpan in
	// pkg/storage/duckdb_test.go). Here we confirm Watcher takes no close
	// action when the store correctly reports nothing to close.
	store := newFakeStore() // rootArrived and staleOpen both empty
	store.spansByTrace["t3"] = []storage.Span{
		span("root", "", 0, 100, "ok"),
	}

	w := NewWatcher(store, 30*time.Second, 0.30)
	require.NoError(t, w.tickOnce(context.Background()))

	require.Empty(t, store.closedVerdicts)
}

func TestWatcher_EmptySpans_SkipsWithoutError(t *testing.T) {
	t.Parallel()
	store := newFakeStore()
	store.rootArrived = []string{"t-no-spans"}
	// spansByTrace intentionally has no entry for "t-no-spans"

	w := NewWatcher(store, 30*time.Second, 0.30)
	require.NoError(t, w.tickOnce(context.Background()))
	require.Empty(t, store.closedVerdicts)
}
