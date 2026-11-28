#!/usr/bin/env bun
/** Enforce Iroha's small, shared responsive breakpoint vocabulary. */

import fs from "node:fs";
import path from "node:path";

export const REPO_ROOT = path.resolve(import.meta.dir, "..");
const ALLOWED_MAX_WIDTHS = new Set([640, 768, 1024]);
const MEDIA_RE = /@media\s*\(\s*max-width\s*:\s*(\d+)px\s*\)/g;
const SOURCE_ROOTS = ["apps/iroha-web/src", "packages/iroha-shared/src"];
const SOURCE_SUFFIXES = new Set([".css", ".svelte"]);

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
  for (const sourceRootName of SOURCE_ROOTS) {
    const sourceRoot = path.join(root, sourceRootName);
    if (!fs.existsSync(sourceRoot) || !fs.statSync(sourceRoot).isDirectory()) {
      continue;
    }
    for (const filePath of walkFiles(sourceRoot)) {
      if (!SOURCE_SUFFIXES.has(path.extname(filePath))) {
        continue;
      }
      const lines = fs.readFileSync(filePath, "utf8").split(/\r?\n/);
      for (let idx = 0; idx < lines.length; idx++) {
        for (const match of lines[idx].matchAll(MEDIA_RE)) {
          const width = Number.parseInt(match[1], 10);
          if (!ALLOWED_MAX_WIDTHS.has(width)) {
            const relative = toPosixRelative(filePath, root);
            violations.push(
              `${relative}:${idx + 1}: max-width ${width}px is not in [640, 768, 1024]`,
            );
          }
        }
      }
    }
  }
  return violations.sort();
}

export function main(): number {
  const violations = findViolations();
  if (violations.length > 0) {
    console.error("Responsive breakpoint check failed:");
    for (const violation of violations) {
      console.error(`- ${violation}`);
    }
    return 1;
  }
  console.log("Responsive breakpoints: 640px / 768px / 1024px");
  return 0;
}

if (import.meta.main) {
  process.exit(main());
}
