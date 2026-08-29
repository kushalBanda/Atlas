import { TimeRangePicker } from "./TimeRangePicker";

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
      <TimeRangePicker sinceMs={sinceMs} onSinceChange={onSinceChange} />

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
