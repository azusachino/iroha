import { describe, it, expect } from "vitest";
import {
  listActivities,
  getActivity,
  getActivityRoute,
  getActivitySamplings,
  getActivityLaps,
  getPublicSummary,
  listPublicActivities,
  getPublicRoutes,
  type Activity,
  type Page,
  type RoutePoint,
  type SamplingPoint,
  type Lap,
  type Summary,
  type PublicActivity,
  type RouteFeatureCollection,
} from "./api";

const emptyPage: Page<Activity> = {
  items: [],
  next_cursor: null,
  has_more: false,
};

// Helper to create a fake fetch function that captures the URL and returns a response
function createFakeFetch(
  responseData: any,
  ok = true,
  status = 200,
  statusText = "OK",
) {
  let capturedUrl = "";

  const fakeFetch = (async (input: string | Request | URL) => {
    const url = typeof input === "string" ? input : input.toString();
    capturedUrl = url;
    return {
      ok,
      status,
      statusText,
      json: async () => responseData,
    };
  }) as typeof fetch;

  return { fakeFetch, getCapturedUrl: () => capturedUrl };
}

describe("listActivities", () => {
  it("builds URL without params", async () => {
    const { fakeFetch, getCapturedUrl } = createFakeFetch([]);
    await listActivities({}, fakeFetch);

    const url = getCapturedUrl();
    expect(url).toContain("/api/v1/activities");
    expect(url).not.toContain("?");
  });

  it("adds sport_type param when provided", async () => {
    const { fakeFetch, getCapturedUrl } = createFakeFetch([]);
    await listActivities({ sport_type: "run" }, fakeFetch);

    const url = getCapturedUrl();
    expect(url).toContain("sport_type=run");
  });

  it("adds limit param when provided", async () => {
    const { fakeFetch, getCapturedUrl } = createFakeFetch([]);
    await listActivities({ limit: 10 }, fakeFetch);

    const url = getCapturedUrl();
    expect(url).toContain("limit=10");
  });

  it("adds both sport_type and limit params", async () => {
    const { fakeFetch, getCapturedUrl } = createFakeFetch([]);
    await listActivities({ sport_type: "run", limit: 20 }, fakeFetch);

    const url = getCapturedUrl();
    expect(url).toContain("sport_type=run");
    expect(url).toContain("limit=20");
  });

  it("adds cursor param when provided", async () => {
    const { fakeFetch, getCapturedUrl } = createFakeFetch(emptyPage);
    await listActivities({ cursor: "abc123" }, fakeFetch);

    const url = getCapturedUrl();
    expect(url).toContain("cursor=abc123");
  });

  it("returns the activities page envelope", async () => {
    const mockPage: Page<Activity> = {
      items: [
        {
          id: "act1",
          sport_type: "run",
          title: "Morning Run",
          started_at: "2024-01-01T08:00:00Z",
          timezone: "UTC",
          source_kind: "manual",
          first_raw_file_id: "file1",
          created_at: "2024-01-01T08:00:00Z",
          updated_at: "2024-01-01T08:00:00Z",
        },
      ],
      next_cursor: "next123",
      has_more: true,
    };
    const { fakeFetch } = createFakeFetch(mockPage);

    const result = await listActivities({}, fakeFetch);
    expect(result).toEqual(mockPage);
  });
});

describe("getActivity", () => {
  it("builds correct path with id", async () => {
    const { fakeFetch, getCapturedUrl } = createFakeFetch({});
    await getActivity("test-id", fakeFetch);

    const url = getCapturedUrl();
    expect(url).toContain("/api/v1/activities/test-id");
  });

  it("encodes special characters in id", async () => {
    const { fakeFetch, getCapturedUrl } = createFakeFetch({});
    await getActivity("test id with spaces", fakeFetch);

    const url = getCapturedUrl();
    expect(url).toContain(encodeURIComponent("test id with spaces"));
    expect(url).not.toContain("test id with spaces");
  });

  it("returns the activity object", async () => {
    const mockActivity: Activity = {
      id: "act1",
      sport_type: "run",
      title: "Morning Run",
      started_at: "2024-01-01T08:00:00Z",
      timezone: "UTC",
      distance_m: 5000,
      source_kind: "manual",
      first_raw_file_id: "file1",
      created_at: "2024-01-01T08:00:00Z",
      updated_at: "2024-01-01T08:00:00Z",
    };
    const { fakeFetch } = createFakeFetch(mockActivity);

    const result = await getActivity("test-id", fakeFetch);
    expect(result).toEqual(mockActivity);
  });
});

