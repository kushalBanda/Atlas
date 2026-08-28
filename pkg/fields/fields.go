// Package fields extracts self-describing data out of a span's raw
// Attributes into Span's typed columns, independent of which plugin module
// the span is routed to (see pkg/plugin.Registry.Dispatch). Each extractor
// owns one family of fields — llm.go today, future families (db, tool, ...)
// add their own file and register in All the same way.
package fields

import "atlas/pkg/storage"

// Extractor populates typed fields on s from s.Attributes. Extractors MUST
// be idempotent and touch only their own family of fields.
type Extractor func(s *storage.Span)

// All extractors, applied to every span at ingest regardless of atlas.module.
var All = []Extractor{
	ExtractSpanKind,
	ExtractLevel,
	ExtractLLMFields,
}

// Apply runs every registered extractor over spans in place.
func Apply(spans []storage.Span) {
	for i := range spans {
		for _, extract := range All {
			extract(&spans[i])
		}
	}
}
