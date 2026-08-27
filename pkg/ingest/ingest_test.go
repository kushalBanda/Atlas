package ingest

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"

	"atlas/pkg/storage"
)

// fakeDispatcher is an in-memory Dispatcher stub for ingest tests, avoiding
// a real plugin registry for handler-level HTTP tests.
type fakeDispatcher struct {
	dispatchErr error
	written     []storage.Span
}

func (f *fakeDispatcher) Dispatch(_ context.Context, spans []storage.Span) error {
	if f.dispatchErr != nil {
		return f.dispatchErr
	}
	f.written = append(f.written, spans...)
	return nil
}

func newSingleSpanTraces(t *testing.T) ptrace.Traces {
	t.Helper()
	traces := ptrace.NewTraces()
	rs := traces.ResourceSpans().AppendEmpty()
	rs.Resource().Attributes().PutStr("service.name", "checkout")

	span := rs.ScopeSpans().AppendEmpty().Spans().AppendEmpty()
	span.SetTraceID(pcommon.TraceID{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16})
	span.SetSpanID(pcommon.SpanID{1, 2, 3, 4, 5, 6, 7, 8})
	span.SetName("charge-card")
	now := time.Now()
	span.SetStartTimestamp(pcommon.NewTimestampFromTime(now))
	span.SetEndTimestamp(pcommon.NewTimestampFromTime(now.Add(50 * time.Millisecond)))
	span.Status().SetCode(ptrace.StatusCodeError)
	return traces
}

// TestServeOTLP_K8sResourceAttributes_SurviveIngestUntouched proves Slice
// 7's discovery-to-span claim in the only place it's currently wired: a
// real K8s-instrumented workload's OTel SDK attaches k8s.pod.name/
// k8s.namespace.name/k8s.node.name as resource attributes on its own
// (independent of pkg/discovery, which only reports candidates — v1 does
// not auto-wire the collector, see docs/plans/atlas/03-program-design.md
// "Least confident decisions" #3). This confirms Atlas's ingest path
// preserves those attributes byte-for-byte rather than dropping or
// renaming them.
func TestServeOTLP_K8sResourceAttributes_SurviveIngestUntouched(t *testing.T) {
	t.Parallel()
	traces := ptrace.NewTraces()
	rs := traces.ResourceSpans().AppendEmpty()
	rs.Resource().Attributes().PutStr("service.name", "checkout-db")
	rs.Resource().Attributes().PutStr("k8s.pod.name", "checkout-db-0")
	rs.Resource().Attributes().PutStr("k8s.namespace.name", "prod")
	rs.Resource().Attributes().PutStr("k8s.node.name", "node-a")

	span := rs.ScopeSpans().AppendEmpty().Spans().AppendEmpty()
	span.SetTraceID(pcommon.TraceID{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16})
	span.SetSpanID(pcommon.SpanID{1, 2, 3, 4, 5, 6, 7, 8})
	span.SetName("query")
	now := time.Now()
	span.SetStartTimestamp(pcommon.NewTimestampFromTime(now))
	span.SetEndTimestamp(pcommon.NewTimestampFromTime(now.Add(10 * time.Millisecond)))

	var marshaler ptrace.ProtoMarshaler
	body, err := marshaler.MarshalTraces(traces)
	require.NoError(t, err)

	dispatcher := &fakeDispatcher{}
	srv := NewServer(dispatcher)

	req := httptest.NewRequest(http.MethodPost, "/v1/traces", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/x-protobuf")
	w := httptest.NewRecorder()
	srv.ServeOTLP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Len(t, dispatcher.written, 1)
	attrs := dispatcher.written[0].ResourceAttributes
	require.Equal(t, "checkout-db-0", attrs["k8s.pod.name"])
	require.Equal(t, "prod", attrs["k8s.namespace.name"])
	require.Equal(t, "node-a", attrs["k8s.node.name"])
}

func TestServeOTLP_ProtobufBody_WritesSpans(t *testing.T) {
	t.Parallel()
	traces := newSingleSpanTraces(t)
	var marshaler ptrace.ProtoMarshaler
	body, err := marshaler.MarshalTraces(traces)
	require.NoError(t, err)

	store := &fakeDispatcher{}
	srv := NewServer(store)

	req := httptest.NewRequest(http.MethodPost, "/v1/traces", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/x-protobuf")
	w := httptest.NewRecorder()

	srv.ServeOTLP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Len(t, store.written, 1)
	require.Equal(t, "charge-card", store.written[0].Name)
	require.Equal(t, "checkout", store.written[0].ServiceName)
	require.Equal(t, "error", store.written[0].StatusCode)
	require.Equal(t, "", store.written[0].ParentSpanID)
}

func TestServeOTLP_JSONBody_WritesSpans(t *testing.T) {
	t.Parallel()
	traces := newSingleSpanTraces(t)
	var marshaler ptrace.JSONMarshaler
	body, err := marshaler.MarshalTraces(traces)
	require.NoError(t, err)

	store := &fakeDispatcher{}
	srv := NewServer(store)

	req := httptest.NewRequest(http.MethodPost, "/v1/traces", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.ServeOTLP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Len(t, store.written, 1)
}

func TestServeOTLP_MalformedBody_Returns400(t *testing.T) {
	t.Parallel()
	store := &fakeDispatcher{}
	srv := NewServer(store)

	req := httptest.NewRequest(http.MethodPost, "/v1/traces", strings.NewReader("not-a-valid-otlp-payload"))
	req.Header.Set("Content-Type", "application/x-protobuf")
	w := httptest.NewRecorder()

	srv.ServeOTLP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
	require.Empty(t, store.written)
}

func TestServeOTLP_WrongMethod_Returns405(t *testing.T) {
	t.Parallel()
	store := &fakeDispatcher{}
	srv := NewServer(store)

	req := httptest.NewRequest(http.MethodGet, "/v1/traces", nil)
	w := httptest.NewRecorder()

	srv.ServeOTLP(w, req)

	require.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

func TestServeOTLP_StoreWriteError_Returns500(t *testing.T) {
	t.Parallel()
	traces := newSingleSpanTraces(t)
	var marshaler ptrace.ProtoMarshaler
	body, err := marshaler.MarshalTraces(traces)
	require.NoError(t, err)

	store := &fakeDispatcher{dispatchErr: assert.AnError}
	srv := NewServer(store)

	req := httptest.NewRequest(http.MethodPost, "/v1/traces", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/x-protobuf")
	w := httptest.NewRecorder()

	srv.ServeOTLP(w, req)

	require.Equal(t, http.StatusInternalServerError, w.Code)
}
