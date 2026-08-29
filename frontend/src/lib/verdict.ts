import type { Trace } from "../api/types";

export type VerdictState = "open" | "ok" | "found";

// Shared by TraceList's status column and VerdictLine (Slice 5) so the
// two screens can't drift on what "open" means.
export function deriveVerdictState(trace: Trace): VerdictState {
  if (!trace.ClosedAt) return "open";
  if (!trace.LikelyRootCauseSpanID) return "ok";
  return "found";
}
