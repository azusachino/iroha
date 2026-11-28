import { afterEach, describe, expect, test } from "bun:test";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";

import { METRICS, SOURCE_DIRS, main, missingSources } from "../check-web-coverage.ts";

const tempDirs: string[] = [];

function setupFixture(): {
  root: string;
  sources: string[];
  report: Record<string, unknown>;
} {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), "iroha-cov-"));
  tempDirs.push(root);
  const sources: string[] = [];
  for (const directory of SOURCE_DIRS) {
    const filePath = path.join(root, directory, "unexecuted.svelte");
    fs.mkdirSync(path.dirname(filePath), { recursive: true });
    fs.writeFileSync(filePath, "<p>Never imported</p>\n");
    sources.push(filePath);
  }
  const metrics = Object.fromEntries(
    METRICS.map((metric) => [metric, { total: 1, covered: 0, pct: 0 }]),
  );
  const report: Record<string, unknown> = Object.fromEntries(
    sources.map((filePath) => [filePath, metrics]),
  );
  return { root, sources, report };
}

afterEach(() => {
  while (tempDirs.length > 0) {
    const dir = tempDirs.pop();
    if (dir) {
      fs.rmSync(dir, { recursive: true, force: true });
    }
  }
});

describe("check-web-coverage", () => {
  test("unexecuted files count as measured", () => {
    const { root, report } = setupFixture();
    expect(missingSources(root, report)).toEqual([]);
  });

  test("app-only report fails shared inventory", () => {
    const { root, sources, report } = setupFixture();
    delete report[sources[1]];
    expect(missingSources(root, report)).toEqual([
      path.relative(root, sources[1]).split(path.sep).join("/"),
    ]);
  });

  test("new source and invalid file metadata fail", () => {
    const { root, report } = setupFixture();
    const newFile = path.join(root, SOURCE_DIRS[1], "new.ts");
    fs.writeFileSync(newFile, "export const value = 1;\n");
    report[newFile] = {};
    expect(missingSources(root, report)).toEqual([
      path.relative(root, newFile).split(path.sep).join("/"),
    ]);
  });

  test("test and non-source files are not required", () => {
    const { root, sources, report } = setupFixture();
    const dir = path.dirname(sources[1]);
    fs.writeFileSync(path.join(dir, "value.test.ts"), "test code\n");
    fs.writeFileSync(path.join(dir, "theme.css"), "body {}\n");
    expect(missingSources(root, report)).toEqual([]);
  });

  test("empty inventory and invalid report fail closed", () => {
    const { root } = setupFixture();
    expect(() => missingSources(path.join(root, "absent"), {})).toThrow();
    expect(() => missingSources(root, [])).toThrow();
  });

  test("cli missing, corrupt, then valid report", () => {
    const { root, report } = setupFixture();
    const summaryPath = path.join(root, "apps/iroha-web/coverage/coverage-summary.json");
    fs.mkdirSync(path.dirname(summaryPath), { recursive: true });

    const origLog = console.log;
    const origErr = console.error;
    console.log = () => {};
    console.error = () => {};
    try {
      expect(main(root)).toBe(1);
      fs.writeFileSync(summaryPath, "invalid json");
      expect(main(root)).toBe(1);
      fs.writeFileSync(summaryPath, JSON.stringify(report));
      expect(main(root)).toBe(0);
    } finally {
      console.log = origLog;
      console.error = origErr;
    }
  });

  test("planted violation: omitted shared source file fails CLI with exit 1", () => {
    const { root, report } = setupFixture();
    const planted = path.join(root, "packages/iroha-shared/src/planted.ts");
    fs.writeFileSync(planted, "export const planted = true;\n");
    const summaryPath = path.join(root, "apps/iroha-web/coverage/coverage-summary.json");
    fs.mkdirSync(path.dirname(summaryPath), { recursive: true });
    fs.writeFileSync(summaryPath, JSON.stringify(report));

    const origErr = console.error;
    console.error = () => {};
    try {
      expect(main(root)).toBe(1);
    } finally {
      console.error = origErr;
    }
  });
});
