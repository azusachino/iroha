import type {
  Activity,
  Meta,
  RouteFeatureCollection,
  Summary,
} from "$lib/public/types";
import type { PageLoad } from "./$types";
import { error } from "@sveltejs/kit";

// The root route is the public activity showcase, reading live sanitized data
// from iroha-server's anonymous /public/v1 API (ADR-0008).
export const ssr = false;
export const prerender = false;

export const load: PageLoad = async ({ fetch }) => {
  const get = async <T>(path: string): Promise<T> => {
    const res = await fetch(`/public/v1/${path}`);
    if (!res.ok) {
      error(res.status, "Public activity data is temporarily unavailable.");
    }
    return (await res.json()) as T;
  };

  const [summary, activities, meta] = await Promise.all([
    get<Summary>("summary"),
    get<Activity[]>("activities"),
    get<Meta>("meta"),
  ]);

  let _routesPromise: Promise<RouteFeatureCollection> | null = null;
  const loadRoutes = () => {
    if (!_routesPromise) {
      _routesPromise = get<RouteFeatureCollection>("routes");
    }
    return _routesPromise;
  };

  return { summary, activities, meta, loadRoutes };
};
