import { afterEach, describe, expect, it, vi } from "vitest";
import { screen } from "@testing-library/react";
import { Discovery } from "./Discovery";
import { renderWithProviders } from "../test/renderWithProviders";

function jsonResponse(body: unknown, status = 200) {
  return Promise.resolve(
    new Response(JSON.stringify(body), { status, headers: { "Content-Type": "application/json" } }),
  );
}

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("Discovery", () => {
  it("shows empty states when matched/unrecognized are null (Go nil-slice on the wire)", async () => {
    vi.stubGlobal("fetch", vi.fn(() => jsonResponse({ matched: null, unrecognized: null })));
    renderWithProviders(<Discovery />);

    expect(await screen.findByText("No recognized services found yet.")).toBeInTheDocument();
    expect(screen.getByText("No unrecognized services found.")).toBeInTheDocument();
  });

  it("shows an error with a retry button on fetch failure", async () => {
    vi.stubGlobal("fetch", vi.fn(() => jsonResponse({ error: "boom" }, 500)));
    renderWithProviders(<Discovery />);

    expect(await screen.findByText(/Could not load discovery targets/)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Retry" })).toBeInTheDocument();
  });
});
