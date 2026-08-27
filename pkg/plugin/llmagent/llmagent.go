// Package llmagent is the second plugin module: prompt/tool-call/agent
// spans, carrying gen_ai.* semantic-convention attributes. Structurally
// identical to otelcore aside from Name() and gen_ai.* extraction.
package llmagent

import (
	"context"
	"fmt"

	"atlas/pkg/api"
	"atlas/pkg/storage"
)

// gen_ai.* semantic-convention attribute keys read out of Span.Attributes
// into the typed LLM columns.
const (
	attrRequestModel     = "gen_ai.request.model"
	attrResponseModel    = "gen_ai.response.model"
	attrPromptTokens     = "gen_ai.usage.prompt_tokens"
	attrCompletionTokens = "gen_ai.usage.completion_tokens"
	attrInputTokens      = "gen_ai.usage.input_tokens"
	attrOutputTokens     = "gen_ai.usage.output_tokens"
	attrCost             = "gen_ai.usage.cost"
	attrPrompt           = "gen_ai.prompt"
	attrCompletion       = "gen_ai.completion"
)

// Name is the module name spans use in their atlas.module resource
// attribute to route to this module instead of the otelcore default.
const Name = "llmagent"

// Module is the llmagent plugin module.
type Module struct {
	store storage.Store
}

// New returns an llmagent Module writing to store.
func New(store storage.Store) *Module {
	return &Module{store: store}
}

// Name implements plugin.Module.
func (m *Module) Name() string { return Name }

// RegisterSchema implements plugin.Module. No-op: prompt/tool-call data
// lives in the existing spans.attributes JSON column (gen_ai.* keys), not
// a new table.
func (m *Module) RegisterSchema(_ storage.SchemaRegistrar) error { return nil }

// RegisterRoutes implements plugin.Module. No-op in v1: llmagent has no
// routes of its own yet (query still goes through the shared /traces path).
func (m *Module) RegisterRoutes(_ api.RouteRegistrar) error { return nil }

// HandleSpans implements plugin.Module: extracts gen_ai.* attributes into
// the typed LLM columns, then writes spans to storage, same path as otelcore.
func (m *Module) HandleSpans(ctx context.Context, spans []storage.Span) error {
	for i := range spans {
		extractLLMFields(&spans[i])
	}
	if err := m.store.WriteSpans(ctx, spans); err != nil {
		return fmt.Errorf("llmagent writing spans: %w", err)
	}
	return nil
}

// extractLLMFields populates s's typed LLM columns from gen_ai.* keys in s.Attributes.
func extractLLMFields(s *storage.Span) {
	if model := stringAttr(s.Attributes, attrRequestModel, attrResponseModel); model != "" {
		s.LLMModel = &model
	}
	if v, ok := intAttr(s.Attributes, attrPromptTokens, attrInputTokens); ok {
		s.LLMPromptTokens = &v
	}
	if v, ok := intAttr(s.Attributes, attrCompletionTokens, attrOutputTokens); ok {
		s.LLMCompletionTokens = &v
	}
	if v, ok := floatAttr(s.Attributes, attrCost); ok {
		s.LLMCost = &v
	}
	if v := stringAttr(s.Attributes, attrPrompt); v != "" {
		s.LLMPrompt = &v
	}
	if v := stringAttr(s.Attributes, attrCompletion); v != "" {
		s.LLMCompletion = &v
	}
}

// stringAttr returns the first non-empty string value found under keys.
func stringAttr(attrs map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, ok := attrs[k].(string); ok && v != "" {
			return v
		}
	}
	return ""
}

// intAttr returns the first int-like value found under keys.
func intAttr(attrs map[string]any, keys ...string) (int64, bool) {
	for _, k := range keys {
		switch v := attrs[k].(type) {
		case float64:
			return int64(v), true
		case int64:
			return v, true
		}
	}
	return 0, false
}

// floatAttr returns the first float-like value found under keys.
func floatAttr(attrs map[string]any, keys ...string) (float64, bool) {
	for _, k := range keys {
		switch v := attrs[k].(type) {
		case float64:
			return v, true
		case int64:
			return float64(v), true
		}
	}
	return 0, false
}
