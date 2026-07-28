import { describe, expect, it } from "vitest";

const routePages = import.meta.glob("./**/+page.svelte", { eager: true });

describe("root route", () => {
  it("renders the day-aggregator cockpit directly (no redirect)", () => {
    expect(routePages["./+page.svelte"]).toBeDefined();
  });
});

describe("cockpit route layout", () => {
  it("has concrete route files for the daily, activities, and media domains", () => {
    expect(routePages["./daily/+page.svelte"]).toBeDefined();
    expect(routePages["./activities/+page.svelte"]).toBeDefined();
    expect(routePages["./media/+page.svelte"]).toBeDefined();
    expect(routePages["./media/[id]/+page.svelte"]).toBeDefined();
  });

  it("does not keep the old /u page route", () => {
    expect(routePages["./u/+page.svelte"]).toBeUndefined();
  });
});
