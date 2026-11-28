import { describe, expect, test } from "bun:test";

import { findFloorViolations, parseExceptions, pathMatchesScope } from "../quality-floor-guard.ts";

describe("quality-floor-guard", () => {
  test("added suppression is flagged without echoing value", () => {
    const diff = ["--- a/example.go", "+++ b/example.go", "@@", "+//" + "nolint:gosec"].join("\n");
    expect(findFloorViolations(diff)).toEqual([["silenced-checker", "example.go"]]);
  });

  test("added skip and stub are flagged", () => {
    const diff = [
      "--- a/example_test.ts",
      "+++ b/example_test.ts",
      "@@",
      "+it" + "." + 'skip("does not run", () => {});',
      '+throw new Error("Not ' + 'implemented");',
    ].join("\n");
    const rules = new Set(findFloorViolations(diff).map(([rule]) => rule));
    expect(rules).toEqual(new Set(["test-made-easier", "unfinished-work"]));
  });

  test("deleted test and removed assertion are flagged", () => {
    const deleted = [
      "--- a/check_test.py",
      "+++ /dev/null",
      "@@",
      "-assert result == expected",
      "",
    ].join("\n");
    const deletedRules = new Set(findFloorViolations(deleted).map(([rule]) => rule));
    expect(deletedRules.has("test-deleted")).toBe(true);

    const edited = [
      "--- a/check_test.py",
      "+++ b/check_test.py",
      "@@",
      "-        self.assertEqual(result, expected)",
      "+        return result",
      "",
    ].join("\n");
    const editedRules = new Set(findFloorViolations(edited).map(([rule]) => rule));
    expect(editedRules.has("assertion-removed")).toBe(true);
  });

  test("lowered constraint threshold is flagged", () => {
    const diff = [
      "--- a/CONSTRAINTS.md",
      "+++ b/CONSTRAINTS.md",
      "@@",
      "-| Web line coverage | >= 10.80% |",
      "+| Web line coverage | >= 10.00% |",
      "",
    ].join("\n");
    const rules = new Set(findFloorViolations(diff).map(([rule]) => rule));
    expect(rules.has("threshold-weakened")).toBe(true);
  });

  test("tightened constraint threshold is allowed", () => {
    const diff = [
      "--- a/CONSTRAINTS.md",
      "+++ b/CONSTRAINTS.md",
      "@@",
      "-| Web line coverage | >= 10.80% |",
      "+| Web line coverage | >= 11.00% |",
      "",
    ].join("\n");
    expect(findFloorViolations(diff)).toEqual([]);
  });

  test("new exception row is flagged", () => {
    const diff = [
      "--- a/CONSTRAINTS.md",
      "+++ b/CONSTRAINTS.md",
      "@@",
      "+| E1 | Lower line coverage | scripts/** | Temporary | Owner | 2026-12-31 |",
      "",
    ].join("\n");
    const rules = new Set(findFloorViolations(diff).map(([rule]) => rule));
    expect(rules.has("new-exception")).toBe(true);
  });

  test("tracked exception exempts matching deleted test", () => {
    const diff = [
      "--- a/check_test.py",
      "+++ /dev/null",
      "@@",
      "-assert result == expected",
      "",
    ].join("\n");
    const constraints = "| E1 | test-deleted | check_test.py | Retire check | haru | 2099-01-01 |\n";
    expect(findFloorViolations(diff, constraints)).toEqual([]);
  });

  test("tracked exception exempts matching removed assertion", () => {
    const diff = [
      "--- a/apps/iroha-web/e2e/test.spec.ts",
      "+++ b/apps/iroha-web/e2e/test.spec.ts",
      "@@",
      "-expect(value).toBe(true);",
      "+console.log(value);",
      "",
    ].join("\n");
    const constraints =
      "| E1 | assertion-removed | apps/iroha-web/** | Retire feature | haru | 2099-01-01 |\n";
    expect(findFloorViolations(diff, constraints)).toEqual([]);
  });

  test("expired tracked exception does not exempt", () => {
    const diff = [
      "--- a/check_test.py",
      "+++ /dev/null",
      "@@",
      "-assert result == expected",
      "",
    ].join("\n");
    const constraints = "| E1 | test-deleted | check_test.py | Expired | haru | 2020-01-01 |\n";
    const rules = new Set(findFloorViolations(diff, constraints).map(([rule]) => rule));
    expect(rules.has("test-deleted")).toBe(true);
  });

  test("planted violation: removed assertion in test file is flagged", () => {
    const diff = [
      "--- a/tools/tests/sample.test.ts",
      "+++ b/tools/tests/sample.test.ts",
      "@@",
      "-expect(actual).toBe(42);",
      "+void actual;",
      "",
    ].join("\n");
    expect(findFloorViolations(diff, "")).toEqual([
      ["assertion-removed", "tools/tests/sample.test.ts"],
    ]);
  });

  test("planted violation: added suppression is flagged", () => {
    const diff = [
      "--- a/apps/iroha-web/src/lib/sample.ts",
      "+++ b/apps/iroha-web/src/lib/sample.ts",
      "@@",
      "+// " + "@ts-" + "ignore",
      "",
    ].join("\n");
    expect(findFloorViolations(diff, "")).toEqual([
      ["silenced-checker", "apps/iroha-web/src/lib/sample.ts"],
    ]);
  });

  test("planted violation: deleted test file is flagged", () => {
    const diff = [
      "--- a/apps/iroha-server/pkg/sample/handler_test.go",
      "+++ /dev/null",
      "@@",
      "-package sample",
      "",
    ].join("\n");
    expect(findFloorViolations(diff, "")).toEqual([
      ["test-deleted", "apps/iroha-server/pkg/sample/handler_test.go"],
    ]);
  });

  test("planted violation: CONSTRAINTS.md threshold weakened is flagged", () => {
    const diff = [
      "--- a/CONSTRAINTS.md",
      "+++ b/CONSTRAINTS.md",
      "@@",
      "-| Go statement coverage | >= 30.84% | `make test` |",
      "+| Go statement coverage | >= 29.00% | `make test` |",
      "",
    ].join("\n");
    expect(findFloorViolations(diff, "")).toEqual([["threshold-weakened", "CONSTRAINTS.md"]]);
  });

  test("max threshold weakened, rule removed, and glob normalization parity", () => {
    const maxWeakened = [
      "--- a/CONSTRAINTS.md",
      "+++ b/CONSTRAINTS.md",
      "@@",
      "-| API cold p95 | <= 13.33 ms |",
      "+| API cold p95 | <= 20.00 ms |",
      "",
    ].join("\n");
    expect(findFloorViolations(maxWeakened, "")).toEqual([
      ["threshold-weakened", "CONSTRAINTS.md"],
    ]);

    const ruleRemoved = [
      "--- a/CONSTRAINTS.md",
      "+++ b/CONSTRAINTS.md",
      "@@",
      "-| Go statement coverage | >= 30.84% |",
      "",
    ].join("\n");
    expect(findFloorViolations(ruleRemoved, "")).toEqual([["rule-removed", "CONSTRAINTS.md"]]);

    expect(pathMatchesScope("apps/iroha-web/src/a.ts", "apps/**")).toBe(true);
    expect(
      parseExceptions("| ID | Rule | Scope | Reason | Owner | Expires |\n| W1 | * | a | r | haru | 2099-01-01 |\n").length,
    ).toBe(1);
  });
});
