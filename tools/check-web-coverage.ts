#!/usr/bin/env bun
/** Fail when Vitest silently omits app or shared source from its coverage report. */

import fs from "node:fs";
import path from "node:path";

export const ROOT = path.resolve(import.meta.dir, "..");
export const SOURCE_DIRS = ["apps/iroha-web/src", "packages/iroha-shared/src"] as const;
export const METRICS = ["statements", "branches", "functions", "lines"] as const;
const SOURCE_SUFFIXES = new Set([".js", ".ts", ".svelte"]);
const TEST_SUFFIXES = [".test.js", ".spec.js", ".test.ts", ".spec.ts"];

function isPlainObject(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function resolveCanonical(targetPath: string): string {
  const abs = path.resolve(targetPath);
  if (fs.existsSync(abs)) {
    return fs.realpathSync(abs);
  }
  return abs;
}

function walkFiles(dir: string): string[] {
  if (!fs.existsSync(dir) || !fs.statSync(dir).isDirectory()) {
    return [];
  }
  const results: string[] = [];
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    const fullPath = path.join(dir, entry.name);
    if (entry.isDirectory()) {
      results.push(...walkFiles(fullPath));
    } else if (entry.isFile()) {
      results.push(fullPath);
    }
  }
  return results;
}

export function missingSources(root: string, report: unknown): string[] {
  const resolvedRoot = resolveCanonical(root);
  if (!isPlainObject(report)) {
    throw new Error("report must be an object");
  }

  const measured = new Set<string>();
  for (const [name, metrics] of Object.entries(report)) {
    if (
      name !== "total" &&
      isPlainObject(metrics) &&
      METRICS.every((metric) => isPlainObject(metrics[metric]))
    ) {
      measured.add(resolveCanonical(name));
    }
  }

  const expected = new Set<string>();
  for (const directory of SOURCE_DIRS) {
    for (const filePath of walkFiles(path.join(resolvedRoot, directory))) {
      const ext = path.extname(filePath);
      if (
        SOURCE_SUFFIXES.has(ext) &&
        !TEST_SUFFIXES.some((suffix) => filePath.endsWith(suffix))
      ) {
        expected.add(resolveCanonical(filePath));
      }
    }
  }

  if (expected.size === 0) {
    throw new Error("source inventory is empty");
  }

  const missing: string[] = [];
  for (const filePath of expected) {
    if (!measured.has(filePath)) {
      missing.push(path.relative(resolvedRoot, filePath).split(path.sep).join("/"));
    }
  }
  return missing.sort();
}

export function main(root: string = ROOT): number {
  let missing: string[];
  try {
    const summaryPath = path.join(root, "apps/iroha-web/coverage/coverage-summary.json");
    const report: unknown = JSON.parse(fs.readFileSync(summaryPath, "utf8"));
    missing = missingSources(root, report);
  } catch (error) {
    const name = error instanceof Error ? error.name : "Error";
    console.error(`web-coverage: cannot verify report (${name})`);
    return 1;
  }

  if (missing.length > 0) {
    console.error(`web-coverage: ${missing.length} source files omitted`);
    for (const filePath of missing) {
      console.error(`  ${filePath}`);
    }
    return 1;
  }

  console.log("web-coverage: all app and shared source files measured");
  return 0;
}

if (import.meta.main) {
  process.exit(main());
}
