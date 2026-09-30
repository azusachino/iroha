// Every route asks the Go backend for the session before it renders. Specs run
// against the Vite dev server alone, so they fake that call with page.route.
import type { Page } from "@playwright/test";
import type { AuthSession } from "../src/lib/api";

export const signedOut: AuthSession = {
  setup_required: false,
  authenticated: false,
};

export const firstRun: AuthSession = {
  setup_required: true,
  authenticated: false,
};

export async function fakeSession(page: Page, session: AuthSession) {
  await page.route("**/api/v1/auth/session", (route) =>
    route.fulfill({ json: session }),
  );
}
