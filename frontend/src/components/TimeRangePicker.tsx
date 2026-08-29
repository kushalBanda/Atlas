import { useEffect, useRef, useState } from "react";

const RELATIVE_OPTIONS = [
  { label: "5m", ms: 5 * 60 * 1000 },
  { label: "15m", ms: 15 * 60 * 1000 },
  { label: "30m", ms: 30 * 60 * 1000 },
  { label: "1h", ms: 60 * 60 * 1000 },
  { label: "6h", ms: 6 * 60 * 60 * 1000 },
  { label: "1d", ms: 24 * 60 * 60 * 1000 },
  { label: "3d", ms: 3 * 24 * 60 * 60 * 1000 },
  { label: "1w", ms: 7 * 24 * 60 * 60 * 1000 },
];

export interface TimeRangePickerProps {
  sinceMs: number;
  onSinceChange: (ms: number) => void;
}

function toDatetimeLocalValue(ms: number): string {
  const d = new Date(Date.now() - ms);
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

function labelForSince(sinceMs: number): string {
  const match = RELATIVE_OPTIONS.find((opt) => opt.ms === sinceMs);
  if (match) return match.label;
  const start = new Date(Date.now() - sinceMs);
  return start.toLocaleString(undefined, {
    month: "short",
    day: "numeric",
    hour: "numeric",
    minute: "2-digit",
  });
}

export function TimeRangePicker({ sinceMs, onSinceChange }: TimeRangePickerProps) {
  const [open, setOpen] = useState(false);
  const [customMode, setCustomMode] = useState(false);
  const [customValue, setCustomValue] = useState(() => toDatetimeLocalValue(sinceMs));
  const rootRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!open) return;
    function onPointerDown(e: MouseEvent) {
      if (rootRef.current && !rootRef.current.contains(e.target as Node)) {
        setOpen(false);
        setCustomMode(false);
      }
    }
    function onKeyDown(e: KeyboardEvent) {
      if (e.key === "Escape") {
        setOpen(false);
        setCustomMode(false);
      }
    }
    document.addEventListener("mousedown", onPointerDown);
    document.addEventListener("keydown", onKeyDown);
    return () => {
      document.removeEventListener("mousedown", onPointerDown);
      document.removeEventListener("keydown", onKeyDown);
    };
  }, [open]);

  function applyRelative(ms: number) {
    onSinceChange(ms);
    setOpen(false);
    setCustomMode(false);
  }

  function applyCustom() {
    const startMs = new Date(customValue).getTime();
    if (Number.isNaN(startMs)) return;
    const ms = Math.max(Date.now() - startMs, 1000);
    onSinceChange(ms);
    setOpen(false);
    setCustomMode(false);
  }

  return (
    <div ref={rootRef} className="relative">
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        className="flex items-center gap-1.5 rounded-md border border-border bg-surface px-2.5 py-1.5 text-xs text-text-muted hover:border-text-faint"
      >
        Since
        <span className="font-plex-mono text-text-primary">{labelForSince(sinceMs)}</span>
      </button>

      <div
        className={`grid transition-[grid-template-rows,opacity] duration-150 ease-out motion-reduce:transition-none ${
          open ? "grid-rows-[1fr] opacity-100" : "grid-rows-[0fr] opacity-0"
        }`}
        style={{ position: "absolute", top: "calc(100% + 4px)", left: 0, zIndex: 20 }}
      >
        <div className="overflow-hidden">
          <div className="w-[180px] rounded-md border border-border bg-elevated py-1 shadow-lg">
            {RELATIVE_OPTIONS.map((opt) => (
              <button
                key={opt.label}
                type="button"
                onClick={() => applyRelative(opt.ms)}
                className={`flex w-full items-center px-3 py-1.5 text-left text-xs hover:bg-surface ${
                  opt.ms === sinceMs ? "text-accent" : "text-text-muted"
                }`}
              >
                Last {opt.label}
              </button>
            ))}
            <div className="my-1 border-t border-border" />
            {!customMode ? (
              <button
                type="button"
                onClick={() => setCustomMode(true)}
                className="flex w-full items-center px-3 py-1.5 text-left text-xs text-text-muted hover:bg-surface"
              >
                Custom
              </button>
            ) : (
              <div className="flex flex-col gap-1.5 px-3 py-1.5">
                <input
                  type="datetime-local"
                  value={customValue}
                  onChange={(e) => setCustomValue(e.target.value)}
                  className="rounded border border-border bg-canvas px-1.5 py-1 text-[11px] text-text-primary outline-none"
                />
                <button
                  type="button"
                  onClick={applyCustom}
                  className="rounded bg-accent-dim px-2 py-1 text-[11px] text-accent hover:bg-accent/20"
                >
                  Apply
                </button>
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
