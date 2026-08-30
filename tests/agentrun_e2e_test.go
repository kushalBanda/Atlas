package tests

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"

	"atlas/pkg/ingest"
	"atlas/pkg/plugin"
	"atlas/pkg/plugin/otelcore"
	"atlas/pkg/query"
	"atlas/pkg/storage"
)

// TestAgentRun_TwoTraceRunGroupsIntoOneConnectedGraph posts two OTLP trace
// batches that share an agent.run.id and session.id, then asserts that
// GET /runs, GET /runs/{id}, and GET /sessions group them into one run and
// one session with a cross-trace edge connecting the two traces. This is
// the derived-on-read path from
// docs/superpowers/specs/2026-08-29-agent-run-debugging-design.md: no
// runs table, no new write path, no new close trigger.
func TestAgentRun_TwoTraceRunGroupsIntoOneConnectedGraph(t *testing.T) {
	store, err := storage.NewDuckDB(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, store.Close()) })

	queryHandlers := query.NewHandlers(store, 3)

	registry := plugin.NewRegistry(store, noopRegistrar{})
	require.NoError(t, registry.Register(otelcore.New(store)))
	ingestSrv := ingest.NewServer(registry)

	trace1ID := [16]byte{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1}
	trace2ID := [16]byte{2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2}
	now := time.Now()

	// Trace 1: root (plan) -> child (handoff), t=0..2s.
	postOTLP(t, ingestSrv, buildAgentTrace(t, trace1ID, []agentSpanSpec{
		{spanID: [8]byte{1, 1, 1, 1, 1, 1, 1, 1}, name: "plan", start: now, end: now.Add(time.Second)},
		{spanID: [8]byte{1, 1, 1, 1, 1, 1, 1, 2}, parentID: [8]byte{1, 1, 1, 1, 1, 1, 1, 1},
			name: "handoff", start: now.Add(time.Second), end: now.Add(2 * time.Second)},
	}))

	// Trace 2: root (write-summary), starting after trace 1's child ends —
	// the handoff case: it attaches to trace 1's latest span.
	postOTLP(t, ingestSrv, buildAgentTrace(t, trace2ID, []agentSpanSpec{
		{spanID: [8]byte{2, 2, 2, 2, 2, 2, 2, 1}, name: "write-summary",
			start: now.Add(3 * time.Second), end: now.Add(4 * time.Second)},
	}))

	// GET /runs: exactly one run, three spans.
	runsRec := httptest.NewRecorder()
	queryHandlers.ListRuns(runsRec, httptest.NewRequest(http.MethodGet, "/runs", nil))
	require.Equal(t, http.StatusOK, runsRec.Code)
	var runsBody struct {
		Runs []storage.RunSummary `json:"runs"`
	}
	require.NoError(t, json.Unmarshal(runsRec.Body.Bytes(), &runsBody))
	require.Len(t, runsBody.Runs, 1)
	require.Equal(t, "run-e2e", runsBody.Runs[0].RunID)
	require.EqualValues(t, 3, runsBody.Runs[0].SpanCount)

	// GET /runs/run-e2e: 3 nodes, 1 within-trace edge, 1 cross-trace edge.
	runReq := httptest.NewRequest(http.MethodGet, "/runs/run-e2e", nil)
	runReq.SetPathValue("run_id", "run-e2e")
	runRec := httptest.NewRecorder()
	queryHandlers.GetRun(runRec, runReq)
	require.Equal(t, http.StatusOK, runRec.Code, runRec.Body.String())

	var runBody struct {
		Graph struct {
			Nodes []json.RawMessage `json:"nodes"`
			Edges []struct {
				From       string `json:"from"`
				To         string `json:"to"`
				CrossTrace bool   `json:"cross_trace"`
			} `json:"edges"`
		} `json:"graph"`
	}
	require.NoError(t, json.Unmarshal(runRec.Body.Bytes(), &runBody))
	require.Len(t, runBody.Graph.Nodes, 3)
	require.Len(t, runBody.Graph.Edges, 2)

	var crossEdges int
	for _, e := range runBody.Graph.Edges {
		if e.CrossTrace {
			crossEdges++
			require.Equal(t, hex.EncodeToString([]byte{1, 1, 1, 1, 1, 1, 1, 2}), e.From)
			require.Equal(t, hex.EncodeToString([]byte{2, 2, 2, 2, 2, 2, 2, 1}), e.To)
		}
	}
	require.Equal(t, 1, crossEdges)

	// GET /sessions: one session, one run.
	sessionsRec := httptest.NewRecorder()
	queryHandlers.ListSessions(sessionsRec, httptest.NewRequest(http.MethodGet, "/sessions", nil))
	require.Equal(t, http.StatusOK, sessionsRec.Code)
	var sessionsBody struct {
		Sessions []storage.SessionSummary `json:"sessions"`
	}
	require.NoError(t, json.Unmarshal(sessionsRec.Body.Bytes(), &sessionsBody))
	require.Len(t, sessionsBody.Sessions, 1)
	require.Equal(t, "sess-e2e", sessionsBody.Sessions[0].SessionID)
	require.EqualValues(t, 1, sessionsBody.Sessions[0].RunCount)
}

// noopRegistrar satisfies api.RouteRegistrar without mounting real routes;
// this test drives the query handlers directly instead of through a router.
type noopRegistrar struct{}

func (noopRegistrar) Handle(string, http.Handler) error { return nil }

func postOTLP(t *testing.T, srv *ingest.Server, body io.Reader) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/v1/traces", body)
	req.Header.Set("Content-Type", "application/x-protobuf")
	w := httptest.NewRecorder()
	srv.ServeOTLP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
}

type agentSpanSpec struct {
	spanID   [8]byte
	parentID [8]byte
	name     string
	start    time.Time
	end      time.Time
}

// buildAgentTrace returns a protobuf-encoded OTLP export for one trace,
// every span tagged agent.run.id=run-e2e and session.id=sess-e2e.
func buildAgentTrace(t *testing.T, traceID [16]byte, specs []agentSpanSpec) io.Reader {
	t.Helper()
	traces := ptrace.NewTraces()
	rs := traces.ResourceSpans().AppendEmpty()
	rs.Resource().Attributes().PutStr("service.name", "agent-svc")

	spans := rs.ScopeSpans().AppendEmpty().Spans()
	for _, spec := range specs {
		s := spans.AppendEmpty()
		s.SetTraceID(pcommon.TraceID(traceID))
		s.SetSpanID(pcommon.SpanID(spec.spanID))
		if spec.parentID != ([8]byte{}) {
			s.SetParentSpanID(pcommon.SpanID(spec.parentID))
		}
		s.SetName(spec.name)
		s.SetStartTimestamp(pcommon.NewTimestampFromTime(spec.start))
		s.SetEndTimestamp(pcommon.NewTimestampFromTime(spec.end))
		s.Status().SetCode(ptrace.StatusCodeOk)
		s.Attributes().PutStr("agent.run.id", "run-e2e")
		s.Attributes().PutStr("session.id", "sess-e2e")
		s.Attributes().PutStr("agent.name", "researcher")
	}

	var marshaler ptrace.ProtoMarshaler
	data, err := marshaler.MarshalTraces(traces)
	require.NoError(t, err)
	return bytes.NewReader(data)
}
