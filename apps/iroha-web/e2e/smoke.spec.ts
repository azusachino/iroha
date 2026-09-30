// Smoke check cloned from the playwright-verify skill: the first route renders
// its main landmark without console errors. Signed out, that is the sign-in
// screen; behavior checks live in their own specs.
import { expect, test } from "@playwright/test";
import { fakeSession, signedOut } from "./session";

const ROUTE = "/";

test("first route renders without console errors", async ({ page }) => {
  const errors: string[] = [];
  page.on("console", (message) => {
    if (message.type() === "error") errors.push(message.text());
  });
  page.on("pageerror", (error) => errors.push(error.message));
  await fakeSession(page, signedOut);

  await page.goto(ROUTE);

  await expect(page.getByRole("main")).toBeVisible();
  expect(errors).toEqual([]);
});
