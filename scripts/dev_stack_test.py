import io
import os
import tempfile
import unittest
from pathlib import Path
from unittest import mock

import dev_stack


def _make_fake_bin(directory: Path, *names: str) -> dict[str, Path]:
    created: dict[str, Path] = {}
    for name in names:
        exe = directory / name
        exe.write_text("#!/bin/sh\nexit 0\n")
        exe.chmod(0o755)
        created[name] = exe
    return created


class DevStackScriptTest(unittest.TestCase):
    def test_container_cmd_prefers_configured_podman_path(self) -> None:
        with mock.patch.dict(os.environ, {"PODMAN": "/tmp/custom-podman"}, clear=True):
            self.assertEqual(dev_stack.container_cmd(), ["/tmp/custom-podman"])

    def test_container_cmd_override_takes_precedence_over_path_and_podman(self) -> None:
        with tempfile.TemporaryDirectory() as tmpdir:
            _make_fake_bin(Path(tmpdir), "podman", "podman-compose")
            with mock.patch.dict(
                os.environ,
                {
                    "PATH": tmpdir,
                    "PODMAN": "/tmp/custom-podman",
                    "IROHA_CONTAINER": "colima nerdctl",
                },
                clear=True,
            ):
                self.assertEqual(dev_stack.container_cmd(), ["colima", "nerdctl"])
                self.assertEqual(
                    dev_stack.compose_cmd(),
                    [
                        "colima",
                        "nerdctl",
                        "compose",
                        "-f",
                        str(dev_stack.COMPOSE_FILE),
                        "-f",
                        str(dev_stack.APP_COMPOSE_FILE),
                        "-p",
                        "iroha-dev",
                    ],
                )

    def test_container_cmd_resolves_podman_from_path(self) -> None:
        with tempfile.TemporaryDirectory() as tmpdir:
            bins = _make_fake_bin(Path(tmpdir), "podman", "podman-compose")
            with mock.patch.dict(os.environ, {"PATH": tmpdir}, clear=True):
                self.assertEqual(dev_stack.container_cmd(), [str(bins["podman"])])
                self.assertEqual(
                    dev_stack.compose_cmd(),
                    [
                        str(bins["podman-compose"]),
                        "-f",
                        str(dev_stack.COMPOSE_FILE),
                        "-f",
                        str(dev_stack.APP_COMPOSE_FILE),
                        "-p",
                        "iroha-dev",
                    ],
                )

    def test_container_cmd_resolves_colima_nerdctl_when_podman_absent(self) -> None:
        with tempfile.TemporaryDirectory() as tmpdir:
            bins = _make_fake_bin(Path(tmpdir), "colima")
            with mock.patch.dict(os.environ, {"PATH": tmpdir}, clear=True):
                self.assertEqual(dev_stack.container_cmd(), [str(bins["colima"]), "nerdctl"])
                self.assertEqual(
                    dev_stack.compose_cmd(),
                    [
                        str(bins["colima"]),
                        "nerdctl",
                        "compose",
                        "-f",
                        str(dev_stack.COMPOSE_FILE),
                        "-f",
                        str(dev_stack.APP_COMPOSE_FILE),
                        "-p",
                        "iroha-dev",
                    ],
                )

    def test_container_cmd_prefers_podman_when_both_on_path(self) -> None:
        with tempfile.TemporaryDirectory() as tmpdir:
            bins = _make_fake_bin(Path(tmpdir), "podman", "colima")
            with mock.patch.dict(os.environ, {"PATH": tmpdir}, clear=True):
                self.assertEqual(dev_stack.container_cmd(), [str(bins["podman"])])

    def test_container_cmd_exits_when_missing_with_clear_single_line_error(self) -> None:
        stderr = io.StringIO()
        with tempfile.TemporaryDirectory() as tmpdir:
            with (
                mock.patch.dict(os.environ, {"PATH": tmpdir}, clear=True),
                mock.patch.object(dev_stack.sys, "stderr", stderr),
                self.assertRaises(SystemExit) as raised,
            ):
                dev_stack.container_cmd()
        self.assertEqual(raised.exception.code, 2)
        message = stderr.getvalue().strip()
        self.assertNotIn("\n", message)
        self.assertIn("podman", message)
        self.assertIn("colima", message)
        self.assertIn("IROHA_CONTAINER", message)

    def test_compose_cmd_uses_both_compose_files(self) -> None:
        with (
            mock.patch.dict(os.environ, {"PODMAN": "podman", "PODMAN_COMPOSE": "podman-compose"}, clear=True),
        ):
            self.assertEqual(
                dev_stack.compose_cmd(),
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
            mock.patch.object(dev_stack, "container_cmd", return_value=["podman"]),
        ):
            dev_stack.ensure_machine()

        self.assertEqual(run.call_args.args[0], ["podman", "machine", "start"])

    def test_ensure_machine_starts_colima_when_resolved_to_colima(self) -> None:
        stopped = mock.Mock(returncode=1, stderr="not running")
        started = mock.Mock(returncode=0)
        with (
            mock.patch.object(dev_stack.subprocess, "run", side_effect=[stopped, started]) as run,
            mock.patch.object(dev_stack, "container_cmd", return_value=["/opt/homebrew/bin/colima", "nerdctl"]),
        ):
            dev_stack.ensure_machine()

        self.assertEqual(run.call_args.args[0], ["/opt/homebrew/bin/colima", "start"])

    def test_main_container_cmd_prints_resolved_prefix(self) -> None:
        stdout = io.StringIO()
        with (
            mock.patch.object(dev_stack.sys, "argv", ["dev_stack.py", "container-cmd"]),
            mock.patch.object(dev_stack, "container_cmd", return_value=["/opt/homebrew/bin/colima", "nerdctl"]),
            mock.patch.object(dev_stack.sys, "stdout", stdout),
        ):
            self.assertEqual(dev_stack.main(), 0)
        self.assertEqual(stdout.getvalue().strip(), "/opt/homebrew/bin/colima nerdctl")

    def test_migrate_sets_database_url_for_db_script(self) -> None:
        captured_env: dict[str, str] = {}

        def fake_run(cmd: list[str], env: dict[str, str] | None = None) -> int:
            self.assertEqual(cmd, [dev_stack.sys.executable, "scripts/db.py", "apply"])
            self.assertIsNotNone(env)
            captured_env.update(env or {})
            return 0

        with mock.patch.object(dev_stack, "run", side_effect=fake_run):
            self.assertEqual(dev_stack.migrate(), 0)

        self.assertEqual(captured_env["DATABASE_URL"], dev_stack.DATABASE_URL)

    def test_recreate_test_db_drops_creates_then_migrates_the_test_database(self) -> None:
        calls: list[tuple[list[str], dict[str, str] | None]] = []

        def fake_run(cmd: list[str], env: dict[str, str] | None = None) -> int:
            calls.append((cmd, env))
            return 0

        with (
            mock.patch.object(dev_stack, "run", side_effect=fake_run),
            mock.patch.object(dev_stack, "compose_cmd", return_value=["compose"]),
        ):
            self.assertEqual(dev_stack.recreate_test_db(), 0)

        statements = [cmd[cmd.index("-c") + 1] for cmd, _ in calls[:2]]
        self.assertEqual(
            statements,
            [
                f"drop database if exists {dev_stack.TEST_DB_NAME} with (force)",
                f"create database {dev_stack.TEST_DB_NAME} owner {dev_stack.DB_USER}",
            ],
        )
        # Neither statement may connect to, or target, the dev database itself.
        for cmd, _ in calls[:2]:
            self.assertNotIn(f"drop database if exists {dev_stack.DB_NAME} ", " ".join(cmd) + " ")
        migrate_cmd, migrate_env = calls[2]
        self.assertEqual(migrate_cmd, [dev_stack.sys.executable, "scripts/db.py", "apply"])
        self.assertEqual((migrate_env or {})["DATABASE_URL"], dev_stack.TEST_DATABASE_URL)
        self.assertIn(f"/{dev_stack.TEST_DB_NAME}?", dev_stack.TEST_DATABASE_URL)

    def test_recreate_test_db_stops_when_a_statement_fails(self) -> None:
        with (
            mock.patch.object(dev_stack, "run", return_value=1) as run,
            mock.patch.object(dev_stack, "compose_cmd", return_value=["compose"]),
        ):
            self.assertEqual(dev_stack.recreate_test_db(), 1)
        self.assertEqual(run.call_count, 1)

    def test_wait_for_db_returns_success_after_ready_probe(self) -> None:
        result = mock.Mock(returncode=0, stdout="ready\n")
        with (
            mock.patch.object(dev_stack.subprocess, "run", return_value=result) as run,
            mock.patch.object(dev_stack, "compose_cmd", return_value=["compose"]),
        ):
            self.assertEqual(dev_stack.wait_for_db(timeout_s=1), 0)

        cmd = run.call_args.args[0]
        self.assertEqual(cmd[:5], ["compose", "exec", "-T", "db", "pg_isready"])

    def test_main_wait_delegates_to_wait_for_db(self) -> None:
        with (
            mock.patch.object(dev_stack.sys, "argv", ["dev_stack.py", "wait"]),
            mock.patch.object(dev_stack, "wait_for_db", return_value=0) as wait_for_db,
        ):
            self.assertEqual(dev_stack.main(), 0)
        wait_for_db.assert_called_once_with()

    def test_start_database_waits_after_bringing_up_db(self) -> None:
        with (
            mock.patch.object(dev_stack, "compose_cmd", return_value=["compose"]),
            mock.patch.object(dev_stack, "check_call") as check_call,
            mock.patch.object(dev_stack, "wait_for_db", return_value=0) as wait_for_db,
        ):
            dev_stack.start_database()

        check_call.assert_called_once_with(["compose", "up", "-d", "db"])
        wait_for_db.assert_called_once_with()

    def test_start_app_only_starts_application_services(self) -> None:
        with (
            mock.patch.object(dev_stack, "compose_cmd", return_value=["compose"]),
            mock.patch.object(dev_stack, "run", return_value=0) as run,
        ):
            self.assertEqual(dev_stack.start_app(), 0)

        run.assert_called_once_with(["compose", "up", "--build", "-d", "server", "job", "web"])

    def test_compose_files_stay_repo_local(self) -> None:
        self.assertEqual(dev_stack.COMPOSE_FILE, Path(dev_stack.ROOT) / "ops" / "local-dev" / "compose.yaml")
        self.assertEqual(dev_stack.APP_COMPOSE_FILE, Path(dev_stack.ROOT) / "ops" / "local-dev" / "compose.app.yaml")


if __name__ == "__main__":
    unittest.main()
