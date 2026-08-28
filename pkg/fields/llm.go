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
	attrTemperature      = "gen_ai.request.temperature"
	attrTopP             = "gen_ai.request.top_p"
	attrMaxTokens        = "gen_ai.request.max_tokens"
	attrTimeToFirstToken = "gen_ai.usage.time_to_first_token_ms"
	attrPromptID         = "gen_ai.prompt.id"
	attrPromptName       = "gen_ai.prompt.name"
	attrPromptVersion    = "gen_ai.prompt.version"
)

// usageDetailKeys and costDetailKeys are gen_ai.* attributes beyond the
// core prompt/completion/cost fields, collected into LLMUsageDetails and
// LLMCostDetails respectively when present. Provider-specific (cache reads,
// reasoning tokens, ...), so kept as an open map rather than more columns.
var (
	usageDetailKeys = map[string]string{
		"gen_ai.usage.cache_read_tokens":  "cache_read_tokens",
		"gen_ai.usage.cache_write_tokens": "cache_write_tokens",
		"gen_ai.usage.reasoning_tokens":   "reasoning_tokens",
		"gen_ai.usage.audio_tokens":       "audio_tokens",
	}
	costDetailKeys = map[string]string{
		"gen_ai.usage.cost.input":  "input",
		"gen_ai.usage.cost.output": "output",
		"gen_ai.usage.cost.cache":  "cache",
	}
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
	if v, ok := floatAttr(s.Attributes, attrTemperature); ok {
		s.LLMTemperature = &v
	}
	if v, ok := floatAttr(s.Attributes, attrTopP); ok {
		s.LLMTopP = &v
	}
	if v, ok := intAttr(s.Attributes, attrMaxTokens); ok {
		s.LLMMaxTokens = &v
	}
	if v, ok := floatAttr(s.Attributes, attrTimeToFirstToken); ok {
		nanos := int64(v * 1e6)
		s.LLMTimeToFirstTokenNano = &nanos
	}
	if v := stringAttr(s.Attributes, attrPromptID); v != "" {
		s.LLMPromptID = &v
	}
	if v := stringAttr(s.Attributes, attrPromptName); v != "" {
		s.LLMPromptName = &v
	}
	if v, ok := intAttr(s.Attributes, attrPromptVersion); ok {
		s.LLMPromptVersion = &v
	}
	if m := detailMap(s.Attributes, usageDetailKeys); m != nil {
		s.LLMUsageDetails = m
	}
	if m := detailMap(s.Attributes, costDetailKeys); m != nil {
		s.LLMCostDetails = m
	}
}

// detailMap collects attrs[attrKey] under outKey for every entry in keys
// that's actually present, or nil if none are.
func detailMap(attrs map[string]any, keys map[string]string) map[string]any {
	var out map[string]any
	for attrKey, outKey := range keys {
		if v, ok := attrs[attrKey]; ok {
			if out == nil {
				out = make(map[string]any)
			}
			out[outKey] = v
		}
	}
	return out
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
