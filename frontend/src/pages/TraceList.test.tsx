import { afterEach, describe, expect, it, vi } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { TraceList } from "./TraceList";
import { renderWithProviders } from "../test/renderWithProviders";

function jsonResponse(body: unknown, status = 200) {
  return Promise.resolve(
    new Response(JSON.stringify(body), { status, headers: { "Content-Type": "application/json" } }),
  );
}

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("TraceList", () => {
  it("shows a loading skeleton before data arrives", () => {
    vi.stubGlobal("fetch", vi.fn(() => new Promise(() => {})));
    renderWithProviders(<TraceList />);
    expect(screen.getByLabelText("Loading")).toBeInTheDocument();
  });

  it("shows an empty state when there are no traces", async () => {
    vi.stubGlobal("fetch", vi.fn(() => jsonResponse({ traces: [] })));
    renderWithProviders(<TraceList />);
    expect(await screen.findByText(/No traces yet/)).toBeInTheDocument();
  });

  it("shows an error with a retry button on fetch failure", async () => {
    const fetchMock = vi.fn(() => jsonResponse({ error: "boom" }, 500));
    vi.stubGlobal("fetch", fetchMock);
    renderWithProviders(<TraceList />);

    const retry = await screen.findByRole("button", { name: "Retry" });
    expect(screen.getByText(/Could not load traces/)).toBeInTheDocument();

    const callsBefore = fetchMock.mock.calls.length;
    await userEvent.click(retry);
    await waitFor(() => expect(fetchMock.mock.calls.length).toBeGreaterThan(callsBefore));
  });

  it("has no Limit control in the DOM, pagination is Prev/Next only", async () => {
    vi.stubGlobal("fetch", vi.fn(() => jsonResponse({ traces: [] })));
    renderWithProviders(<TraceList />);
    await screen.findByText(/No traces yet/);
    expect(screen.queryByText(/Limit/i)).not.toBeInTheDocument();
  });
});
