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
	trace    *storage.Trace
	traceErr error
	spans    []storage.Span
	spansErr error
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

func requestWithTraceID(traceID string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/traces/"+traceID, nil)
	req.SetPathValue("trace_id", traceID)
	return req
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
