// The owner's way in: which screen each session state shows before the cockpit.
import { expect, test } from "@playwright/test";
import type { Page } from "@playwright/test";
import { fakeSession, firstRun, signedOut } from "./session";

async function fakeAuthenticatedCockpit(
  page: Page,
  options: { passkeys_enabled?: boolean } = {},
) {
  await fakeSession(page, {
    setup_required: false,
    authenticated: true,
    username: "synthetic-owner",
    display_name: "Synthetic Owner",
    csrf_token: "synthetic-csrf",
    passkeys_enabled: options.passkeys_enabled ?? true,
  });
  await page.route("**/api/v1/briefing**", (route) =>
    route.fulfill({ json: { sections: [] } }),
  );
  await page.route("**/api/v1/tasks**", (route) => route.fulfill({ json: [] }));
  await page.route("**/api/v1/daily/dates**", (route) =>
    route.fulfill({ json: [] }),
  );
}

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

test("passkeys are offered on sign-in when enabled on server and supported by browser", async ({
  page,
}) => {
  await fakeSession(page, {
    setup_required: false,
    authenticated: false,
    passkeys_enabled: true,
  });
  await page.goto("/");

  await expect(
    page.getByRole("button", { name: "Sign in with a passkey" }),
  ).toBeVisible();
});

test("passkeys are omitted when disabled on server", async ({ page }) => {
  await fakeSession(page, {
    setup_required: false,
    authenticated: false,
    passkeys_enabled: false,
  });
  await page.goto("/");

  await expect(
    page.getByRole("button", { name: "Sign in with a passkey" }),
  ).toHaveCount(0);
});

test("passkeys are omitted during first-run owner setup even if enabled", async ({
  page,
}) => {
  await fakeSession(page, {
    setup_required: true,
    authenticated: false,
    passkeys_enabled: true,
  });
  await page.goto("/");

  await expect(
    page.getByRole("button", { name: "Sign in with a passkey" }),
  ).toHaveCount(0);
  await expect(page.getByRole("heading", { name: "Welcome" })).toBeVisible();
});

test("passkey login cancellation clears error and retains form", async ({
  page,
}) => {
  await page.addInitScript(() => {
    if (typeof PublicKeyCredential !== "undefined") {
      PublicKeyCredential.parseRequestOptionsFromJSON = (opts) => opts as any;
    }
    navigator.credentials.get = () =>
      Promise.reject(
        new DOMException("User cancelled passkey sheet", "NotAllowedError"),
      );
  });
  await fakeSession(page, {
    setup_required: false,
    authenticated: false,
    passkeys_enabled: true,
  });
  await page.route("**/api/v1/auth/passkey/begin", (route) =>
    route.fulfill({
      json: { publicKey: { challenge: "synthetic-challenge" } },
    }),
  );
  await page.goto("/");

  const passkeyButton = page.getByRole("button", {
    name: "Sign in with a passkey",
  });
  await expect(passkeyButton).toBeVisible();
  await passkeyButton.click();

  await expect(page.getByRole("alert")).toHaveCount(0);
  await expect(
    page.getByRole("button", { name: "Sign in", exact: true }),
  ).toBeEnabled();
});

test("passkey login failure displays error alert and retains form", async ({
  page,
}) => {
  await fakeSession(page, {
    setup_required: false,
    authenticated: false,
    passkeys_enabled: true,
  });
  await page.route("**/api/v1/auth/passkey/begin", (route) =>
    route.fulfill({
      status: 503,
      json: {
        code: "temporarily_unavailable",
        message: "Passkey authentication temporarily unavailable.",
      },
    }),
  );
  await page.goto("/");

  await page.getByRole("button", { name: "Sign in with a passkey" }).click();
  await expect(page.getByRole("alert")).toHaveText(
    "Passkey authentication temporarily unavailable.",
  );
  await expect(page.getByRole("heading", { name: "Sign in" })).toBeVisible();
});

test("successful passkey login transitions to authenticated cockpit", async ({
  page,
}) => {
  await page.addInitScript(() => {
    if (typeof PublicKeyCredential !== "undefined") {
      PublicKeyCredential.parseRequestOptionsFromJSON = (opts) => opts as any;
    }
    const cred = {
      id: "cred-synthetic",
      type: "public-key",
      rawId: new ArrayBuffer(8),
      response: {},
      toJSON: () => ({
        id: "cred-synthetic",
        rawId: "abc",
        response: {},
        type: "public-key",
      }),
    };
    Object.setPrototypeOf(cred, PublicKeyCredential.prototype);
    navigator.credentials.get = () =>
      Promise.resolve(cred as unknown as Credential);
  });

  await fakeSession(page, {
    setup_required: false,
    authenticated: false,
    passkeys_enabled: true,
  });
  await page.route("**/api/v1/auth/passkey/begin", (route) =>
    route.fulfill({
      json: { publicKey: { challenge: "synthetic-challenge" } },
    }),
  );
  await page.route("**/api/v1/auth/passkey/finish", (route) =>
    route.fulfill({
      json: {
        setup_required: false,
        authenticated: true,
        username: "synthetic-owner",
        display_name: "Synthetic Owner",
      },
    }),
  );
  await page.route("**/api/v1/briefing**", (route) =>
    route.fulfill({ json: { sections: [] } }),
  );
  await page.route("**/api/v1/tasks**", (route) => route.fulfill({ json: [] }));
  await page.route("**/api/v1/daily/dates**", (route) =>
    route.fulfill({ json: [] }),
  );

  await page.goto("/");
  await page.getByRole("button", { name: "Sign in with a passkey" }).click();

  await expect(
    page.getByRole("navigation", { name: "Primary navigation" }),
  ).toBeVisible();
});

