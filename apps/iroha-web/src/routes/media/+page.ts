import { redirect } from "@sveltejs/kit";

export function load({ url }) {
  redirect(308, `/library${url.search}`);
}
