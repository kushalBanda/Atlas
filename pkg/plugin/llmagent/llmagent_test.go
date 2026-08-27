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

func TestModule_HandleSpans_ExtractsLLMFields(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		attrs         map[string]any
		wantModel     *string
		wantPromptTok *int64
		wantComplTok  *int64
		wantCost      *float64
	}{
		"full gen_ai attributes": {
			attrs: map[string]any{
				"gen_ai.request.model":           "gpt-4",
				"gen_ai.usage.prompt_tokens":     float64(10),
				"gen_ai.usage.completion_tokens": float64(20),
				"gen_ai.usage.cost":              float64(0.01),
			},
			wantModel:     ptr("gpt-4"),
			wantPromptTok: ptrInt(10),
			wantComplTok:  ptrInt(20),
			wantCost:      ptrFloat(0.01),
		},
		"response model falls back when request model absent": {
			attrs:     map[string]any{"gen_ai.response.model": "gpt-4o"},
			wantModel: ptr("gpt-4o"),
		},
		"input/output token aliases": {
			attrs: map[string]any{
				"gen_ai.usage.input_tokens":  float64(5),
				"gen_ai.usage.output_tokens": float64(7),
			},
			wantPromptTok: ptrInt(5),
			wantComplTok:  ptrInt(7),
		},
		"no gen_ai attributes stays nil": {
			attrs: map[string]any{"some.other.key": "value"},
		},
		"empty attributes stays nil": {
			attrs: nil,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			store := &fakeStore{}
			m := New(store)

			require.NoError(t, m.HandleSpans(context.Background(), []storage.Span{
				{SpanID: "s1", Attributes: tt.attrs},
			}))

			require.Len(t, store.written, 1)
			got := store.written[0]
			require.Equal(t, tt.wantModel, got.LLMModel)
			require.Equal(t, tt.wantPromptTok, got.LLMPromptTokens)
			require.Equal(t, tt.wantComplTok, got.LLMCompletionTokens)
			require.Equal(t, tt.wantCost, got.LLMCost)
		})
	}
}

func ptr(s string) *string        { return &s }
func ptrInt(i int64) *int64       { return &i }
func ptrFloat(f float64) *float64 { return &f }

func TestModule_HandleSpans_PropagatesStoreError(t *testing.T) {
	t.Parallel()
	store := &fakeStore{writeErr: errors.New("write failed")}
	m := New(store)

	err := m.HandleSpans(context.Background(), []storage.Span{{SpanID: "s1"}})
	require.Error(t, err)
}
