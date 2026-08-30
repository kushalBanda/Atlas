package query

import (
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"atlas/pkg/agentrun"
	"atlas/pkg/storage"
)

// runListResponse is the run-list payload: summaries, not span trees.
type runListResponse struct {
	Runs []storage.RunSummary `json:"runs"`
}

// runResponse is one run: its summary, its raw spans, and the derived
// graph. Kept view-agnostic like traceResponse — the graph carries nodes
// and edges, never layout coordinates.
type runResponse struct {
	Run   *storage.RunSummary `json:"run"`
	Spans []spanView          `json:"spans"`
	Graph agentrun.Graph      `json:"graph"`
}

type sessionListResponse struct {
	Sessions []storage.SessionSummary `json:"sessions"`
}

type sessionResponse struct {
	SessionID string               `json:"session_id"`
	Runs      []storage.RunSummary `json:"runs"`
}

// ListRuns handles GET /runs?session_id=&user_id=&agent=&since=&until=&limit=.
// All parameters are optional; an unparseable one is a 400.
func (h *Handlers) ListRuns(w http.ResponseWriter, r *http.Request) {
	var f storage.RunFilter

	if v := r.URL.Query().Get("session_id"); v != "" {
		f.SessionID = &v
	}
	if v := r.URL.Query().Get("user_id"); v != "" {
		f.UserID = &v
	}
	if v := r.URL.Query().Get("agent"); v != "" {
		f.AgentName = &v
	}
	since, until, ok := parseWindow(w, r)
	if !ok {
		return
	}
	f.Since, f.Until = since, until
	limit, ok := parseLimit(w, r)
	if !ok {
		return
	}
	f.Limit = limit

	runs, err := h.store.ListRuns(r.Context(), f)
	if err != nil {
		slog.ErrorContext(r.Context(), "list runs failed", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if runs == nil {
		runs = []storage.RunSummary{}
	}

	writeJSON(w, http.StatusOK, runListResponse{Runs: runs})
}

// GetRun handles GET /runs/{run_id}: summary, spans, and derived graph. A
// run with no spans is a 404 — runs are derived, so "no spans" is the only
// meaning "not found" can have.
func (h *Handlers) GetRun(w http.ResponseWriter, r *http.Request) {
	runID := r.PathValue("run_id")
	if runID == "" {
		writeJSONError(w, http.StatusBadRequest, "missing run_id")
		return
	}

	spans, err := h.store.GetRunSpans(r.Context(), runID)
	if err != nil {
		slog.ErrorContext(r.Context(), "get run spans failed", "error", err, "run_id", runID)
		writeJSONError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if len(spans) == 0 {
		writeJSONError(w, http.StatusNotFound, "run not found")
		return
	}

	runs, err := h.store.ListRuns(r.Context(), storage.RunFilter{})
	if err != nil {
		slog.ErrorContext(r.Context(), "list runs for summary failed", "error", err, "run_id", runID)
		writeJSONError(w, http.StatusInternalServerError, "internal error")
		return
	}
	var summary *storage.RunSummary
	for i := range runs {
		if runs[i].RunID == runID {
			summary = &runs[i]
			break
		}
	}

	spanViews := make([]spanView, len(spans))
	for i, s := range spans {
		spanViews[i] = newSpanView(s)
	}

	writeJSON(w, http.StatusOK, runResponse{
		Run:   summary,
		Spans: spanViews,
		Graph: agentrun.Build(runID, spans, h.repeatThreshold),
	})
}

// ListSessions handles GET /sessions?user_id=&since=&until=&limit=.
func (h *Handlers) ListSessions(w http.ResponseWriter, r *http.Request) {
	var f storage.SessionFilter

	if v := r.URL.Query().Get("user_id"); v != "" {
		f.UserID = &v
	}
	since, until, ok := parseWindow(w, r)
	if !ok {
		return
	}
	f.Since, f.Until = since, until
	limit, ok := parseLimit(w, r)
	if !ok {
		return
	}
	f.Limit = limit

	sessions, err := h.store.ListSessions(r.Context(), f)
	if err != nil {
		slog.ErrorContext(r.Context(), "list sessions failed", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if sessions == nil {
		sessions = []storage.SessionSummary{}
	}

	writeJSON(w, http.StatusOK, sessionListResponse{Sessions: sessions})
}

// GetSession handles GET /sessions/{session_id}: the session's runs, most
// recent first. A session with no runs is a 404, matching GetRun.
func (h *Handlers) GetSession(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("session_id")
	if sessionID == "" {
		writeJSONError(w, http.StatusBadRequest, "missing session_id")
		return
	}

	runs, err := h.store.ListRuns(r.Context(), storage.RunFilter{SessionID: &sessionID})
	if err != nil {
		slog.ErrorContext(r.Context(), "list session runs failed", "error", err, "session_id", sessionID)
		writeJSONError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if len(runs) == 0 {
		writeJSONError(w, http.StatusNotFound, "session not found")
		return
	}

	writeJSON(w, http.StatusOK, sessionResponse{SessionID: sessionID, Runs: runs})
}

// parseWindow reads ?since= and ?until= as RFC3339 timestamps. It writes a
// 400 and reports false when either is present but unparseable.
func parseWindow(w http.ResponseWriter, r *http.Request) (*time.Time, *time.Time, bool) {
	var since, until *time.Time

	if v := r.URL.Query().Get("since"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid since, want RFC3339")
			return nil, nil, false
		}
		since = &t
	}
	if v := r.URL.Query().Get("until"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid until, want RFC3339")
			return nil, nil, false
		}
		until = &t
	}
	return since, until, true
}

// parseLimit reads ?limit=, returning 0 (no limit) when absent. It writes a
// 400 and reports false when present but unparseable or negative.
func parseLimit(w http.ResponseWriter, r *http.Request) (int, bool) {
	v := r.URL.Query().Get("limit")
	if v == "" {
		return 0, true
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 0 {
		writeJSONError(w, http.StatusBadRequest, "invalid limit")
		return 0, false
	}
	return n, true
}
