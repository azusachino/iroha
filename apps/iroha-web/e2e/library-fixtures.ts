import type { Page } from "@playwright/test";
import type { MediaAggregates, MediaRow } from "@iroha/shared/domain/media";
import { fakeSession } from "./session";

type Read = "records" | "aggregates";
const rows: MediaRow[] = [
  {
    id: "00000000-0000-4000-8000-000000000003",
    title: "Synthetic anime",
    media_type: "anime_season",
    item_role: "standalone",
    status: "completed",
    rating: 8,
    completed_on: "2026-08-14",
    last_update_at: "2026-09-15T12:00:00Z",
  },
  {
    id: "00000000-0000-4000-8000-000000000002",
    title: "Synthetic book",
    media_type: "book",
    item_role: "standalone",
    status: "in_progress",
    position: 5,
    total: 20,
    unit: "chapters",
    last_update_at: "2026-09-14T12:00:00Z",
  },
  {
    id: "00000000-0000-4000-8000-000000000001",
    title: "Synthetic game",
    media_type: "game",
    item_role: "standalone",
    status: "planned",
    last_update_at: "2026-09-13T12:00:00Z",
  },
];
const families: Record<string, string[]> = {
  anime: ["anime_season", "movie", "ona", "ova", "special"],
  manga_book: ["manga", "one_shot", "light_novel"],
  book: ["book"],
  game: ["game"],
};
// Latest completion is historical evidence, not the current status or progress date.
const completionEvents: Record<string, string[]> = {
  "00000000-0000-4000-8000-000000000002": ["2024-08-14", "2025-08-14"],
};
function latestCompletion(row: MediaRow) {
  return [
    ...(row.completed_on ? [row.completed_on] : []),
    ...(completionEvents[row.id] ?? []),
  ]
    .sort()
    .at(-1);
}
export async function installLibraryFixtures(page: Page) {
  const fixture = {
    failures: new Set<Read>(),
    holds: new Map<Read, Promise<void>>(),
    requests: [] as string[],
    unknown: [] as string[],
    empty: false,
    zeroRating: false,
    paused: false,
    animeType: "anime_season",
    bookType: "book",
    paginated: false,
    pageFailures: new Set<string>(),
    pageHolds: new Map<string, Promise<void>>(),
    pageHold: null as Promise<void> | null,
  };
  await page.route("**/api/v1/**", async (route) => {
    const url = new URL(route.request().url());
    fixture.requests.push(url.pathname + url.search);
    const read: Read | null =
      url.pathname === "/api/v1/media"
        ? "records"
        : url.pathname === "/api/v1/media/aggregates"
          ? "aggregates"
          : null;
    if (!read) {
      fixture.unknown.push(url.pathname + url.search);
      return route.fulfill({
        status: 404,
        json: { error: "Unknown synthetic endpoint" },
      });
    }
    const family = url.searchParams.get("family");
    const year = url.searchParams.get("completed_year");
    if (
      (family && !families[family]) ||
      (year &&
        (!Number.isInteger(Number(year)) ||
          Number(year) < 1 ||
          Number(year) > 9999))
    )
      return route.fulfill({
        status: 400,
        json: { error: "Invalid synthetic facet" },
      });
    if (url.searchParams.has("cursor")) {
      const key = family || "all";
      await (fixture.pageHolds.get(key) ??
        fixture.pageHold ??
        fixture.holds.get(read));
      if (fixture.pageFailures.has(key))
        return route.fulfill({
          status: 503,
          json: { error: "Synthetic library cursor failure" },
        });
    } else if (fixture.holds.has(read)) await fixture.holds.get(read);
    if (fixture.failures.has(read))
      return route.fulfill({
        status: 503,
        json: { error: "Synthetic unavailable read" },
      });
    const facet = (
      fixture.empty
        ? []
        : rows.map((row) => ({
            ...row,
            media_type:
              row.media_type === "anime_season"
                ? fixture.animeType
                : row.media_type === "book"
                  ? fixture.bookType
                  : row.media_type,
            hidden_from_continue: row.id.endsWith("0002") && fixture.paused,
            rating:
              row.rating == null
                ? undefined
                : fixture.zeroRating
                  ? 0
                  : row.rating,
          }))
    ).filter((row) => {
      return (
        (!family || families[family].includes(row.media_type)) &&
        (!year || latestCompletion(row)?.slice(0, 4) === year)
      );
    });
    const filtered = facet.filter(
      (row) =>
        !url.searchParams.get("status") ||
        row.status === url.searchParams.get("status"),
    );
    if (read === "records") {
      const counts: Record<string, number> = {};
      for (const row of facet) {
        const key =
          row.status === "in_progress" && row.hidden_from_continue
            ? "paused"
            : (row.status ?? "unknown");
        counts[key] = (counts[key] ?? 0) + 1;
      }
      const cursor = url.searchParams.get("cursor");
      if (cursor && cursor !== "synthetic-next")
        return route.fulfill({
          status: 400,
          json: { error: "Invalid synthetic cursor" },
        });
      const items = fixture.paginated
        ? filtered.slice(cursor ? 1 : 0, cursor ? undefined : 1)
        : filtered;
      return route.fulfill({
        json: {
          items,
          next_cursor:
            fixture.paginated && !cursor && filtered.length > 1
              ? "synthetic-next"
              : null,
          has_more: fixture.paginated && !cursor && filtered.length > 1,
          status_counts: counts,
          active_count: facet.filter(
            (row) => row.status === "in_progress" && !row.hidden_from_continue,
          ).length,
        },
      });
    }
    const completions = new Map<number, number>();
    const scores = new Map<number, number>();
    const types = new Map<string, number>();
    const ratings: number[] = [];
    for (const row of filtered) {
      const completion = latestCompletion(row);
      if (completion) {
        const year = Number(completion.slice(0, 4));
        completions.set(year, (completions.get(year) ?? 0) + 1);
      }
      if (row.rating != null) {
        ratings.push(row.rating);
        const score = Math.round(Math.max(0, Math.min(10, row.rating)));
        scores.set(score, (scores.get(score) ?? 0) + 1);
      }
      types.set(row.media_type, (types.get(row.media_type) ?? 0) + 1);
    }
    const aggregates: MediaAggregates = {
      totals: {
        item_count: filtered.length,
        completed_count: filtered.filter((row) => latestCompletion(row)).length,
        current_completed_count: filtered.filter(
          (row) => row.status === "completed",
        ).length,
        this_year_completed: completions.get(new Date().getUTCFullYear()) ?? 0,
        average_rating: ratings.length
          ? ratings.reduce((a, b) => a + b, 0) / ratings.length
          : 0,
      },
      completions_by_year: [...completions]
        .sort(([a], [b]) => a - b)
        .map(([year, count]) => ({ year, count })),
      score_distribution: [...scores]
        .sort(([a], [b]) => a - b)
        .map(([score, count]) => ({ score, count })),
      type_split: [...types]
        .sort(([a], [b]) => a.localeCompare(b))
        .map(([type, count]) => ({ type, count })),
    };
    return route.fulfill({ json: aggregates });
  });
  await fakeSession(page, {
    setup_required: false,
    authenticated: true,
    username: "synthetic-owner",
  });
  return fixture;
}
