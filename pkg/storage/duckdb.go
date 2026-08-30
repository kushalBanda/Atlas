package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	_ "github.com/marcboeker/go-duckdb/v2" // registers the "duckdb" database/sql driver
)

// ErrTraceNotFound is returned by GetTrace when no row matches the trace ID.
var ErrTraceNotFound = errors.New("trace not found")

// spanColumns is the canonical span SELECT list, shared by GetTraceSpans
// and GetRunSpans. scanSpanRows scans exactly this order.
const spanColumns = `trace_id, span_id, COALESCE(parent_span_id, ''), service_name, name,
	start_time, end_time, status_code, attributes, resource_attributes,
	span_kind, level,
	llm_model, llm_prompt_tokens, llm_completion_tokens, llm_cost,
	llm_temperature, llm_top_p, llm_max_tokens, llm_usage_details, llm_cost_details,
	llm_time_to_first_token_nano, llm_prompt_id, llm_prompt_name, llm_prompt_version,
	session_id, user_id, agent_run_id, agent_name, agent_step_kind`

// DuckDB is the Store implementation backed by an embedded DuckDB file.
type DuckDB struct {
	db *sql.DB
}

// NewDuckDB opens (creating if absent) a DuckDB database at path and applies
// the core schema. path may be ":memory:" for tests.
func NewDuckDB(path string) (*DuckDB, error) {
	db, err := sql.Open("duckdb", path)
	if err != nil {
		return nil, fmt.Errorf("opening duckdb at %q: %w", path, err)
	}

	// Single-writer engine: one connection avoids concurrent-writer errors:
	// see 03-program-design.md "Write contention note".
	db.SetMaxOpenConns(1)

	if _, err := db.Exec(schemaDDL); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("applying core schema: %w", err)
	}

	if _, err := db.Exec(migrationDDL); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("applying schema migrations: %w", err)
	}

	return &DuckDB{db: db}, nil
}

// CreateTable implements storage.SchemaRegistrar for plugin modules.
func (d *DuckDB) CreateTable(ddl string) error {
	if _, err := d.db.Exec(ddl); err != nil {
		return fmt.Errorf("creating table: %w", err)
	}
	return nil
}

func marshalJSON(m map[string]any) (string, error) {
	if len(m) == 0 {
		return "{}", nil
	}
	b, err := json.Marshal(m)
	if err != nil {
		return "", fmt.Errorf("marshaling attributes: %w", err)
	}
	return string(b), nil
}

func unmarshalJSON(s string) (map[string]any, error) {
	if s == "" {
		return map[string]any{}, nil
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		return nil, fmt.Errorf("unmarshaling attributes: %w", err)
	}
	return m, nil
}

