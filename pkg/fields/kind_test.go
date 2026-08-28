package fields

import (
	"testing"

	"github.com/stretchr/testify/require"

	"atlas/pkg/storage"
)

func TestExtractSpanKind(t *testing.T) {
	t.Parallel()

	t.Run("present", func(t *testing.T) {
		t.Parallel()
		s := storage.Span{Attributes: map[string]any{"openinference.span.kind": "LLM"}}
		ExtractSpanKind(&s)
		require.NotNil(t, s.SpanKind)
		require.Equal(t, "LLM", *s.SpanKind)
	})

	t.Run("absent stays nil", func(t *testing.T) {
		t.Parallel()
		s := storage.Span{}
		ExtractSpanKind(&s)
		require.Nil(t, s.SpanKind)
	})
}

func TestExtractLevel(t *testing.T) {
	t.Parallel()

	t.Run("present", func(t *testing.T) {
		t.Parallel()
		s := storage.Span{Attributes: map[string]any{"level": "ERROR"}}
		ExtractLevel(&s)
		require.NotNil(t, s.Level)
		require.Equal(t, "ERROR", *s.Level)
	})

	t.Run("absent stays nil", func(t *testing.T) {
		t.Parallel()
		s := storage.Span{}
		ExtractLevel(&s)
		require.Nil(t, s.Level)
	})
}
