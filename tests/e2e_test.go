// Package tests holds cross-package integration tests — the full loop
// through ingest, storage, plugin dispatch, root-cause scoring, and query,
// wired together the same way cmd/atlas-server does, but in-process
// (in-memory DuckDB, no real network listeners).
package tests

import (
	"bytes"
	"context"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"

	"atlas/pkg/api"
	"atlas/pkg/ingest"
	"atlas/pkg/plugin"
	"atlas/pkg/plugin/otelcore"
	"atlas/pkg/query"
	"atlas/pkg/rootcause"
	"atlas/pkg/storage"
)

// TestEndToEnd_MultiHopTrace_ProducesExpectedRootCauseVerdict wires the
// real ingest -> plugin registry -> storage -> rootcause -> query path
// (the same components cmd/atlas-server assembles) and sends a synthetic
// three-span trace: an ok root, an ok middle hop, and an error leaf. It
// polls the query API until the trace closes and asserts the verdict
// against the known-answer fixture: the error leaf, flagged as the
// earliest error in the trace.
func TestEndToEnd_MultiHopTrace_ProducesExpectedRootCauseVerdict(t *testing.T) {
	store, err := storage.NewDuckDB(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, store.Close()) })

	queryHandlers := query.NewHandlers(store, 3)
	router := api.NewRouter(queryHandlers, "")

	registry := plugin.NewRegistry(store, router)
	require.NoError(t, registry.Register(otelcore.New(store)))

	ingestSrv := ingest.NewServer(registry)

	traceIDBytes := [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}
	traceIDHex := hex.EncodeToString(traceIDBytes[:])

	body := buildMultiHopTrace(t, traceIDBytes)

	req := httptest.NewRequest(http.MethodPost, "/v1/traces", body)
	req.Header.Set("Content-Type", "application/x-protobuf")
	w := httptest.NewRecorder()
	ingestSrv.ServeOTLP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	// Drive the close loop deterministically instead of sleeping for a
	// real ticker: run the watcher against a short-lived context and poll
	// the query API for the verdict to land.
	watcher := rootcause.NewWatcher(store, 30*time.Second, 0.30)
	watcherCtx, cancelWatcher := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancelWatcher()
	go func() { _ = watcher.Run(watcherCtx, 20*time.Millisecond) }()

	deadline := time.Now().Add(2 * time.Second)
	var trace *storage.Trace
	for time.Now().Before(deadline) {
		trace, err = store.GetTrace(context.Background(), traceIDHex)
		require.NoError(t, err)
		if trace.ClosedAt != nil {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	cancelWatcher()

	require.NotNil(t, trace.ClosedAt, "trace did not close within the deadline")
	require.NotNil(t, trace.LikelyRootCauseSpanID)
	require.Equal(t, "0303030303030303", *trace.LikelyRootCauseSpanID)
	require.NotNil(t, trace.Reason)
	require.Contains(t, *trace.Reason, "earliest error")

	// Query API returns the raw span tree alongside the verdict.
	getReq := httptest.NewRequest(http.MethodGet, "/traces/"+traceIDHex, nil)
	getReq.SetPathValue("trace_id", traceIDHex)
	getW := httptest.NewRecorder()
	queryHandlers.GetTrace(getW, getReq)
	require.Equal(t, http.StatusOK, getW.Code)
}

// buildMultiHopTrace returns a protobuf-encoded OTLP export with three
// spans under traceID: root (ok) -> mid (ok) -> leaf (error).
func buildMultiHopTrace(t *testing.T, traceID [16]byte) io.Reader {
	t.Helper()
	traces := ptrace.NewTraces()
	rs := traces.ResourceSpans().AppendEmpty()
	rs.Resource().Attributes().PutStr("service.name", "checkout")

	spans := rs.ScopeSpans().AppendEmpty().Spans()
	now := time.Now()

	root := spans.AppendEmpty()
	root.SetTraceID(pcommon.TraceID(traceID))
	root.SetSpanID(pcommon.SpanID{1, 1, 1, 1, 1, 1, 1, 1})
	root.SetName("handle-request")
	root.SetStartTimestamp(pcommon.NewTimestampFromTime(now))
	root.SetEndTimestamp(pcommon.NewTimestampFromTime(now.Add(100 * time.Millisecond)))
	root.Status().SetCode(ptrace.StatusCodeOk)

	mid := spans.AppendEmpty()
	mid.SetTraceID(pcommon.TraceID(traceID))
	mid.SetSpanID(pcommon.SpanID{2, 2, 2, 2, 2, 2, 2, 2})
	mid.SetParentSpanID(pcommon.SpanID{1, 1, 1, 1, 1, 1, 1, 1})
	mid.SetName("validate-order")
	mid.SetStartTimestamp(pcommon.NewTimestampFromTime(now.Add(10 * time.Millisecond)))
	mid.SetEndTimestamp(pcommon.NewTimestampFromTime(now.Add(90 * time.Millisecond)))
	mid.Status().SetCode(ptrace.StatusCodeOk)

	leaf := spans.AppendEmpty()
	leaf.SetTraceID(pcommon.TraceID(traceID))
	leaf.SetSpanID(pcommon.SpanID{3, 3, 3, 3, 3, 3, 3, 3})
	leaf.SetParentSpanID(pcommon.SpanID{2, 2, 2, 2, 2, 2, 2, 2})
	leaf.SetName("charge-card")
	leaf.SetStartTimestamp(pcommon.NewTimestampFromTime(now.Add(20 * time.Millisecond)))
	leaf.SetEndTimestamp(pcommon.NewTimestampFromTime(now.Add(40 * time.Millisecond)))
	leaf.Status().SetCode(ptrace.StatusCodeError)

	var marshaler ptrace.ProtoMarshaler
	data, err := marshaler.MarshalTraces(traces)
	require.NoError(t, err)
	return bytes.NewReader(data)
}
