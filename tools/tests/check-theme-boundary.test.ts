import { afterEach, describe, expect, test } from "bun:test";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";

import { findViolations } from "../check-theme-boundary.ts";

const tempDirs: string[] = [];

function makeRoot(): string {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), "iroha-theme-"));
  tempDirs.push(root);
  fs.mkdirSync(path.join(root, "apps/iroha-web/src/lib"), { recursive: true });
  fs.mkdirSync(path.join(root, "packages/iroha-shared/src"), { recursive: true });
  return root;
}

afterEach(() => {
  while (tempDirs.length > 0) {
    const dir = tempDirs.pop();
    if (dir) {
      fs.rmSync(dir, { recursive: true, force: true });
    }
  }
});

describe("check-theme-boundary", () => {
  test("current boundary is clean", () => {
    expect(findViolations()).toEqual([]);
  });

  test("rejects new app theme asset", () => {
    const root = makeRoot();
    const asset = path.join(root, "apps/iroha-web/src/lib/themes/NewTheme.svelte");
    fs.mkdirSync(path.dirname(asset), { recursive: true });
    fs.writeFileSync(asset, "<div />\n");

    expect(findViolations(root)).toContain(
      "apps/iroha-web/src/lib/themes/NewTheme.svelte: themed assets belong under packages/iroha-shared",
    );
  });

  test("rejects shared app import", () => {
    const root = makeRoot();
    const sharedFile = path.join(root, "packages/iroha-shared/src/Bad.svelte");
    fs.writeFileSync(sharedFile, 'import api from "$lib/api";\n');

    expect(findViolations(root)).toContain(
      "packages/iroha-shared/src/Bad.svelte:1: shared code cannot import app sources",
    );
  });

  test("rejects app theme directory", () => {
    const root = makeRoot();
    const legacy = path.join(root, "apps/iroha-web/src/lib/themes/registry.ts");
    fs.mkdirSync(path.dirname(legacy), { recursive: true });
    fs.writeFileSync(legacy, 'export * from "@iroha/shared/theme-ui/registry";\n');

    expect(findViolations(root)).toEqual([
      "apps/iroha-web/src/lib/themes/registry.ts: themed assets belong under packages/iroha-shared",
    ]);
  });

  test("planted violation: shared module importing relative app path is rejected", () => {
    const root = makeRoot();
    const planted = path.join(root, "packages/iroha-shared/src/Planted.ts");
    fs.writeFileSync(planted, 'import { x } from "../../apps/iroha-web/src/lib/foo";\n');

    expect(findViolations(root)).toEqual([
      "packages/iroha-shared/src/Planted.ts:1: shared code cannot import app sources",
    ]);
  });
});
