#!/usr/bin/env bun
/** Enforce the shared theme asset and import-direction boundary. */

import fs from "node:fs";
import path from "node:path";

export const REPO_ROOT = path.resolve(import.meta.dir, "..");

const SOURCE_SUFFIXES = new Set([".css", ".js", ".svelte", ".ts", ".tsx"]);
const FORBIDDEN_SHARED_IMPORTS: RegExp[] = [
  /from\s+['"]\$lib(?:\/|['"])/,
  /from\s+['"](?:\.\.\/)+apps\/iroha-(?:web|public-site)(?:\/|['"])/,
  /from\s+['"]apps\/iroha-(?:web|public-site)(?:\/|['"])/,
];

function toPosixRelative(filePath: string, root: string): string {
  return path.relative(root, filePath).split(path.sep).join("/");
}

function walkFiles(dir: string): string[] {
  const results: string[] = [];
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    const fullPath = path.join(dir, entry.name);
    if (entry.isDirectory()) {
      results.push(...walkFiles(fullPath));
    } else if (entry.isFile()) {
      results.push(fullPath);
    }
  }
  return results.sort();
}

export function findViolations(root: string = REPO_ROOT): string[] {
  const violations: string[] = [];
  const appsRoot = path.join(root, "apps");
  if (fs.existsSync(appsRoot) && fs.statSync(appsRoot).isDirectory()) {
    const appDirs = fs
      .readdirSync(appsRoot, { withFileTypes: true })
      .filter((entry) => entry.isDirectory())
      .map((entry) => entry.name)
      .sort();
    for (const appName of appDirs) {
      const themeDir = path.join(appsRoot, appName, "src", "lib", "themes");
      if (!fs.existsSync(themeDir) || !fs.statSync(themeDir).isDirectory()) {
        continue;
      }
      for (const filePath of walkFiles(themeDir)) {
        const relative = toPosixRelative(filePath, root);
        violations.push(`${relative}: themed assets belong under packages/iroha-shared`);
      }
    }
  }

  const sharedRoot = path.join(root, "packages", "iroha-shared", "src");
  if (fs.existsSync(sharedRoot) && fs.statSync(sharedRoot).isDirectory()) {
    for (const filePath of walkFiles(sharedRoot)) {
      if (!SOURCE_SUFFIXES.has(path.extname(filePath))) {
        continue;
      }
      const lines = fs.readFileSync(filePath, "utf8").split(/\r?\n/);
      for (let idx = 0; idx < lines.length; idx++) {
        const line = lines[idx];
        if (FORBIDDEN_SHARED_IMPORTS.some((pattern) => pattern.test(line))) {
          violations.push(
            `${toPosixRelative(filePath, root)}:${idx + 1}: shared code cannot import app sources`,
          );
        }
      }
    }
  }

  return violations.sort();
}

export function main(): number {
  const violations = findViolations();
  if (violations.length > 0) {
    console.error("Theme asset boundary check failed:");
    for (const violation of violations) {
      console.error(`- ${violation}`);
    }
    return 1;
  }
  console.log("Theme asset boundary: clean");
  return 0;
}

if (import.meta.main) {
  process.exit(main());
}
