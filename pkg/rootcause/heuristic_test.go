package rootcause

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"atlas/pkg/storage"
)

func span(spanID, parentID string, startOffsetMs, durationMs int, statusCode string) storage.Span {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	start := base.Add(time.Duration(startOffsetMs) * time.Millisecond)
	return storage.Span{
		SpanID:       spanID,
		ParentSpanID: parentID,
		StartTime:    start,
		EndTime:      start.Add(time.Duration(durationMs) * time.Millisecond),
		StatusCode:   statusCode,
	}
}

func TestScore_LinearChain_NoError_FlagsHighestSelfTime(t *testing.T) {
	t.Parallel()
	// root [0,100) -> mid [10,90) self=10ms -> leaf [20,30) self=10ms (leaf's
	// own 10ms is 10% of the 100ms trace; mid's self is also 10ms/100ms=10%).
	// Give mid extra self-time so it, not the trivial leaf, wins: mid runs
	// [10,90) with leaf only occupying [20,30) inside it -> mid self = 70ms (70%).
	spans := []storage.Span{
		span("root", "", 0, 100, "ok"),
		span("mid", "root", 10, 80, "ok"),
		span("leaf", "mid", 20, 10, "ok"),
	}

	v, err := Score(spans, 0.30)
	require.NoError(t, err)
	require.Equal(t, "mid", v.SpanID)
	require.InDelta(t, 0.70, v.SelfTimePct, 0.01)
}

func TestScore_ErrorInChild_FlagsEarliestError(t *testing.T) {
	t.Parallel()
	spans := []storage.Span{
		span("root", "", 0, 100, "ok"),
		span("child-a", "root", 10, 20, "error"),
		span("child-b", "root", 40, 20, "error"),
	}

	v, err := Score(spans, 0.30)
	require.NoError(t, err)
	require.Equal(t, "child-a", v.SpanID)
	require.Contains(t, v.Reason, "earliest error")
}

func TestScore_FanOut_NonOverlappingChildren_ComputesSelfTimeCorrectly(t *testing.T) {
	t.Parallel()
	// root [0,100) fans out to two SEQUENTIAL (non-overlapping) children:
	// a [0,80) self=80ms(80%), b [80,100) self=20ms(20%). a should win.
	// This does NOT cover concurrent/overlapping siblings — selfTime
	// (heuristic.go) subtracts each child's full duration regardless of
	// overlap between them, so overlapping concurrent children would be
	// double-counted. That's a known, documented v1 simplification (see
	// heuristic.go's selfTime doc comment and future.md's scorer
	// deep-dive), not covered by a test here.
	spans := []storage.Span{
		span("root", "", 0, 100, "ok"),
		span("a", "root", 0, 80, "ok"),
		span("b", "root", 80, 20, "ok"),
	}

	v, err := Score(spans, 0.30)
	require.NoError(t, err)
	require.Equal(t, "a", v.SpanID)
	require.InDelta(t, 0.80, v.SelfTimePct, 0.01)
}

func TestScore_NoErrorBelowThreshold_ReturnsLowConfidenceVerdict(t *testing.T) {
	t.Parallel()
	// root [0,100) split across 5 equal sequential children, each 20ms
	// self-time = 20% of the trace, all below a 30% threshold.
	spans := []storage.Span{
		span("root", "", 0, 100, "ok"),
		span("c1", "root", 0, 20, "ok"),
		span("c2", "root", 20, 20, "ok"),
		span("c3", "root", 40, 20, "ok"),
		span("c4", "root", 60, 20, "ok"),
		span("c5", "root", 80, 20, "ok"),
	}

	v, err := Score(spans, 0.30)
	require.NoError(t, err)
	require.NotNil(t, v) // always a verdict, never nil, for a non-empty span set
	require.InDelta(t, 0.20, v.SelfTimePct, 0.01)
	require.Contains(t, v.Reason, "low confidence")
}

func TestScore_EmptySpans_ReturnsError(t *testing.T) {
	t.Parallel()
	_, err := Score(nil, 0.30)
	require.Error(t, err)
}

func TestSelfTime_SubtractsChildDurations(t *testing.T) {
	t.Parallel()
	parent := span("p", "", 0, 100, "ok")
	children := []storage.Span{
		span("c1", "p", 0, 30, "ok"),
		span("c2", "p", 30, 20, "ok"),
	}

	require.Equal(t, 50*time.Millisecond, selfTime(parent, children))
}

func TestSelfTime_NoChildren_EqualsFullDuration(t *testing.T) {
	t.Parallel()
	leaf := span("leaf", "p", 0, 10, "ok")
	require.Equal(t, 10*time.Millisecond, selfTime(leaf, nil))
}
