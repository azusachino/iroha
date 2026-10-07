import { expect, test } from "@playwright/test";
import { fakeSession } from "./session";

const credential = {
  id: "credential-synthetic",
  name: "harus-phone",
  created_at: "2026-01-01T00:00:00Z",
  last_used_at: null,
  revoked_at: null,
};

async function openAdmin(
  page: Parameters<typeof fakeSession>[0],
  tab: string,
  headingName: string,
) {
  await fakeSession(page, {
    setup_required: false,
    authenticated: true,
    username: "test",
    csrf_token: "csrf-synthetic",
  });
  await page.goto(`/admin?tab=${tab}`);
  await page.getByRole("heading", { name: headingName }).waitFor();
}

async function openAdminIntake(page: Parameters<typeof fakeSession>[0]) {
  await openAdmin(page, "intake", "Intake tokens");
}

const intakeReads = "**/api/v1/admin/intake-credentials";
const panelFor = (page: Parameters<typeof fakeSession>[0]) =>
  page.getByRole("region", { name: "Intake tokens" });

test("Intake read stays truthful through pending, repeated failure, and recovery", async ({
  page,
}) => {
  let resolveInitial!: () => void;
  const initialHold = new Promise<void>((resolve) => {
    resolveInitial = resolve;
  });
  let reads = 0;
  await page.route(intakeReads, async (route) => {
    reads += 1;
    if (reads === 1) {
      await initialHold;
      await route.fulfill({ status: 503, json: { error: "unavailable" } });
    } else if (reads === 2) {
      await route.fulfill({
        status: 503,
        json: { error: "still unavailable" },
      });
    } else {
      await route.fulfill({ json: { items: [credential] } });
    }
  });
  await openAdminIntake(page);
  const panel = panelFor(page);
  await expect(panel.getByRole("status")).toContainText(
    "Loading intake tokens",
  );
  await expect(panel.getByText("0 active", { exact: true })).toHaveCount(0);
  await expect(panel.getByText(/No tokens yet/)).toHaveCount(0);

  resolveInitial();
  await expect(panel.getByRole("alert")).toContainText(
    "Could not load intake tokens",
  );
  await expect(panel.getByText("0 active", { exact: true })).toHaveCount(0);
  await expect(panel.getByText(/No tokens yet/)).toHaveCount(0);

  const retry = panel.getByRole("button", { name: "Retry intake tokens" });
  const failedRetry = page.waitForResponse(
    (response) =>
      response.url().includes("/api/v1/admin/intake-credentials") &&
      response.status() === 503,
  );
  await retry.focus();
  await expect(retry).toBeFocused();
  await retry.click();
  await failedRetry;
  await expect.poll(() => reads).toBe(2);
  await expect(panel.getByRole("alert")).toContainText(
    "Could not load intake tokens",
  );
  await expect(panel.getByText("0 active", { exact: true })).toHaveCount(0);
  await expect(panel.getByText(/No tokens yet/)).toHaveCount(0);

  const recovered = page.waitForResponse(
    (response) =>
      response.url().includes("/api/v1/admin/intake-credentials") &&
      response.status() === 200,
  );
  await retry.focus();
  await page.keyboard.press("Enter");
  await recovered;
  await expect.poll(() => reads).toBe(3);
  await expect(panel.getByText("1 active", { exact: true })).toBeVisible();
  await expect(panel.locator("li strong")).toHaveText("harus-phone");
  await expect(panel.getByRole("alert")).toHaveCount(0);
});

test("Intake read recovery does not clear a separate issue error", async ({
  page,
}) => {
  let reads = 0;
  await page.route(intakeReads, (route) => {
    if (route.request().method() === "POST") {
      return route.fulfill({ status: 503, json: { error: "issue rejected" } });
    }
    reads += 1;
    return reads === 1
      ? route.fulfill({ status: 503, json: { error: "unavailable" } })
      : route.fulfill({ json: { items: [] } });
  });
  await openAdminIntake(page);
  const panel = panelFor(page);
  await expect(
    panel.getByRole("button", { name: "Retry intake tokens" }),
  ).toBeVisible();
  await page.getByRole("button", { name: "Issue token" }).click();
  const mutationAlert = panel.locator("p.error[role=alert]");
  await expect(mutationAlert).toBeVisible();
  await expect(panel.getByRole("alert")).toHaveCount(2);
  await page.getByRole("button", { name: "Retry intake tokens" }).click();
  await expect(panel.getByText("0 active", { exact: true })).toBeVisible();
  await expect(mutationAlert).toBeVisible();
  await expect(panel.getByRole("alert")).toHaveCount(1);
});

