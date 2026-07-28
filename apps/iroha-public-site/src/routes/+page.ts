import type { Activity, RouteFeatureCollection, Summary } from "$lib/types";
import type { PageLoad } from "./$types";

// This whole site is one known page over one known snapshot -- prerender it
// into static HTML at build time rather than shipping a client-rendered
// shell that fetches on mount. `fetch` here resolves against SvelteKit's
// own static-file server during prerendering, so this reads straight from
// static/data/ without any live backend.
export const prerender = true;

export const load: PageLoad = async ({ fetch }) => {
  const [summaryRes, activitiesRes, routesRes] = await Promise.all([
    fetch("data/summary.json"),
    fetch("data/activities.json"),
    fetch("data/routes.geojson"),
  ]);

  const summary: Summary = await summaryRes.json();
  const activities: Activity[] = await activitiesRes.json();
  const routes: RouteFeatureCollection = await routesRes.json();

  return { summary, activities, routes };
};
