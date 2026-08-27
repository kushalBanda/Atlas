// Package query implements the read-side HTTP handlers: trace-by-id and
// trace-list. Slice 1 ships GetTrace only; ListTraces, logs, and metrics
// handlers land in later slices.
package query

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

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
	Spans []storage.Span `json:"spans"`
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

	writeJSON(w, http.StatusOK, traceResponse{Trace: trace, Spans: spans})
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
