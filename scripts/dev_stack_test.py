import os
import unittest
from pathlib import Path
from unittest import mock

import dev_stack


class DevStackScriptTest(unittest.TestCase):
    def test_bianpai_bin_prefers_configured_path(self) -> None:
        with mock.patch.dict(os.environ, {"BIANPAI": "/tmp/custom-bianpai"}):
            self.assertEqual(dev_stack.bianpai_bin(), "/tmp/custom-bianpai")

    def test_bianpai_bin_uses_path_lookup(self) -> None:
        with (
            mock.patch.dict(os.environ, {}, clear=True),
            mock.patch.object(dev_stack.shutil, "which", return_value="/usr/local/bin/bianpai"),
        ):
            self.assertEqual(dev_stack.bianpai_bin(), "/usr/local/bin/bianpai")

    def test_bianpai_cmd_exits_when_missing(self) -> None:
        with mock.patch.object(dev_stack, "bianpai_bin", return_value=""):
            with self.assertRaises(SystemExit) as raised:
                dev_stack.bianpai_cmd()
        self.assertEqual(raised.exception.code, 2)

    def test_migrate_sets_database_url_for_db_script(self) -> None:
        captured_env: dict[str, str] = {}

        def fake_run(cmd: list[str], env: dict[str, str] | None = None) -> int:
            self.assertEqual(cmd, ["uv", "run", "python", "scripts/db.py", "apply"])
            self.assertIsNotNone(env)
            captured_env.update(env or {})
            return 0

        with mock.patch.object(dev_stack, "run", side_effect=fake_run):
            self.assertEqual(dev_stack.migrate(), 0)

        self.assertEqual(captured_env["DATABASE_URL"], dev_stack.DATABASE_URL)

    def test_wait_for_db_returns_success_after_ready_probe(self) -> None:
        result = mock.Mock(returncode=0, stdout="ready\n")
        with mock.patch.object(dev_stack.subprocess, "run", return_value=result) as run:
            self.assertEqual(dev_stack.wait_for_db(timeout_s=1), 0)

        cmd = run.call_args.args[0]
        self.assertEqual(cmd[:2], ["pg_isready", "-h"])

    def test_main_wait_delegates_to_wait_for_db(self) -> None:
        with (
            mock.patch.object(dev_stack.sys, "argv", ["dev_stack.py", "wait"]),
            mock.patch.object(dev_stack, "wait_for_db", return_value=0) as wait_for_db,
        ):
            self.assertEqual(dev_stack.main(), 0)
        wait_for_db.assert_called_once_with()

    def test_compose_file_stays_repo_local(self) -> None:
        self.assertEqual(dev_stack.COMPOSE_FILE, Path(dev_stack.ROOT) / "ops" / "local-dev" / "compose.yaml")


if __name__ == "__main__":
    unittest.main()
