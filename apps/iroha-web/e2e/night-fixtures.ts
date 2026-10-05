import type { Page } from "@playwright/test";
import type {
  SleepSession,
  SleepAggregateBucket,
} from "@iroha/shared/domain/sleep";
import { fakeSession } from "./session";

type Read = "records" | "year" | "month" | "lifetime" | "bounds";

function aggregate(period: string, rows: SleepSession[]): SleepAggregateBucket {
  const main = rows.filter((row) => row.is_main_sleep);
  const average = (key: "asleep_s" | "time_in_bed_s" | "efficiency") =>
    main.length
      ? main.reduce((sum, row) => sum + row[key], 0) / main.length
      : 0;
  const sum = (
    key: "core_s" | "deep_s" | "rem_s" | "awake_s" | "unspecified_s",
  ) => main.reduce((sum, row) => sum + row[key], 0);
  return {
    period,
    session_count: rows.length,
    main_sleep_count: main.length,
    nap_count: rows.length - main.length,
    observed_wake_dates: new Set(rows.map((row) => row.wake_date)).size,
    average_asleep_s: average("asleep_s"),
    average_time_in_bed_s: average("time_in_bed_s"),
    average_efficiency: average("efficiency"),
    core_s: sum("core_s"),
    deep_s: sum("deep_s"),
    rem_s: sum("rem_s"),
    awake_s: sum("awake_s"),
    unspecified_s: sum("unspecified_s"),
  };
}

export async function installNightFixtures(page: Page) {
  const fixture = {
    failures: new Set<Read>(),
    holds: new Map<Read, Promise<void>>(),
    requests: [] as string[],
    unknown: [] as string[],
    empty: false,
    napsOnly: false,
    paginated: false,
    pageFailures: new Set<string>(),
    pageHolds: new Map<string, Promise<void>>(),
    pageHold: null as Promise<void> | null,
  };
  await page.route("**/api/v1/**", async (route) => {
    const url = new URL(route.request().url());
    fixture.requests.push(url.pathname + url.search);
    const read: Read | null =
      url.pathname === "/api/v1/sleep"
        ? "records"
        : url.pathname === "/api/v1/sleep/bounds"
          ? "bounds"
          : url.pathname === "/api/v1/sleep/aggregates"
            ? (url.searchParams.get("granularity") as Read)
            : null;
    if (
      !read ||
      !["records", "year", "month", "lifetime", "bounds"].includes(read)
    ) {
      fixture.unknown.push(url.pathname + url.search);
      return route.fulfill({
        status: 404,
        json: { error: "Unknown synthetic endpoint" },
      });
    }
    if (url.searchParams.has("cursor")) {
      const key = url.searchParams.get("date") || "lifetime";
      await (fixture.pageHolds.get(key) ??
        fixture.pageHold ??
        fixture.holds.get(read));
      if (fixture.pageFailures.has(key))
        return route.fulfill({
          status: 503,
          json: { error: "Synthetic sleep cursor failure" },
        });
    } else if (fixture.holds.has(read)) await fixture.holds.get(read);
    if (fixture.failures.has(read))
      return route.fulfill({
        status: 503,
        json: { error: "Synthetic sleep failure" },
      });
    const sessions: SleepSession[] = fixture.empty
      ? []
      : ["2025-08", "2025-09", "2026-08", "2026-09"].flatMap((month) => {
          const asleep = month.endsWith("09") ? 21600 : 25200;
          const session: SleepSession = {
            id: `synthetic-${month}`,
            wake_date: `${month}-14`,
            started_at: `${month}-13T22:00:00Z`,
            ended_at: `${month}-14T06:00:00Z`,
            time_in_bed_s: 28800,
            asleep_s: asleep,
            efficiency: asleep / 28800,
            is_main_sleep: !fixture.napsOnly,
            core_s: asleep - 7200,
            deep_s: 3600,
            rem_s: 3600,
            awake_s: 28800 - asleep,
            unspecified_s: 0,
            source: "synthetic",
            first_raw_file_id: "synthetic-raw",
            created_at: "2026-10-01T12:00:00Z",
            updated_at: "2026-10-01T12:00:00Z",
          };
          if (fixture.napsOnly)
            Object.assign(session, {
              started_at: `${month}-14T12:00:00Z`,
              ended_at: `${month}-14T13:00:00Z`,
              time_in_bed_s: 3600,
              asleep_s: 3000,
              efficiency: 3000 / 3600,
              core_s: 3000,
              deep_s: 0,
              rem_s: 0,
              awake_s: 600,
            });
          return fixture.paginated
            ? [
                session,
                {
                  ...session,
                  id: `${session.id}-next`,
                  wake_date: `${month}-15`,
                  started_at: `${month}-14T22:00:00Z`,
                  ended_at: `${month}-15T06:00:00Z`,
                },
              ]
            : [session];
        });
    if (read === "bounds")
      return route.fulfill({
        json: sessions.length
          ? {
              min: sessions.map((row) => row.wake_date).sort()[0],
              max: sessions
                .map((row) => row.wake_date)
                .sort()
                .at(-1),
            }
          : {},
      });
    const date = url.searchParams.get("date");
    const selected = sessions.filter(
      (session) => !date || session.wake_date.startsWith(date),
    );
    if (read === "records") {
      selected.sort((a, b) => b.wake_date.localeCompare(a.wake_date));
      const next = url.searchParams.has("cursor");
      const items = fixture.paginated
        ? selected.slice(next ? 1 : 0, next ? undefined : 1)
        : selected;
      const more = fixture.paginated && !next && selected.length > items.length;
      return route.fulfill({
        json: {
          items,
          has_more: more,
          next_cursor: more ? "synthetic-cursor" : null,
        },
      });
    }
    const groups = new Map<string, SleepSession[]>();
    for (const session of selected) {
      const period =
        read === "lifetime"
          ? "lifetime"
          : session.wake_date.slice(0, read === "year" ? 4 : 7);
      groups.set(period, [...(groups.get(period) ?? []), session]);
    }
    const buckets = [...groups]
      .sort(([a], [b]) => a.localeCompare(b))
      .map(([period, rows]) => aggregate(period, rows));
    return route.fulfill({ json: { granularity: read, buckets } });
  });
  await fakeSession(page, {
    setup_required: false,
    authenticated: true,
    username: "synthetic-owner",
  });
  return fixture;
}