test("Intake retains observed credentials through failed refresh", async ({
  page,
}) => {
  let reads = 0;
  await page.route(intakeReads, (route) => {
    if (route.request().method() === "POST") {
      return route.fulfill({
        json: { credential, token: "synthetic-one-time-token" },
      });
    }
    reads += 1;
    return reads === 1
      ? route.fulfill({ json: { items: [credential] } })
      : route.fulfill({ status: 503, json: { error: "refresh unavailable" } });
  });
  await openAdminIntake(page);
  const panel = panelFor(page);
  await expect(panel.locator("li strong")).toHaveText("harus-phone");
  await page.getByRole("button", { name: "Issue token" }).click();
  await expect(panel.getByRole("alert")).toContainText(
    "Could not load intake tokens",
  );
  await expect(panel.getByText("1 active", { exact: true })).toBeVisible();
  await expect(panel.locator("li strong")).toHaveText("harus-phone");
  await expect(panel.getByText(/No tokens yet/)).toHaveCount(0);
});

test("Intake tokens show empty only after a confirmed empty read", async ({
  page,
}) => {
  await page.route(intakeReads, (route) =>
    route.fulfill({ json: { items: [] } }),
  );
  await openAdminIntake(page);
  const panel = panelFor(page);
  await expect(panel.getByText("0 active", { exact: true })).toBeVisible();
  await expect(panel.getByText(/No tokens yet/)).toBeVisible();
});

test("Intake issued token displays role=status banner with copy and dismissal", async ({
  page,
  context,
}) => {
  await context.grantPermissions(["clipboard-read", "clipboard-write"]);
  await page.addInitScript(() => {
    Object.defineProperty(navigator, "clipboard", {
      value: {
        writeText: async () => {},
      },
      configurable: true,
    });
  });

  let issuedCount = 0;
  await page.route(intakeReads, (route) => {
    if (route.request().method() === "POST") {
      issuedCount += 1;
      return route.fulfill({
        status: 201,
        json: {
          credential: {
            id: `cred-${issuedCount}`,
            name: "primary",
            created_at: "2026-10-01T12:00:00Z",
            last_used_at: null,
            revoked_at: null,
          },
          token: "hae-synthetic-secret-token",
        },
      });
    }
    return route.fulfill({
      json: {
        items: issuedCount
          ? [
              {
                id: "cred-1",
                name: "primary",
                created_at: "2026-10-01T12:00:00Z",
                last_used_at: null,
                revoked_at: null,
              },
            ]
          : [],
      },
    });
  });

  await openAdminIntake(page);
  const panel = panelFor(page);
  await page.getByRole("button", { name: "Issue token" }).click();

  const statusBanner = panel.locator(".issued[role=status]");
  await expect(statusBanner).toBeVisible();
  await expect(statusBanner).toContainText("Token for “primary”");
  await expect(statusBanner).toContainText("hae-synthetic-secret-token");

  // Copy token
  const copyButton = statusBanner.getByRole("button", { name: /Copy token/ });
  await copyButton.click();
  await expect(
    statusBanner.getByRole("button", { name: "Copied" }),
  ).toBeVisible();

  // Dismiss issued banner
  await statusBanner.getByRole("button", { name: "Done" }).click();
  await expect(statusBanner).toHaveCount(0);
  await expect(panel.getByText("1 active", { exact: true })).toBeVisible();
});

test("Admin System browser-status observers reflect Healthy API, passkey configuration, and recoverable alert", async ({
  page,
}) => {
  let fail = false;
  await page.route("**/api/v1/admin/system", (route) =>
    fail
      ? route.fulfill({
          status: 503,
          json: { code: "service_unavailable", message: "Database offline" },
        })
      : route.fulfill({
          json: {
            parser_version: "v1.0.0",
            migration_version: 42,
            database_bytes: 52428800,
            raw_file_count: 10,
            raw_file_bytes: 20971520,
            raw_file_purged_count: 1,
            cache_backend: "memory",
            timezone: "Asia/Tokyo",
            passkeys_enabled: true,
            server_time: "2026-10-01T12:00:00Z",
          },
        }),
  );
  await page.route("**/api/v1/metrics", (route) =>
    route.fulfill({
      json: {
        schema: "metric-catalog.v1",
        metrics: [
          {
            id: "daily.steps",
            domain: "daily",
            label: "Steps",
            kind: "canonical",
            unit: "count",
          },
        ],
      },
    }),
  );

  await openAdmin(page, "system", "System");

  // 1. API Healthy badge indicator
  const healthyBadge = page.locator(".status.good");
  await expect(healthyBadge).toHaveText("Healthy");

  // 2. Passkeys configuration status
  await expect(page.getByText("Enabled", { exact: true })).toBeVisible();

  // 3. Metric catalog status
  await expect(page.getByText("1 total", { exact: true })).toBeVisible();

  // 4. Recoverable failure and alert
  fail = true;
  await page.getByRole("button", { name: "Refresh" }).click();
  const alert = page.locator(".admin-error[role=alert]");
  await expect(alert).toContainText("API unavailable: Database offline");

  fail = false;
  await page.getByRole("button", { name: "Refresh" }).click();
  await expect(alert).toHaveCount(0);
  await expect(healthyBadge).toHaveText("Healthy");
});