// WriteSpans inserts a batch of spans and upserts each affected trace's
// first_seen/last_seen bounds. Batched per ingest call, not per-span, to
// limit write contention with the root-cause poll loop.
func (d *DuckDB) WriteSpans(ctx context.Context, spans []Span) error {
	if len(spans) == 0 {
		return nil
	}

	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning write-spans transaction: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck // no-op if already committed

	spanStmt, err := tx.PrepareContext(ctx, `
		INSERT INTO spans (trace_id, span_id, parent_span_id, service_name, name,
			start_time, end_time, status_code, attributes, resource_attributes,
			span_kind, level,
			llm_model, llm_prompt_tokens, llm_completion_tokens, llm_cost,
			llm_temperature, llm_top_p, llm_max_tokens, llm_usage_details, llm_cost_details,
			llm_time_to_first_token_nano, llm_prompt_id, llm_prompt_name, llm_prompt_version,
			session_id, user_id, agent_run_id, agent_name, agent_step_kind)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (trace_id, span_id) DO NOTHING
	`)
	if err != nil {
		return fmt.Errorf("preparing span insert: %w", err)
	}
	defer spanStmt.Close()

	traceStmt, err := tx.PrepareContext(ctx, `
		INSERT INTO traces (trace_id, first_seen, last_seen)
		VALUES (?, ?, ?)
		ON CONFLICT (trace_id) DO UPDATE SET
			first_seen = LEAST(traces.first_seen, EXCLUDED.first_seen),
			last_seen = GREATEST(traces.last_seen, EXCLUDED.last_seen)
	`)
	if err != nil {
		return fmt.Errorf("preparing trace upsert: %w", err)
	}
	defer traceStmt.Close()

	for _, s := range spans {
		attrs, err := marshalJSON(s.Attributes)
		if err != nil {
			return err
		}
		resAttrs, err := marshalJSON(s.ResourceAttributes)
		if err != nil {
			return err
		}

		var parentSpanID any
		if s.ParentSpanID != "" {
			parentSpanID = s.ParentSpanID
		}

		usageDetails, err := marshalJSON(s.LLMUsageDetails)
		if err != nil {
			return err
		}
		costDetails, err := marshalJSON(s.LLMCostDetails)
		if err != nil {
			return err
		}

		if _, err := spanStmt.ExecContext(ctx, s.TraceID, s.SpanID, parentSpanID,
			s.ServiceName, s.Name, s.StartTime, s.EndTime, s.StatusCode, attrs, resAttrs,
			s.SpanKind, s.Level,
			s.LLMModel, s.LLMPromptTokens, s.LLMCompletionTokens, s.LLMCost,
			s.LLMTemperature, s.LLMTopP, s.LLMMaxTokens, usageDetails, costDetails,
			s.LLMTimeToFirstTokenNano, s.LLMPromptID, s.LLMPromptName, s.LLMPromptVersion,
			s.SessionID, s.UserID, s.AgentRunID, s.AgentName, s.AgentStepKind); err != nil {
			return fmt.Errorf("inserting span %s/%s: %w", s.TraceID, s.SpanID, err)
		}

		if _, err := traceStmt.ExecContext(ctx, s.TraceID, s.StartTime, s.EndTime); err != nil {
			return fmt.Errorf("upserting trace %s: %w", s.TraceID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing write-spans transaction: %w", err)
	}
	return nil
}

// GetTraceSpans returns every span belonging to traceID, ordered by start time.
func (d *DuckDB) GetTraceSpans(ctx context.Context, traceID string) ([]Span, error) {
	rows, err := d.db.QueryContext(ctx, `SELECT `+spanColumns+`
		FROM spans
		WHERE trace_id = ?
		ORDER BY start_time
	`, traceID)
	if err != nil {
		return nil, fmt.Errorf("querying spans for trace %s: %w", traceID, err)
	}
	defer rows.Close()

	spans, err := scanSpanRows(rows)
	if err != nil {
		return nil, fmt.Errorf("reading spans for trace %s: %w", traceID, err)
	}
	return spans, nil
}

// GetRunSpans returns every span in runID, ordered by start_time. A run can
// cross traces, so this deliberately does not filter by trace_id.
func (d *DuckDB) GetRunSpans(ctx context.Context, runID string) ([]Span, error) {
	rows, err := d.db.QueryContext(ctx, `SELECT `+spanColumns+`
		FROM spans
		WHERE agent_run_id = ?
		ORDER BY start_time, span_id
	`, runID)
	if err != nil {
		return nil, fmt.Errorf("querying spans for run %s: %w", runID, err)
	}
	defer rows.Close()

	spans, err := scanSpanRows(rows)
	if err != nil {
		return nil, fmt.Errorf("reading spans for run %s: %w", runID, err)
	}
	return spans, nil
}

// scanSpanRows scans rows selected with spanColumns into Spans.
func scanSpanRows(rows *sql.Rows) ([]Span, error) {
	var spans []Span
	for rows.Next() {
		var s Span
		var attrs, resAttrs, usageDetails, costDetails string
		if err := rows.Scan(&s.TraceID, &s.SpanID, &s.ParentSpanID, &s.ServiceName, &s.Name,
			&s.StartTime, &s.EndTime, &s.StatusCode, &attrs, &resAttrs,
			&s.SpanKind, &s.Level,
			&s.LLMModel, &s.LLMPromptTokens, &s.LLMCompletionTokens, &s.LLMCost,
			&s.LLMTemperature, &s.LLMTopP, &s.LLMMaxTokens, &usageDetails, &costDetails,
			&s.LLMTimeToFirstTokenNano, &s.LLMPromptID, &s.LLMPromptName, &s.LLMPromptVersion,
			&s.SessionID, &s.UserID, &s.AgentRunID, &s.AgentName, &s.AgentStepKind); err != nil {
			return nil, fmt.Errorf("scanning span row: %w", err)
		}
		var err error
		if s.Attributes, err = unmarshalJSON(attrs); err != nil {
			return nil, err
		}
		if s.ResourceAttributes, err = unmarshalJSON(resAttrs); err != nil {
			return nil, err
		}
		if s.LLMUsageDetails, err = unmarshalJSON(usageDetails); err != nil {
			return nil, err
		}
		if s.LLMCostDetails, err = unmarshalJSON(costDetails); err != nil {
			return nil, err
		}
		spans = append(spans, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating spans: %w", err)
	}
	return spans, nil
}

// ListRuns aggregates spans into run summaries, most recent first.
func (d *DuckDB) ListRuns(ctx context.Context, f RunFilter) ([]RunSummary, error) {
	query := `
		SELECT agent_run_id,
			MAX(agent_name), MAX(session_id), MAX(user_id),
			MIN(start_time), MAX(end_time),
			COUNT(*),
			COUNT(*) FILTER (WHERE status_code = 'error'),
			COALESCE(SUM(llm_prompt_tokens), 0),
			COALESCE(SUM(llm_completion_tokens), 0),
			COALESCE(SUM(llm_cost), 0)
		FROM spans
		WHERE agent_run_id IS NOT NULL
	`
	var args []any

	if f.SessionID != nil {
		query += " AND session_id = ?"
		args = append(args, *f.SessionID)
	}
	if f.UserID != nil {
		query += " AND user_id = ?"
		args = append(args, *f.UserID)
	}
	if f.AgentName != nil {
		query += " AND agent_name = ?"
		args = append(args, *f.AgentName)
	}
	if f.Since != nil {
		query += " AND start_time >= ?"
		args = append(args, *f.Since)
	}
	if f.Until != nil {
		query += " AND start_time <= ?"
		args = append(args, *f.Until)
	}
	query += " GROUP BY agent_run_id ORDER BY MAX(end_time) DESC"
	if f.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, f.Limit)
	}

	rows, err := d.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("listing runs: %w", err)
	}
	defer rows.Close()

	var runs []RunSummary
	for rows.Next() {
		var r RunSummary
		if err := rows.Scan(&r.RunID, &r.AgentName, &r.SessionID, &r.UserID,
			&r.FirstSeen, &r.LastSeen, &r.SpanCount, &r.ErrorCount,
			&r.PromptTokens, &r.CompletionTokens, &r.Cost); err != nil {
			return nil, fmt.Errorf("scanning run summary: %w", err)
		}
		runs = append(runs, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating runs: %w", err)
	}
	return runs, nil
}

// ListSessions aggregates runs into session summaries, most recent first.
func (d *DuckDB) ListSessions(ctx context.Context, f SessionFilter) ([]SessionSummary, error) {
	query := `
		SELECT session_id, MAX(user_id),
			MIN(start_time), MAX(end_time),
			COUNT(DISTINCT agent_run_id),
			COUNT(*) FILTER (WHERE status_code = 'error'),
			COALESCE(SUM(llm_cost), 0)
		FROM spans
		WHERE session_id IS NOT NULL
	`
	var args []any

	if f.UserID != nil {
		query += " AND user_id = ?"
		args = append(args, *f.UserID)
	}
	if f.Since != nil {
		query += " AND start_time >= ?"
		args = append(args, *f.Since)
	}
	if f.Until != nil {
		query += " AND start_time <= ?"
		args = append(args, *f.Until)
	}
	query += " GROUP BY session_id ORDER BY MAX(end_time) DESC"
	if f.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, f.Limit)
	}

	rows, err := d.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("listing sessions: %w", err)
	}
	defer rows.Close()

	var sessions []SessionSummary
	for rows.Next() {
		var s SessionSummary
		if err := rows.Scan(&s.SessionID, &s.UserID, &s.FirstSeen, &s.LastSeen,
			&s.RunCount, &s.ErrorCount, &s.Cost); err != nil {
			return nil, fmt.Errorf("scanning session summary: %w", err)
		}
		sessions = append(sessions, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating sessions: %w", err)
	}
	return sessions, nil
}

