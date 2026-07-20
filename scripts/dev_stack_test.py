import os
import unittest
from pathlib import Path
from unittest import mock

import dev_stack


class DevStackScriptTest(unittest.TestCase):
    def test_podman_bin_prefers_configured_path(self) -> None:
        with mock.patch.dict(os.environ, {"PODMAN": "/tmp/custom-podman"}):
            self.assertEqual(dev_stack.podman_bin(), "/tmp/custom-podman")

    def test_compose_bin_uses_path_lookup(self) -> None:
        with (
            mock.patch.dict(os.environ, {}, clear=True),
            mock.patch.object(dev_stack.shutil, "which", return_value="/usr/local/bin/podman-compose"),
        ):
            self.assertEqual(dev_stack.compose_bin(), "/usr/local/bin/podman-compose")

    def test_podman_cmd_exits_when_missing(self) -> None:
        with mock.patch.object(dev_stack, "podman_bin", return_value=""):
            with self.assertRaises(SystemExit) as raised:
                dev_stack.podman_cmd()
        self.assertEqual(raised.exception.code, 2)

    def test_podman_cmd_uses_both_compose_files(self) -> None:
        with (
            mock.patch.object(dev_stack, "podman_bin", return_value="podman"),
            mock.patch.object(dev_stack, "compose_bin", return_value="podman-compose"),
        ):
            self.assertEqual(
                dev_stack.podman_cmd(),
                [
                    "podman-compose",
                    "-f",
                    str(dev_stack.COMPOSE_FILE),
                    "-f",
                    str(dev_stack.APP_COMPOSE_FILE),
                    "-p",
                    "iroha-dev",
                ],
            )

    def test_ensure_machine_starts_existing_machine(self) -> None:
        stopped = mock.Mock(returncode=1, stderr="not running")
        started = mock.Mock(returncode=0)
        with (
            mock.patch.object(dev_stack.subprocess, "run", side_effect=[stopped, started]) as run,
            mock.patch.object(dev_stack, "podman_bin", return_value="podman"),
        ):
            dev_stack.ensure_machine()

        self.assertEqual(run.call_args.args[0], ["podman", "machine", "start"])

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

    def test_start_database_waits_after_bringing_up_db(self) -> None:
        with (
            mock.patch.object(dev_stack, "podman_cmd", return_value=["compose"]),
            mock.patch.object(dev_stack, "check_call") as check_call,
            mock.patch.object(dev_stack, "wait_for_db", return_value=0) as wait_for_db,
        ):
            dev_stack.start_database()

        check_call.assert_called_once_with(["compose", "up", "-d", "db"])
        wait_for_db.assert_called_once_with()

    def test_start_app_only_starts_application_services(self) -> None:
        with (
            mock.patch.object(dev_stack, "podman_cmd", return_value=["compose"]),
            mock.patch.object(dev_stack, "run", return_value=0) as run,
        ):
            self.assertEqual(dev_stack.start_app(), 0)

        run.assert_called_once_with(["compose", "up", "--build", "-d", "server", "job", "web"])

    def test_compose_files_stay_repo_local(self) -> None:
        self.assertEqual(dev_stack.COMPOSE_FILE, Path(dev_stack.ROOT) / "ops" / "local-dev" / "compose.yaml")
        self.assertEqual(dev_stack.APP_COMPOSE_FILE, Path(dev_stack.ROOT) / "ops" / "local-dev" / "compose.app.yaml")


if __name__ == "__main__":
    unittest.main()