test("Admin Sources browser-status observers reflect connection state tones, delivery status, and live attention", async ({
  page,
}) => {
  const connections = [
    {
      id: "src_health",
      provider: "health_auto_export",
      display_name: "Health Auto Export",
      availability: "supported",
      collection: "all",
      operation: "idle",
      freshness: "within_cadence",
      coverage: ["steps"],
      last_receipt: { received_at: "2026-10-01T12:00:00Z" },
      next_actions: [],
    },
    {
      id: "src_gpx",
      provider: "gpx",
      display_name: "Manual GPX",
      availability: "supported",
      collection: "activities",
      operation: "failed",
      freshness: "not_scheduled",
      coverage: [],
      last_receipt: null,
      next_actions: [
        {
          kind: "retry_import",
          path: "/api/v1/connections/src_gpx/actions/retry",
        },
      ],
    },
    {
      id: "src_overdue",
      provider: "whoop",
      display_name: "Whoop Sync",
      availability: "supported",
      collection: "recovery",
      operation: "idle",
      freshness: "overdue",
      coverage: [],
      last_receipt: null,
      next_actions: [],
    },
    {
      id: "src_fit",
      provider: "fit",
      display_name: "Live FIT",
      availability: "supported",
      collection: "activities",
      operation: "importing",
      freshness: "not_scheduled",
      coverage: [],
      last_receipt: null,
      next_actions: [],
    },
  ];

  await page.route("**/api/v1/connections", (route) =>
    route.fulfill({ json: { connections } }),
  );
  await page.route("**/api/v1/connections/src_gpx/actions/retry", (route) =>
    route.fulfill({ json: { ok: true } }),
  );

  await openAdmin(page, "sources", "Sources");

  // Summary role=status
  const summaryStatus = page.locator("header small[role=status]");
  await expect(summaryStatus).toContainText("4 sources · 2 need attention");

  // Connection status badges: good, bad, and neutral
  await expect(page.locator(".status.good")).toHaveText("Within cadence");
  await expect(
    page.locator(".status.bad").filter({ hasText: "Import failed" }),
  ).toBeVisible();
  await expect(
    page.locator(".status.bad").filter({ hasText: "Overdue" }),
  ).toBeVisible();
  await expect(page.locator(".status:not(.good):not(.bad)")).toHaveText(
    "Importing",
  );

  // Delivery status descriptions
  await expect(page.getByText(/delivery overdue/)).toBeVisible();
  await expect(page.getByText(/last received/)).toBeVisible();
  await expect(page.getByText(/never received/)).toHaveCount(3);
  await expect(page.getByText(/never received/).first()).toBeVisible();

  // Action button
  const retryBtn = page.getByRole("button", { name: "Retry import" });
  await expect(retryBtn).toBeVisible();
  await retryBtn.click();
});

