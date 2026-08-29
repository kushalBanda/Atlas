import type { Span } from "../api/types";

// Self-null-checking pills (Langfuse pattern): a nil field renders no
// badge, never an empty placeholder.
export function hasLLMFields(span: Span): boolean {
  return (
    span.LLMModel !== null ||
    span.LLMPromptTokens !== null ||
    span.LLMCompletionTokens !== null ||
    span.LLMCost !== null ||
    span.LLMTemperature !== null ||
    span.LLMTopP !== null ||
    span.LLMMaxTokens !== null ||
    span.LLMTimeToFirstTokenNano !== null ||
    span.LLMPromptID !== null
  );
}
