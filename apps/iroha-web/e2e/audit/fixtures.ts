// The audit runs against the seeded stack scripts/e2e_audit.py starts. That
// script signs in a throwaway owner and passes its session cookie here as
// E2E_SESSION_COOKIE=name=value, so every page opens signed in.
import { test as base } from "@playwright/test";

export { expect } from "@playwright/test";

export const test = base.extend({
  context: async ({ context, baseURL }, use) => {
    const cookie = process.env.E2E_SESSION_COOKIE ?? "";
    const split = cookie.indexOf("=");
    if (split < 1 || !baseURL) {
      throw new Error(
        "run the audit through `make e2e-audit`, which sets E2E_SESSION_COOKIE",
      );
    }
    await context.addCookies([
      {
        name: cookie.slice(0, split),
        value: cookie.slice(split + 1),
        url: baseURL,
        httpOnly: true,
        sameSite: "Lax",
      },
    ]);
    await use(context);
  },
});