test("Admin Jobs browser-status observers reflect completed, failed, and queued tones and mutation notices", async ({
  page,
}) => {
  let schedules = [
    {
      kind: "health_export_fetch",
      enabled: true,
      schedule_kind: "interval",
      schedule_expr: "1h",
      next_run_at: "2026-10-01T13:00:00Z",
      last_run_at: "2026-10-01T12:00:00Z",
    },
  ];
  let jobs = [
    {
      id: "j-completed",
      kind: "health_export_fetch",
      status: "completed" as const,
      attempts: 1,
      max_attempts: 3,
      created_at: "2026-10-01T12:00:00Z",
    },
    {
      id: "j-failed",
      kind: "gpx_import",
      status: "failed" as const,
      attempts: 3,
      max_attempts: 3,
      error_message: "Corrupt file",
      created_at: "2026-10-01T12:00:00Z",
    },
    {
      id: "j-queued",
      kind: "fit_import",
      status: "queued" as const,
      attempts: 0,
      max_attempts: 3,
      created_at: "2026-10-01T12:00:00Z",
    },
  ];

  await page.route("**/api/v1/admin/schedules", (route) =>
    route.fulfill({ json: { items: schedules } }),
  );
  await page.route("**/api/v1/admin/schedules/health_export_fetch", (route) => {
    schedules = schedules.map((s) => ({ ...s, enabled: !s.enabled }));
    return route.fulfill({ json: schedules[0] });
  });
  await page.route("**/api/v1/jobs**", (route) => {
    const url = new URL(route.request().url());
    if (url.pathname === "/api/v1/jobs/j-failed/retry") {
      return route.fulfill({ json: { ok: true } });
    }
    if (url.pathname === "/api/v1/jobs/j-queued") {
      jobs = jobs.filter((j) => j.id !== "j-queued");
      return route.fulfill({ status: 204 });
    }
    return route.fulfill({ json: jobs });
  });

  await openAdmin(page, "jobs", "Schedules");

  // Job status badges
  await expect(page.locator(".status.good")).toHaveText("completed");
  await expect(page.locator(".status.bad")).toHaveText("failed");
  await expect(page.locator(".status:not(.good):not(.bad)")).toHaveText(
    "queued",
  );

  // Retry action notice with role=status
  await page.getByRole("button", { name: "Retry" }).click();
  await expect(page.locator(".admin-muted[role=status]")).toHaveText(
    "Queued a new attempt.",
  );

  // Cancel action notice with role=status
  await page.getByRole("button", { name: "Cancel" }).click();
  await expect(page.locator(".admin-muted[role=status]")).toHaveText(
    "Job canceled.",
  );

  // Schedule toggle notice with role=status
  const toggle = page.getByRole("checkbox", { name: "Enabled" });
  await toggle.uncheck();
  await expect(page.locator(".admin-muted[role=status]")).toContainText(
    "health export fetch paused.",
  );
});

test("Admin Imports browser-status observers reflect import tones and reprocess and upload notices", async ({
  page,
}) => {
  let imports = [
    {
      id: "imp-1",
      raw_file_id: "rf-1",
      status: "completed" as const,
      parser_kind: "gpx",
      parser_version: "v1",
      created_at: "2026-10-01T12:00:00Z",
    },
    {
      id: "imp-2",
      raw_file_id: "rf-2",
      status: "failed" as const,
      parser_kind: "fit",
      parser_version: "v1",
      error_message: "Format error",
      created_at: "2026-10-01T12:00:00Z",
    },
  ];
  const files = [
    {
      id: "rf-1",
      original_filename: "morning_run.gpx",
      size_bytes: 1024,
      uploaded_via: "web",
      source_kind: "gpx",
      created_at: "2026-10-01T12:00:00Z",
    },
    {
      id: "rf-2",
      original_filename: "evening_ride.fit",
      size_bytes: 2048,
      uploaded_via: "web",
      source_kind: "fit",
      created_at: "2026-10-01T12:00:00Z",
    },
  ];

  await page.route("**/api/v1/imports", (route) => {
    if (route.request().method() === "POST") {
      return route.fulfill({
        json: {
          id: "imp-3",
          raw_file_id: "rf-3",
          status: "completed",
          parser_kind: "gpx",
          parser_version: "v1",
          created_at: "2026-10-01T12:00:00Z",
        },
      });
    }
    return route.fulfill({ json: imports });
  });
  await page.route("**/api/v1/raw-files", (route) => {
    if (route.request().method() === "POST") {
      return route.fulfill({
        status: 201,
        json: {
          id: "rf-3",
          original_filename: "synthetic.gpx",
          size_bytes: 512,
          uploaded_via: "web",
          source_kind: "gpx",
          created_at: "2026-10-01T12:00:00Z",
          duplicate: false,
        },
      });
    }
    return route.fulfill({ json: files });
  });

  await openAdmin(page, "imports", "Upload");

  // Status badges: completed (good) and failed (bad)
  await expect(page.locator(".status.good")).toHaveText("completed");
  await expect(page.locator(".status.bad")).toHaveText("failed");

  // Reprocess action notice with role=status
  const reprocessButtons = page.getByRole("button", { name: "Reprocess" });
  await reprocessButtons.first().click();
  await expect(page.locator(".admin-muted[role=status]")).toHaveText(
    "Queued a new import of the same file.",
  );

  // Upload action notice with role=status
  const fileInput = page.getByLabel("File", { exact: true });
  await fileInput.setInputFiles({
    name: "synthetic.gpx",
    mimeType: "application/gpx+xml",
    buffer: Buffer.from("<gpx></gpx>"),
  });
  await page.getByRole("button", { name: "Upload and import" }).click();
  await expect(page.locator(".admin-muted[role=status]")).toHaveText(
    "Uploaded “synthetic.gpx” and queued its import.",
  );
});
