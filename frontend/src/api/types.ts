// Mirrors the backend's actual JSON output field-for-field. Trace, Span,
// Target, and Rule (pkg/storage, pkg/discovery) have NO json tags, so they
// serialize as Go's exact PascalCase field names. Only the query-layer
// wrapper structs and computed fields (pkg/query/query.go) carry explicit
// lowercase tags. Verified against a live `curl /traces` response and the
// source, not assumed, do not "fix" these to snake_case.

export interface Span {
  TraceID: string;
  SpanID: string;
  ParentSpanID: string;
  ServiceName: string;
  Name: string;
  StartTime: string; // RFC3339
  EndTime: string; // RFC3339
  StatusCode: "ok" | "error" | "unset";
  Attributes: Record<string, unknown>;
  ResourceAttributes: Record<string, unknown>;
  SpanKind: string | null;
  Level: string | null;
  LLMModel: string | null;
  LLMPromptTokens: number | null;
  LLMCompletionTokens: number | null;
  LLMCost: number | null;
  LLMTemperature: number | null;
  LLMTopP: number | null;
  LLMMaxTokens: number | null;
  LLMUsageDetails: Record<string, unknown> | null;
  LLMCostDetails: Record<string, unknown> | null;
  LLMTimeToFirstTokenNano: number | null;
  LLMPromptID: string | null;
  LLMPromptName: string | null;
  LLMPromptVersion: number | null;
  duration_nano: number; // spanView's one explicitly-tagged field
}

export interface Trace {
  TraceID: string;
  FirstSeen: string;
  LastSeen: string;
  ClosedAt: string | null;
  LikelyRootCauseSpanID: string | null;
  Reason: string | null;
  SelfTimePct: number | null;
}

// traceSummary embeds storage.Trace, Go promotes its fields inline in JSON.
export interface TraceSummary extends Trace {
  duration_nano: number;
}

export interface TraceListResponse {
  traces: TraceSummary[];
}

export interface TraceResponse {
  trace: Trace;
  spans: Span[];
}

export interface DiscoveryRule {
  Port: number;
  ProcessMatch: string;
  ReceiverConfig: string;
}

export interface DiscoveryTarget {
  Host: string;
  Port: number;
  ProcessOrImage: string;
  MatchedRule: DiscoveryRule | null;
  ResourceAttributes: Record<string, string> | null;
}

export interface DiscoveryResponse {
  matched: DiscoveryTarget[];
  unrecognized: DiscoveryTarget[];
}
