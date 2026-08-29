import { formatDurationNano } from "../lib/duration";
import type { TraceStats } from "../lib/stats";

interface StatsStripProps {
  stats: TraceStats;
}

// A small stats summary above the trace list. Computed client-side from
// the same batch the table already fetched, no separate call. See
// lib/stats.ts for why this reflects only the current batch, not a true
// fleet-wide aggregate.
export function StatsStrip({ stats }: StatsStripProps) {
  const items = [
    { label: "Traces in view", value: String(stats.count) },
    { label: "Root cause found", value: String(stats.rootCauseFoundCount) },
    { label: "P50 duration", value: formatDurationNano(stats.p50Nano) },
    { label: "P95 duration", value: formatDurationNano(stats.p95Nano) },
  ];

  return (
    <div className="flex gap-6 border-b border-border px-5 py-3">
      {items.map((item) => (
        <div key={item.label} className="flex flex-col gap-0.5">
          <span className="font-plex-mono text-lg leading-none text-text-primary">{item.value}</span>
          <span className="text-[11px] uppercase tracking-wide text-text-faint">{item.label}</span>
        </div>
      ))}
    </div>
  );
}
