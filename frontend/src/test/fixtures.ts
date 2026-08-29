import type { Span, Trace } from "../api/types";

export function makeSpan(overrides: Partial<Span> = {}): Span {
  return {
    TraceID: "trace-1",
    SpanID: "span-1",
    ParentSpanID: "",
    ServiceName: "svc",
    Name: "op",
    StartTime: "2026-01-01T00:00:00Z",
    EndTime: "2026-01-01T00:00:01Z",
    StatusCode: "ok",
    Attributes: {},
    ResourceAttributes: {},
    SpanKind: null,
    Level: null,
    LLMModel: null,
    LLMPromptTokens: null,
    LLMCompletionTokens: null,
    LLMCost: null,
    LLMTemperature: null,
    LLMTopP: null,
    LLMMaxTokens: null,
    LLMUsageDetails: null,
    LLMCostDetails: null,
    LLMTimeToFirstTokenNano: null,
    LLMPromptID: null,
    LLMPromptName: null,
    LLMPromptVersion: null,
    duration_nano: 1_000_000_000,
    ...overrides,
  };
}

export function makeTrace(overrides: Partial<Trace> = {}): Trace {
  return {
    TraceID: "trace-1",
    FirstSeen: "2026-01-01T00:00:00Z",
    LastSeen: "2026-01-01T00:00:01Z",
    ClosedAt: null,
    LikelyRootCauseSpanID: null,
    Reason: null,
    SelfTimePct: null,
    ...overrides,
  };
}
