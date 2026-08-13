import { describe, expect, it } from "vitest";

const routePages = import.meta.glob("./**/+page.svelte", { eager: true });
const routeLoads = import.meta.glob("./**/+page.ts", { eager: true });

describe("root route", () => {
  it("renders the day-aggregator cockpit directly (no redirect)", () => {
    expect(routePages["./+page.svelte"]).toBeDefined();
  });
});

describe("cockpit route layout", () => {
  it("has concrete page files for the user-facing primary URLs", () => {
    expect(routePages["./patterns/+page.svelte"]).toBeDefined();
    expect(routePages["./motion/+page.svelte"]).toBeDefined();
    expect(routePages["./library/+page.svelte"]).toBeDefined();
    expect(routePages["./library/[id]/+page.svelte"]).toBeDefined();
    expect(routePages["./overview/+page.svelte"]).toBeDefined();
    expect(routePages["./night/+page.svelte"]).toBeDefined();
    expect(routePages["./night/[id]/+page.svelte"]).toBeDefined();
    expect(routePages["./to-go/+page.svelte"]).toBeDefined();
    expect(routePages["./admin/+page.svelte"]).toBeDefined();
    expect(routePages["./expenses/+page.svelte"]).toBeDefined();
    expect(routePages["./reports/+page.svelte"]).toBeDefined();
    expect(routePages["./metrics/+page.svelte"]).toBeDefined();
  });

  it("keeps the old page URLs as redirects", () => {
    expect(routeLoads["./dashboard/+page.ts"]).toBeDefined();
    expect(routeLoads["./daily/+page.ts"]).toBeDefined();
    expect(routeLoads["./activities/+page.ts"]).toBeDefined();
    expect(routeLoads["./activities/[id]/+page.ts"]).toBeDefined();
    expect(routeLoads["./sleep/+page.ts"]).toBeDefined();
    expect(routeLoads["./sleep/[id]/+page.ts"]).toBeDefined();
    expect(routeLoads["./media/+page.ts"]).toBeDefined();
    expect(routeLoads["./media/[id]/+page.ts"]).toBeDefined();
  });

  it("keeps navigation grouped instead of growing the top tab row", async () => {
    const { navigationGroups } = await import("$lib/navigation");
    expect(navigationGroups.map((group) => group.label)).toEqual([
      "Primary",
      "Domains",
      "Analyze",
      "More",
    ]);
    expect(navigationGroups[0].items.map((item) => item.href)).toEqual([
      "/",
      "/overview",
    ]);
    expect(
      navigationGroups
        .slice(1)
        .flatMap((group) => group.items)
        .map((item) => item.href),
    ).toEqual([
      "/motion",
      "/night",
      "/library",
      "/expenses",
      "/patterns",
      "/reports",
      "/to-go",
      "/admin",
      "/design",
    ]);
  });

  it("does not keep the old /u page route", () => {
    expect(routePages["./u/+page.svelte"]).toBeUndefined();
  });
});
