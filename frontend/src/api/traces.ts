import { useQuery } from "@tanstack/react-query";
import { apiGet, ApiError } from "./client";
import type { TraceListResponse, TraceResponse } from "./types";

export interface TraceFilter {
  hasRootCause?: boolean;
  // Relative time window in ms, converted to an absolute `since` timestamp
  // inside queryFn at fetch time (not during render): computing "now" in a
  // render-phase useMemo is an impurity the linter correctly flags, and
  // computing it fresh per fetch is more correct anyway.
  sinceMs?: number;
  // Absolute upper bound (ms epoch), set only by a custom start+end range.
  untilMs?: number;
  limit?: number; // internal, driven by pagination, never user-set directly
}

function filterToParams(filter: TraceFilter): Record<string, string> {
  const params: Record<string, string> = {};
  if (filter.hasRootCause !== undefined) {
    params.has_root_cause = String(filter.hasRootCause);
  }
  if (filter.sinceMs !== undefined) {
    params.since = new Date(Date.now() - filter.sinceMs).toISOString();
  }
  if (filter.untilMs !== undefined) {
    params.until = new Date(filter.untilMs).toISOString();
  }
  if (filter.limit !== undefined) params.limit = String(filter.limit);
  return params;
}

export function useTraces(filter: TraceFilter) {
  return useQuery<TraceListResponse, ApiError>({
    queryKey: ["traces", filter],
    queryFn: () => apiGet<TraceListResponse>("/traces", filterToParams(filter)),
  });
}

export function useTrace(traceId: string) {
  return useQuery<TraceResponse, ApiError>({
    queryKey: ["trace", traceId],
    queryFn: () => apiGet<TraceResponse>(`/traces/${traceId}`),
    enabled: traceId !== "",
  });
}
