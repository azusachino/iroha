import { describe, expect, it } from "vitest";
import { nextTabId } from "./tab-keys";

const IDS = ["a", "b", "c"] as const;

describe("nextTabId", () => {
  it("moves right and left, wrapping at both ends", () => {
    expect(nextTabId(IDS, "a", "ArrowRight")).toBe("b");
    expect(nextTabId(IDS, "c", "ArrowRight")).toBe("a");
    expect(nextTabId(IDS, "b", "ArrowLeft")).toBe("a");
    expect(nextTabId(IDS, "a", "ArrowLeft")).toBe("c");
  });

  it("jumps to the first and last tab", () => {
    expect(nextTabId(IDS, "b", "Home")).toBe("a");
    expect(nextTabId(IDS, "b", "End")).toBe("c");
  });

  it("ignores other keys and unknown tabs", () => {
    expect(nextTabId(IDS, "a", "Enter")).toBeNull();
    expect(nextTabId(IDS, "a", "Tab")).toBeNull();
    expect(nextTabId(IDS, "z" as "a", "ArrowRight")).toBeNull();
  });
});
