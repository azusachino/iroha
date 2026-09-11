import os
import unittest
from unittest import mock

import db


class DBScriptTest(unittest.TestCase):
    def test_usage_requires_known_action(self) -> None:
        with mock.patch.object(db.sys, "argv", ["db.py"]):
            self.assertEqual(db.main(), 2)
        with mock.patch.object(db.sys, "argv", ["db.py", "bad"]):
            self.assertEqual(db.main(), 2)

    def test_requires_database_url(self) -> None:
        with (
            mock.patch.object(db.sys, "argv", ["db.py", "apply"]),
            mock.patch.dict(os.environ, {}, clear=True),
        ):
            self.assertEqual(db.main(), 2)

    def test_apply_invokes_sqlx_run_with_database_url(self) -> None:
        with (
            mock.patch.object(db.sys, "argv", ["db.py", "apply"]),
            mock.patch.dict(os.environ, {"DATABASE_URL": "postgres://example"}, clear=True),
            mock.patch.object(db.subprocess, "call", return_value=0) as call,
        ):
            self.assertEqual(db.main(), 0)

        call.assert_called_once_with(
            [
                "sqlx",
                "migrate",
                "run",
                "--source",
                db.MIGRATIONS_DIR,
                "--no-dotenv",
                "--database-url",
                "postgres://example",
            ]
        )

    def test_rollback_accepts_iroha_database_url(self) -> None:
        with (
            mock.patch.object(db.sys, "argv", ["db.py", "rollback"]),
            mock.patch.dict(os.environ, {"IROHA_DATABASE_URL": "postgres://iroha"}, clear=True),
            mock.patch.object(db.subprocess, "call", return_value=0) as call,
        ):
            self.assertEqual(db.main(), 0)

        self.assertEqual(call.call_args.args[0][2], "revert")
        self.assertEqual(call.call_args.args[0][-1], "postgres://iroha")


if __name__ == "__main__":
    unittest.main()
