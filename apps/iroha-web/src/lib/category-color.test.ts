import { describe, expect, it } from "vitest";
import {
  categoryColor,
  categoryColorVar,
} from "@iroha/shared/domain/category-color";
import { healthMetricColorVar } from "@iroha/shared/domain/health-metric-colors";

describe("canonical category colors", () => {
  it("keeps category identity stable regardless of chart rank", () => {
    expect(categoryColor("food")).toBe("var(--mark-amber)");
    expect(categoryColor("transport")).toBe("var(--sport-swim)");
    expect(categoryColor("food")).not.toBe(categoryColor("transport"));
    const categories = [
      "food",
      "groceries",
      "transport",
      "shopping",
      "housing",
      "utilities",
      "health",
      "entertainment",
      "subscriptions",
      "work",
      "other",
    ];
    const colors = categories.map(categoryColor);
    expect(new Set(colors).size).toBe(categories.length);
    expect([...categories].reverse().map(categoryColor)).toEqual(
      [...colors].reverse(),
    );
    for (const category of categories)
      expect(categoryColor(category)).toBe(
        `var(${categoryColorVar(category)})`,
      );
  });

  it("uses a neutral unknown fallback, never chart rank or interactive accent", () => {
    expect(categoryColor("unrecognized")).toBe("var(--text-muted)");
    expect(categoryColorVar("unrecognized")).toBe("--text-muted");
    expect(healthMetricColorVar("unrecognized")).toBe("--text-muted");
    expect(healthMetricColorVar("hrv_sdnn")).toBe("--ring-stand");
    expect(healthMetricColorVar("respiratory_rate")).toBe("--ring-move");
    expect(healthMetricColorVar("flights")).toBe("--ring-stand");
  });
});
