import { useQuery } from "@tanstack/react-query";
import { apiGet, ApiError } from "./client";
import type {
  RunListResponse,
  RunResponse,
  SessionListResponse,
} from "./types";

export interface RunFilters {
  sessionId?: string;
  userId?: string;
  agent?: string;
  since?: string;
  until?: string;
  limit?: number;
}

function runFiltersToParams(f: RunFilters): Record<string, string> {
  const params: Record<string, string> = {};
  if (f.sessionId) params.session_id = f.sessionId;
  if (f.userId) params.user_id = f.userId;
  if (f.agent) params.agent = f.agent;
  if (f.since) params.since = f.since;
  if (f.until) params.until = f.until;
  params.limit = String(f.limit ?? 200);
  return params;
}

export function useRuns(filters: RunFilters = {}) {
  return useQuery<RunListResponse, ApiError>({
    queryKey: ["runs", filters],
    queryFn: () => apiGet<RunListResponse>("/runs", runFiltersToParams(filters)),
  });
}

export function useRun(runId: string | undefined) {
  return useQuery<RunResponse, ApiError>({
    queryKey: ["run", runId],
    queryFn: () => apiGet<RunResponse>(`/runs/${runId}`),
    enabled: Boolean(runId),
  });
}

export interface SessionFilters {
  userId?: string;
  since?: string;
  limit?: number;
}

export function useSessions(filters: SessionFilters = {}) {
  return useQuery<SessionListResponse, ApiError>({
    queryKey: ["sessions", filters],
    queryFn: () => {
      const params: Record<string, string> = {};
      if (filters.userId) params.user_id = filters.userId;
      if (filters.since) params.since = filters.since;
      params.limit = String(filters.limit ?? 200);
      return apiGet<SessionListResponse>("/sessions", params);
    },
  });
}
