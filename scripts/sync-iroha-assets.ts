import { cp, mkdir } from "node:fs/promises";
import { resolve } from "node:path";

const app = process.argv[2];
const supportedApps = new Set(["iroha-web", "iroha-public-site"]);
const assets = [
  "apple-touch-icon.png",
  "favicon.ico",
  "favicon.svg",
  "icon-192.png",
  "icon-512.png",
];

if (!app || !supportedApps.has(app)) {
  throw new Error(
    `Usage: bun sync-iroha-assets.ts <${[...supportedApps].join("|")}>`,
  );
}

const root = resolve(import.meta.dir, "..");
const source = resolve(root, "packages/iroha-shared/assets");
const target = resolve(root, "apps", app, "static");

await mkdir(target, { recursive: true });
await Promise.all(
  assets.map((asset) =>
    cp(resolve(source, asset), resolve(target, asset), { force: true }),
  ),
);
