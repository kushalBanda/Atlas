import { describe, expect, it } from "vitest";
import { deriveVerdictState } from "./verdict";
import { makeTrace } from "../test/fixtures";

describe("deriveVerdictState", () => {
  it("is open when not closed", () => {
    expect(deriveVerdictState(makeTrace({ ClosedAt: null }))).toBe("open");
  });

  it("is ok when closed with no root cause", () => {
    expect(
      deriveVerdictState(makeTrace({ ClosedAt: "2026-01-01T00:00:02Z", LikelyRootCauseSpanID: null })),
    ).toBe("ok");
  });

  it("is found when closed with a root cause", () => {
    expect(
      deriveVerdictState(
        makeTrace({ ClosedAt: "2026-01-01T00:00:02Z", LikelyRootCauseSpanID: "span-1" }),
      ),
    ).toBe("found");
  });
});
