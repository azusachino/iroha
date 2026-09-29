import { describe, expect, it } from "vitest";
import { buildActivityHeatmap } from "@iroha/shared/theme-ui/components/activity-heatmap";

describe("buildActivityHeatmap", () => {
  it("shows daily activity counts for the trailing year and starts weeks on Monday", () => {
    const model = buildActivityHeatmap(
      [
        { day: "2025-09-29", activity_count: 7 },
        { day: "2025-09-30", activity_count: 5 },
        { day: "2026-09-28", activity_count: 1 },
        { day: "2026-09-29", activity_count: 2 },
        { day: "2026-10-01", activity_count: 9 },
      ],
      "2026-09-29",
    );
    const cells = model.weeks.flat().filter((cell) => cell.date);
    const lastWeek = model.weeks.find((week) =>
      week.some((cell) => cell.date === "2026-09-29"),
    );

    expect(cells).toHaveLength(365);
    expect(model.activeDayCount).toBe(3);
    expect(model.totalActivityCount).toBe(8);
    expect(cells.find((cell) => cell.date === "2026-09-29")?.level).toBe(2);
    expect(lastWeek?.[1].date).toBe("2026-09-29");
    expect(cells.find((cell) => cell.date === "2025-09-30")?.count).toBe(5);
    expect(cells.some((cell) => cell.date === "2025-09-29")).toBe(false);
    expect(cells.some((cell) => cell.date === "2026-10-01")).toBe(false);
  });
});
