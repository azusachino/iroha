// Report collection only; not a replacement for the normal browser gate.
import base from "../../../../apps/iroha-web/playwright.config";
import { fileURLToPath } from "node:url";
import { resolve } from "node:path";

const local = (path: string) => fileURLToPath(new URL(path, import.meta.url));
const output = process.env.IROHA_UX_AUDIT_OUT;
if (!output || !output.startsWith("/")) {
  throw new Error("Set IROHA_UX_AUDIT_OUT to an absolute scratch directory.");
}
if (process.env.E2E_BASE_URL || process.env.E2E_PUBLIC_BASE_URL) {
  throw new Error(
    "This review owns isolated local servers; external overrides are forbidden.",
  );
}
const servers = Array.isArray(base.webServer) ? base.webServer : [];
export default {
  ...base,
  testDir: local("./"),
  testMatch: "route-review.spec.ts",
  outputDir: resolve(output, "browser"),
  projects: [{ name: "chromium", use: base.projects![0].use }],
  reporter: [
    ["list"],
    ["json", { outputFile: resolve(output, "routes.json") }],
  ],
  use: { ...base.use, screenshot: "on" },
  webServer: servers.map((server, index) => ({
    ...server,
    cwd: local(
      index === 0
        ? "../../../../apps/iroha-web"
        : "../../../../apps/iroha-public-site",
    ),
  })),
};
