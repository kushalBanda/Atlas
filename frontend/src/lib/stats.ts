import type { TraceSummary } from "../api/types";
import { deriveVerdictState } from "./verdict";

export interface TraceStats {
  count: number;
  rootCauseFoundCount: number;
  p50Nano: number;
  p95Nano: number;
}

// Percentile over a sorted-ascending array of nanosecond durations.
// Nearest-rank method: fine for a summary strip, not a precision metric.
function percentile(sortedNanos: number[], p: number): number {
  if (sortedNanos.length === 0) return 0;
  const idx = Math.min(sortedNanos.length - 1, Math.floor(p * sortedNanos.length));
  return sortedNanos[idx];
}

// Computed client-side from whatever batch is already fetched. Not a
// backend aggregation, see docs/plans/atlas-frontend/00-status.md: Atlas
// has no aggregate-stats endpoint, so this only ever reflects the current
// batch (BATCH_LIMIT rows), not the true fleet-wide distribution.
export function computeTraceStats(traces: TraceSummary[]): TraceStats {
  const durations = traces.map((t) => t.duration_nano).sort((a, b) => a - b);
  const rootCauseFoundCount = traces.filter((t) => deriveVerdictState(t) === "found").length;
  return {
    count: traces.length,
    rootCauseFoundCount,
    p50Nano: percentile(durations, 0.5),
    p95Nano: percentile(durations, 0.95),
  };
}
