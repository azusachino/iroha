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
    expect(routePages["./expenses/+page.svelte"]).toBeDefined();
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
    expect(routeLoads["./admin/+page.ts"]).toBeDefined();
  });

  it("uses the tab vocabulary in the primary navigation URLs", async () => {
    const { primaryNavigation } = await import("$lib/navigation");
    expect(primaryNavigation).toEqual([
      { label: "Today", href: "/" },
      { label: "Overview", href: "/overview" },
      { label: "Patterns", href: "/patterns" },
      { label: "Motion", href: "/motion" },
      { label: "Night", href: "/night" },
      { label: "Library", href: "/library" },
      { label: "To-go", href: "/to-go" },
      { label: "Expenses", href: "/expenses" },
    ]);
  });

  it("does not keep the old /u page route", () => {
    expect(routePages["./u/+page.svelte"]).toBeUndefined();
  });
});
