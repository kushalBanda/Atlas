import { useState } from "react";
import type { Span } from "../api/types";
import { LLMBadges, MetaBadges } from "./LLMBadges";
import { JsonTree, JsonRaw } from "./JsonTree";
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

function CopyButton({ text }: { text: string }) {
  const [copied, setCopied] = useState(false);
  return (
    <button
      type="button"
      onClick={async () => {
        try {
          await navigator.clipboard.writeText(text);
          setCopied(true);
          setTimeout(() => setCopied(false), 1200);
        } catch {
          // Clipboard may be unavailable (permissions, insecure context); no-op.
        }
      }}
      className="text-[10px] text-text-faint hover:text-text-primary"
    >
      {copied ? "Copied" : "Copy"}
    </button>
  );
}

// Section header shared by Prompt/Completion/Attributes blocks: label on the
// left, optional Tree/Raw toggle plus a copy-to-clipboard action on the right.
function SectionHeader({
  label,
  copyText,
  view,
  onViewChange,
}: {
  label: string;
  copyText: string;
  view?: "tree" | "raw";
  onViewChange?: (view: "tree" | "raw") => void;
}) {
  return (
    <div className="mb-1.5 flex items-center justify-between">
      <div className="text-[11px] uppercase tracking-wide text-text-faint">{label}</div>
      <div className="flex items-center gap-2">
        {view && onViewChange && (
          <div className="flex overflow-hidden rounded border border-border">
            {(["tree", "raw"] as const).map((v) => (
              <button
                key={v}
                type="button"
                onClick={() => onViewChange(v)}
                className={`px-1.5 py-0.5 text-[10px] capitalize ${
                  view === v ? "bg-accent-dim text-accent" : "text-text-faint hover:text-text-primary"
                }`}
              >
                {v}
              </button>
            ))}
          </div>
        )}
        <CopyButton text={copyText} />
      </div>
    </div>
  );
}

export function SpanDetailPanel({ span, onClose }: SpanDetailPanelProps) {
  const isLLM = hasLLMFields(span);
  const prompt = asText(span.Attributes["gen_ai.prompt"]);
  const completion = asText(span.Attributes["gen_ai.completion"]);
  const [attributesView, setAttributesView] = useState<"tree" | "raw">("tree");
  const attributesRaw = JSON.stringify(span.Attributes, null, 2);

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
              <SectionHeader label="Prompt" copyText={prompt} />
              <pre className="max-h-[110px] overflow-auto rounded border border-border bg-canvas p-2 font-plex-mono text-[11.5px] leading-relaxed whitespace-pre-wrap break-words text-text-muted">
                {prompt}
              </pre>
            </div>
          )}
          {completion !== null && (
            <div className="border-b border-border px-4 py-3">
              <SectionHeader label="Completion" copyText={completion} />
              <pre className="max-h-[110px] overflow-auto rounded border border-border bg-canvas p-2 font-plex-mono text-[11.5px] leading-relaxed whitespace-pre-wrap break-words text-text-muted">
                {completion}
              </pre>
            </div>
          )}
        </>
      ) : (
        <div className="overflow-auto px-4 py-3">
          <SectionHeader
            label="Attributes"
            copyText={attributesRaw}
            view={attributesView}
            onViewChange={setAttributesView}
          />
          <div className="max-h-[420px] overflow-auto rounded border border-border bg-canvas p-2">
            {attributesView === "tree" ? (
              <JsonTree data={span.Attributes} />
            ) : (
              <JsonRaw text={attributesRaw} />
            )}
          </div>
        </div>
      )}
    </div>
  );
}
