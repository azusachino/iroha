import contextlib
import io
import json
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch

import check_web_coverage as checks


class WebCoverageTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.sources = []
        for directory in checks.SOURCE_DIRS:
            path = self.root / directory / "unexecuted.svelte"
            path.parent.mkdir(parents=True)
            path.write_text("<p>Never imported</p>\n")
            self.sources.append(path)
        self.metrics = {metric: {"total": 1, "covered": 0, "pct": 0} for metric in checks.METRICS}
        self.report = {str(path): self.metrics for path in self.sources}

    def test_unexecuted_files_count_as_measured(self):
        self.assertEqual(checks.missing_sources(self.root, self.report), [])

    def test_app_only_report_fails_shared_inventory(self):
        del self.report[str(self.sources[1])]
        self.assertEqual(checks.missing_sources(self.root, self.report), [str(self.sources[1].relative_to(self.root))])

    def test_new_source_and_invalid_file_metadata_fail(self):
        new = self.root / checks.SOURCE_DIRS[1] / "new.ts"
        new.write_text("export const value = 1;\n")
        self.report[str(new)] = {}
        self.assertEqual(checks.missing_sources(self.root, self.report), [str(new.relative_to(self.root))])

    def test_test_and_non_source_files_are_not_required(self):
        directory = self.sources[1].parent
        (directory / "value.test.ts").write_text("test code\n")
        (directory / "theme.css").write_text("body {}\n")
        self.assertEqual(checks.missing_sources(self.root, self.report), [])

    def test_empty_inventory_and_invalid_report_fail_closed(self):
        with self.assertRaises(ValueError):
            checks.missing_sources(self.root / "absent", {})
        with self.assertRaises(ValueError):
            checks.missing_sources(self.root, [])

    def test_cli_missing_corrupt_then_valid_report(self):
        path = self.root / "apps/iroha-web/coverage/coverage-summary.json"
        path.parent.mkdir(parents=True)
        with patch.object(checks, "ROOT", self.root), contextlib.redirect_stderr(io.StringIO()), contextlib.redirect_stdout(io.StringIO()):
            self.assertEqual(checks.main(), 1)
            path.write_text("invalid json")
            self.assertEqual(checks.main(), 1)
            path.write_text(json.dumps(self.report))
            self.assertEqual(checks.main(), 0)


if __name__ == "__main__":
    unittest.main()
