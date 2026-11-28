import { afterEach, describe, expect, test } from "bun:test";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";

import { findViolations } from "../check-responsive-contract.ts";

const tempDirs: string[] = [];

function makeTempDir(): string {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), "iroha-responsive-"));
  tempDirs.push(dir);
  return dir;
}

afterEach(() => {
  while (tempDirs.length > 0) {
    const dir = tempDirs.pop();
    if (dir) {
      fs.rmSync(dir, { recursive: true, force: true });
    }
  }
});

describe("check-responsive-contract", () => {
  test("accepts the canonical breakpoints", () => {
    const root = makeTempDir();
    const source = path.join(root, "apps/iroha-web/src/routes/app.css");
    fs.mkdirSync(path.dirname(source), { recursive: true });
    fs.writeFileSync(
      source,
      "@media (max-width: 640px) {}\n@media (max-width: 768px) {}\n@media (max-width: 1024px) {}\n",
    );

    expect(findViolations(root)).toEqual([]);
  });

  test("rejects a route-local breakpoint", () => {
    const root = makeTempDir();
    const source = path.join(root, "packages/iroha-shared/src/Widget.svelte");
    fs.mkdirSync(path.dirname(source), { recursive: true });
    fs.writeFileSync(source, "@media (max-width: 700px) {}\n");

    expect(findViolations(root)).toEqual([
      "packages/iroha-shared/src/Widget.svelte:1: max-width 700px is not in [640, 768, 1024]",
    ]);
  });

  test("planted violation: non-canonical 900px max-width is rejected", () => {
    const root = makeTempDir();
    const planted = path.join(root, "apps/iroha-web/src/routes/planted.css");
    fs.mkdirSync(path.dirname(planted), { recursive: true });
    fs.writeFileSync(planted, "@media (max-width: 900px) { .grid { display: block; } }\n");

    expect(findViolations(root)).toEqual([
      "apps/iroha-web/src/routes/planted.css:1: max-width 900px is not in [640, 768, 1024]",
    ]);
  });
});
