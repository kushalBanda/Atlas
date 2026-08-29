import { afterEach, describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import App from "./App";

function jsonResponse(body: unknown, status = 200) {
  return Promise.resolve(
    new Response(JSON.stringify(body), { status, headers: { "Content-Type": "application/json" } }),
  );
}

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("App", () => {
  it("keeps the nav rail visible when discovery fails to load", async () => {
    // App renders its own BrowserRouter, so drive location via history
    // rather than nesting a second router around it.
    window.history.pushState({}, "", "/discovery");
    vi.stubGlobal("fetch", vi.fn(() => jsonResponse({ error: "boom" }, 500)));
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });

    render(
      <QueryClientProvider client={queryClient}>
        <App />
      </QueryClientProvider>,
    );

    expect(await screen.findByText(/Could not load discovery targets/)).toBeInTheDocument();
    // Nav rail lives outside <main>, a failed page must not take it down.
    expect(screen.getByTitle("Traces")).toBeInTheDocument();
    expect(screen.getByTitle("Discovery")).toBeInTheDocument();
  });
});
