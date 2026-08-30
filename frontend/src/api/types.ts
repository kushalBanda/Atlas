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

export interface ModelStat {
  Model: string;
  Calls: number;
  PromptTokens: number;
  CompletionTokens: number;
  Cost: number;
}

export interface LLMStats {
  TotalCost: number;
  TotalPromptTokens: number;
  TotalCompletionTokens: number;
  Models: ModelStat[] | null;
}

export interface Stats {
  TotalTraces: number;
  TracesWithRootCause: number;
  TotalSpans: number;
  LLM: LLMStats;
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

// Backend structs have no JSON tags, so storage.RunSummary and
// storage.SessionSummary serialize as Go PascalCase field names. The
// agentrun graph types DO carry explicit lowercase tags. Do not
// "normalize" either side — check a live curl before changing a name.
export interface RunSummary {
  RunID: string;
  AgentName: string | null;
  SessionID: string | null;
  UserID: string | null;
  FirstSeen: string;
  LastSeen: string;
  SpanCount: number;
  ErrorCount: number;
  PromptTokens: number;
  CompletionTokens: number;
  Cost: number;
}

export interface SessionSummary {
  SessionID: string;
  UserID: string | null;
  FirstSeen: string;
  LastSeen: string;
  RunCount: number;
  ErrorCount: number;
  Cost: number;
}

export interface GraphNode {
  span_id: string;
  trace_id: string;
  name: string;
  step_kind: string;
  agent_name: string;
  service_name: string;
  status_code: string;
  start_time: string;
  duration_nano: number;
  repeat_group: number | null;
}

export interface GraphEdge {
  from: string;
  to: string;
  cross_trace: boolean;
}

export interface RunRepeat {
  index: number;
  agent_name: string;
  name: string;
  count: number;
  span_ids: string[];
}

export interface RunGraph {
  run_id: string;
  nodes: GraphNode[];
  edges: GraphEdge[];
  repeats: RunRepeat[];
}

export interface RunListResponse {
  runs: RunSummary[];
}

export interface RunResponse {
  run: RunSummary | null;
  spans: Span[];
  graph: RunGraph;
}

export interface SessionListResponse {
  sessions: SessionSummary[];
}
