package query

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"atlas/pkg/storage"
)

type fakeStore struct {
	storage.Store
	trace     *storage.Trace
	traceErr  error
	spans     []storage.Span
	spansErr  error
	traces    []storage.Trace
	tracesErr error
	gotFilter storage.TraceFilter
}

func (f *fakeStore) GetTrace(_ context.Context, _ string) (*storage.Trace, error) {
	if f.traceErr != nil {
		return nil, f.traceErr
	}
	return f.trace, nil
}

func (f *fakeStore) GetTraceSpans(_ context.Context, _ string) ([]storage.Span, error) {
	if f.spansErr != nil {
		return nil, f.spansErr
	}
	return f.spans, nil
}

func (f *fakeStore) ListTraces(_ context.Context, filter storage.TraceFilter) ([]storage.Trace, error) {
	f.gotFilter = filter
	if f.tracesErr != nil {
		return nil, f.tracesErr
	}
	return f.traces, nil
}

func requestWithTraceID(traceID string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/traces/"+traceID, nil)
	req.SetPathValue("trace_id", traceID)
	return req
}

func TestHandlers_ListTraces_ReturnsSummariesWithDuration(t *testing.T) {
	t.Parallel()
	first := time.Now().UTC()
	last := first.Add(2 * time.Second)
	store := &fakeStore{traces: []storage.Trace{{TraceID: "t1", FirstSeen: first, LastSeen: last}}}
	h := NewHandlers(store)

	w := httptest.NewRecorder()
	h.ListTraces(w, httptest.NewRequest(http.MethodGet, "/traces", nil))

	require.Equal(t, http.StatusOK, w.Code)
	var resp traceListResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Len(t, resp.Traces, 1)
	assert.Equal(t, "t1", resp.Traces[0].TraceID)
	assert.Equal(t, (2 * time.Second).Nanoseconds(), resp.Traces[0].DurationNano)
}

func TestHandlers_ListTraces_ParsesFilterParams(t *testing.T) {
	t.Parallel()
	store := &fakeStore{}
	h := NewHandlers(store)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/traces?has_root_cause=true&since=2026-01-01T00:00:00Z&limit=5", nil)
	h.ListTraces(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, store.gotFilter.HasRootCause)
	assert.True(t, *store.gotFilter.HasRootCause)
	require.NotNil(t, store.gotFilter.Since)
	assert.Equal(t, 2026, store.gotFilter.Since.Year())
	assert.Equal(t, 5, store.gotFilter.Limit)
}

func TestHandlers_ListTraces_InvalidParams_Returns400(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		query string
	}{
		{"bad has_root_cause", "has_root_cause=maybe"},
		{"bad since", "since=not-a-date"},
		{"bad limit", "limit=-1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			h := NewHandlers(&fakeStore{})
			w := httptest.NewRecorder()
			h.ListTraces(w, httptest.NewRequest(http.MethodGet, "/traces?"+tt.query, nil))
			require.Equal(t, http.StatusBadRequest, w.Code)
		})
	}
}

func TestHandlers_ListTraces_StoreError_Returns500(t *testing.T) {
	t.Parallel()
	store := &fakeStore{tracesErr: assert.AnError}
	h := NewHandlers(store)

	w := httptest.NewRecorder()
	h.ListTraces(w, httptest.NewRequest(http.MethodGet, "/traces", nil))

	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHandlers_GetTrace_ReturnsWaterfallAndVerdict(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC()
	rootSpanID := "root"
	reason := "earliest error"
	selfTimePct := 0.42

	store := &fakeStore{
		trace: &storage.Trace{
			TraceID: "t1", FirstSeen: now, LastSeen: now, ClosedAt: &now,
			LikelyRootCauseSpanID: &rootSpanID, Reason: &reason, SelfTimePct: &selfTimePct,
		},
		spans: []storage.Span{{TraceID: "t1", SpanID: "root", ServiceName: "s", Name: "n"}},
	}
	h := NewHandlers(store)

	w := httptest.NewRecorder()
	h.GetTrace(w, requestWithTraceID("t1"))

	require.Equal(t, http.StatusOK, w.Code)
	var resp traceResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, "t1", resp.Trace.TraceID)
	require.Equal(t, "earliest error", *resp.Trace.Reason)
	require.Len(t, resp.Spans, 1)
}

func TestHandlers_GetTrace_UnknownTraceID_Returns404(t *testing.T) {
	t.Parallel()
	store := &fakeStore{traceErr: storage.ErrTraceNotFound}
	h := NewHandlers(store)

	w := httptest.NewRecorder()
	h.GetTrace(w, requestWithTraceID("missing"))

	require.Equal(t, http.StatusNotFound, w.Code)
	var body map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Equal(t, "trace not found", body["error"])
}

func TestHandlers_GetTrace_StoreError_Returns500(t *testing.T) {
	t.Parallel()
	store := &fakeStore{traceErr: assert.AnError}
	h := NewHandlers(store)

	w := httptest.NewRecorder()
	h.GetTrace(w, requestWithTraceID("t1"))

	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHandlers_GetTrace_MissingTraceIDPathValue_Returns400(t *testing.T) {
	t.Parallel()
	store := &fakeStore{}
	h := NewHandlers(store)

	w := httptest.NewRecorder()
	h.GetTrace(w, requestWithTraceID(""))

	require.Equal(t, http.StatusBadRequest, w.Code)
}
