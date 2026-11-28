import { afterEach, describe, expect, test } from "bun:test";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";

import { findViolations } from "../check-motion-tokens.ts";

const tempDirs: string[] = [];

function makeRoot(): string {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), "iroha-motion-"));
  tempDirs.push(root);
  const themes = path.join(root, "packages/iroha-shared/src/theme/themes.css");
  fs.mkdirSync(path.dirname(themes), { recursive: true });
  fs.writeFileSync(themes, ":root {\n  --motion-quick-state: 220ms ease;\n}\n");
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

describe("check-motion-tokens", () => {
  test("current tokens are clean", () => {
    expect(findViolations()).toEqual([]);
  });

  test("rejects reference to undefined token", () => {
    const root = makeRoot();
    const consumer = path.join(root, "apps/iroha-web/src/routes/app.css");
    fs.mkdirSync(path.dirname(consumer), { recursive: true });
    fs.writeFileSync(consumer, ".appbar { transition: opacity var(--motion-quick-statee); }\n");

    expect(findViolations(root)).toContain(
      "apps/iroha-web/src/routes/app.css:1: var(--motion-quick-statee) has no " +
        "matching --motion-quick-statee: definition in " +
        "packages/iroha-shared/src/theme/themes.css",
    );
  });

  test("allows reference to defined token", () => {
    const root = makeRoot();
    const consumer = path.join(root, "packages/iroha-shared/src/components/Widget.svelte");
    fs.mkdirSync(path.dirname(consumer), { recursive: true });
    fs.writeFileSync(
      consumer,
      "<style>a { transition: opacity var(--motion-quick-state); }</style>\n",
    );

    expect(findViolations(root)).toEqual([]);
  });

  test("deleted token still referenced elsewhere is caught", () => {
    const root = makeRoot();
    const themes = path.join(root, "packages/iroha-shared/src/theme/themes.css");
    fs.writeFileSync(themes, ":root {\n}\n");
    const consumer = path.join(root, "apps/iroha-web/src/routes/app.css");
    fs.mkdirSync(path.dirname(consumer), { recursive: true });
    fs.writeFileSync(consumer, ".appbar { transition: opacity var(--motion-quick-state); }\n");

    expect(findViolations(root)).toHaveLength(1);
  });

  test("planted violation: undefined --motion-planted-missing token is rejected", () => {
    const root = makeRoot();
    const planted = path.join(root, "apps/iroha-web/src/routes/planted.css");
    fs.mkdirSync(path.dirname(planted), { recursive: true });
    fs.writeFileSync(planted, ".card { transition: transform var(--motion-planted-missing); }\n");

    expect(findViolations(root)).toEqual([
      "apps/iroha-web/src/routes/planted.css:1: var(--motion-planted-missing) has no " +
        "matching --motion-planted-missing: definition in " +
        "packages/iroha-shared/src/theme/themes.css",
    ]);
  });
});
