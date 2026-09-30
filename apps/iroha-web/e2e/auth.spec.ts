// The owner's way in: which screen each session state shows before the cockpit.
import { expect, test } from "@playwright/test";
import { fakeSession, firstRun, signedOut } from "./session";

test("a signed-out visitor is asked to sign in", async ({ page }) => {
  await fakeSession(page, signedOut);
  await page.goto("/");

  await expect(page.getByRole("heading", { name: "Sign in" })).toBeVisible();
  await expect(page.getByLabel("Username")).toBeVisible();
  await expect(page.getByRole("button", { name: "Sign in" })).toBeVisible();
});

test("a fresh install asks for the owner account", async ({ page }) => {
  await fakeSession(page, firstRun);
  await page.goto("/");

  await expect(page.getByRole("heading", { name: "Welcome" })).toBeVisible();
  await expect(page.getByLabel("Confirm password")).toBeVisible();
  await expect(
    page.getByRole("button", { name: "Create account" }),
  ).toBeVisible();
});

test("a rejected sign-in says so and keeps the form", async ({ page }) => {
  await fakeSession(page, signedOut);
  await page.route("**/api/v1/auth/login", (route) =>
    route.fulfill({
      status: 401,
      json: { code: "invalid_credentials", message: "invalid credentials" },
    }),
  );
  await page.goto("/");

  await page.getByLabel("Username").fill("owner");
  await page.getByLabel("Password", { exact: true }).fill("not-the-password");
  await page.getByRole("button", { name: "Sign in" }).click();

  await expect(page.getByRole("alert")).toHaveText(
    "Wrong username or password.",
  );
  await expect(page.getByRole("heading", { name: "Sign in" })).toBeVisible();
});

test("an unreachable backend offers a retry that recovers", async ({
  page,
}) => {
  let calls = 0;
  await page.route("**/api/v1/auth/session", (route) =>
    ++calls === 1
      ? route.fulfill({ status: 502, body: "Bad Gateway" })
      : route.fulfill({ json: signedOut }),
  );
  await page.goto("/");

  await expect(page.getByRole("alert")).toHaveText("Couldn't reach Iroha.");
  await page.getByRole("button", { name: "Try again" }).click();
  await expect(page.getByRole("heading", { name: "Sign in" })).toBeVisible();
});
