import type { Page } from "@playwright/test";
import type { Expense, MetricSeriesResponse } from "../src/lib/api";
import { fakeSession } from "./session";

const instant = "2026-08-14T12:00:00Z";
const rows: Expense[] = [
  ["august", "2026-08-14", "JPY", 0, 12345, "food"],
  ["september", "2026-09-14", "USD", 2, 123456, "shopping"],
  ["previous", "2025-03-11", "EUR", 2, 2500, "housing"],
].map(([id, date, currency, exponent, amount, category]) => ({
  id: `synthetic-${id}`,
  occurred_on: String(date),
  account_key: "synthetic",
  kind: "expense",
  currency: currency as Expense["currency"],
  currency_exponent: Number(exponent),
  amount_minor: Number(amount),
  category: category as Expense["category"],
  merchant: `Synthetic ${id}`,
  note: "Synthetic fixture only",
  items: [],
  source: { kind: "manual", ref: `synthetic-${id}` },
  created_at: instant,
  updated_at: instant,
}));

// Mirror metricseries.expenseDimensionSeries: every calendar bucket exists,
// unobserved money/count is null, refunds subtract, genuine net zero is zero.
function series(url: URL, expenses: Expense[]): MetricSeriesResponse {
  const id = decodeURIComponent(url.pathname.split("/")[4]);
  const from = url.searchParams.get("from")!;
  const to = url.searchParams.get("to")!;
  const grain = url.searchParams.get("grain") as "day" | "month" | "year";
  const periods: string[] = [];
  for (let day = from; day < to;) {
    const key =
      grain === "year"
        ? day.slice(0, 4)
        : grain === "month"
          ? day.slice(0, 7)
          : day;
    if (!periods.includes(key)) periods.push(key);
    const next = new Date(`${day}T00:00:00Z`);
    next.setUTCDate(next.getUTCDate() + 1);
    day = next.toISOString().slice(0, 10);
  }
  const dimensions = url.searchParams.getAll("dimension");
  const currencies = dimensions
    .filter((v) => v.startsWith("currency:"))
    .map((v) => v.slice(9));
  const categories = dimensions
    .filter((v) => v.startsWith("category:"))
    .map((v) => v.slice(9));
  const groups = currencies.flatMap<Record<string, string>>((currency) =>
    categories.length
      ? categories.map((category) => ({ currency, category }))
      : [{ currency }],
  );
  const money = id === "expenses.amount_minor";
  return {
    schema: "metric-series.v1",
    metric_id: id,
    label: "Expenses",
    unit: money ? "minor currency unit" : "count",
    value_type: money ? "money" : "number",
    period: { grain, from, to, timezone: "Asia/Tokyo" },
    series: groups.map((group) => {
      const matching = expenses.filter(
        (row) =>
          row.occurred_on >= from &&
          row.occurred_on < to &&
          row.currency === group.currency &&
          (!group.category || row.category === group.category),
      );
      const observed = new Set(
        matching.map((row) =>
          grain === "year"
            ? row.occurred_on.slice(0, 4)
            : grain === "month"
              ? row.occurred_on.slice(0, 7)
              : row.occurred_on,
        ),
      );
      return {
        dimensions: group,
        points: periods.map((period) => {
          const selected = matching.filter((row) =>
            row.occurred_on.startsWith(period),
          );
          return {
            period,
            observed_days: new Set(selected.map((row) => row.occurred_on)).size,
            ...(money
              ? {
                  value_minor: selected.length
                    ? selected.reduce(
                        (sum, row) =>
                          sum +
                          (row.kind === "refund"
                            ? -row.amount_minor
                            : row.amount_minor),
                        0,
                      )
                    : null,
                }
              : { value: selected.length ? selected.length : null }),
          };
        }),
        coverage: {
          expected_periods: periods.length,
          observed_periods: observed.size,
          observation_state: observed.size ? "observed" : "empty",
          collection_completeness: "unknown",
        },
        source: {
          kind: "derived",
          method: money ? "expenses.sum_active.v1" : "expenses.count_active.v1",
          source_kinds: matching.length ? ["manual"] : [],
        },
      };
    }),
  };
}

export async function installExpensesFixtures(page: Page) {
  const fixture = {
    failures: new Set<string>(),
    holds: new Map<string, Promise<void>>(),
    requests: [] as string[],
    unknown: [] as string[],
    empty: false,
    netZero: false,
  };
  await page.route("**/api/v1/**", async (route) => {
    const url = new URL(route.request().url());
    const path = url.pathname;
    const dependency =
      path === "/api/v1/expenses/bounds"
        ? "bounds"
        : path.endsWith("/series")
          ? "spending"
          : route.request().method() === "DELETE"
            ? "delete"
            : path === "/api/v1/expenses"
              ? "ledger"
              : "unknown";
    const scope =
      url.searchParams.get("date") ??
      url.searchParams.get("from")?.slice(0, 7) ??
      "";
    const key = `${dependency}:${scope}`;
    fixture.requests.push(`${route.request().method()} ${path}${url.search}`);
    await (fixture.holds.get(key) ?? fixture.holds.get(dependency));
    if (fixture.failures.has(key) || fixture.failures.has(dependency))
      return route.fulfill({
        status: 503,
        json: { error: `Synthetic ${dependency} failure` },
      });
    const expenses = fixture.empty
      ? []
      : fixture.netZero
        ? [
            ...rows,
            {
              ...rows[0],
              id: "synthetic-refund",
              kind: "refund" as const,
              refund_of_expense_id: rows[0].id,
              merchant: "Synthetic refund",
            },
          ]
        : rows;
    if (dependency === "bounds")
      return route.fulfill({
        json: fixture.empty ? {} : { min: "2025-03-11", max: "2026-09-14" },
      });
    if (dependency === "spending")
      return route.fulfill({ json: series(url, expenses) });
    if (dependency === "ledger")
      return route.fulfill({
        json: {
          items: expenses.filter(
            (row) =>
              row.occurred_on.startsWith(scope) &&
              (!url.searchParams.get("currency") ||
                row.currency === url.searchParams.get("currency")) &&
              (!url.searchParams.get("category") ||
                row.category === url.searchParams.get("category")),
          ),
          has_more: false,
          next_cursor: null,
        },
      });
    if (dependency === "delete") return route.fulfill({ status: 204 });
    fixture.unknown.push(path);
    return route.fulfill({
      status: 404,
      json: { error: "Unmodelled endpoint" },
    });
  });
  await fakeSession(page, {
    setup_required: false,
    authenticated: true,
    username: "synthetic-owner",
    display_name: "Synthetic owner",
  });
  return fixture;
}
