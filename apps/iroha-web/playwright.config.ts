// Headless Chromium checks for both consumers of the shared UI.
// E2E_BASE_URL / E2E_PUBLIC_BASE_URL opt into independently managed servers.
import { defineConfig, devices } from "@playwright/test";

const PORT = 5183; // off 5173, which a person's `vite dev` uses
const COMMAND = `bun run dev --host 127.0.0.1 --port ${PORT} --strictPort`; // /api calls are faked per spec with page.route; no Go backend
const external = process.env.E2E_BASE_URL;
const PUBLIC_PORT = 5184;
const publicExternal = process.env.E2E_PUBLIC_BASE_URL;
export const PUBLIC_BASE_URL =
  publicExternal ?? `http://127.0.0.1:${PUBLIC_PORT}`;
const CI = !!process.env.CI;
const chrome = devices["Desktop Chrome"];

export default defineConfig({
  testDir: "./e2e",
  timeout: 30_000,
  expect: { timeout: 5_000 },
  // Playwright's recommended CI settings: no stray test.only, retries that
  // expose flaky tests instead of hiding them, one worker.
  forbidOnly: CI,
  retries: CI ? 2 : 0,
  failOnFlakyTests: CI,
  workers: CI ? 1 : undefined,
  // A text reporter, because agents read the terminal; the html report needs a browser.
  reporter: "list",
  use: {
    baseURL: external ?? `http://127.0.0.1:${PORT}`,
    // Local runs do not retry, so keep a trace of every failure rather than of the first retry.
    trace: "retain-on-failure",
    screenshot: "only-on-failure",
  },
  // `make e2e` runs the checks; `make e2e-probe` runs only the probe.
  projects: [
    {
      name: "chromium",
      use: chrome,
      testIgnore: /(?:probe|pilot-audit)\.spec\.ts$/,
    },
    { name: "pilot-audit", use: chrome, testMatch: /pilot-audit\.spec\.ts$/ },
    { name: "probe", use: chrome, testMatch: /probe\.spec\.ts$/ },
  ],
  // No reuse: a port already in use fails the run instead of testing a server someone else is using.
  webServer: [
    ...(external
      ? []
      : [
          {
            command: COMMAND,
            url: `http://127.0.0.1:${PORT}`,
            reuseExistingServer: false,
            timeout: 60_000,
          },
        ]),
    ...(publicExternal
      ? []
      : [
          {
            command: `bun run dev --host 127.0.0.1 --port ${PUBLIC_PORT} --strictPort`,
            cwd: "../iroha-public-site",
            url: PUBLIC_BASE_URL,
            reuseExistingServer: false,
            timeout: 60_000,
          },
        ]),
  ],
});
