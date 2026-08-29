const SINCE_OPTIONS = [
  { label: "15m", ms: 15 * 60 * 1000 },
  { label: "1h", ms: 60 * 60 * 1000 },
  { label: "6h", ms: 6 * 60 * 60 * 1000 },
  { label: "24h", ms: 24 * 60 * 60 * 1000 },
];

export type RootCauseFilter = "any" | "found" | "not-found";

export interface FilterBarProps {
  sinceMs: number;
  onSinceChange: (ms: number) => void;
  rootCause: RootCauseFilter;
  onRootCauseChange: (value: RootCauseFilter) => void;
}

// Time filter only, no Limit control (Gate 1 decision: page size is a
// pagination concern, not a user-facing filter).
export function FilterBar({ sinceMs, onSinceChange, rootCause, onRootCauseChange }: FilterBarProps) {
  return (
    <div className="flex items-center gap-2 border-b border-border px-5 py-2.5">
      <div className="flex items-center gap-1.5 rounded-md border border-border bg-surface px-2.5 py-1.5 text-xs text-text-muted">
        Since
        <select
          className="bg-transparent font-plex-mono text-text-primary outline-none"
          value={sinceMs}
          onChange={(e) => onSinceChange(Number(e.target.value))}
        >
          {SINCE_OPTIONS.map((opt) => (
            <option key={opt.label} value={opt.ms} className="bg-surface">
              {opt.label}
            </option>
          ))}
        </select>
      </div>

      <div className="flex gap-0.5">
        {(
          [
            ["any", "Any"],
            ["found", "Root cause found"],
            ["not-found", "No root cause"],
          ] as const
        ).map(([value, label]) => (
          <button
            key={value}
            type="button"
            onClick={() => onRootCauseChange(value)}
            className={`border border-border px-2.5 py-1.5 text-xs first:rounded-l-md last:rounded-r-md ${
              rootCause === value ? "border-accent bg-accent-dim text-accent" : "text-text-muted"
            }`}
          >
            {label}
          </button>
        ))}
      </div>
    </div>
  );
}
