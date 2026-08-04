import type { RoutePoint } from "$lib/api";

function haversineDistanceM(
  lat1: number,
  lon1: number,
  lat2: number,
  lon2: number,
): number {
  const earthRadiusM = 6_371_000;
  const phi1 = (lat1 * Math.PI) / 180;
  const phi2 = (lat2 * Math.PI) / 180;
  const deltaPhi = ((lat2 - lat1) * Math.PI) / 180;
  const deltaLambda = ((lon2 - lon1) * Math.PI) / 180;
  const a =
    Math.sin(deltaPhi / 2) ** 2 +
    Math.cos(phi1) * Math.cos(phi2) * Math.sin(deltaLambda / 2) ** 2;
  return earthRadiusM * 2 * Math.atan2(Math.sqrt(a), Math.sqrt(1 - a));
}

export function populateRouteDistances(
  routePoints: readonly RoutePoint[],
): RoutePoint[] {
  if (routePoints.length === 0) return [];
  const points = routePoints.map((point) => ({ ...point }));
  points[0].distance_m ??= 0;
  for (let index = 1; index < points.length; index += 1) {
    if (points[index].distance_m != null) continue;
    const previous = points[index - 1];
    const current = points[index];
    points[index].distance_m =
      (previous.distance_m ?? 0) +
      haversineDistanceM(previous.lat, previous.lon, current.lat, current.lon);
  }
  return points;
}

// The imported activity may omit distance while retaining GPS fixes. This is
// a presentation-only distance derived from those source coordinates.
export function deriveRouteDistanceM(
  routePoints: readonly RoutePoint[],
): number | undefined {
  const points = populateRouteDistances(routePoints);
  return points.length > 1 ? points.at(-1)?.distance_m : undefined;
}
