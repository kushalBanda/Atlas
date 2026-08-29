package storage

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func newTestDB(t *testing.T) *DuckDB {
	t.Helper()
	db, err := NewDuckDB(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	return db
}

func TestWriteAndGetTraceSpans(t *testing.T) {
	t.Parallel()
	db := newTestDB(t)
	ctx := context.Background()

	start := time.Now().UTC().Truncate(time.Millisecond)
	spans := []Span{
		{
			TraceID: "t1", SpanID: "root", ParentSpanID: "",
			ServiceName: "svc-a", Name: "root-op",
			StartTime: start, EndTime: start.Add(100 * time.Millisecond),
			StatusCode: "ok",
			Attributes: map[string]any{"k": "v"},
		},
		{
			TraceID: "t1", SpanID: "child", ParentSpanID: "root",
			ServiceName: "svc-b", Name: "child-op",
			StartTime: start.Add(10 * time.Millisecond), EndTime: start.Add(50 * time.Millisecond),
			StatusCode: "error",
		},
	}

	require.NoError(t, db.WriteSpans(ctx, spans))

	got, err := db.GetTraceSpans(ctx, "t1")
	require.NoError(t, err)
	require.Len(t, got, 2)
	require.Equal(t, "root", got[0].SpanID)
	require.Equal(t, "", got[0].ParentSpanID)
	require.Equal(t, map[string]any{"k": "v"}, got[0].Attributes)
	require.Equal(t, "child", got[1].SpanID)
	require.Equal(t, "root", got[1].ParentSpanID)
	require.Equal(t, "error", got[1].StatusCode)
}

func TestWriteAndGetTraceSpans_LLMFieldsRoundTrip(t *testing.T) {
	t.Parallel()
	db := newTestDB(t)
	ctx := context.Background()
	start := time.Now().UTC().Truncate(time.Millisecond)

	model := "gpt-5.6"
	promptTokens := int64(12)
	completionTokens := int64(34)
	cost := 0.0056

	spans := []Span{
		{
			TraceID: "t1", SpanID: "llm-span", ServiceName: "svc-a", Name: "chat",
			StartTime: start, EndTime: start.Add(time.Second), StatusCode: "ok",
			LLMModel: &model, LLMPromptTokens: &promptTokens, LLMCompletionTokens: &completionTokens,
			LLMCost: &cost,
		},
		{
			TraceID: "t1", SpanID: "plain-span", ParentSpanID: "llm-span", ServiceName: "svc-a", Name: "op",
			StartTime: start, EndTime: start.Add(time.Second), StatusCode: "ok",
		},
	}
	require.NoError(t, db.WriteSpans(ctx, spans))

	got, err := db.GetTraceSpans(ctx, "t1")
	require.NoError(t, err)
	require.Len(t, got, 2)

	require.Equal(t, "llm-span", got[0].SpanID)
	require.Equal(t, &model, got[0].LLMModel)
	require.Equal(t, &promptTokens, got[0].LLMPromptTokens)
	require.Equal(t, &completionTokens, got[0].LLMCompletionTokens)
	require.Equal(t, &cost, got[0].LLMCost)

	require.Equal(t, "plain-span", got[1].SpanID)
	require.Nil(t, got[1].LLMModel)
	require.Nil(t, got[1].LLMPromptTokens)
	require.Nil(t, got[1].LLMCompletionTokens)
	require.Nil(t, got[1].LLMCost)
}

func TestWriteAndGetTraceSpans_ExtendedLLMFieldsRoundTrip(t *testing.T) {
	t.Parallel()
	db := newTestDB(t)
	ctx := context.Background()
	start := time.Now().UTC().Truncate(time.Millisecond)

	kind := "LLM"
	level := "ERROR"
	temp := 0.7
	topP := 0.9
	maxTokens := int64(512)
	ttft := int64(120_000_000)
	promptID := "pr-1"
	promptName := "support-agent"
	promptVersion := int64(3)

	spans := []Span{
		{
			TraceID: "t1", SpanID: "llm-span", ServiceName: "svc-a", Name: "chat",
			StartTime: start, EndTime: start.Add(time.Second), StatusCode: "error",
			SpanKind: &kind, Level: &level,
			LLMTemperature: &temp, LLMTopP: &topP, LLMMaxTokens: &maxTokens,
			LLMUsageDetails:         map[string]any{"cache_read_tokens": float64(4)},
			LLMCostDetails:          map[string]any{"input": float64(0.002)},
			LLMTimeToFirstTokenNano: &ttft,
			LLMPromptID:             &promptID, LLMPromptName: &promptName, LLMPromptVersion: &promptVersion,
		},
		{
			TraceID: "t1", SpanID: "plain-span", ParentSpanID: "llm-span", ServiceName: "svc-a", Name: "op",
			StartTime: start, EndTime: start.Add(time.Second), StatusCode: "ok",
		},
	}
	require.NoError(t, db.WriteSpans(ctx, spans))

	got, err := db.GetTraceSpans(ctx, "t1")
	require.NoError(t, err)
	require.Len(t, got, 2)

	require.Equal(t, "llm-span", got[0].SpanID)
	require.Equal(t, &kind, got[0].SpanKind)
	require.Equal(t, &level, got[0].Level)
	require.Equal(t, &temp, got[0].LLMTemperature)
	require.Equal(t, &topP, got[0].LLMTopP)
	require.Equal(t, &maxTokens, got[0].LLMMaxTokens)
	require.Equal(t, map[string]any{"cache_read_tokens": float64(4)}, got[0].LLMUsageDetails)
	require.Equal(t, map[string]any{"input": float64(0.002)}, got[0].LLMCostDetails)
	require.Equal(t, &ttft, got[0].LLMTimeToFirstTokenNano)
	require.Equal(t, &promptID, got[0].LLMPromptID)
	require.Equal(t, &promptName, got[0].LLMPromptName)
	require.Equal(t, &promptVersion, got[0].LLMPromptVersion)

	require.Equal(t, "plain-span", got[1].SpanID)
	require.Nil(t, got[1].SpanKind)
	require.Nil(t, got[1].Level)
	require.Nil(t, got[1].LLMTemperature)
	require.Nil(t, got[1].LLMTopP)
	require.Nil(t, got[1].LLMMaxTokens)
	require.Equal(t, map[string]any{}, got[1].LLMUsageDetails)
	require.Equal(t, map[string]any{}, got[1].LLMCostDetails)
	require.Nil(t, got[1].LLMTimeToFirstTokenNano)
	require.Nil(t, got[1].LLMPromptID)
	require.Nil(t, got[1].LLMPromptName)
	require.Nil(t, got[1].LLMPromptVersion)
}

func TestListRootArrivedTraces_OnlyReturnsTracesWithRootSpan(t *testing.T) {
	t.Parallel()
	db := newTestDB(t)
	ctx := context.Background()
	now := time.Now().UTC()

	require.NoError(t, db.WriteSpans(ctx, []Span{
		{TraceID: "has-root", SpanID: "root", ParentSpanID: "", ServiceName: "s", Name: "n", StartTime: now, EndTime: now, StatusCode: "ok"},
		{TraceID: "no-root", SpanID: "child", ParentSpanID: "missing-parent", ServiceName: "s", Name: "n", StartTime: now, EndTime: now, StatusCode: "ok"},
	}))

	ids, err := db.ListRootArrivedTraces(ctx)
	require.NoError(t, err)
	require.Equal(t, []string{"has-root"}, ids)
}

func TestListStaleOpenTraces_ExcludesTracesWithRootSpan(t *testing.T) {
	t.Parallel()
	db := newTestDB(t)
	ctx := context.Background()

	// Regression guard: a trace old enough to look "idle" must still be
	// excluded from the fallback list once its root span has arrived — the
	// primary trigger owns it, not the idle-timeout fallback. Without this,
	// a long-running child span (e.g. a 60s LLM call) risks an early close
	// via the fallback path racing the primary one.
	old := time.Now().UTC().Add(-time.Hour)
	require.NoError(t, db.WriteSpans(ctx, []Span{
		{TraceID: "has-root-but-old", SpanID: "root", ParentSpanID: "", ServiceName: "s", Name: "n", StartTime: old, EndTime: old, StatusCode: "ok"},
	}))

	ids, err := db.ListStaleOpenTraces(ctx, time.Now().UTC().Add(-time.Minute))
	require.NoError(t, err)
	require.Empty(t, ids)
}

func TestListStaleOpenTraces_ExcludesRecentlyActive(t *testing.T) {
	t.Parallel()
	db := newTestDB(t)
	ctx := context.Background()

	old := time.Now().UTC().Add(-time.Hour)
	recent := time.Now().UTC()

	require.NoError(t, db.WriteSpans(ctx, []Span{
		// stale: no root span, last_seen far in the past
		{TraceID: "stale", SpanID: "child", ParentSpanID: "missing", ServiceName: "s", Name: "n", StartTime: old, EndTime: old, StatusCode: "ok"},
		// recently active: no root span, but last_seen is now
		{TraceID: "active", SpanID: "child", ParentSpanID: "missing", ServiceName: "s", Name: "n", StartTime: recent, EndTime: recent, StatusCode: "ok"},
	}))

	ids, err := db.ListStaleOpenTraces(ctx, time.Now().UTC().Add(-time.Minute))
	require.NoError(t, err)
	require.Equal(t, []string{"stale"}, ids)
}

func TestMarkTraceClosed_PersistsVerdict(t *testing.T) {
	t.Parallel()
	db := newTestDB(t)
	ctx := context.Background()
	now := time.Now().UTC()

	require.NoError(t, db.WriteSpans(ctx, []Span{
		{TraceID: "t1", SpanID: "root", ParentSpanID: "", ServiceName: "s", Name: "n", StartTime: now, EndTime: now, StatusCode: "error"},
	}))

	require.NoError(t, db.MarkTraceClosed(ctx, "t1", CloseVerdict{SpanID: "root", Reason: "earliest error", SelfTimePct: 0.42}))

	trace, err := db.GetTrace(ctx, "t1")
	require.NoError(t, err)
	require.NotNil(t, trace.ClosedAt)
	require.NotNil(t, trace.LikelyRootCauseSpanID)
	require.Equal(t, "root", *trace.LikelyRootCauseSpanID)
	require.NotNil(t, trace.Reason)
	require.Equal(t, "earliest error", *trace.Reason)
	require.NotNil(t, trace.SelfTimePct)
	require.InDelta(t, 0.42, *trace.SelfTimePct, 0.0001)
}

func TestMarkTraceClosed_PreservesFirstSeenLastSeen(t *testing.T) {
	t.Parallel()
	db := newTestDB(t)
	ctx := context.Background()
	first := time.Now().UTC().Add(-time.Hour)
	last := time.Now().UTC()

	require.NoError(t, db.WriteSpans(ctx, []Span{
		{TraceID: "t1", SpanID: "root", ParentSpanID: "", ServiceName: "s", Name: "n", StartTime: first, EndTime: last, StatusCode: "ok"},
	}))

	before, err := db.GetTrace(ctx, "t1")
	require.NoError(t, err)

	require.NoError(t, db.MarkTraceClosed(ctx, "t1", CloseVerdict{SpanID: "root", Reason: "r", SelfTimePct: 0.1}))

	after, err := db.GetTrace(ctx, "t1")
	require.NoError(t, err)
	require.WithinDuration(t, before.FirstSeen, after.FirstSeen, time.Millisecond)
	require.WithinDuration(t, before.LastSeen, after.LastSeen, time.Millisecond)
}

func TestMarkTraceClosed_UnknownTraceReturnsNotFound(t *testing.T) {
	t.Parallel()
	db := newTestDB(t)
	ctx := context.Background()

	err := db.MarkTraceClosed(ctx, "does-not-exist", CloseVerdict{SpanID: "x", Reason: "r", SelfTimePct: 0})
	require.ErrorIs(t, err, ErrTraceNotFound)
}

func TestListTraces_FiltersByHasRootCause(t *testing.T) {
	t.Parallel()
	db := newTestDB(t)
	ctx := context.Background()
	now := time.Now().UTC()

	require.NoError(t, db.WriteSpans(ctx, []Span{
		{TraceID: "closed", SpanID: "root", ParentSpanID: "", ServiceName: "s", Name: "n", StartTime: now, EndTime: now, StatusCode: "ok"},
		{TraceID: "open", SpanID: "root", ParentSpanID: "", ServiceName: "s", Name: "n", StartTime: now, EndTime: now, StatusCode: "ok"},
	}))
	require.NoError(t, db.MarkTraceClosed(ctx, "closed", CloseVerdict{SpanID: "root", Reason: "r", SelfTimePct: 0.5}))

	hasRootCause := true
	traces, err := db.ListTraces(ctx, TraceFilter{HasRootCause: &hasRootCause})
	require.NoError(t, err)
	require.Len(t, traces, 1)
	require.Equal(t, "closed", traces[0].TraceID)
}

func TestListTraces_FiltersByUntil(t *testing.T) {
	t.Parallel()
	db := newTestDB(t)
	ctx := context.Background()
	old := time.Now().UTC().Add(-2 * time.Hour)
	recent := time.Now().UTC()

	require.NoError(t, db.WriteSpans(ctx, []Span{
		{TraceID: "old", SpanID: "root", ParentSpanID: "", ServiceName: "s", Name: "n", StartTime: old, EndTime: old, StatusCode: "ok"},
		{TraceID: "recent", SpanID: "root", ParentSpanID: "", ServiceName: "s", Name: "n", StartTime: recent, EndTime: recent, StatusCode: "ok"},
	}))

	cutoff := old.Add(time.Hour)
	traces, err := db.ListTraces(ctx, TraceFilter{Until: &cutoff})
	require.NoError(t, err)
	require.Len(t, traces, 1)
	require.Equal(t, "old", traces[0].TraceID)
}

func TestGetStats_EmptyStore(t *testing.T) {
	t.Parallel()
	db := newTestDB(t)
	ctx := context.Background()

	stats, err := db.GetStats(ctx, TraceFilter{})
	require.NoError(t, err)
	require.Equal(t, int64(0), stats.TotalTraces)
	require.Equal(t, int64(0), stats.TracesWithRootCause)
	require.Equal(t, int64(0), stats.TotalSpans)
	require.Empty(t, stats.LLM.Models)
}

func TestGetStats_CountsAndLLMBreakdown(t *testing.T) {
	t.Parallel()
	db := newTestDB(t)
	ctx := context.Background()
	now := time.Now().UTC()

	model := "gpt-5.6"
	promptTokens, completionTokens := int64(10), int64(20)
	cost := 0.5

	otherModel := "claude-opus-5"
	otherPromptTokens, otherCompletionTokens := int64(5), int64(7)
	otherCost := 1.5

	require.NoError(t, db.WriteSpans(ctx, []Span{
		{TraceID: "t1", SpanID: "root", ServiceName: "s", Name: "n", StartTime: now, EndTime: now, StatusCode: "ok"},
		{
			TraceID: "t1", SpanID: "llm1", ParentSpanID: "root", ServiceName: "s", Name: "chat",
			StartTime: now, EndTime: now, StatusCode: "ok",
			LLMModel: &model, LLMPromptTokens: &promptTokens, LLMCompletionTokens: &completionTokens, LLMCost: &cost,
		},
		{TraceID: "t2", SpanID: "root", ServiceName: "s", Name: "n", StartTime: now, EndTime: now, StatusCode: "ok"},
		{
			TraceID: "t2", SpanID: "llm2", ParentSpanID: "root", ServiceName: "s", Name: "chat",
			StartTime: now, EndTime: now, StatusCode: "ok",
			LLMModel: &otherModel, LLMPromptTokens: &otherPromptTokens, LLMCompletionTokens: &otherCompletionTokens, LLMCost: &otherCost,
		},
	}))
	require.NoError(t, db.MarkTraceClosed(ctx, "t1", CloseVerdict{SpanID: "root", Reason: "r", SelfTimePct: 0.5}))

	stats, err := db.GetStats(ctx, TraceFilter{})
	require.NoError(t, err)
	require.Equal(t, int64(2), stats.TotalTraces)
	require.Equal(t, int64(1), stats.TracesWithRootCause)
	require.Equal(t, int64(4), stats.TotalSpans)

	require.InDelta(t, 2.0, stats.LLM.TotalCost, 0.0001)
	require.Equal(t, int64(15), stats.LLM.TotalPromptTokens)
	require.Equal(t, int64(27), stats.LLM.TotalCompletionTokens)
	require.Len(t, stats.LLM.Models, 2)
	// Ordered by cost descending: otherModel (1.5) before model (0.5).
	require.Equal(t, otherModel, stats.LLM.Models[0].Model)
	require.Equal(t, int64(1), stats.LLM.Models[0].Calls)
	require.Equal(t, model, stats.LLM.Models[1].Model)
}

func TestGetTrace_UnknownTraceReturnsNotFound(t *testing.T) {
	t.Parallel()
	db := newTestDB(t)

	_, err := db.GetTrace(context.Background(), "does-not-exist")
	require.ErrorIs(t, err, ErrTraceNotFound)
}

// TestConcurrentWritesAndScans_NoErrors is a smoke test for the write-vs-
// scan contention note in 03-program-design.md: WriteSpans (ingest) and
// the root-cause poll loop's list/close calls hit the same single-writer
// DuckDB connection concurrently. This does not establish a throughput
// number — see docs/design-considerations.md for the scope of what's
// actually verified here — it only confirms concurrent access doesn't
// error or deadlock at a modest volume.
func TestConcurrentWritesAndScans_NoErrors(t *testing.T) {
	t.Parallel()
	db := newTestDB(t)
	ctx := context.Background()

	const goroutines = 8
	const spansPerGoroutine = 25

	var wg sync.WaitGroup
	errCh := make(chan error, goroutines*3)

	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := 0; i < spansPerGoroutine; i++ {
				traceID := fmt.Sprintf("load-trace-%d-%d", g, i)
				now := time.Now().UTC()
				err := db.WriteSpans(ctx, []Span{
					{TraceID: traceID, SpanID: "root", ParentSpanID: "", ServiceName: "s", Name: "n",
						StartTime: now, EndTime: now, StatusCode: "ok"},
				})
				if err != nil {
					errCh <- err
				}
			}
		}(g)
	}

	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < spansPerGoroutine; i++ {
				if _, err := db.ListRootArrivedTraces(ctx); err != nil {
					errCh <- err
				}
				if _, err := db.ListStaleOpenTraces(ctx, time.Now().UTC().Add(-time.Hour)); err != nil {
					errCh <- err
				}
			}
		}()
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		require.NoError(t, err)
	}
}
