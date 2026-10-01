import { expect, test } from "@playwright/test";
import { fakeSession } from "./session";

const health = {
  id: "src_synthetic",
  provider: "health_auto_export",
  instance_key: "iphone-hae:test",
  display_name: "Test Health delivery",
  availability: "supported",
  collection: "unknown",
  operation: "idle",
  freshness: "within_cadence",
  expected_interval_s: 86400,
  next_expected_at: "2026-01-02T00:00:00Z",
  last_receipt: {
    id: "receipt_synthetic",
    raw_file_id: "raw_synthetic",
    source_kind: "health_auto_export",
    ingestion_mode: "incremental",
    received_at: "2026-01-01T00:00:00Z",
  },
  coverage: [],
  next_actions: [],
};

test("Sources shows delivery freshness separately from coverage and refreshes overdue state", async ({
  page,
}) => {
  await page.clock.install({ time: new Date("2026-01-01T23:59:30Z") });
  await fakeSession(page, {
    setup_required: false,
    authenticated: true,
    username: "test",
  });
  let overdue = false;
  await page.route("**/api/v1/connections", (route) =>
    route.fulfill({
      json: {
        connections: [
          { ...health, freshness: overdue ? "overdue" : "within_cadence" },
          {
            ...health,
            id: "src_manual",
            display_name: "Manual GPX",
            provider: "gpx",
            freshness: "not_scheduled",
            last_receipt: null,
            next_expected_at: undefined,
          },
        ],
      },
    }),
  );
  await page.goto("/admin?tab=sources");
  await expect(page.getByText("Within cadence", { exact: true })).toBeVisible();
  await expect(page.getByText("Not scheduled", { exact: true })).toBeVisible();
  await expect(page.getByText(/never received/)).toBeVisible();
  await expect(page.getByText(/last received/)).toBeVisible();
  await expect(page.getByText(/no coverage recorded/)).toHaveCount(2);
  overdue = true;
  await page.clock.fastForward(60_000);
  await expect(page.getByText("Overdue", { exact: true })).toBeVisible();
  await expect(page.getByRole("status")).toContainText("1 needs attention");
});

test("Sources can retry a failed refresh on a compact viewport", async ({
  page,
}) => {
  await page.setViewportSize({ width: 320, height: 844 });
  await fakeSession(page, {
    setup_required: false,
    authenticated: true,
    username: "test",
  });
  let failed = true;
  await page.route("**/api/v1/connections", (route) =>
    failed
      ? route.fulfill({
          status: 503,
          json: { error: "source service unavailable" },
        })
      : route.fulfill({
          json: { connections: [{ ...health, freshness: "overdue" }] },
        }),
  );
  await page.goto("/admin?tab=sources");
  await expect(page.getByRole("alert")).toBeVisible();
  failed = false;
  await page.getByRole("button", { name: "Refresh", exact: true }).click();
  await expect(page.getByText("Overdue", { exact: true })).toBeVisible();
  await expect(page.getByRole("alert")).toHaveCount(0);
  await expect(page.getByRole("status")).toContainText("1 needs attention");
});
