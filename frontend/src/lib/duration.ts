// Shared by every screen that shows a duration (trace list, waterfall).
export function formatDurationNano(nanos: number): string {
  if (nanos < 1_000) return `${nanos}ns`;
  if (nanos < 1_000_000) return `${Math.round(nanos / 1_000)}us`;
  if (nanos < 1_000_000_000) return `${Math.round(nanos / 1_000_000)}ms`;
  return `${(nanos / 1_000_000_000).toFixed(1)}s`;
}
