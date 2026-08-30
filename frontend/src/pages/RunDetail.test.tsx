import { afterEach, describe, expect, it, vi } from "vitest";
import { screen } from "@testing-library/react";
import { RunDetail } from "./RunDetail";
import { renderWithProviders } from "../test/renderWithProviders";
import type { RunResponse } from "../api/types";

function jsonResponse(body: unknown, status = 200) {
  return Promise.resolve(
    new Response(JSON.stringify(body), { status, headers: { "Content-Type": "application/json" } }),
  );
}

const runResponseFixture: RunResponse = {
  run: {
    RunID: "run-a",
    AgentName: "researcher",
    SessionID: "sess-1",
    UserID: "user-1",
    FirstSeen: "2026-08-29T12:00:00Z",
    LastSeen: "2026-08-29T12:00:05Z",
    SpanCount: 4,
    ErrorCount: 0,
    PromptTokens: 120,
    CompletionTokens: 45,
    Cost: 0.0021,
  },
  spans: [],
  graph: {
    run_id: "run-a",
    nodes: [
      {
        span_id: "s1",
        trace_id: "t1",
        name: "plan",
        step_kind: "chain",
        agent_name: "researcher",
        service_name: "svc",
        status_code: "ok",
        start_time: "2026-08-29T12:00:00.000000000Z",
        duration_nano: 1_000_000,
        repeat_group: null,
      },
      {
        span_id: "s2",
        trace_id: "t1",
        name: "search",
        step_kind: "tool",
        agent_name: "researcher",
        service_name: "svc",
        status_code: "ok",
        start_time: "2026-08-29T12:00:01.000000000Z",
        duration_nano: 2_000_000,
        repeat_group: 0,
      },
    ],
    edges: [{ from: "s1", to: "s2", cross_trace: false }],
    repeats: [
      { index: 0, agent_name: "researcher", name: "search", count: 3, span_ids: ["s2", "s3", "s4"] },
    ],
  },
};

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("RunDetail", () => {
  it("renders one node per graph node", async () => {
    vi.stubGlobal("fetch", vi.fn(() => jsonResponse(runResponseFixture)));
    renderWithProviders(<RunDetail />, { route: "/runs/run-a", path: "/runs/:runId" });

    expect(await screen.findByText("plan")).toBeInTheDocument();
    expect(await screen.findByText("search")).toBeInTheDocument();
  });

  it("shows a repeat badge when the graph reports one", async () => {
    vi.stubGlobal("fetch", vi.fn(() => jsonResponse(runResponseFixture)));
    renderWithProviders(<RunDetail />, { route: "/runs/run-a", path: "/runs/:runId" });

    expect(await screen.findByText(/3×/)).toBeInTheDocument();
  });

  it("shows not-found, no retry, when the run is missing (404)", async () => {
    vi.stubGlobal("fetch", vi.fn(() => jsonResponse({ error: "run not found" }, 404)));
    renderWithProviders(<RunDetail />, { route: "/runs/missing", path: "/runs/:runId" });

    expect(await screen.findByText("Run not found.")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Retry" })).not.toBeInTheDocument();
  });
});
