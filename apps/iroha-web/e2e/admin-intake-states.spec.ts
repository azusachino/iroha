import { expect, test } from "@playwright/test";
import { fakeSession } from "./session";

const credential = {
  id: "credential-synthetic",
  name: "harus-phone",
  created_at: "2026-01-01T00:00:00Z",
  last_used_at: null,
  revoked_at: null,
};

async function openAdminIntake(page: Parameters<typeof fakeSession>[0]) {
  await fakeSession(page, {
    setup_required: false,
    authenticated: true,
    username: "test",
    csrf_token: "csrf-synthetic",
  });
  await page.goto("/admin?tab=intake");
  await page.getByRole("heading", { name: "Intake tokens" }).waitFor();
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
