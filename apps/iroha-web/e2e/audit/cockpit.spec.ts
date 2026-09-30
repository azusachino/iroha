// Cockpit quality audit (iroha#79, workstream 3): each navigation route, at
// each viewport, in each state, checked with axe (WCAG 2.2 AA) and for
// horizontal overflow; populated pages also get a keyboard walk. Findings are
// soft assertions and annotations, so one run reports the whole matrix; each
// cell also attaches a screenshot and axe's full result. Run it with
// `make e2e-audit`, which starts a seeded throwaway stack.
import AxeBuilder from "@axe-core/playwright";
import type { Page, TestInfo } from "@playwright/test";
import { expect, test } from "./fixtures";

// `static` routes make no API calls, so they have no loading or error state.
type Route = { name: string; path: string; empty?: string; static?: boolean };

// The seed's data is in 2026-08; 2026-03 has none. Routes without a date
// parameter cannot be emptied without faking their API, so they have no
// empty state here.
const ROUTES: Route[] = [
  { name: "today", path: "/?date=2026-08-15", empty: "/?date=2026-03-15" },
  { name: "overview", path: "/overview" },
  {
    name: "motion",
    path: "/motion?date=2026-08",
    empty: "/motion?date=2026-03",
  },
  { name: "night", path: "/night?date=2026-08", empty: "/night?date=2026-03" },
  { name: "library", path: "/library" },
  {
    name: "expenses",
    path: "/expenses?date=2026-08",
    empty: "/expenses?date=2026-03",
  },
  { name: "patterns", path: "/patterns" },
  {
    name: "reports",
    path: "/reports?date=2026-08",
    empty: "/reports?date=2026-03",
  },
  { name: "to-go", path: "/to-go" },
  { name: "admin", path: "/admin" },
  { name: "manual", path: "/manual", static: true },
];

const VIEWPORTS = [
  // WCAG 1.4.10 reflow: a 1280 px window at 400% zoom.
  { name: "320", viewport: { width: 320, height: 640 }, scale: 1 },
  { name: "390", viewport: { width: 390, height: 844 }, scale: 3 },
  { name: "768", viewport: { width: 768, height: 1024 }, scale: 2 },
  { name: "1280", viewport: { width: 1280, height: 800 }, scale: 1 },
  // A 1280x800 window at 200% zoom: half the CSS pixels, twice the density.
  { name: "zoom200", viewport: { width: 640, height: 400 }, scale: 2 },
];

// Loading, error and empty states are checked at one phone and one desktop size.
const STATE_VIEWPORTS = new Set(["390", "1280"]);
const WCAG_TAGS = ["wcag2a", "wcag2aa", "wcag21a", "wcag21aa", "wcag22aa"];
const API = "**/api/v1/**";
const isSession = (url: string) => url.includes("/api/v1/auth/session");

async function signedIn(page: Page) {
  // The skip link renders only for a signed-in owner.
  await expect(
    page.getByRole("link", { name: "Skip to main content" }),
  ).toBeAttached();
}

// A route has settled when no data region is busy and no first-load surface remains.
async function settled(page: Page) {
  await signedIn(page);
  await expect(
    page.locator('[aria-busy="true"], .loading-surface'),
  ).toHaveCount(0, {
    timeout: 20_000,
  });
}

async function audit(page: Page, testInfo: TestInfo) {
  // Let fades finish first: axe reads a mid-transition color as the real one.
  // Infinite animations (the loading spinner) never finish, so skip them.
  await page.waitForFunction(() =>
    document
      .getAnimations()
      .every(
        (a) =>
          a.effect?.getTiming().iterations === Infinity ||
          a.playState !== "running",
      ),
  );
  const result = await new AxeBuilder({ page }).withTags(WCAG_TAGS).analyze();
  const violations = result.violations.map((v) => `${v.id}(${v.nodes.length})`);
  testInfo.annotations.push({
    type: "axe",
    description: violations.join(" ") || "none",
  });
  await testInfo.attach("axe.json", {
    body: JSON.stringify(result.violations, null, 2),
    contentType: "application/json",
  });
  expect.soft(violations, "axe WCAG 2.2 AA violations").toEqual([]);

  const overflow = await page.evaluate(
    () => document.documentElement.scrollWidth - window.innerWidth,
  );
  testInfo.annotations.push({ type: "overflow", description: `${overflow}px` });
  expect.soft(overflow, "horizontal overflow in CSS px").toBeLessThanOrEqual(0);

  await testInfo.attach("page.png", {
    body: await page.screenshot({ fullPage: true }),
    contentType: "image/png",
  });
}

