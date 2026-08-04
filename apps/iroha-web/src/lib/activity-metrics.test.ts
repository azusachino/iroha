import { describe, expect, it } from "vitest";
import {
  deriveRouteDistanceM,
  populateRouteDistances,
} from "./activity-metrics";

describe("populateRouteDistances", () => {
  it("derives cumulative GPS distance when source points omit it", () => {
    const points = populateRouteDistances([
      { seq: 0, lat: 35, lon: 139 },
      { seq: 1, lat: 35.001, lon: 139 },
    ]);

    expect(points[0].distance_m).toBe(0);
    expect(points[1].distance_m).toBeGreaterThan(100);
    expect(points[1].distance_m).toBeLessThan(120);
  });

  it("preserves an imported cumulative distance", () => {
    const points = populateRouteDistances([
      { seq: 0, lat: 35, lon: 139, distance_m: 12 },
      { seq: 1, lat: 35.001, lon: 139 },
    ]);

    expect(points[0].distance_m).toBe(12);
    expect(points[1].distance_m).toBeGreaterThan(120);
  });
});

describe("deriveRouteDistanceM", () => {
  it("needs at least two GPS fixes", () => {
    expect(deriveRouteDistanceM([])).toBeUndefined();
    expect(
      deriveRouteDistanceM([{ seq: 0, lat: 35, lon: 139 }]),
    ).toBeUndefined();
  });
});
