import type { Span } from "../api/types";
import { LLMBadges, MetaBadges } from "./LLMBadges";
import { hasLLMFields } from "../lib/llm";

export interface SpanDetailPanelProps {
  span: Span;
  onClose: () => void;
}

// Prompt/completion text is deliberately not a typed column (see
// CLAUDE.md), read it straight from Attributes.
function asText(value: unknown): string | null {
  if (typeof value === "string") return value;
  if (value === undefined || value === null) return null;
  return JSON.stringify(value, null, 2);
}

export function SpanDetailPanel({ span, onClose }: SpanDetailPanelProps) {
  const isLLM = hasLLMFields(span);
  const prompt = asText(span.Attributes["gen_ai.prompt"]);
  const completion = asText(span.Attributes["gen_ai.completion"]);

  return (
    <div className="fixed right-0 top-0 bottom-0 flex w-[380px] flex-col border-l border-border bg-surface shadow-[-8px_0_24px_rgba(0,0,0,0.4)]">
      <header className="flex items-center justify-between border-b border-border px-4 py-3">
        <div>
          <div className="text-sm font-semibold text-text-primary">{span.Name}</div>
          <div className="font-plex-mono text-[11px] text-text-faint">{span.ServiceName}</div>
        </div>
        <button type="button" onClick={onClose} className="text-text-faint hover:text-text-primary">
          &times;
        </button>
      </header>

      <MetaBadges span={span} />
      {isLLM && <LLMBadges span={span} />}

      {isLLM ? (
        <>
          {prompt !== null && (
            <div className="border-b border-border px-4 py-3">
              <div className="mb-1.5 text-[11px] uppercase tracking-wide text-text-faint">Prompt</div>
              <pre className="max-h-[110px] overflow-auto rounded border border-border bg-canvas p-2 font-plex-mono text-[11.5px] whitespace-pre-wrap text-text-muted">
                {prompt}
              </pre>
            </div>
          )}
          {completion !== null && (
            <div className="border-b border-border px-4 py-3">
              <div className="mb-1.5 text-[11px] uppercase tracking-wide text-text-faint">Completion</div>
              <pre className="max-h-[110px] overflow-auto rounded border border-border bg-canvas p-2 font-plex-mono text-[11.5px] whitespace-pre-wrap text-text-muted">
                {completion}
              </pre>
            </div>
          )}
        </>
      ) : (
        <div className="overflow-auto px-4 py-3">
          <div className="mb-1.5 text-[11px] uppercase tracking-wide text-text-faint">Attributes</div>
          <pre className="rounded border border-border bg-canvas p-2 font-plex-mono text-[11.5px] whitespace-pre-wrap text-text-muted">
            {JSON.stringify(span.Attributes, null, 2)}
          </pre>
        </div>
      )}
    </div>
  );
}
