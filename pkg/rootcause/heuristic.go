// Package rootcause scores a closed trace to find the likely root-cause
// span: a rule-based heuristic (self-time threshold + earliest-error), not
// a finished design — see docs/plans/atlas/future.md.
package rootcause

import (
	"errors"
	"fmt"
	"time"

	"atlas/pkg/storage"
)

// Verdict is the root-cause scoring result for one trace.
type Verdict struct {
	SpanID      string
	Reason      string
	SelfTimePct float64
}

var errEmptySpanSet = errors.New("scoring empty span set")

// Score picks the likely root-cause span among spans. selfTimeThreshold is
// a fraction (e.g. 0.30) of the *total trace duration*, not of the
// candidate span's own duration. Always returns a verdict for a non-empty
// span set — nil only on error — with Reason explaining low confidence
// when nothing clears the threshold (no upstream reference tool computes
// this verdict at all; see 03-program-design.md "Least confident
// decisions" #4).
func Score(spans []storage.Span, selfTimeThreshold float64) (*Verdict, error) {
	if len(spans) == 0 {
		return nil, errEmptySpanSet
	}

	total := traceDuration(spans)
	childrenByParent := groupByParent(spans)

	if errSpan := earliestError(spans); errSpan != nil {
		pct := selfTimePercent(*errSpan, childrenByParent[errSpan.SpanID], total)
		return &Verdict{
			SpanID:      errSpan.SpanID,
			Reason:      "earliest error span in trace",
			SelfTimePct: pct,
		}, nil
	}

	best, bestPct := highestSelfTimeSpan(spans, childrenByParent, total)

	if bestPct >= selfTimeThreshold {
		return &Verdict{
			SpanID:      best.SpanID,
			Reason:      fmt.Sprintf("highest self-time span (%.1f%% of trace duration)", bestPct*100),
			SelfTimePct: bestPct,
		}, nil
	}
	return &Verdict{
		SpanID: best.SpanID,
		Reason: fmt.Sprintf("low confidence: highest self-time span is only %.1f%% of trace duration (below %.0f%% threshold)",
			bestPct*100, selfTimeThreshold*100),
		SelfTimePct: bestPct,
	}, nil
}

// earliestError returns the error-status span with the earliest StartTime,
// or nil if no span has StatusCode "error".
func earliestError(spans []storage.Span) *storage.Span {
	var earliest *storage.Span
	for i := range spans {
		if spans[i].StatusCode != "error" {
			continue
		}
		if earliest == nil || spans[i].StartTime.Before(earliest.StartTime) {
			earliest = &spans[i]
		}
	}
	return earliest
}

// selfTime is the duration span spends outside of its direct children.
// Assumes children are non-overlapping and fully contained within span —
// an acceptable v1 simplification (see future.md).
func selfTime(span storage.Span, children []storage.Span) time.Duration {
	self := span.EndTime.Sub(span.StartTime)
	for _, c := range children {
		self -= c.EndTime.Sub(c.StartTime)
	}
	if self < 0 {
		return 0
	}
	return self
}

func selfTimePercent(span storage.Span, children []storage.Span, total time.Duration) float64 {
	if total <= 0 {
		return 0
	}
	return float64(selfTime(span, children)) / float64(total)
}

func highestSelfTimeSpan(spans []storage.Span, childrenByParent map[string][]storage.Span, total time.Duration) (storage.Span, float64) {
	best := spans[0]
	bestPct := selfTimePercent(best, childrenByParent[best.SpanID], total)
	for _, s := range spans[1:] {
		pct := selfTimePercent(s, childrenByParent[s.SpanID], total)
		if pct > bestPct {
			best, bestPct = s, pct
		}
	}
	return best, bestPct
}

func groupByParent(spans []storage.Span) map[string][]storage.Span {
	byParent := make(map[string][]storage.Span)
	for _, s := range spans {
		if s.ParentSpanID == "" {
			continue
		}
		byParent[s.ParentSpanID] = append(byParent[s.ParentSpanID], s)
	}
	return byParent
}

// traceDuration is the wall-clock span of the whole trace: earliest start
// to latest end across all spans.
func traceDuration(spans []storage.Span) time.Duration {
	start := spans[0].StartTime
	end := spans[0].EndTime
	for _, s := range spans[1:] {
		if s.StartTime.Before(start) {
			start = s.StartTime
		}
		if s.EndTime.After(end) {
			end = s.EndTime
		}
	}
	return end.Sub(start)
}