// ListRootArrivedTraces returns trace IDs whose root span has arrived
// (parent_span_id IS NULL) but the trace is not yet closed. Primary
// trace-close trigger — see 03-program-design.md.
func (d *DuckDB) ListRootArrivedTraces(ctx context.Context) ([]string, error) {
	rows, err := d.db.QueryContext(ctx, `
		SELECT DISTINCT t.trace_id
		FROM traces t
		JOIN spans s ON s.trace_id = t.trace_id
		WHERE t.closed_at IS NULL AND s.parent_span_id IS NULL
	`)
	if err != nil {
		return nil, fmt.Errorf("listing root-arrived traces: %w", err)
	}
	defer rows.Close()
	return scanTraceIDs(rows)
}

// ListStaleOpenTraces returns trace IDs with no new span since idleSince and
// no root span yet — the crashed-hop fallback close trigger.
func (d *DuckDB) ListStaleOpenTraces(ctx context.Context, idleSince time.Time) ([]string, error) {
	rows, err := d.db.QueryContext(ctx, `
		SELECT t.trace_id
		FROM traces t
		WHERE t.closed_at IS NULL
			AND t.last_seen < ?
			AND NOT EXISTS (
				SELECT 1 FROM spans s WHERE s.trace_id = t.trace_id AND s.parent_span_id IS NULL
			)
	`, idleSince)
	if err != nil {
		return nil, fmt.Errorf("listing stale open traces: %w", err)
	}
	defer rows.Close()
	return scanTraceIDs(rows)
}

