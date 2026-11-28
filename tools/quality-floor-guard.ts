#!/usr/bin/env bun
/** Fail when a diff weakens the quality floor recorded in CONSTRAINTS.md. */

import { spawnSync } from "node:child_process";
import fs from "node:fs";

const SUPPRESSION_LITERALS = [
  "@ts-" + "ignore",
  "@ts-" + "nocheck",
  "eslint-" + "disable",
  "biome-" + "ignore",
  "svelte-" + "ignore",
  "# no" + "qa",
  "#" + "noqa",
  "# type:" + " ignore",
  "# type:" + "ignore",
  "# pragma:" + " no cover",
  "# pragma:" + "no cover",
  "istanbul " + "ignore",
  "nosem" + "grep",
  "gitleaks:" + "allow",
  "Stryker " + "disable",
  "//" + " nolint",
  "//" + "nolint",
];

function escapeRegex(value: string): string {
  return value.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}

const SUPPRESSIONS = new RegExp(SUPPRESSION_LITERALS.map(escapeRegex).join("|"));
const STUBS = new RegExp(
  "throw new (?:Error|NotImplementedError).*not\\s+implemented|" +
    "raise Not" +
    "ImplementedError|catch\\s*\\([^)]*\\)\\s*\\{\\s*\\}|" +
    "catch\\s*\\{\\s*\\}|\\bTO" +
    "DO\\b|\\bpass\\s+#\\s*stub",
  "i",
);
const SKIPS = new RegExp(
  "\\.(?:sk" +
    "ip|to" +
    "do)\\b|\\bxit\\s*\\(|\\bxdescribe\\s*\\(|" +
    "@pytest\\.mark\\.sk" +
    "ip|" +
    "@unittest\\.sk" +
    "ip|\\bt\\.Skip\\s*\\(",
);
const MIN_BEFORE = /(?:>=|>|≥|at least|minimum|\bmin\b|no less than)\s*$/i;
const MAX_BEFORE = /(?:<=|<|≤|at most|maximum|\bmax\b|no more than|under|below)\s*$/i;
const MIN_AFTER = /^\s*\S*\s*(?:or more|or higher|must not fall|must not drop)/i;
const MAX_AFTER = /^\s*\S*\s*(?:or less|or lower|must not grow|must not exceed)/i;
const NUMBER = /\d+(?:\.\d+)?/g;

export interface Change {
  path: string;
  text: string;
}

export interface TrackedException {
  id: string;
  rule: string;
  scope: string;
  reason: string;
  owner: string;
  expires: string;
}

export type Direction = "min" | "max" | null;
export type Violation = [rule: string, path: string];

function stripPrefix(value: string, prefix: string): string {
  return value.startsWith(prefix) ? value.slice(prefix.length) : value;
}

export function parseDiff(diff: string): {
  added: Change[];
  removed: Change[];
  deleted: Set<string>;
} {
  const added: Change[] = [];
  const removed: Change[] = [];
  const deleted = new Set<string>();
  let currentPath = "";
  let oldPath = "";

  for (const line of diff.split(/\r?\n/)) {
    if (line.startsWith("--- ")) {
      oldPath = stripPrefix(line.slice(4), "a/");
    } else if (line.startsWith("+++ ")) {
      const newPath = stripPrefix(line.slice(4), "b/");
      currentPath = newPath === "/dev/null" ? oldPath : newPath;
      if (newPath === "/dev/null") {
        deleted.add(currentPath);
      }
    } else if (line.startsWith("+") && !line.startsWith("+++")) {
      added.push({ path: currentPath, text: line.slice(1) });
    } else if (line.startsWith("-") && !line.startsWith("---")) {
      removed.push({ path: currentPath, text: line.slice(1) });
    }
  }

  return { added, removed, deleted };
}

export function isTest(filePath: string): boolean {
  return /(?:\.test\.|\.spec\.|_test\.|(?:^|\/)test_)/.test(filePath);
}

export function ruleKey(text: string): string | null {
  const stripped = text.trim();
  if (stripped.startsWith("|")) {
    const cells = stripped
      .split("|")
      .map((cell) => cell.trim())
      .filter((cell) => cell.length > 0);
    return cells.length > 0 ? cells[0] : "";
  }
  if (stripped.startsWith("- ") || stripped.startsWith("* ")) {
    const body = stripped.slice(2);
    const colonIdx = body.indexOf(":");
    return (colonIdx === -1 ? body : body.slice(0, colonIdx)).trim();
  }
  return null;
}

export function thresholds(text: string): Array<[number, Direction]> {
  const values: Array<[number, Direction]> = [];
  for (const match of text.matchAll(NUMBER)) {
    const start = match.index ?? 0;
    const end = start + match[0].length;
    const before = text.slice(Math.max(0, start - 24), start);
    const after = text.slice(end, end + 40);
    const direction: Direction =
      MIN_BEFORE.test(before) || MIN_AFTER.test(after)
        ? "min"
        : MAX_BEFORE.test(before) || MAX_AFTER.test(after)
          ? "max"
          : null;
    values.push([Number.parseFloat(match[0]), direction]);
  }
  return values;
}

