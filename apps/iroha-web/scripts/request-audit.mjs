#!/usr/bin/env node

// Capture browser API traffic for every top-level cockpit route. This is an
// opt-in regression check because it needs a running web/API deployment.
// Usage: bun run request-audit -- --base https://iroha.h.azusachino.icu

import { chromium } from "playwright";

function parseArgs(argv) {
  const args = { base: "http://127.0.0.1:5173", apiBase: "" };
  for (let i = 0; i < argv.length; i += 1) {
    if (argv[i] === "--base") args.base = argv[++i];
    if (argv[i] === "--api-base") args.apiBase = argv[++i];
  }
  return args;
}

const routes = [
  "/",
  "/dashboard",
  "/activities",
  "/daily",
  "/design",
  "/media",
  "/sleep",
  "/admin",
];

async function capture(page, base, path) {
  const calls = [];
  const listener = (request) => {
    const url = new URL(request.url());
    if (url.pathname.startsWith("/api/"))
      calls.push(`${request.method()} ${url.pathname}${url.search}`);
  };
  page.on("request", listener);
  await page.goto(base + path, { waitUntil: "networkidle" });
  await page.waitForTimeout(300);
  page.off("request", listener);
  return calls;
}

async function captureDailyTab(page, name) {
  const calls = [];
  const listener = (request) => {
    const url = new URL(request.url());
    if (url.pathname.startsWith("/api/"))
      calls.push(`${request.method()} ${url.pathname}${url.search}`);
  };
  page.on("request", listener);
  await page
    .getByRole("button", { name: new RegExp(`^${name}$`, "i") })
    .last()
    .click();
  await page.waitForTimeout(500);
  page.off("request", listener);
  return calls;
}

function assert(condition, message) {
  if (!condition) throw new Error(message);
}

async function main() {
  const { base, apiBase } = parseArgs(process.argv.slice(2));
  const browser = await chromium.launch();
  const context = await browser.newContext();
  const page = await context.newPage();
  if (apiBase) {
    await page.route("**/api/**", async (route) => {
      const request = route.request();
      const source = new URL(request.url());
      const target =
        apiBase.replace(/\/$/, "") + source.pathname + source.search;
      const response = await context.request.fetch(target, {
        method: request.method(),
        headers: {
          accept: "application/json",
          ...(request.method() === "POST"
            ? { "content-type": "application/json" }
            : {}),
        },
        data: request.postData() ?? undefined,
      });
      await route.fulfill({
        status: response.status(),
        headers: response.headers(),
        body: await response.body(),
      });
    });
  }
  const report = {};
  try {
    for (const path of routes) report[path] = await capture(page, base, path);

    const admin = report["/admin"];
    assert(
      admin.filter((call) => call.includes("/api/v1/tasks")).length === 1,
      `admin should make one task request, got ${admin.join(" | ")}`,
    );
    assert(
      admin.filter((call) => call.includes("/api/v1/jobs")).length === 1,
      `admin should make one scoped job request, got ${admin.join(" | ")}`,
    );
    assert(
      admin.some((call) =>
        call.includes("media_sync_anilist%2Cmedia_sync_bangumi"),
      ),
      "admin job request is not scoped to top-level media sync actions",
    );

    const daily = report["/daily"];
    assert(
      daily.filter((call) => call.includes("/api/v1/daily?")).length <= 1,
      `daily initial load should not sweep history, got ${daily.join(" | ")}`,
    );
    assert(
      daily.filter((call) => call.includes("granularity=year")).length === 0,
      `daily should lazy-load yearly aggregates, got ${daily.join(" | ")}`,
    );

    await page.goto(base + "/daily", { waitUntil: "networkidle" });
    const yearCalls = await captureDailyTab(page, "Year");
    assert(
      yearCalls.filter((call) => call.includes("granularity=year")).length ===
        1,
      `daily Year tab should make one yearly aggregate request, got ${yearCalls.join(" | ")}`,
    );
    const dayCalls = await captureDailyTab(page, "Day");
    assert(
      dayCalls.some(
        (call) => /\/api\/v1\/daily\?/.test(call) && call.includes("limit=31"),
      ),
      `daily Day tab should request one bounded month, got ${dayCalls.join(" | ")}`,
    );

    report["/daily Year tab"] = yearCalls;
    report["/daily Day tab"] = dayCalls;
    console.log(JSON.stringify({ base, routes: report }, null, 2));
  } finally {
    await browser.close();
  }
}

main().catch((error) => {
  console.error(error.message);
  process.exitCode = 1;
});
