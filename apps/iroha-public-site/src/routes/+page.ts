import type {
  Activity,
  Meta,
  RouteFeatureCollection,
  Summary,
} from "$lib/types";
import type { PageLoad } from "./$types";

// The site reads the live, sanitized projection from iroha-server's
// anonymous /public/v1 API in the browser (ADR-0008). The server caches it
// for up to a day and rebuilds when activities change, so there is no build
// job and no data baked into the HTML.
export const ssr = false;
export const prerender = false;

export const load: PageLoad = async ({ fetch }) => {
  const get = async <T>(path: string): Promise<T> => {
    const res = await fetch(`/public/v1/${path}`);
    if (!res.ok) throw new Error(`public ${path}: HTTP ${res.status}`);
    return (await res.json()) as T;
  };
  const [summary, activities, routes, meta] = await Promise.all([
    get<Summary>("summary"),
    get<Activity[]>("activities"),
    get<RouteFeatureCollection>("routes"),
    get<Meta>("meta"),
  ]);
  return { summary, activities, routes, meta };
};
