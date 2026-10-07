import { describe, expect, it } from "vitest";
import { activeDaysFromActivities, monthlyBuckets } from "./aggregate";
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

describe("monthlyBuckets", () => {
  it("distinguishes known distance from missing distance on distance sports", () => {
    const activities: Activity[] = [
      {
        id: "1",
        started_at: "2026-01-10T08:00:00Z",
        sport_type: "run",
        title: "Run with distance",
        timezone: "UTC",
        distance_m: 5000,
      },
      {
        id: "2",
        started_at: "2026-01-15T08:00:00Z",
        sport_type: "run",
        title: "Run without distance",
        timezone: "UTC",
      },
    ];

    const buckets = monthlyBuckets(activities);
    expect(buckets).toHaveLength(1);
    expect(buckets[0]).toMatchObject({
      key: "2026-01",
      activity_count: 2,
      distance_m: 5000,
      distance_known_count: 1,
      distance_unknown_count: 1,
    });
  });

  it("does not count non-distance sports without distance as distance_unknown_count", () => {
    const activities: Activity[] = [
      {
        id: "1",
        started_at: "2026-01-10T08:00:00Z",
        sport_type: "run",
        title: "Morning Run",
        timezone: "UTC",
        distance_m: 10000,
      },
      {
        id: "2",
        started_at: "2026-01-12T18:00:00Z",
        sport_type: "fitness_gaming",
        title: "Ring Fit Adventure",
        timezone: "UTC",
      },
      {
        id: "3",
        started_at: "2026-01-14T19:00:00Z",
        sport_type: "CardioDance",
        title: "Dance Workout",
        timezone: "UTC",
      },
      {
        id: "4",
        started_at: "2026-01-16T07:00:00Z",
        sport_type: "Yoga",
        title: "Morning Yoga",
        timezone: "UTC",
      },
    ];

    const buckets = monthlyBuckets(activities);
    expect(buckets).toHaveLength(1);
    expect(buckets[0]).toMatchObject({
      key: "2026-01",
      activity_count: 4,
      distance_m: 10000,
      distance_known_count: 1,
      distance_unknown_count: 0,
    });
  });
});
