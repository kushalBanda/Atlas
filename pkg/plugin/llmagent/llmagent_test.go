package llmagent

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"atlas/pkg/storage"
)

type fakeStore struct {
	storage.Store
	writeErr error
	written  []storage.Span
}

func (f *fakeStore) WriteSpans(_ context.Context, spans []storage.Span) error {
	if f.writeErr != nil {
		return f.writeErr
	}
	f.written = append(f.written, spans...)
	return nil
}

func TestModule_Name(t *testing.T) {
	t.Parallel()
	require.Equal(t, "llmagent", New(&fakeStore{}).Name())
}

func TestModule_RegisterSchema_NoOp(t *testing.T) {
	t.Parallel()
	require.NoError(t, New(&fakeStore{}).RegisterSchema(nil))
}

func TestModule_RegisterRoutes_NoOp(t *testing.T) {
	t.Parallel()
	require.NoError(t, New(&fakeStore{}).RegisterRoutes(nil))
}

func TestModule_HandleSpans_WritesToStore(t *testing.T) {
	t.Parallel()
	store := &fakeStore{}
	m := New(store)

	spans := []storage.Span{
		{SpanID: "s1", Attributes: map[string]any{"gen_ai.system": "openai", "gen_ai.request.model": "gpt-4"}},
	}
	require.NoError(t, m.HandleSpans(context.Background(), spans))
	require.Len(t, store.written, 1)
	require.Equal(t, "openai", store.written[0].Attributes["gen_ai.system"])
}

func TestModule_HandleSpans_PropagatesStoreError(t *testing.T) {
	t.Parallel()
	store := &fakeStore{writeErr: errors.New("write failed")}
	m := New(store)

	err := m.HandleSpans(context.Background(), []storage.Span{{SpanID: "s1"}})
	require.Error(t, err)
}