type Stop = { name: string; indicator: boolean; hidden: boolean };

// Tab through the page and record each stop. A stop "without indicator" has
// neither an outline nor a box-shadow when focused; check its screenshot
// before calling it a failure, since a background change also counts.
async function keyboardWalk(page: Page, testInfo: TestInfo) {
  const stops: Stop[] = [];
  for (let i = 0; i < 80; i++) {
    await page.keyboard.press("Tab");
    const stop = await page.evaluate(() => {
      const el = document.activeElement as HTMLElement | null;
      if (!el || el === document.body) return null;
      const style = getComputedStyle(el);
      const box = el.getBoundingClientRect();
      const label =
        el.getAttribute("aria-label") || el.textContent || el.tagName;
      return {
        name: label.replace(/\s+/g, " ").trim().slice(0, 48),
        indicator:
          (style.outlineStyle !== "none" &&
            parseFloat(style.outlineWidth) > 0) ||
          style.boxShadow !== "none",
        hidden:
          box.width === 0 ||
          box.height === 0 ||
          el.closest('[aria-hidden="true"], [inert]') !== null,
      };
    });
    if (!stop || (stops.length > 0 && stop.name === stops[0].name)) break;
    stops.push(stop);
    if (!stop.indicator) {
      await testInfo.attach(`focus-${stops.length}.png`, {
        body: await page.screenshot(),
        contentType: "image/png",
      });
    }
  }
  const bare = stops.filter((s) => !s.indicator).map((s) => s.name);
  const hidden = stops.filter((s) => s.hidden).map((s) => s.name);
  testInfo.annotations.push(
    { type: "tab-stops", description: String(stops.length) },
    { type: "no-indicator", description: bare.join(" | ") || "none" },
    { type: "hidden-focus", description: hidden.join(" | ") || "none" },
  );
  expect.soft(stops[0]?.name, "first tab stop").toBe("Skip to main content");
  expect.soft(hidden, "focus reached hidden or inert elements").toEqual([]);
  expect
    .soft(bare, "focused elements without an outline or shadow")
    .toEqual([]);
}

for (const vp of VIEWPORTS) {
  test.describe(`viewport ${vp.name}`, () => {
    test.use({ viewport: vp.viewport, deviceScaleFactor: vp.scale });

    for (const route of ROUTES) {
      test(`${route.name} populated @ ${vp.name}`, async ({
        page,
      }, testInfo) => {
        await page.goto(route.path);
        await settled(page);
        await audit(page, testInfo);
        if (vp.name === "1280" || vp.name === "390")
          await keyboardWalk(page, testInfo);
      });

      if (!STATE_VIEWPORTS.has(vp.name) || route.static) continue;

      test(`${route.name} loading @ ${vp.name}`, async ({ page }, testInfo) => {
        let release = () => {};
        const held = new Promise<void>((resolve) => (release = resolve));
        await page.route(API, async (r) => {
          if (!isSession(r.request().url())) await held;
          await r.continue().catch(() => {});
        });
        try {
          await page.goto(route.path);
          await signedIn(page);
          await expect(page.getByRole("main")).toBeVisible();
          await expect
            .soft(
              page.getByRole("status").first(),
              "a pending load is announced with role=status",
            )
            .toBeVisible();
          await audit(page, testInfo);
        } finally {
          release();
        }
      });

      test(`${route.name} error @ ${vp.name}`, async ({ page }, testInfo) => {
        await page.route(API, (r) =>
          isSession(r.request().url())
            ? r.continue()
            : r.fulfill({
                status: 500,
                json: {
                  code: "internal_error",
                  message: "Forced by the audit",
                },
              }),
        );
        await page.goto(route.path);
        await settled(page);
        const alert = await page.getByRole("alert").count();
        testInfo.annotations.push({
          type: "alerts",
          description: String(alert),
        });
        expect
          .soft(alert, "an API failure is announced with role=alert")
          .toBeGreaterThan(0);
        await audit(page, testInfo);
      });

      if (!route.empty) continue;
      const empty = route.empty;
      test(`${route.name} empty @ ${vp.name}`, async ({ page }, testInfo) => {
        await page.goto(empty);
        await settled(page);
        await audit(page, testInfo);
      });
    }
  });
}
