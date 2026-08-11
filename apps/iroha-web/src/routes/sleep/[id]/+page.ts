import { redirect } from "@sveltejs/kit";

export function load({ params, url }) {
  redirect(308, `/night/${encodeURIComponent(params.id)}${url.search}`);
}