export function parseExceptions(content: string): TrackedException[] {
  const exceptions: TrackedException[] = [];
  for (const line of content.split(/\r?\n/)) {
    const match = line
      .trim()
      .match(/^\|\s*([WE]\d+)\s*\|\s*([^|]+)\|\s*([^|]+)\|\s*([^|]+)\|\s*([^|]+)\|\s*([^|]+)\|/);
    if (!match) {
      continue;
    }
    const id = match[1].trim();
    if (id === "ID" || id === "---") {
      continue;
    }
    exceptions.push({
      id,
      rule: match[2].trim(),
      scope: match[3].trim(),
      reason: match[4].trim(),
      owner: match[5].trim(),
      expires: match[6].trim(),
    });
  }
  return exceptions;
}

function parseIsoDateKey(value: string): number | null {
  if (!/^\d{4}-\d{2}-\d{2}$/.test(value)) {
    return null;
  }
  const [y, m, d] = value.split("-").map((part) => Number.parseInt(part, 10));
  const dt = new Date(Date.UTC(y, m - 1, d));
  if (dt.getUTCFullYear() !== y || dt.getUTCMonth() !== m - 1 || dt.getUTCDate() !== d) {
    return null;
  }
  return y * 10000 + m * 100 + d;
}

function localTodayKey(): number {
  const now = new Date();
  return now.getFullYear() * 10000 + (now.getMonth() + 1) * 100 + now.getDate();
}

export function isActiveException(exc: TrackedException, todayKey?: number): boolean {
  const refKey = todayKey ?? localTodayKey();
  if (!exc.owner || exc.owner === "Owner" || !exc.reason) {
    return false;
  }
  const expiryKey = parseIsoDateKey(exc.expires);
  if (expiryKey === null) {
    return false;
  }
  return expiryKey >= refKey;
}

function fnmatchToRegex(pattern: string): RegExp {
  let i = 0;
  let regex = "^";
  while (i < pattern.length) {
    const ch = pattern[i];
    i += 1;
    if (ch === "*") {
      regex += ".*";
    } else if (ch === "?") {
      regex += ".";
    } else if (ch === "[") {
      let j = i;
      if (j < pattern.length && pattern[j] === "!") {
        j += 1;
      }
      if (j < pattern.length && pattern[j] === "]") {
        j += 1;
      }
      while (j < pattern.length && pattern[j] !== "]") {
        j += 1;
      }
      if (j >= pattern.length) {
        regex += "\\[";
      } else {
        let stuff = pattern.slice(i, j).replace(/\\/g, "\\\\");
        i = j + 1;
        if (stuff.startsWith("!")) {
          stuff = "^" + stuff.slice(1);
        } else if (stuff.startsWith("^")) {
          stuff = "\\" + stuff;
        }
        regex += `[${stuff}]`;
      }
    } else {
      regex += escapeRegex(ch);
    }
  }
  regex += "$";
  return new RegExp(regex, "s");
}

function fnmatch(value: string, pattern: string): boolean {
  return fnmatchToRegex(pattern).test(value);
}

export function pathMatchesScope(filePath: string, scope: string): boolean {
  if (filePath === scope) {
    return true;
  }
  if (fnmatch(filePath, scope)) {
    return true;
  }
  const normalized = scope.replace(/\*\*+/g, "*");
  return fnmatch(filePath, normalized);
}

function sameException(a: TrackedException, b: TrackedException): boolean {
  return (
    a.id === b.id &&
    a.rule === b.rule &&
    a.scope === b.scope &&
    a.reason === b.reason &&
    a.owner === b.owner &&
    a.expires === b.expires
  );
}

