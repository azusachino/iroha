#!/usr/bin/env bun
/**
 * Fail if a --motion-* custom property is referenced but never defined.
 *
 * One-directional check (referenced-but-undefined only); unused definitions
 * are allowed.
 */

import fs from "node:fs";
import path from "node:path";

export const REPO_ROOT = path.resolve(import.meta.dir, "..");
export const DEFINITION_FILE_RELATIVE = "packages/iroha-shared/src/theme/themes.css";
const DEFINE_RE = /--(motion-[\w-]+)\s*:/g;
const USE_RE = /var\(\s*--(motion-[\w-]+)/g;
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

export function definedTokens(root: string = REPO_ROOT): Set<string> {
  const definitionFile = path.join(root, DEFINITION_FILE_RELATIVE);
  if (!fs.existsSync(definitionFile) || !fs.statSync(definitionFile).isFile()) {
    return new Set();
  }
  const text = fs.readFileSync(definitionFile, "utf8");
  const tokens = new Set<string>();
  for (const match of text.matchAll(DEFINE_RE)) {
    tokens.add(match[1]);
  }
  return tokens;
}

export function findViolations(root: string = REPO_ROOT): string[] {
  const known = definedTokens(root);
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
        for (const match of lines[idx].matchAll(USE_RE)) {
          const name = match[1];
          if (!known.has(name)) {
            const relative = toPosixRelative(filePath, root);
            violations.push(
              `${relative}:${idx + 1}: var(--${name}) has no matching ` +
                `--${name}: definition in ${DEFINITION_FILE_RELATIVE}`,
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
    console.error("Motion token check failed:");
    for (const violation of violations) {
      console.error(`- ${violation}`);
    }
    return 1;
  }
  console.log(`Motion tokens: ${definedTokens().size} defined, all references resolve`);
  return 0;
}

if (import.meta.main) {
  process.exit(main());
}
