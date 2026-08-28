package fields

import "atlas/pkg/storage"

// Attribute keys read into general (non-LLM-specific) span fields.
const (
	attrSpanKind = "openinference.span.kind" // Atlas convention: LLM/TOOL/CHAIN/RETRIEVER/...
	attrLevel    = "level"                   // DEBUG/DEFAULT/WARNING/ERROR
)

// ExtractSpanKind populates s.SpanKind from attrSpanKind, if present.
func ExtractSpanKind(s *storage.Span) {
	if v := stringAttr(s.Attributes, attrSpanKind); v != "" {
		s.SpanKind = &v
	}
}

// ExtractLevel populates s.Level from attrLevel, if present.
func ExtractLevel(s *storage.Span) {
	if v := stringAttr(s.Attributes, attrLevel); v != "" {
		s.Level = &v
	}
}