describe("getActivityRoute", () => {
  it("builds correct path with id", async () => {
    const { fakeFetch, getCapturedUrl } = createFakeFetch([]);
    await getActivityRoute("test-id", fakeFetch);

    const url = getCapturedUrl();
    expect(url).toContain("/api/v1/activities/test-id/route");
  });

  it("encodes special characters in id", async () => {
    const { fakeFetch, getCapturedUrl } = createFakeFetch([]);
    await getActivityRoute("test/id", fakeFetch);

    const url = getCapturedUrl();
    expect(url).toContain(encodeURIComponent("test/id"));
  });

  it("returns route points array", async () => {
    const mockRoute: RoutePoint[] = [
      {
        seq: 1,
        lat: 51.5,
        lon: -0.1,
        elevation_m: 100,
        distance_m: 0,
        speed_mps: 0,
      },
      {
        seq: 2,
        lat: 51.51,
        lon: -0.11,
        elevation_m: 105,
        distance_m: 100,
        speed_mps: 5,
      },
    ];
    const { fakeFetch } = createFakeFetch(mockRoute);

    const result = await getActivityRoute("test-id", fakeFetch);
    expect(result).toEqual(mockRoute);
  });
});

describe("getActivitySamplings", () => {
  it("builds correct path with id", async () => {
    const { fakeFetch, getCapturedUrl } = createFakeFetch([]);
    await getActivitySamplings("test-id", undefined, fakeFetch);

    const url = getCapturedUrl();
    expect(url).toContain("/api/v1/activities/test-id/samplings");
  });

  it("encodes special characters in id", async () => {
    const { fakeFetch, getCapturedUrl } = createFakeFetch([]);
    await getActivitySamplings("test?id", undefined, fakeFetch);

    const url = getCapturedUrl();
    expect(url).toContain(encodeURIComponent("test?id"));
  });

  it("returns sampling points array", async () => {
    const mockSamplings: SamplingPoint[] = [
      {
        id: "samp1",
        sampling_type: "heart_rate",
        ts: "2024-01-01T08:00:00Z",
        value: 120,
        unit: "bpm",
      },
      {
        id: "samp2",
        sampling_type: "heart_rate",
        ts: "2024-01-01T08:01:00Z",
        value: 125,
        unit: "bpm",
      },
    ];
    const { fakeFetch } = createFakeFetch(mockSamplings);

    const result = await getActivitySamplings("test-id", undefined, fakeFetch);
    expect(result).toEqual(mockSamplings);
  });
});

describe("getActivityLaps", () => {
  it("builds correct path with id", async () => {
    const { fakeFetch, getCapturedUrl } = createFakeFetch([]);
    await getActivityLaps("test-id", fakeFetch);

    const url = getCapturedUrl();
    expect(url).toContain("/api/v1/activities/test-id/laps");
  });

  it("encodes special characters in id", async () => {
    const { fakeFetch, getCapturedUrl } = createFakeFetch([]);
    await getActivityLaps("test&id", fakeFetch);

    const url = getCapturedUrl();
    expect(url).toContain(encodeURIComponent("test&id"));
  });

  it("returns laps array", async () => {
    const mockLaps: Lap[] = [
      {
        id: "lap1",
        lap_no: 1,
        distance_m: 1000,
        duration_s: 300,
        avg_hr: 120,
        avg_pace_s_per_km: 300,
      },
      {
        id: "lap2",
        lap_no: 2,
        distance_m: 1000,
        duration_s: 295,
        avg_hr: 125,
        avg_pace_s_per_km: 295,
      },
    ];
    const { fakeFetch } = createFakeFetch(mockLaps);

    const result = await getActivityLaps("test-id", fakeFetch);
    expect(result).toEqual(mockLaps);
  });
});

describe("getPublicSummary", () => {
  it("builds correct path", async () => {
    const mockSummary: Summary = {
      totals: {
        activity_count: 0,
        distance_m: 0,
        duration_s: 0,
        moving_time_s: 0,
      },
      by_year: [],
      by_month: [],
      by_sport: [],
    };
    const { fakeFetch, getCapturedUrl } = createFakeFetch(mockSummary);
    await getPublicSummary({}, fakeFetch);

    const url = getCapturedUrl();
    expect(url).toContain("/public/v1/summary");
  });

  it("returns the summary object", async () => {
    const mockSummary: Summary = {
      totals: {
        activity_count: 10,
        distance_m: 50000,
        duration_s: 18000,
        moving_time_s: 0,
      },
      by_year: [
        {
          key: "2026",
          activity_count: 10,
          distance_m: 50000,
          duration_s: 18000,
          moving_time_s: 0,
        },
      ],
      by_month: [
        {
          key: "2026-07",
          activity_count: 5,
          distance_m: 25000,
          duration_s: 9000,
          moving_time_s: 0,
        },
      ],
      by_sport: [
        {
          key: "run",
          activity_count: 8,
          distance_m: 40000,
          duration_s: 14000,
          moving_time_s: 0,
        },
      ],
    };
    const { fakeFetch } = createFakeFetch(mockSummary);

    const result = await getPublicSummary({}, fakeFetch);
    expect(result).toEqual(mockSummary);
  });

  it("encodes year and sport params", async () => {
    const mockSummary: Summary = {
      totals: {
        activity_count: 0,
        distance_m: 0,
        duration_s: 0,
        moving_time_s: 0,
      },
      by_year: [],
      by_month: [],
      by_sport: [],
    };
    const { fakeFetch, getCapturedUrl } = createFakeFetch(mockSummary);
    await getPublicSummary({ year: "2025", sport: "run" }, fakeFetch);

    const url = getCapturedUrl();
    expect(url).toContain("year=2025");
    expect(url).toContain("sport=run");
  });
});

