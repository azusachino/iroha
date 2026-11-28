import { afterEach, describe, expect, test } from "bun:test";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { gzipSync } from "node:zlib";

import { assets, main, sizes } from "../web-bundle-report.ts";

const tempDirs: string[] = [];

function makeTempDir(): string {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), "iroha-bundle-"));
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

describe("web-bundle-report", () => {
  test("sums raw and per-asset gzip sizes", () => {
    const enc = new TextEncoder();
    const contents = [enc.encode("first asset"), enc.encode("second asset")];
    const { count, rawBytes, gzipBytes } = sizes(contents);

    expect(count).toBe(2);
    expect(rawBytes).toBe(contents.reduce((acc, item) => acc + item.byteLength, 0));
    expect(gzipBytes).toBe(
      contents.reduce((acc, item) => acc + gzipSync(item).byteLength, 0),
    );
  });

  test("discovers .js and .css assets recursively", () => {
    const dir = makeTempDir();
    fs.mkdirSync(path.join(dir, "_app"), { recursive: true });
    fs.writeFileSync(path.join(dir, "_app/a.js"), "console.log(1);");
    fs.writeFileSync(path.join(dir, "_app/b.css"), "body {}");

    expect(assets(dir, ".js")).toEqual([path.join(dir, "_app/a.js")]);
    expect(assets(dir, ".css")).toEqual([path.join(dir, "_app/b.css")]);
  });

  test("planted violation: missing build directory exits with code 2", () => {
    const missing = path.join(makeTempDir(), "does-not-exist");
    const origErr = console.error;
    console.error = () => {};
    try {
      expect(main([missing])).toBe(2);
    } finally {
      console.error = origErr;
    }
  });
});
