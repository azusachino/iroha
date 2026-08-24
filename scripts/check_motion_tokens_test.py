import tempfile
import unittest
from pathlib import Path

import check_motion_tokens


class MotionTokensTest(unittest.TestCase):
    def _root(self) -> Path:
        root = Path(tempfile.mkdtemp())
        themes = root / "packages/iroha-shared/src/theme/themes.css"
        themes.parent.mkdir(parents=True)
        themes.write_text(":root {\n  --motion-quick-state: 220ms ease;\n}\n")
        return root

    def test_current_tokens_are_clean(self):
        self.assertEqual(check_motion_tokens.find_violations(), [])

    def test_rejects_reference_to_undefined_token(self):
        root = self._root()
        consumer = root / "apps/iroha-web/src/routes/app.css"
        consumer.parent.mkdir(parents=True, exist_ok=True)
        consumer.write_text(".appbar { transition: opacity var(--motion-quick-statee); }\n")

        violations = check_motion_tokens.find_violations(root)

        self.assertIn(
            "apps/iroha-web/src/routes/app.css:1: var(--motion-quick-statee) has no "
            "matching --motion-quick-statee: definition in "
            "packages/iroha-shared/src/theme/themes.css",
            violations,
        )

    def test_allows_reference_to_defined_token(self):
        root = self._root()
        consumer = root / "packages/iroha-shared/src/components/Widget.svelte"
        consumer.parent.mkdir(parents=True, exist_ok=True)
        consumer.write_text("<style>a { transition: opacity var(--motion-quick-state); }</style>\n")

        self.assertEqual(check_motion_tokens.find_violations(root), [])

    def test_a_deleted_token_still_referenced_elsewhere_is_caught(self):
        # Simulates Task 2/3's own scenario: a token renamed or removed from
        # themes.css while a consumer still references the old name.
        root = self._root()
        themes = root / "packages/iroha-shared/src/theme/themes.css"
        themes.write_text(":root {\n}\n")
        consumer = root / "apps/iroha-web/src/routes/app.css"
        consumer.parent.mkdir(parents=True, exist_ok=True)
        consumer.write_text(".appbar { transition: opacity var(--motion-quick-state); }\n")

        violations = check_motion_tokens.find_violations(root)

        self.assertEqual(len(violations), 1)


if __name__ == "__main__":
    unittest.main()
