import { describe, expect, it } from "vitest";
import { categoryColor } from "@iroha/shared/category-color";

describe("canonical category colors", () => {
  it("keeps category identity stable regardless of chart rank", () => {
    expect(categoryColor("food")).toBe("var(--mark-amber)");
    expect(categoryColor("transport")).toBe("var(--sport-swim)");
    expect(categoryColor("food")).not.toBe(categoryColor("transport"));
  });
});
