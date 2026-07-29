#!/usr/bin/env node
// Screenshots iroha-web's themed routes with a real Chromium instance and
// reports any console/page errors. This exists because svelte-check/vitest/
// build all pass without ever rendering a component in a JS engine -- for a
// visual/theme change, that's necessary but not sufficient. Reusable across
// every theme in the theme-maturity epic, not just field-journal.
//
// One-time setup: `bunx playwright install chromium` (or `--with-deps` on a
// bare Linux box) after `bun install` -- the browser binary isn't a bun
// package and isn't committed.
//
// Usage:
//   node scripts/visual-check.mjs [--theme field-journal] [--base http://127.0.0.1:5173]
//     [--routes today,dashboard,daily,activities,sleep,media] [--out .visual-check]
//
// Expects the dev server (and iroha-server + the local DB) already running.

import { chromium } from "playwright";
import { mkdirSync } from "node:fs";

function parseArgs(argv) {
  const args = {
    theme: "field-journal",
    base: "http://127.0.0.1:5173",
    out: ".visual-check",
  };
  args.routes = ["today", "dashboard", "daily", "activities", "sleep", "media"];
  for (let i = 0; i < argv.length; i += 1) {
    const key = argv[i];
    if (!key.startsWith("--")) continue;
    const value = argv[i + 1];
    i += 1;
    if (key === "--routes") args.routes = value.split(",").map((r) => r.trim());
    else args[key.slice(2)] = value;
  }
  return args;
}

const ROUTE_PATHS = {
  today: "/",
  dashboard: "/dashboard",
  daily: "/daily",
  activities: "/activities",
  sleep: "/sleep",
  media: "/media",
};

async function main() {
  const { theme, base, out, routes } = parseArgs(process.argv.slice(2));
  mkdirSync(out, { recursive: true });

  const browser = await chromium.launch();
  const context = await browser.newContext({
    viewport: { width: 1280, height: 900 },
  });
  const page = await context.newPage();

  await page.addInitScript((designLanguage) => {
    localStorage.setItem("iroha-design-language", designLanguage);
  }, theme);

  let failures = 0;
  for (const name of routes) {
    const path = ROUTE_PATHS[name];
    if (!path) {
      console.log(`skip: unknown route "${name}"`);
      continue;
    }
    const consoleErrors = [];
    const onConsole = (msg) => {
      if (msg.type() === "error") consoleErrors.push(msg.text());
    };
    const onPageError = (err) =>
      consoleErrors.push(`pageerror: ${err.message}`);
    page.on("console", onConsole);
    page.on("pageerror", onPageError);

    await page.goto(base + path, { waitUntil: "networkidle" });
    await page.waitForTimeout(400);

    const viewports = [
      { width: 1280, label: "desktop" },
      { width: 768, label: "768" },
      { width: 640, label: "mobile" },
      { width: 414, label: "414" },
      { width: 375, label: "375" },
      { width: 320, label: "320" },
    ];

    for (const vp of viewports) {
      await page.setViewportSize({ width: vp.width, height: 900 });
      await page.waitForTimeout(200);
      await page.screenshot({
        path: `${out}/${theme}-${name}-${vp.label}.png`,
        fullPage: true,
      });
    }
    await page.setViewportSize({ width: 1280, height: 900 });

    page.off("console", onConsole);
    page.off("pageerror", onPageError);
    console.log(`${name}: ${consoleErrors.length} console errors`);
    for (const e of consoleErrors) {
      console.log(`  ERROR: ${e}`);
      failures += 1;
    }
  }

  // Detail routes need a real id from the currently imported dataset; skip
  // gracefully (not a failure) when the local dev DB has no rows yet.
  if (routes.includes("activities")) {
    await page.goto(base + "/activities", { waitUntil: "networkidle" });
    const href = await page
      .locator('a[href^="/activities/"]')
      .first()
      .getAttribute("href")
      .catch(() => null);
    if (href) {
      await page.goto(base + href, { waitUntil: "networkidle" });
      await page.waitForTimeout(400);
      await page.screenshot({
        path: `${out}/${theme}-activity-detail-desktop.png`,
        fullPage: true,
      });
      console.log("activity-detail: screenshot taken");
    } else {
      console.log("activity-detail: no activities in dev DB, skipped");
    }
  }

  if (routes.includes("media")) {
    await page.goto(base + "/media", { waitUntil: "networkidle" });
    const href = await page
      .locator('a[href^="/media/"]')
      .first()
      .getAttribute("href")
      .catch(() => null);
    if (href) {
      await page.goto(base + href, { waitUntil: "networkidle" });
      await page.waitForTimeout(400);
      await page.screenshot({
        path: `${out}/${theme}-media-detail-desktop.png`,
        fullPage: true,
      });
      console.log("media-detail: screenshot taken");
    } else {
      console.log("media-detail: no media items in dev DB, skipped");
    }
  }

  await browser.close();
  if (failures > 0) {
    console.error(`\n${failures} console error(s) across the checked routes.`);
    process.exit(1);
  }
  console.log(`\nScreenshots written to ${out}/`);
}

main();
