import { describe, expect, it } from "vitest";
import { activeDaysFromActivities } from "./aggregate";
import type { Activity } from "./types";

function createMockActivity(id: string, started_at: string): Activity {
  return {
    id,
    started_at,
    sport_type: "run",
    title: "Morning Run",
    timezone: "UTC",
  };
}

describe("activeDaysFromActivities", () => {
  it("handles empty list of activities", () => {
    expect(activeDaysFromActivities([])).toEqual([]);
  });

  it("handles multiple dates and returns them sorted ascending by day", () => {
    const activities = [
      createMockActivity("1", "2026-05-15T08:00:00Z"),
      createMockActivity("2", "2026-01-03T10:00:00Z"),
      createMockActivity("3", "2026-03-20T12:00:00Z"),
    ];

    const result = activeDaysFromActivities(activities);

    expect(result).toEqual([
      { day: "2026-01-03", activity_count: 1 },
      { day: "2026-03-20", activity_count: 1 },
      { day: "2026-05-15", activity_count: 1 },
    ]);
  });

  it("tallies multiple activities on the same day", () => {
    const activities = [
      createMockActivity("1", "2026-04-10T07:00:00Z"),
      createMockActivity("2", "2026-04-10T12:30:00Z"),
      createMockActivity("3", "2026-04-10T18:00:00Z"),
      createMockActivity("4", "2026-04-09T08:00:00Z"),
      createMockActivity("5", "2026-04-11T09:00:00Z"),
      createMockActivity("6", "2026-04-11T16:00:00Z"),
    ];

    const result = activeDaysFromActivities(activities);

    expect(result).toEqual([
      { day: "2026-04-09", activity_count: 1 },
      { day: "2026-04-10", activity_count: 3 },
      { day: "2026-04-11", activity_count: 2 },
    ]);
  });

  it("ignores activities with empty started_at timestamp", () => {
    const activities = [
      createMockActivity("1", ""),
      createMockActivity("2", "2026-06-01T10:00:00Z"),
    ];

    const result = activeDaysFromActivities(activities);

    expect(result).toEqual([{ day: "2026-06-01", activity_count: 1 }]);
  });
});
