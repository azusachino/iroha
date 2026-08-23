import { describe, expect, it } from "vitest";
import { latestRecordedDay } from "./today-state.svelte";

describe("Today recorded-day navigation", () => {
  it("chooses the latest canonical day without depending on API order", () => {
    expect(latestRecordedDay(["2026-08-12", "2026-08-16", "2026-08-14"])).toBe(
      "2026-08-16",
    );
  });

  it("has no destination when the canonical index is empty", () => {
    expect(latestRecordedDay([])).toBeNull();
  });
});
