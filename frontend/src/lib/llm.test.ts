import { describe, expect, it } from "vitest";
import { hasLLMFields } from "./llm";
import { makeSpan } from "../test/fixtures";

describe("hasLLMFields", () => {
  it("is true when model is set", () => {
    expect(hasLLMFields(makeSpan({ LLMModel: "gpt-4.1-mini" }))).toBe(true);
  });

  it("is true when only cost is set", () => {
    expect(hasLLMFields(makeSpan({ LLMCost: 0.001 }))).toBe(true);
  });

  it("is false when all LLM fields are nil", () => {
    expect(hasLLMFields(makeSpan())).toBe(false);
  });
});
