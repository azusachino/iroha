import unittest

from quality_floor_guard import find_floor_violations


class QualityFloorGuardTest(unittest.TestCase):
    def test_added_suppression_is_flagged_without_echoing_value(self):
        diff = "\n".join(
            ["--- a/example.go", "+++ b/example.go", "@@", "+//" + "nolint:gosec"]
        )
        self.assertEqual(find_floor_violations(diff), [("silenced-checker", "example.go")])

    def test_added_skip_and_stub_are_flagged(self):
        diff = "\n".join(
            [
                "--- a/example_test.ts",
                "+++ b/example_test.ts",
                "@@",
                "+it" + "." + "skip(\"does not run\", () => {});",
                '+throw new Error("Not ' + 'implemented");',
            ]
        )
        rules = {rule for rule, _ in find_floor_violations(diff)}
        self.assertEqual(rules, {"test-made-easier", "unfinished-work"})

    def test_deleted_test_and_removed_assertion_are_flagged(self):
        deleted = """--- a/check_test.py
+++ /dev/null
@@
-assert result == expected
"""
        self.assertIn("test-deleted", {rule for rule, _ in find_floor_violations(deleted)})
        edited = """--- a/check_test.py
+++ b/check_test.py
@@
-        self.assertEqual(result, expected)
+        return result
"""
        self.assertIn("assertion-removed", {rule for rule, _ in find_floor_violations(edited)})

    def test_lowered_constraint_threshold_is_flagged(self):
        diff = """--- a/CONSTRAINTS.md
+++ b/CONSTRAINTS.md
@@
-| Web line coverage | >= 10.80% |
+| Web line coverage | >= 10.00% |
"""
        self.assertIn("threshold-weakened", {rule for rule, _ in find_floor_violations(diff)})

    def test_tightened_constraint_threshold_is_allowed(self):
        diff = """--- a/CONSTRAINTS.md
+++ b/CONSTRAINTS.md
@@
-| Web line coverage | >= 10.80% |
+| Web line coverage | >= 11.00% |
"""
        self.assertEqual(find_floor_violations(diff), [])

    def test_new_exception_row_is_flagged(self):
        diff = """--- a/CONSTRAINTS.md
+++ b/CONSTRAINTS.md
@@
+| E1 | Lower line coverage | scripts/** | Temporary | Owner | 2026-12-31 |
"""
        self.assertIn("new-exception", {rule for rule, _ in find_floor_violations(diff)})

    def test_tracked_exception_exempts_matching_deleted_test(self):
        diff = """--- a/check_test.py
+++ /dev/null
@@
-assert result == expected
"""
        constraints = "| E1 | test-deleted | check_test.py | Retire check | haru | 2099-01-01 |\n"
        self.assertEqual(find_floor_violations(diff, constraints_content=constraints), [])

    def test_tracked_exception_exempts_matching_removed_assertion(self):
        diff = """--- a/apps/iroha-web/e2e/test.spec.ts
+++ b/apps/iroha-web/e2e/test.spec.ts
@@
-expect(value).toBe(true);
+console.log(value);
"""
        constraints = "| E1 | assertion-removed | apps/iroha-web/** | Retire feature | haru | 2099-01-01 |\n"
        self.assertEqual(find_floor_violations(diff, constraints_content=constraints), [])

    def test_expired_tracked_exception_does_not_exempt(self):
        diff = """--- a/check_test.py
+++ /dev/null
@@
-assert result == expected
"""
        constraints = "| E1 | test-deleted | check_test.py | Expired | haru | 2020-01-01 |\n"
        self.assertIn("test-deleted", {rule for rule, _ in find_floor_violations(diff, constraints_content=constraints)})


if __name__ == "__main__":
    unittest.main()
