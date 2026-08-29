import { describe, expect, it } from "vitest";
import { render, screen } from "@testing-library/react";
import { LLMBadges } from "./LLMBadges";
import { makeSpan } from "../test/fixtures";

describe("LLMBadges", () => {
  it("renders only populated fields", () => {
    render(<LLMBadges span={makeSpan({ LLMModel: "gpt-4.1-mini", LLMCost: 0.0031 })} />);
    expect(screen.getByText("gpt-4.1-mini")).toBeInTheDocument();
    expect(screen.getByText("$0.0031")).toBeInTheDocument();
    expect(screen.queryByText(/tok/)).not.toBeInTheDocument();
  });

  it("renders no badges when all fields are nil", () => {
    const { container } = render(<LLMBadges span={makeSpan()} />);
    expect(container.querySelectorAll("span > b")).toHaveLength(0);
  });
});
