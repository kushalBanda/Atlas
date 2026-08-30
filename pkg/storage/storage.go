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

	// Agent-run fields, populated by pkg/fields.ExtractAgentFields; nil
	// otherwise. Runs and sessions are grouped by these values at read
	// time — nothing about them is materialized. AgentStepKind falls back
	// to SpanKind when no explicit agent.step.kind attribute is present.
	SessionID     *string
	UserID        *string
	AgentRunID    *string
	AgentName     *string
	AgentStepKind *string
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
	Until        *time.Time
	Limit        int
}

// Stats is the Home-page aggregate: trace counts plus LLM usage, computed
// over traces matching f's since/until window (HasRootCause/Limit ignored).
type Stats struct {
	TotalTraces         int64
	TracesWithRootCause int64
	TotalSpans          int64
	LLM                 LLMStats
}

// LLMStats aggregates over spans with llm_model IS NOT NULL.
type LLMStats struct {
	TotalCost             float64
	TotalPromptTokens     int64
	TotalCompletionTokens int64
	Models                []ModelStat
}

// ModelStat is the per-model breakdown row within LLMStats.
type ModelStat struct {
	Model            string
	Calls            int64
	PromptTokens     int64
	CompletionTokens int64
	Cost             float64
}

// RunFilter narrows ListRuns results. All pointer fields are optional.
type RunFilter struct {
	SessionID *string
	UserID    *string
	AgentName *string
	Since     *time.Time
	Until     *time.Time
	Limit     int
}

// SessionFilter narrows ListSessions results.
type SessionFilter struct {
	UserID *string
	Since  *time.Time
	Until  *time.Time
	Limit  int
}

// RunSummary is one agent run, aggregated over spans sharing an
// agent_run_id. Derived by SQL at read time — no runs table exists.
// SessionID/UserID/AgentName are the max value across the run's spans, so
// a run whose spans disagree reports one of them rather than failing.
type RunSummary struct {
	RunID            string
	AgentName        *string
	SessionID        *string
	UserID           *string
	FirstSeen        time.Time
	LastSeen         time.Time
	SpanCount        int64
	ErrorCount       int64
	PromptTokens     int64
	CompletionTokens int64
	Cost             float64
}

// SessionSummary aggregates the runs sharing a session_id.
type SessionSummary struct {
	SessionID  string
	UserID     *string
	FirstSeen  time.Time
	LastSeen   time.Time
	RunCount   int64
	ErrorCount int64
	Cost       float64
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

	// GetStats returns the Home-page aggregate, filtered by f.Since/f.Until.
	GetStats(ctx context.Context, f TraceFilter) (*Stats, error)

	// ListRuns returns agent-run summaries matching f, most recent first.
	// Spans with a NULL agent_run_id are never part of a run.
	ListRuns(ctx context.Context, f RunFilter) ([]RunSummary, error)

	// GetRunSpans returns every span sharing runID, ordered by start_time.
	// A run may cross trace boundaries, so this is not trace-scoped.
	GetRunSpans(ctx context.Context, runID string) ([]Span, error)

	// ListSessions returns session summaries matching f, most recent first.
	ListSessions(ctx context.Context, f SessionFilter) ([]SessionSummary, error)

	Close() error
}

// SchemaRegistrar lets a plugin module declare the tables it owns at
// registration time; storage.DuckDB implements it. Collision on table name
// is an error.
type SchemaRegistrar interface {
	CreateTable(ddl string) error
}
