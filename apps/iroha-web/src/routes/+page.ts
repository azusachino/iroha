import type { PageLoad } from "./$types";
import type { Activity, Meta, Summary } from "$lib/public/types";

export const ssr = false;
export const prerender = false;

export const load: PageLoad = async ({ fetch, url }) => {
  if (url.searchParams.has("date")) {
    return { isCockpit: true };
  }
  try {
    const get = async <T>(path: string): Promise<T> => {
      const res = await fetch(`/public/v1/${path}`);
      if (!res.ok) {
        throw new Error("Public activity data is temporarily unavailable.");
      }
      return (await res.json()) as T;
    };
    const [summary, activities, meta] = await Promise.all([
      get<Summary>("summary"),
      get<Activity[]>("activities"),
      get<Meta>("meta"),
    ]);
    return { summary, activities, meta, isCockpit: false };
  } catch {
    return {
      isCockpit: false,
      error: "Public activity data is temporarily unavailable.",
    };
  }
};
