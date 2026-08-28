// Package storage defines the backend-agnostic persistence interface for
// Atlas spans and traces. The DuckDB implementation lives in duckdb.go;
// ClickHouse may implement this same interface later.
package storage

import (
	"context"
	"time"
)

// Span is one OTel span as Atlas stores it.
type Span struct {
	TraceID            string
	SpanID             string
	ParentSpanID       string
	ServiceName        string
	Name               string
	StartTime          time.Time
	EndTime            time.Time
	StatusCode         string // "ok" | "error" | "unset"
	Attributes         map[string]any
	ResourceAttributes map[string]any

	// SpanKind and Level apply to any span, not just LLM calls (e.g.
	// SpanKind "tool"/"chain"/"retriever", Level "ERROR"/"WARNING").
	// Populated by pkg/fields.ExtractSpanKind/ExtractLevel; nil otherwise.
	SpanKind *string
	Level    *string

	// LLM fields, populated by pkg/fields.ExtractLLMFields; nil otherwise.
	// Prompt/completion text isn't duplicated here — read it from Attributes.
	LLMModel                *string
	LLMPromptTokens         *int64
	LLMCompletionTokens     *int64
	LLMCost                 *float64
	LLMTemperature          *float64
	LLMTopP                 *float64
	LLMMaxTokens            *int64
	LLMUsageDetails         map[string]any // usage breakdown beyond prompt/completion tokens (cache, reasoning, ...)
	LLMCostDetails          map[string]any // cost breakdown beyond total (input/output/cache, ...)
	LLMTimeToFirstTokenNano *int64
	LLMPromptID             *string
	LLMPromptName           *string
	LLMPromptVersion        *int64
}

// Trace is one row per trace, written/updated at trace-close time.
type Trace struct {
	TraceID               string
	FirstSeen             time.Time
	LastSeen              time.Time
	ClosedAt              *time.Time
	LikelyRootCauseSpanID *string
	Reason                *string
	SelfTimePct           *float64
}

// CloseVerdict avoids storage importing pkg/rootcause (would cycle back
// through pkg/rootcause's own storage.Store dependency). pkg/rootcause.Verdict
// converts to this shape at the call site in Watcher.Run.
type CloseVerdict struct {
	SpanID      string
	Reason      string
	SelfTimePct float64
}

// TraceFilter narrows ListTraces results.
type TraceFilter struct {
	HasRootCause *bool
	Since        *time.Time
	Limit        int
}

// Store is the backend-agnostic persistence interface.
type Store interface {
	WriteSpans(ctx context.Context, spans []Span) error
	GetTraceSpans(ctx context.Context, traceID string) ([]Span, error)

	// ListRootArrivedTraces returns trace_ids whose root span (parent_span_id
	// IS NULL) has been written but the trace row is not yet closed_at.
	// Primary close trigger — checked by pkg/rootcause after every WriteSpans batch.
	ListRootArrivedTraces(ctx context.Context) ([]string, error)

	// ListStaleOpenTraces returns trace_ids with no new span since idleSince
	// AND no root span yet. Fallback close trigger only (crashed-hop case).
	ListStaleOpenTraces(ctx context.Context, idleSince time.Time) ([]string, error)

	// MarkTraceClosed is a targeted update: closed_at, likely_root_cause_span_id,
	// reason, self_time_pct only — first_seen/last_seen untouched.
	MarkTraceClosed(ctx context.Context, traceID string, verdict CloseVerdict) error

	ListTraces(ctx context.Context, f TraceFilter) ([]Trace, error)
	GetTrace(ctx context.Context, traceID string) (*Trace, error)

	Close() error
}

// SchemaRegistrar lets a plugin module declare the tables it owns at
// registration time; storage.DuckDB implements it. Collision on table name
// is an error.
type SchemaRegistrar interface {
	CreateTable(ddl string) error
}
