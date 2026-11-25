import tempfile
import unittest
from pathlib import Path

import check_theme_boundary


class ThemeBoundaryTest(unittest.TestCase):
    def _root(self) -> Path:
        root = Path(tempfile.mkdtemp())
        (root / "apps/iroha-web/src/lib").mkdir(parents=True)
        (root / "packages/iroha-shared/src").mkdir(parents=True)
        return root

    def test_current_boundary_is_clean(self):
        self.assertEqual(check_theme_boundary.find_violations(), [])

    def test_rejects_new_app_theme_asset(self):
        root = self._root()
        asset = root / "apps/iroha-web/src/lib/themes/NewTheme.svelte"
        asset.parent.mkdir(parents=True, exist_ok=True)
        asset.write_text("<div />\n")

        violations = check_theme_boundary.find_violations(root)

        self.assertIn(
            "apps/iroha-web/src/lib/themes/NewTheme.svelte: themed assets belong under packages/iroha-shared",
            violations,
        )

    def test_rejects_shared_app_import(self):
        root = self._root()
        shared_file = root / "packages/iroha-shared/src/Bad.svelte"
        shared_file.write_text('import api from "$lib/api";\n')

        violations = check_theme_boundary.find_violations(root)

        self.assertIn(
            "packages/iroha-shared/src/Bad.svelte:1: shared code cannot import app sources",
            violations,
        )

    def test_rejects_app_theme_directory(self):
        root = self._root()
        legacy = root / "apps/iroha-web/src/lib/themes/registry.ts"
        legacy.parent.mkdir(parents=True, exist_ok=True)
        legacy.write_text('export * from "@iroha/shared/theme-ui/registry";\n')

        self.assertEqual(
            check_theme_boundary.find_violations(root),
            [
                "apps/iroha-web/src/lib/themes/registry.ts: themed assets belong under packages/iroha-shared",
            ],
        )


if __name__ == "__main__":
    unittest.main()