func scanTraceIDs(rows *sql.Rows) ([]string, error) {
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scanning trace id: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating trace ids: %w", err)
	}
	return ids, nil
}

// MarkTraceClosed is a targeted update: only closed_at, likely_root_cause_span_id,
// reason, and self_time_pct change. first_seen/last_seen are left untouched.
func (d *DuckDB) MarkTraceClosed(ctx context.Context, traceID string, verdict CloseVerdict) error {
	res, err := d.db.ExecContext(ctx, `
		UPDATE traces
		SET closed_at = ?, likely_root_cause_span_id = ?, reason = ?, self_time_pct = ?
		WHERE trace_id = ?
	`, time.Now().UTC(), verdict.SpanID, verdict.Reason, verdict.SelfTimePct, traceID)
	if err != nil {
		return fmt.Errorf("marking trace %s closed: %w", traceID, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking rows affected for trace %s: %w", traceID, err)
	}
	if n == 0 {
		return fmt.Errorf("marking trace %s closed: %w", traceID, ErrTraceNotFound)
	}
	return nil
}

// ListTraces returns traces matching f, most recent first.
func (d *DuckDB) ListTraces(ctx context.Context, f TraceFilter) ([]Trace, error) {
	query := `
		SELECT trace_id, first_seen, last_seen, closed_at, likely_root_cause_span_id, reason, self_time_pct
		FROM traces
		WHERE 1=1
	`
	var args []any

	if f.HasRootCause != nil {
		if *f.HasRootCause {
			query += " AND likely_root_cause_span_id IS NOT NULL"
		} else {
			query += " AND likely_root_cause_span_id IS NULL"
		}
	}
	if f.Since != nil {
		query += " AND last_seen >= ?"
		args = append(args, *f.Since)
	}
	if f.Until != nil {
		query += " AND last_seen <= ?"
		args = append(args, *f.Until)
	}
	query += " ORDER BY last_seen DESC"
	if f.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, f.Limit)
	}

	rows, err := d.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("listing traces: %w", err)
	}
	defer rows.Close()

	var traces []Trace
	for rows.Next() {
		t, err := scanTrace(rows)
		if err != nil {
			return nil, err
		}
		traces = append(traces, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating traces: %w", err)
	}
	return traces, nil
}