export function findFloorViolations(diff: string, constraintsContent?: string): Violation[] {
  const { added, removed, deleted } = parseDiff(diff);
  const violations: Violation[] = [];

  let resolvedConstraints = constraintsContent;
  if (resolvedConstraints === undefined) {
    try {
      resolvedConstraints = fs.readFileSync("CONSTRAINTS.md", "utf8");
    } catch {
      resolvedConstraints = "";
    }
  }

  const exceptions = resolvedConstraints ? parseExceptions(resolvedConstraints) : [];
  const diffAddedConstraints = added
    .filter((change) => change.path.endsWith("CONSTRAINTS.md"))
    .map((change) => change.text)
    .join("\n");

  if (diffAddedConstraints) {
    for (const exc of parseExceptions(diffAddedConstraints)) {
      if (!exceptions.some((existing) => sameException(existing, exc))) {
        exceptions.push(exc);
      }
    }
  }

  const activeExceptions = exceptions.filter((exc) => isActiveException(exc));

  function isExempted(rule: string, filePath: string): boolean {
    return activeExceptions.some(
      (exc) => (exc.rule === "*" || exc.rule === rule) && pathMatchesScope(filePath, exc.scope),
    );
  }

  function flag(rule: string, filePath: string): void {
    if (isExempted(rule, filePath)) {
      return;
    }
    if (!violations.some(([existingRule, existingPath]) => existingRule === rule && existingPath === filePath)) {
      violations.push([rule, filePath]);
    }
  }

  for (const change of added) {
    if (SUPPRESSIONS.test(change.text)) {
      flag("silenced-checker", change.path);
    }
    if (STUBS.test(change.text)) {
      flag("unfinished-work", change.path);
    }
    if (SKIPS.test(change.text)) {
      flag("test-made-easier", change.path);
    }
    if (change.path.endsWith("CONSTRAINTS.md") && /^\|\s*(?:W|E)\d+\s*\|/.test(change.text)) {
      const rowExc = parseExceptions(change.text);
      if (rowExc.length === 0 || !isActiveException(rowExc[0])) {
        flag("new-exception", change.path);
      }
    }
  }

  for (const filePath of deleted) {
    if (isTest(filePath)) {
      flag("test-deleted", filePath);
    }
  }

  for (const change of removed) {
    if (
      isTest(change.path) &&
      !deleted.has(change.path) &&
      /\b(?:expect|assert(?:[A-Z]\w*)?|require|should)\b/.test(change.text)
    ) {
      flag("assertion-removed", change.path);
    }
  }

  const removedRules = removed.filter(
    (item) => item.path.endsWith("CONSTRAINTS.md") && ruleKey(item.text) !== null,
  );
  const addedRules = added.filter(
    (item) => item.path.endsWith("CONSTRAINTS.md") && ruleKey(item.text) !== null,
  );

  for (const old of removedRules) {
    const oldKey = ruleKey(old.text);
    const replacement = addedRules.find((item) => ruleKey(item.text) === oldKey);
    if (!replacement) {
      if (!/^\|\s*(?:W|E)\d+\s*\|/.test(old.text.trim())) {
        flag("rule-removed", old.path);
      }
      continue;
    }
    const before = thresholds(old.text);
    const after = thresholds(replacement.text);
    for (const direction of ["min", "max", null] as const) {
      const previous = before.filter(([, dir]) => dir === direction);
      const current = after.filter(([, dir]) => dir === direction);
      for (let index = 0; index < previous.length; index++) {
        if (index >= current.length) {
          flag("threshold-removed", old.path);
          break;
        }
        const oldValue = previous[index][0];
        const newValue = current[index][0];
        const weakened =
          direction === "min"
            ? newValue < oldValue
            : direction === "max"
              ? newValue > oldValue
              : newValue !== oldValue;
        if (weakened) {
          flag(direction ? "threshold-weakened" : "threshold-changed", old.path);
          break;
        }
      }
    }
  }

  return violations;
}

function git(args: string[], diffExit = false): string | null {
  const result = spawnSync("git", args, {
    encoding: "utf8",
    maxBuffer: 64 * 1024 * 1024,
  });
  if (result.status === 0 || (diffExit && result.status === 1)) {
    return result.stdout ?? "";
  }
  return null;
}

export function workingDiff(base: string): string {
  const mergeBase = git(["merge-base", base, "HEAD"]);
  if (!mergeBase || !mergeBase.trim()) {
    throw new Error(`no merge base against ${base}`);
  }
  const tracked = git(["diff", "--unified=0", mergeBase.trim(), "--"]);
  if (tracked === null) {
    throw new Error(`could not diff against ${mergeBase.trim()}`);
  }
  const untracked = git(["ls-files", "--others", "--exclude-standard"]);
  if (untracked === null) {
    throw new Error("could not list untracked files");
  }
  const diffs = [tracked];
  for (const filePath of untracked.split(/\r?\n/)) {
    if (!filePath) {
      continue;
    }
    const diff = git(["diff", "--no-index", "--unified=0", "/dev/null", "--", filePath], true);
    if (diff === null) {
      throw new Error(`could not diff untracked file ${filePath}`);
    }
    diffs.push(diff);
  }
  return diffs.join("\n");
}

function parseBaseArg(argv: string[]): string {
  let base = "origin/main";
  for (let i = 0; i < argv.length; i++) {
    const arg = argv[i];
    if (arg === "--base" && i + 1 < argv.length) {
      base = argv[i + 1];
      i += 1;
    } else if (arg.startsWith("--base=")) {
      base = arg.slice("--base=".length);
    }
  }
  return base;
}

export function main(argv: string[] = process.argv.slice(2)): number {
  const base = parseBaseArg(argv);
  let violations: Violation[];
  try {
    violations = findFloorViolations(workingDiff(base));
  } catch (error) {
    const message = error instanceof Error ? error.message : String(error);
    console.error(`quality-floor-guard: ${message}`);
    return 2;
  }
  if (violations.length === 0) {
    console.log("quality-floor-guard: clean");
    return 0;
  }
  console.error(`quality-floor-guard: ${violations.length} floor violation(s):`);
  for (const [rule, filePath] of violations) {
    console.error(`  [${rule}] ${filePath}`);
  }
  console.error("Fix the code or route a reviewed change through a tracked exception.");
  return 1;
}

if (import.meta.main) {
  process.exit(main());
}
