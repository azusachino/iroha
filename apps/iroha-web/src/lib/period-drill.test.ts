import { describe, expect, it } from "vitest";
import { periodDrillAccessibleName } from "@iroha/shared/theme-ui/components/PeriodDrill.svelte";

describe("PeriodDrill", () => {
  it.each([
    [12345, "2026-08, 12,345 steps"],
    [0, "2026-08, 0 steps"],
    [null, "2026-08, no steps recorded"],
  ])("names the period and evidence for %s", (value, expected) => {
    expect(periodDrillAccessibleName("2026-08", value)).toBe(expected);
  });
});
