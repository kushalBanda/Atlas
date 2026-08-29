import { afterEach, describe, expect, it, vi } from "vitest";
import { screen } from "@testing-library/react";
import { TraceDetail } from "./TraceDetail";
import { renderWithProviders } from "../test/renderWithProviders";

function jsonResponse(body: unknown, status = 200) {
  return Promise.resolve(
    new Response(JSON.stringify(body), { status, headers: { "Content-Type": "application/json" } }),
  );
}

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("TraceDetail", () => {
  it("shows not-found, no retry, when the trace is missing (404)", async () => {
    vi.stubGlobal("fetch", vi.fn(() => jsonResponse({ error: "trace not found" }, 404)));
    renderWithProviders(<TraceDetail />, { route: "/traces/missing", path: "/traces/:traceId" });

    expect(await screen.findByText("Trace not found.")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Retry" })).not.toBeInTheDocument();
  });

  it("shows an error with retry on a non-404 failure", async () => {
    vi.stubGlobal("fetch", vi.fn(() => jsonResponse({ error: "boom" }, 500)));
    renderWithProviders(<TraceDetail />, { route: "/traces/trace-1", path: "/traces/:traceId" });

    expect(await screen.findByText("Could not load trace.")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Retry" })).toBeInTheDocument();
  });
});