// GetTrace returns the trace row for traceID, or ErrTraceNotFound.
func (d *DuckDB) GetTrace(ctx context.Context, traceID string) (*Trace, error) {
	row := d.db.QueryRowContext(ctx, `
		SELECT trace_id, first_seen, last_seen, closed_at, likely_root_cause_span_id, reason, self_time_pct
		FROM traces
		WHERE trace_id = ?
	`, traceID)

	t, err := scanTrace(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrTraceNotFound
		}
		return nil, fmt.Errorf("querying trace %s: %w", traceID, err)
	}
	return &t, nil
}

// GetStats returns the Home-page aggregate over traces/spans in f's
// since/until window. f.HasRootCause and f.Limit are ignored.
func (d *DuckDB) GetStats(ctx context.Context, f TraceFilter) (*Stats, error) {
	traceQuery := `SELECT COUNT(*), COUNT(likely_root_cause_span_id) FROM traces WHERE 1=1`
	var args []any
	if f.Since != nil {
		traceQuery += " AND last_seen >= ?"
		args = append(args, *f.Since)
	}
	if f.Until != nil {
		traceQuery += " AND last_seen <= ?"
		args = append(args, *f.Until)
	}

	var stats Stats
	if err := d.db.QueryRowContext(ctx, traceQuery, args...).Scan(&stats.TotalTraces, &stats.TracesWithRootCause); err != nil {
		return nil, fmt.Errorf("counting traces: %w", err)
	}

	spanQuery := `SELECT COUNT(*) FROM spans s JOIN traces t ON t.trace_id = s.trace_id WHERE 1=1`
	spanArgs := append([]any{}, args...)
	if f.Since != nil {
		spanQuery += " AND t.last_seen >= ?"
		spanArgs = append(spanArgs, *f.Since)
	}
	if f.Until != nil {
		spanQuery += " AND t.last_seen <= ?"
		spanArgs = append(spanArgs, *f.Until)
	}
	if err := d.db.QueryRowContext(ctx, spanQuery, spanArgs...).Scan(&stats.TotalSpans); err != nil {
		return nil, fmt.Errorf("counting spans: %w", err)
	}

	modelQuery := `
		SELECT s.llm_model, COUNT(*), COALESCE(SUM(s.llm_prompt_tokens), 0),
			COALESCE(SUM(s.llm_completion_tokens), 0), COALESCE(SUM(s.llm_cost), 0)
		FROM spans s
		JOIN traces t ON t.trace_id = s.trace_id
		WHERE s.llm_model IS NOT NULL
	`
	modelArgs := append([]any{}, args...)
	if f.Since != nil {
		modelQuery += " AND t.last_seen >= ?"
		modelArgs = append(modelArgs, *f.Since)
	}
	if f.Until != nil {
		modelQuery += " AND t.last_seen <= ?"
		modelArgs = append(modelArgs, *f.Until)
	}
	modelQuery += " GROUP BY s.llm_model ORDER BY SUM(s.llm_cost) DESC"

	rows, err := d.db.QueryContext(ctx, modelQuery, modelArgs...)
	if err != nil {
		return nil, fmt.Errorf("aggregating llm stats: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var m ModelStat
		if err := rows.Scan(&m.Model, &m.Calls, &m.PromptTokens, &m.CompletionTokens, &m.Cost); err != nil {
			return nil, fmt.Errorf("scanning model stat: %w", err)
		}
		stats.LLM.Models = append(stats.LLM.Models, m)
		stats.LLM.TotalPromptTokens += m.PromptTokens
		stats.LLM.TotalCompletionTokens += m.CompletionTokens
		stats.LLM.TotalCost += m.Cost
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating model stats: %w", err)
	}

	return &stats, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanTrace(row rowScanner) (Trace, error) {
	var t Trace
	if err := row.Scan(&t.TraceID, &t.FirstSeen, &t.LastSeen, &t.ClosedAt,
		&t.LikelyRootCauseSpanID, &t.Reason, &t.SelfTimePct); err != nil {
		return Trace{}, err
	}
	return t, nil
}

var _ Store = (*DuckDB)(nil)

// Close releases the underlying database connection.
func (d *DuckDB) Close() error {
	if err := d.db.Close(); err != nil {
		return fmt.Errorf("closing duckdb: %w", err)
	}
	return nil
}
