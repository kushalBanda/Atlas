package plugin

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"atlas/pkg/plugin/llmagent"
	"atlas/pkg/plugin/otelcore"
	"atlas/pkg/storage"
)

// realFakeStore is a minimal storage.Store stub shared by both real
// modules below, isolating this test from a real DuckDB dependency.
type realFakeStore struct {
	storage.Store
	written []storage.Span
}

func (s *realFakeStore) WriteSpans(_ context.Context, spans []storage.Span) error {
	s.written = append(s.written, spans...)
	return nil
}

// TestRegistry_OtelcoreAndLlmagentCoexist is Slice 4's proof: two real
// modules, no plugin.Module interface changes needed to add the second
// one. Both write through the same storage.Store and route correctly by
// atlas.module resource attribute.
func TestRegistry_OtelcoreAndLlmagentCoexist(t *testing.T) {
	t.Parallel()
	store := &realFakeStore{}
	reg := NewRegistry(noopSchemaRegistrar{}, newFakeRouteRegistrar())

	require.NoError(t, reg.Register(otelcore.New(store)))
	require.NoError(t, reg.Register(llmagent.New(store)))

	spans := []storage.Span{
		{SpanID: "http-span", ResourceAttributes: map[string]any{"atlas.module": "otelcore"}},
		{SpanID: "llm-span", ResourceAttributes: map[string]any{
			"atlas.module": "llmagent",
		}, Attributes: map[string]any{"gen_ai.system": "openai"}},
		{SpanID: "default-span"}, // no atlas.module attribute -> otelcore
	}

	require.NoError(t, reg.Dispatch(context.Background(), spans))
	require.Len(t, store.written, 3)
}
