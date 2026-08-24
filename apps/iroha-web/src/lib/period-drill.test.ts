import { render } from "svelte/server";
import { describe, expect, it, vi } from "vitest";
import PeriodDrill from "@iroha/shared/theme-ui/components/PeriodDrill.svelte";
import FallbackPatternsTable from "../routes/patterns/FallbackPatternsTable.svelte";

describe("PeriodDrill", () => {
  it.each([
    [12345, "2026-08, 12,345 steps"],
    [0, "2026-08, 0 steps"],
    [null, "2026-08, no steps recorded"],
  ])("names the period and evidence for %s", (value, expected) => {
    const { body } = render(PeriodDrill, {
      props: {
        label: "2026-08",
        period: "2026-08",
        value,
        onDrill: vi.fn(),
      },
    });

    expect(body).toContain(`aria-label="${expected}"`);
    expect(body).toContain("data-period-drill");
  });

  it("renders fallback evidence from the real Steps column", () => {
    const format = (value: number | null | undefined, digits: number) =>
      value == null
        ? "—"
        : value.toLocaleString(undefined, {
            minimumFractionDigits: digits,
            maximumFractionDigits: digits,
          });
    const { body } = render(FallbackPatternsTable, {
      props: {
        gran: "month",
        aggregated: true,
        rows: [
          {
            label: "2026-08",
            period: "2026-08",
            days: 31,
            move: 500,
            exercise: 30,
            stand: 12,
            moveClosedPct: 75,
            steps: 12345,
            distance: 8.2,
            resting_hr: 54,
            hrv_sdnn: 42,
            spo2_avg: 98,
            respiratory_rate: 13,
            vo2max: 51,
            body_mass_kg: 68,
          },
          {
            label: "2026-07",
            period: "2026-07",
            days: 31,
            move: null,
            exercise: null,
            stand: null,
            moveClosedPct: null,
            steps: null,
            distance: null,
            resting_hr: null,
            hrv_sdnn: null,
            spo2_avg: null,
            respiratory_rate: null,
            vo2max: null,
            body_mass_kg: null,
          },
        ],
        format,
        onDrill: vi.fn(),
      },
    });

    expect(body).toContain('aria-label="2026-08, 12,345 steps"');
    expect(body).toContain('aria-label="2026-07, no steps recorded"');
    expect(body).toMatch(/<th[^>]*>Steps\/d<\/th>/);
    expect(body).toMatch(/<td[^>]*data-period-evidence[^>]*>12,345<\/td>/);
    expect(body).toMatch(/<td[^>]*data-period-evidence[^>]*>—<\/td>/);
  });
});
