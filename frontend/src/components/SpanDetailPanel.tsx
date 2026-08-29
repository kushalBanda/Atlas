import { useEffect, useRef, useState, type ReactNode } from "react";
import type { Span } from "../api/types";
import { LLMBadges, MetaBadges } from "./LLMBadges";
import { JsonTree, JsonRaw } from "./JsonTree";
import { hasLLMFields } from "../lib/llm";

export interface SpanDetailPanelProps {
  span: Span | null;
  onClose: () => void;
}

const WIDTH_STORAGE_KEY = "atlas.spanDetailPanel.width";
const MIN_WIDTH = 320;
const MAX_WIDTH = 800;
const DEFAULT_WIDTH = 380;

function readStoredWidth(): number {
  try {
    const raw = Number(localStorage.getItem(WIDTH_STORAGE_KEY));
    if (raw >= MIN_WIDTH && raw <= MAX_WIDTH) return raw;
  } catch {
    // Storage may be unavailable (private mode); fall back to default.
  }
  return DEFAULT_WIDTH;
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

// Collapsible block shared by Prompt/Completion/Attributes. Animates open/
// closed via a grid-template-rows 0fr↔1fr transition on an always-mounted
// wrapper — the browser can animate that smoothly because it's a real layout
// property, unlike height (which has no intrinsic "auto" to animate to)
// or a conditional unmount (which just snaps, no transition at all).
function CollapsibleSection({
  label,
  copyText,
  view,
  onViewChange,
  defaultOpen = true,
  children,
}: {
  label: string;
  copyText: string;
  view?: "tree" | "raw";
  onViewChange?: (view: "tree" | "raw") => void;
  defaultOpen?: boolean;
  children: ReactNode;
}) {
  const [open, setOpen] = useState(defaultOpen);
  return (
    <div className="border-b border-border px-4 py-3">
      <div className="mb-1.5 flex flex-shrink-0 items-center justify-between">
        <button
          type="button"
          onClick={() => setOpen((v) => !v)}
          className="flex items-center gap-1.5 text-[11px] uppercase tracking-wide text-text-faint hover:text-text-primary"
        >
          <span
            aria-hidden="true"
            className={`inline-block text-[9px] transition-transform duration-150 ease-out ${
              open ? "rotate-0" : "-rotate-90"
            }`}
          >
            ▾
          </span>
          {label}
        </button>
        <div
          className={`flex items-center gap-2 transition-opacity duration-150 ${
            open ? "opacity-100" : "pointer-events-none opacity-0"
          }`}
        >
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
      <div
        className="grid transition-[grid-template-rows] duration-200 ease-out motion-reduce:transition-none"
        style={{ gridTemplateRows: open ? "1fr" : "0fr" }}
      >
        <div className="overflow-hidden">{children}</div>
      </div>
    </div>
  );
}

export function SpanDetailPanel({ span, onClose }: SpanDetailPanelProps) {
  // Keep the last non-null span rendered while the panel slides shut, so
  // closing animates instead of snapping to empty content mid-transition.
  const [displaySpan, setDisplaySpan] = useState(span);
  useEffect(() => {
    if (span) setDisplaySpan(span);
  }, [span]);
  const open = span !== null;

  const isLLM = displaySpan ? hasLLMFields(displaySpan) : false;
  const prompt = displaySpan ? asText(displaySpan.Attributes["gen_ai.prompt"]) : null;
  const completion = displaySpan ? asText(displaySpan.Attributes["gen_ai.completion"]) : null;
  const [attributesView, setAttributesView] = useState<"tree" | "raw">("tree");
  const attributesRaw = displaySpan ? JSON.stringify(displaySpan.Attributes, null, 2) : "";

  const [width, setWidth] = useState(readStoredWidth);
  const [resizing, setResizing] = useState(false);
  const widthRef = useRef(width);
  widthRef.current = width;

  useEffect(() => {
    if (!resizing) return;
    const onMove = (e: MouseEvent) => {
      const next = Math.min(MAX_WIDTH, Math.max(MIN_WIDTH, window.innerWidth - e.clientX));
      setWidth(next);
    };
    const onUp = () => {
      setResizing(false);
      try {
        localStorage.setItem(WIDTH_STORAGE_KEY, String(widthRef.current));
      } catch {
        // Storage may be unavailable (private mode); resize just won't persist.
      }
    };
    document.addEventListener("mousemove", onMove);
    document.addEventListener("mouseup", onUp);
    return () => {
      document.removeEventListener("mousemove", onMove);
      document.removeEventListener("mouseup", onUp);
    };
  }, [resizing]);

  if (!displaySpan) return null;

  return (
    <div
      className={`fixed right-0 top-0 bottom-0 flex flex-col border-l border-border bg-surface shadow-[-8px_0_24px_rgba(0,0,0,0.4)] transition-transform duration-200 ease-out motion-reduce:transition-none ${
        open ? "translate-x-0" : "translate-x-full"
      }`}
      style={{ width }}
      inert={open ? undefined : true}
    >
      <div
        onMouseDown={() => setResizing(true)}
        className={`absolute left-0 top-0 bottom-0 w-1 -translate-x-1/2 cursor-col-resize ${
          resizing ? "bg-accent" : "hover:bg-accent/50"
        }`}
      />

      <header className="flex flex-shrink-0 items-center justify-between border-b border-border px-4 py-3">
        <div>
          <div className="text-sm font-semibold text-text-primary">{displaySpan.Name}</div>
          <div className="font-plex-mono text-[11px] text-text-faint">{displaySpan.ServiceName}</div>
        </div>
        <button type="button" onClick={onClose} className="text-text-faint hover:text-text-primary">
          &times;
        </button>
      </header>

      <div className="flex-shrink-0">
        <MetaBadges span={displaySpan} />
        {isLLM && <LLMBadges span={displaySpan} />}
      </div>

      {isLLM ? (
        <div className="min-h-0 flex-1 overflow-y-auto">
          {prompt !== null && (
            <CollapsibleSection label="Prompt" copyText={prompt}>
              <pre className="max-h-[45vh] overflow-auto rounded border border-border bg-canvas p-2.5 font-plex-mono text-[11.5px] leading-relaxed whitespace-pre-wrap break-words text-text-muted">
                {prompt}
              </pre>
            </CollapsibleSection>
          )}
          {completion !== null && (
            <CollapsibleSection label="Completion" copyText={completion}>
              <pre className="max-h-[45vh] overflow-auto rounded border border-border bg-canvas p-2.5 font-plex-mono text-[11.5px] leading-relaxed whitespace-pre-wrap break-words text-text-muted">
                {completion}
              </pre>
            </CollapsibleSection>
          )}
        </div>
      ) : (
        <div className="min-h-0 flex-1 overflow-y-auto">
          <CollapsibleSection
            label="Attributes"
            copyText={attributesRaw}
            view={attributesView}
            onViewChange={setAttributesView}
          >
            <div className="max-h-[60vh] overflow-auto rounded border border-border bg-canvas p-2.5">
              {attributesView === "tree" ? (
                <JsonTree data={displaySpan.Attributes} />
              ) : (
                <JsonRaw text={attributesRaw} />
              )}
            </div>
          </CollapsibleSection>
        </div>
      )}
    </div>
  );
}
