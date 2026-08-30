package query

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"atlas/pkg/storage"
)

func newTestStore(t *testing.T) *storage.DuckDB {
	t.Helper()
	store, err := storage.NewDuckDB(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close() })
	return store
}

func TestListRuns_ReturnsRunsAndEmptySliceNotNull(t *testing.T) {
	store := newTestStore(t)
	h := NewHandlers(store, 3)

	rec := httptest.NewRecorder()
	h.ListRuns(rec, httptest.NewRequest(http.MethodGet, "/runs", nil))

	require.Equal(t, http.StatusOK, rec.Code)
	var body struct {
		Runs []storage.RunSummary `json:"runs"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.NotNil(t, body.Runs, "runs is null, want []")
}

func TestGetRun_ReturnsGraph(t *testing.T) {
	store := newTestStore(t)
	base := time.Now().UTC().Truncate(time.Second)
	run, sess, agent, kind := "run-a", "sess-1", "researcher", "tool"
	spans := []storage.Span{
		{TraceID: "t1", SpanID: "s1", ServiceName: "svc", Name: "plan",
			StartTime: base, EndTime: base.Add(time.Second), StatusCode: "ok",
			AgentRunID: &run, SessionID: &sess, AgentName: &agent, AgentStepKind: &kind},
		{TraceID: "t1", SpanID: "s2", ParentSpanID: "s1", ServiceName: "svc", Name: "search",
			StartTime: base.Add(time.Second), EndTime: base.Add(2 * time.Second), StatusCode: "ok",
			AgentRunID: &run, SessionID: &sess, AgentName: &agent, AgentStepKind: &kind},
	}
	require.NoError(t, store.WriteSpans(context.Background(), spans))

	h := NewHandlers(store, 3)
	req := httptest.NewRequest(http.MethodGet, "/runs/run-a", nil)
	req.SetPathValue("run_id", "run-a")
	rec := httptest.NewRecorder()
	h.GetRun(rec, req)

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var body struct {
		Run   *storage.RunSummary `json:"run"`
		Spans []json.RawMessage   `json:"spans"`
		Graph struct {
			Nodes []json.RawMessage `json:"nodes"`
			Edges []struct {
				From string `json:"from"`
				To   string `json:"to"`
			} `json:"edges"`
		} `json:"graph"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.NotNil(t, body.Run)
	require.EqualValues(t, 2, body.Run.SpanCount)
	require.Len(t, body.Graph.Nodes, 2)
	require.Len(t, body.Graph.Edges, 1)
	require.Equal(t, "s1", body.Graph.Edges[0].From)
}

func TestGetRun_UnknownRunIs404(t *testing.T) {
	h := NewHandlers(newTestStore(t), 3)

	req := httptest.NewRequest(http.MethodGet, "/runs/nope", nil)
	req.SetPathValue("run_id", "nope")
	rec := httptest.NewRecorder()
	h.GetRun(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestListSessions_ReturnsEmptySliceNotNull(t *testing.T) {
	h := NewHandlers(newTestStore(t), 3)

	rec := httptest.NewRecorder()
	h.ListSessions(rec, httptest.NewRequest(http.MethodGet, "/sessions", nil))

	require.Equal(t, http.StatusOK, rec.Code)
	var body struct {
		Sessions []storage.SessionSummary `json:"sessions"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.NotNil(t, body.Sessions, "sessions is null, want []")
}
