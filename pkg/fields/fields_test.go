package fields

import (
	"testing"

	"github.com/stretchr/testify/require"

	"atlas/pkg/storage"
)

func TestApply_RunsExtractorsOverEverySpan(t *testing.T) {
	t.Parallel()
	spans := []storage.Span{
		{SpanID: "s1", Attributes: map[string]any{"gen_ai.request.model": "gpt-4"}},
		{SpanID: "s2", Attributes: map[string]any{"some.other.key": "value"}},
	}

	Apply(spans)

	require.NotNil(t, spans[0].LLMModel)
	require.Equal(t, "gpt-4", *spans[0].LLMModel)
	require.Nil(t, spans[1].LLMModel)
}
