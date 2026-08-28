// Package query implements the read-side HTTP handlers: trace-by-id and
// trace-list. Slice 1 ships GetTrace only; ListTraces, logs, and metrics
// handlers land in later slices.
package query

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"atlas/pkg/storage"
)

// Handlers wires the query HTTP endpoints to a storage.Store.
type Handlers struct {
	store storage.Store
}

// NewHandlers returns query Handlers backed by store.
func NewHandlers(store storage.Store) *Handlers {
	return &Handlers{store: store}
}

// traceResponse is the raw span tree plus verdict — deliberately
// view-agnostic so future trace-view types can render off the same
// response without an API change. See docs/plans/atlas/future.md.
type traceResponse struct {
	Trace *storage.Trace `json:"trace"`
	Spans []spanView     `json:"spans"`
}

// spanView adds DurationNano, computed from EndTime-StartTime, not stored.
type spanView struct {
	storage.Span
	DurationNano int64 `json:"duration_nano"`
}

func newSpanView(s storage.Span) spanView {
	return spanView{Span: s, DurationNano: s.EndTime.Sub(s.StartTime).Nanoseconds()}
}

// traceListResponse is the dashboard-list payload: summaries, not full span trees.
type traceListResponse struct {
	Traces []traceSummary `json:"traces"`
}

// traceSummary adds DurationNano, computed from LastSeen-FirstSeen, not stored.
type traceSummary struct {
	storage.Trace
	DurationNano int64 `json:"duration_nano"`
}

func newTraceSummary(t storage.Trace) traceSummary {
	return traceSummary{Trace: t, DurationNano: t.LastSeen.Sub(t.FirstSeen).Nanoseconds()}
}

// ListTraces handles GET /traces?has_root_cause=&since=&limit=. All
// parameters are optional; an unparseable one is a 400, not a silent ignore.
func (h *Handlers) ListTraces(w http.ResponseWriter, r *http.Request) {
	var f storage.TraceFilter

	if v := r.URL.Query().Get("has_root_cause"); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid has_root_cause")
			return
		}
		f.HasRootCause = &b
	}
	if v := r.URL.Query().Get("since"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid since, want RFC3339")
			return
		}
		f.Since = &t
	}
	if v := r.URL.Query().Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			writeJSONError(w, http.StatusBadRequest, "invalid limit")
			return
		}
		f.Limit = n
	}

	traces, err := h.store.ListTraces(r.Context(), f)
	if err != nil {
		slog.ErrorContext(r.Context(), "list traces failed", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "internal error")
		return
	}

	summaries := make([]traceSummary, len(traces))
	for i, t := range traces {
		summaries[i] = newTraceSummary(t)
	}

	writeJSON(w, http.StatusOK, traceListResponse{Traces: summaries})
}

// GetTrace handles GET /traces/{trace_id}. Returns 404 with a JSON error
// body if the trace is not found.
func (h *Handlers) GetTrace(w http.ResponseWriter, r *http.Request) {
	traceID := r.PathValue("trace_id")
	if traceID == "" {
		writeJSONError(w, http.StatusBadRequest, "missing trace_id")
		return
	}

	trace, err := h.store.GetTrace(r.Context(), traceID)
	if err != nil {
		if errors.Is(err, storage.ErrTraceNotFound) {
			writeJSONError(w, http.StatusNotFound, "trace not found")
			return
		}
		slog.ErrorContext(r.Context(), "get trace failed", "error", err, "trace_id", traceID)
		writeJSONError(w, http.StatusInternalServerError, "internal error")
		return
	}

	spans, err := h.store.GetTraceSpans(r.Context(), traceID)
	if err != nil {
		slog.ErrorContext(r.Context(), "get trace spans failed", "error", err, "trace_id", traceID)
		writeJSONError(w, http.StatusInternalServerError, "internal error")
		return
	}

	spanViews := make([]spanView, len(spans))
	for i, s := range spans {
		spanViews[i] = newSpanView(s)
	}

	writeJSON(w, http.StatusOK, traceResponse{Trace: trace, Spans: spanViews})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("encoding json response failed", "error", err)
	}
}

func writeJSONError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
