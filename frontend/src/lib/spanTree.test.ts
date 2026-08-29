import { describe, expect, it } from "vitest";
import { buildSpanTree, flattenSpanTree } from "./spanTree";
import { makeSpan } from "../test/fixtures";

describe("buildSpanTree", () => {
  it("returns a single root with no children", () => {
    const tree = buildSpanTree([makeSpan({ SpanID: "a", ParentSpanID: "" })]);
    expect(tree).toHaveLength(1);
    expect(tree[0].span.SpanID).toBe("a");
    expect(tree[0].children).toHaveLength(0);
  });

  it("nests three levels correctly", () => {
    const tree = buildSpanTree([
      makeSpan({ SpanID: "a", ParentSpanID: "" }),
      makeSpan({ SpanID: "b", ParentSpanID: "a" }),
      makeSpan({ SpanID: "c", ParentSpanID: "b" }),
    ]);
    expect(tree[0].span.SpanID).toBe("a");
    expect(tree[0].children[0].span.SpanID).toBe("b");
    expect(tree[0].children[0].depth).toBe(1);
    expect(tree[0].children[0].children[0].span.SpanID).toBe("c");
    expect(tree[0].children[0].children[0].depth).toBe(2);
  });

  it("produces multiple trees when there are multiple roots", () => {
    const tree = buildSpanTree([
      makeSpan({ SpanID: "a", ParentSpanID: "" }),
      makeSpan({ SpanID: "b", ParentSpanID: "" }),
    ]);
    expect(tree).toHaveLength(2);
  });

  it("treats an orphaned parent reference as a root instead of dropping it", () => {
    const tree = buildSpanTree([makeSpan({ SpanID: "a", ParentSpanID: "does-not-exist" })]);
    expect(tree).toHaveLength(1);
    expect(tree[0].span.SpanID).toBe("a");
    expect(tree[0].depth).toBe(0);
  });
});

describe("flattenSpanTree", () => {
  it("flattens depth-first", () => {
    const tree = buildSpanTree([
      makeSpan({ SpanID: "a", ParentSpanID: "" }),
      makeSpan({ SpanID: "b", ParentSpanID: "a" }),
      makeSpan({ SpanID: "c", ParentSpanID: "" }),
    ]);
    const flat = flattenSpanTree(tree);
    expect(flat.map((n) => n.span.SpanID)).toEqual(["a", "b", "c"]);
  });
});
