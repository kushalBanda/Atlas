import { describe, expect, it } from "vitest";
import { formatDurationNano } from "./duration";

describe("formatDurationNano", () => {
  it("formats sub-microsecond durations", () => {
    expect(formatDurationNano(500)).toBe("500ns");
  });

  it("formats microseconds", () => {
    expect(formatDurationNano(1_500)).toBe("2us");
  });

  it("formats milliseconds", () => {
    expect(formatDurationNano(842_000_000)).toBe("842ms");
  });

  it("formats seconds", () => {
    expect(formatDurationNano(1_900_000_000)).toBe("1.9s");
  });

  it("formats zero", () => {
    expect(formatDurationNano(0)).toBe("0ns");
  });
});
