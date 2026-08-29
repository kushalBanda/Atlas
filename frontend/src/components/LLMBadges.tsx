import type { Span } from "../api/types";

export interface LLMBadgesProps {
  span: Span;
}

function Badge({ label, value, className }: { label: string; value: string; className?: string }) {
  return (
    <span className={`flex items-center gap-1 rounded px-2 py-0.5 text-[11px] text-text-muted ${className ?? "bg-elevated"}`}>
      {label} <b className="font-semibold text-text-primary">{value}</b>
    </span>
  );
}

export function LLMBadges({ span }: LLMBadgesProps) {
  return (
    <div className="flex flex-wrap gap-1.5 border-b border-border px-4 py-3">
      {span.LLMModel !== null && <Badge label="Model" value={span.LLMModel} className="bg-[#152531] text-[#a3c9e8]" />}
      {span.LLMPromptTokens !== null && <Badge label="Prompt" value={`${span.LLMPromptTokens} tok`} />}
      {span.LLMCompletionTokens !== null && <Badge label="Completion" value={`${span.LLMCompletionTokens} tok`} />}
      {span.LLMCost !== null && <Badge label="Cost" value={`$${span.LLMCost.toFixed(4)}`} className="bg-[#1e2815] text-[#c9e0a3]" />}
      {span.LLMTemperature !== null && <Badge label="Temp" value={String(span.LLMTemperature)} />}
      {span.LLMTopP !== null && <Badge label="Top P" value={String(span.LLMTopP)} />}
      {span.LLMMaxTokens !== null && <Badge label="Max tokens" value={String(span.LLMMaxTokens)} />}
      {span.LLMTimeToFirstTokenNano !== null && (
        <Badge label="TTFT" value={`${Math.round(span.LLMTimeToFirstTokenNano / 1_000_000)}ms`} />
      )}
      {span.LLMPromptName !== null && <Badge label="Prompt name" value={span.LLMPromptName} />}
      {span.LLMPromptVersion !== null && <Badge label="Prompt version" value={String(span.LLMPromptVersion)} />}
    </div>
  );
}

// SpanKind/Level apply to any span, not just LLM ones (Gate 3): kept as a
// separate export so SpanDetailPanel can show it regardless of hasLLMFields.
export function MetaBadges({ span }: { span: Span }) {
  if (span.SpanKind === null && span.Level === null) return null;
  return (
    <div className="flex flex-wrap gap-1.5 border-b border-border px-4 py-3">
      {span.SpanKind !== null && <Badge label="Kind" value={span.SpanKind} />}
      {span.Level !== null && (
        <Badge
          label="Level"
          value={span.Level}
          className={span.Level === "ERROR" ? "bg-error-dim text-[#f3a2a5]" : undefined}
        />
      )}
    </div>
  );
}
