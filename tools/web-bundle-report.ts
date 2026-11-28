#!/usr/bin/env bun
/** Report raw and per-asset gzip sizes for a built private web app. */

import fs from "node:fs";
import path from "node:path";
import { gzipSync } from "node:zlib";

export const REPO_ROOT = path.resolve(import.meta.dir, "..");
export const DEFAULT_BUILD = path.join(REPO_ROOT, "apps", "iroha-web", "build");

export function sizes(contents: Iterable<Uint8Array>): {
  count: number;
  rawBytes: number;
  gzipBytes: number;
} {
  let count = 0;
  let rawBytes = 0;
  let gzipBytes = 0;
  for (const content of contents) {
    count += 1;
    rawBytes += content.byteLength;
    gzipBytes += gzipSync(content).byteLength;
  }
  return { count, rawBytes, gzipBytes };
}

export function assets(root: string, suffix: string): string[] {
  const results: string[] = [];
  for (const entry of fs.readdirSync(root, { withFileTypes: true })) {
    const fullPath = path.join(root, entry.name);
    if (entry.isDirectory()) {
      results.push(...assets(fullPath, suffix));
    } else if (entry.isFile() && entry.name.endsWith(suffix)) {
      results.push(fullPath);
    }
  }
  return results.sort();
}

export function main(argv: string[] = process.argv.slice(2)): number {
  const root = argv[0] ? path.resolve(argv[0]) : DEFAULT_BUILD;
  if (!fs.existsSync(root) || !fs.statSync(root).isDirectory()) {
    console.error(`error: build directory does not exist: ${root}`);
    return 2;
  }

  for (const [label, suffix] of [
    ["JavaScript", ".js"],
    ["CSS", ".css"],
  ] as const) {
    const files = assets(root, suffix);
    const { count, rawBytes, gzipBytes } = sizes(files.map((f) => fs.readFileSync(f)));
    console.log(`${label}: ${rawBytes} raw bytes, ${gzipBytes} gzip bytes across ${count} files`);
  }
  return 0;
}

if (import.meta.main) {
  process.exit(main());
}
