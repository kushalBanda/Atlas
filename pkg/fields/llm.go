package fields

import "atlas/pkg/storage"

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
)

// ExtractLLMFields populates s's typed LLM columns from gen_ai.* keys in
// s.Attributes, if present. A span with no gen_ai.* keys is left untouched.
func ExtractLLMFields(s *storage.Span) {
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