test("session state observer: unauthenticated event expires active session and displays login", async ({
  page,
}) => {
  await fakeAuthenticatedCockpit(page);
  await page.goto("/");

  await expect(
    page.getByRole("navigation", { name: "Primary navigation" }),
  ).toBeVisible();

  // Emulate 401 unauthenticated event dispatched by API client
  await page.evaluate(() => {
    window.dispatchEvent(new CustomEvent("iroha:unauthenticated"));
  });

  await expect(page.getByRole("heading", { name: "Sign in" })).toBeVisible();
});

test("session state observer: signing out from user menu returns to sign in", async ({
  page,
}) => {
  await fakeAuthenticatedCockpit(page);
  await page.route("**/api/v1/auth/logout", (route) =>
    route.fulfill({ json: { ok: true } }),
  );
  await page.goto("/");

  const userMenu = page.locator("summary[aria-label*='Account menu for']");
  await userMenu.click();
  await page.getByRole("menuitem", { name: "Log out" }).click();

  await expect(page.getByRole("heading", { name: "Sign in" })).toBeVisible();
});

test("passkey settings in account dialog reflects server passkeys disabled state", async ({
  page,
}) => {
  await fakeAuthenticatedCockpit(page, { passkeys_enabled: false });
  await page.route("**/api/v1/account/passkeys", (route) =>
    route.fulfill({ json: { items: [] } }),
  );
  await page.goto("/");

  const userMenu = page.locator("summary[aria-label*='Account menu for']");
  await userMenu.click();
  await page.getByRole("menuitem", { name: "Account settings" }).click();

  const dialog = page.getByRole("dialog", { name: "Account settings" });
  await expect(dialog).toBeVisible();
  await dialog.getByRole("tab", { name: "Security" }).click();

  await expect(
    dialog.getByText("Passkeys aren't configured on this server."),
  ).toBeVisible();
});

test("passkey settings in account dialog lists passkeys and supports renaming and removal", async ({
  page,
}) => {
  await fakeAuthenticatedCockpit(page, { passkeys_enabled: true });
  let passkeys = [
    {
      id: "pk-synthetic-1",
      name: "Office Mac",
      created_at: "2026-01-01T00:00:00Z",
      last_used_at: null,
    },
  ];
  await page.route("**/api/v1/account/passkeys", (route) => {
    if (route.request().method() === "GET") {
      return route.fulfill({ json: { items: passkeys } });
    }
    return route.continue();
  });
  await page.route("**/api/v1/account/passkeys/pk-synthetic-1", (route) => {
    if (route.request().method() === "PATCH") {
      passkeys = passkeys.map((p) => ({ ...p, name: "Home Mac" }));
      return route.fulfill({ json: { items: passkeys } });
    }
    if (route.request().method() === "DELETE") {
      passkeys = [];
      return route.fulfill({ status: 204 });
    }
    return route.continue();
  });
  await page.route("**/api/v1/account/reauth", (route) =>
    route.fulfill({ json: { ok: true } }),
  );

  await page.goto("/");

  const userMenu = page.locator("summary[aria-label*='Account menu for']");
  await userMenu.click();
  await page.getByRole("menuitem", { name: "Account settings" }).click();

  const dialog = page.getByRole("dialog", { name: "Account settings" });
  await expect(dialog).toBeVisible();
  await dialog.getByRole("tab", { name: "Security" }).click();

  // Listed passkey
  await expect(dialog.getByText("Office Mac")).toBeVisible();

  // Rename passkey
  await dialog.getByRole("button", { name: "Rename" }).click();
  const nameInput = dialog.getByLabel("Passkey name");
  await nameInput.fill("Home Mac");
  await dialog.getByRole("button", { name: "Save" }).click();
  await expect(dialog.getByText("Home Mac")).toBeVisible();

  // Remove passkey (requires password confirmation)
  await dialog.getByRole("button", { name: "Remove Home Mac" }).click();
  const passwordInput = dialog.locator("input[type=password]");
  await passwordInput.fill("correct-password");
  await dialog.locator("form.confirm button[type=submit]").click();
  await expect(dialog.getByText("No passkeys yet.")).toBeVisible();
});
