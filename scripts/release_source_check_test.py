import contextlib
import io
import json
import pathlib
import subprocess
import tempfile
import unittest
from unittest.mock import patch

from release_source_check import REVISION_LABEL, git, main, verify_image, verify_source


class ReleaseSourceCheckTests(unittest.TestCase):
    def setUp(self):
        self.directory = tempfile.TemporaryDirectory()
        self.addCleanup(self.directory.cleanup)
        self.root = pathlib.Path(self.directory.name)
        git(self.root, "init", "--quiet")
        git(self.root, "config", "user.name", "Synthetic fixture")
        git(self.root, "config", "user.email", "fixture@example.invalid")
        (self.root / "source.txt").write_text("synthetic source\n")
        git(self.root, "add", "source.txt")
        git(self.root, "commit", "--quiet", "-m", "synthetic")
        git(self.root, "tag", "-a", "candidate", "-m", "synthetic annotated candidate")

    def test_clean_annotated_identity_and_frozen_object(self):
        identity = verify_source(self.root, "candidate")
        self.assertEqual(identity["revision"], git(self.root, "rev-parse", "HEAD"))
        self.assertEqual(verify_source(self.root, "candidate", identity["revision"], identity["tag_object"]), identity)

    def test_dirty_tracked_source_fails(self):
        (self.root / "source.txt").write_text("changed")
        with self.assertRaisesRegex(ValueError, "dirty"):
            verify_source(self.root, "candidate")

    def test_untracked_source_fails(self):
        (self.root / "extra.txt").write_text("untracked")
        with self.assertRaisesRegex(ValueError, "dirty"):
            verify_source(self.root, "candidate")

    def test_lightweight_tag_fails(self):
        git(self.root, "tag", "lightweight")
        with self.assertRaisesRegex(ValueError, "annotated"):
            verify_source(self.root, "lightweight")

    def test_tag_not_at_head_fails(self):
        (self.root / "source.txt").write_text("next")
        git(self.root, "commit", "--quiet", "-am", "next")
        with self.assertRaisesRegex(ValueError, "HEAD"):
            verify_source(self.root, "candidate")

    def test_replaced_annotation_fails_frozen_object(self):
        identity = verify_source(self.root, "candidate")
        git(self.root, "tag", "-f", "-a", "candidate", "-m", "changed annotation")
        with self.assertRaisesRegex(ValueError, "object changed"):
            verify_source(self.root, "candidate", identity["revision"], identity["tag_object"])

    def test_wrong_expected_revision_fails(self):
        with self.assertRaisesRegex(ValueError, "revision changed"):
            verify_source(self.root, "candidate", "0" * 40)

    def test_missing_and_wrong_oci_revision_fail(self):
        for labels in [{}, {REVISION_LABEL: "wrong"}]:
            with self.subTest(labels=labels), patch("release_source_check.subprocess.check_output", return_value=json.dumps([{"Config": {"Labels": labels}}])):
                with self.assertRaisesRegex(ValueError, "OCI revision"):
                    verify_image("synthetic", "source")

    def test_matching_oci_revision(self):
        with patch("release_source_check.subprocess.check_output", return_value=json.dumps([{"Config": {"Labels": {REVISION_LABEL: "source"}}}])) as inspect:
            verify_image("synthetic", "source")
            inspect.assert_called_once_with(["podman", "image", "inspect", "synthetic"], text=True)

    def test_cli_reports_missing_tool_and_malformed_inspection(self):
        for failure in [FileNotFoundError("podman missing"), "[3]"]:
            with self.subTest(failure=failure), patch("sys.argv", ["release_source_check", "--tag", "candidate", "--image", "synthetic"]), patch("release_source_check.verify_source", return_value={"revision": "source"}), patch("release_source_check.subprocess.check_output", **({"side_effect": failure} if isinstance(failure, Exception) else {"return_value": failure})):
                output = io.StringIO()
                with contextlib.redirect_stderr(output), self.assertRaises(SystemExit) as stopped:
                    main()
                self.assertEqual(stopped.exception.code, 1)
                self.assertIn("release identity check failed", output.getvalue())
                self.assertNotIn("Traceback", output.getvalue())

    def test_missing_tag_fails(self):
        with self.assertRaises(subprocess.CalledProcessError):
            verify_source(self.root, "missing")
