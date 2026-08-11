import { redirect } from "@sveltejs/kit";

export function load({ params, url }) {
  redirect(308, `/library/${encodeURIComponent(params.id)}${url.search}`);
}
