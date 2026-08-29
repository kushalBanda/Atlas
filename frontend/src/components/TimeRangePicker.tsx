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
  untilMs?: number;
  onRangeChange: (sinceMs: number, untilMs?: number) => void;
}

function toDatetimeLocalValue(date: Date): string {
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}`;
}

function formatAbsolute(ms: number): string {
  return new Date(ms).toLocaleString(undefined, {
    month: "short",
    day: "numeric",
    hour: "numeric",
    minute: "2-digit",
  });
}

function labelForRange(sinceMs: number, untilMs?: number): string {
  if (untilMs !== undefined) {
    return `${formatAbsolute(Date.now() - sinceMs)} → ${formatAbsolute(untilMs)}`;
  }
  const match = RELATIVE_OPTIONS.find((opt) => opt.ms === sinceMs);
  if (match) return match.label;
  return formatAbsolute(Date.now() - sinceMs);
}

export function TimeRangePicker({ sinceMs, untilMs, onRangeChange }: TimeRangePickerProps) {
  const [open, setOpen] = useState(false);
  const [customMode, setCustomMode] = useState(false);
  const [startValue, setStartValue] = useState(() => toDatetimeLocalValue(new Date(Date.now() - sinceMs)));
  const [endValue, setEndValue] = useState(() => toDatetimeLocalValue(untilMs ? new Date(untilMs) : new Date()));
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
    onRangeChange(ms, undefined);
    setOpen(false);
    setCustomMode(false);
  }

  function applyCustom() {
    const startMs = new Date(startValue).getTime();
    const endMs = new Date(endValue).getTime();
    if (Number.isNaN(startMs) || Number.isNaN(endMs) || endMs <= startMs) return;
    onRangeChange(Math.max(Date.now() - startMs, 1000), endMs);
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
        {untilMs === undefined && "Since"}
        <span className="font-plex-mono text-text-primary">{labelForRange(sinceMs, untilMs)}</span>
      </button>

      <div
        className={`grid transition-[grid-template-rows,opacity] duration-150 ease-out motion-reduce:transition-none ${
          open ? "grid-rows-[1fr] opacity-100" : "grid-rows-[0fr] opacity-0"
        }`}
        style={{ position: "absolute", top: "calc(100% + 4px)", left: 0, zIndex: 20, width: 220 }}
      >
        <div className="overflow-hidden">
          <div className="w-[220px] rounded-md border border-border bg-elevated py-1 shadow-lg">
            {RELATIVE_OPTIONS.map((opt) => (
              <button
                key={opt.label}
                type="button"
                onClick={() => applyRelative(opt.ms)}
                className={`flex w-full items-center px-3 py-1.5 text-left text-xs hover:bg-surface ${
                  untilMs === undefined && opt.ms === sinceMs ? "text-accent" : "text-text-muted"
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
                Custom range
              </button>
            ) : (
              <div className="flex flex-col gap-1.5 px-3 py-1.5">
                <label className="flex flex-col gap-0.5 text-[10px] uppercase tracking-wide text-text-faint">
                  Start
                  <input
                    type="datetime-local"
                    value={startValue}
                    onChange={(e) => setStartValue(e.target.value)}
                    className="rounded border border-border bg-canvas px-1.5 py-1 text-[11px] text-text-primary outline-none"
                  />
                </label>
                <label className="flex flex-col gap-0.5 text-[10px] uppercase tracking-wide text-text-faint">
                  End
                  <input
                    type="datetime-local"
                    value={endValue}
                    onChange={(e) => setEndValue(e.target.value)}
                    className="rounded border border-border bg-canvas px-1.5 py-1 text-[11px] text-text-primary outline-none"
                  />
                </label>
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
