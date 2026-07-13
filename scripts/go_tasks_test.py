import tempfile
import unittest
from pathlib import Path

import go_tasks


class DiscoverModulesTest(unittest.TestCase):
    def _write(self, text: str) -> Path:
        tmp = Path(tempfile.mkdtemp()) / "go.work"
        tmp.write_text(text)
        return tmp

    def test_parses_use_block(self):
        work = self._write(
            "go 1.26.4\n\nuse (\n\t./apps/iroha-core\n\t./apps/iroha-server\n)\n"
        )
        self.assertEqual(
            go_tasks.discover_modules(work),
            ["apps/iroha-core", "apps/iroha-server"],
        )

    def test_parses_single_line_use(self):
        work = self._write("go 1.26.4\n\nuse ./apps/iroha-core\n")
        self.assertEqual(go_tasks.discover_modules(work), ["apps/iroha-core"])

    def test_ignores_comments_and_blanks(self):
        work = self._write(
            "go 1.26.4\n\nuse (\n\t// a note\n\n\t./apps/iroha-job\n)\n"
        )
        self.assertEqual(go_tasks.discover_modules(work), ["apps/iroha-job"])

    def test_real_go_work_matches_workspace(self):
        modules = go_tasks.discover_modules(go_tasks.GO_WORK)
        self.assertIn("apps/iroha-server", modules)
        for module in modules:
            self.assertTrue(
                (go_tasks.REPO_ROOT / module / "go.mod").is_file(),
                f"{module} has no go.mod",
            )


if __name__ == "__main__":
    unittest.main()
