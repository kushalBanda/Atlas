package fields

import (
	"testing"

	"github.com/stretchr/testify/require"

	"atlas/pkg/storage"
)

func TestExtractLLMFields(t *testing.T) {
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
			s := storage.Span{SpanID: "s1", Attributes: tt.attrs}

			ExtractLLMFields(&s)

			require.Equal(t, tt.wantModel, s.LLMModel)
			require.Equal(t, tt.wantPromptTok, s.LLMPromptTokens)
			require.Equal(t, tt.wantComplTok, s.LLMCompletionTokens)
			require.Equal(t, tt.wantCost, s.LLMCost)
		})
	}
}

func ptr(s string) *string        { return &s }
func ptrInt(i int64) *int64       { return &i }
func ptrFloat(f float64) *float64 { return &f }