describe("listPublicActivities", () => {
  it("builds URL without params", async () => {
    const { fakeFetch, getCapturedUrl } = createFakeFetch(emptyPage);
    await listPublicActivities({}, fakeFetch);

    const url = getCapturedUrl();
    expect(url).toContain("/public/v1/activities");
    expect(url).not.toContain("?");
  });

  it("adds sport_type param when provided", async () => {
    const { fakeFetch, getCapturedUrl } = createFakeFetch(emptyPage);
    await listPublicActivities({ sport_type: "ride" }, fakeFetch);

    const url = getCapturedUrl();
    expect(url).toContain("sport_type=ride");
  });

  it("adds limit and cursor params when provided", async () => {
    const { fakeFetch, getCapturedUrl } = createFakeFetch(emptyPage);
    await listPublicActivities({ limit: 20, cursor: "abc123" }, fakeFetch);

    const url = getCapturedUrl();
    expect(url).toContain("limit=20");
    expect(url).toContain("cursor=abc123");
  });

  it("adds distance bound params when provided", async () => {
    const { fakeFetch, getCapturedUrl } = createFakeFetch(emptyPage);
    await listPublicActivities(
      { min_distance_m: 1000, max_distance_m: 5000 },
      fakeFetch,
    );

    const url = getCapturedUrl();
    expect(url).toContain("min_distance_m=1000");
    expect(url).toContain("max_distance_m=5000");
  });

  it("returns the public activities page envelope", async () => {
    const mockPage: Page<PublicActivity> = {
      items: [
        {
          id: "act1",
          sport_type: "run",
          title: "Morning Run",
          started_at: "2026-01-01T08:00:00Z",
          timezone: "UTC",
          distance_m: 5000,
          duration_s: 1500,
        },
      ],
      next_cursor: null,
      has_more: false,
    };
    const { fakeFetch } = createFakeFetch(mockPage);

    const result = await listPublicActivities({}, fakeFetch);
    expect(result).toEqual(mockPage);
  });
});

describe("getPublicRoutes", () => {
  it("builds correct path", async () => {
    const emptyCollection: RouteFeatureCollection = {
      type: "FeatureCollection",
      features: [],
    };
    const { fakeFetch, getCapturedUrl } = createFakeFetch(emptyCollection);
    await getPublicRoutes(fakeFetch);

    const url = getCapturedUrl();
    expect(url).toContain("/public/v1/routes");
  });

  it("returns the route feature collection", async () => {
    const mockCollection: RouteFeatureCollection = {
      type: "FeatureCollection",
      features: [
        {
          type: "Feature",
          geometry: {
            type: "LineString",
            coordinates: [
              [-0.1, 51.5],
              [-0.11, 51.51],
            ],
          },
          properties: { sport_type: "run", year: "2026" },
        },
      ],
    };
    const { fakeFetch } = createFakeFetch(mockCollection);

    const result = await getPublicRoutes(fakeFetch);
    expect(result).toEqual(mockCollection);
  });
});

describe("error handling", () => {
  it("rejects when fetch returns ok: false with status and statusText", async () => {
    const { fakeFetch } = createFakeFetch(null, false, 404, "Not Found");

    await expect(listActivities({}, fakeFetch)).rejects.toThrow(
      /request failed: 404 Not Found/,
    );
  });

  it("includes the path in error message", async () => {
    const { fakeFetch } = createFakeFetch(
      null,
      false,
      500,
      "Internal Server Error",
    );

    await expect(getActivity("test-id", fakeFetch)).rejects.toThrow(
      /request failed: 500 Internal Server Error/,
    );
  });

  it("handles 403 Forbidden errors", async () => {
    const { fakeFetch } = createFakeFetch(null, false, 403, "Forbidden");

    await expect(getActivityRoute("restricted-id", fakeFetch)).rejects.toThrow(
      /request failed: 403 Forbidden/,
    );
  });
});
