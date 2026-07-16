import unittest
from pathlib import Path

import dev_stack
import dev_watch


class DevWatchTest(unittest.TestCase):
    def test_go_changes_rebuild_server_and_job(self) -> None:
        path = Path(dev_stack.ROOT) / "apps" / "iroha-server" / "main.go"
        self.assertEqual(dev_watch.changed_services({path: 1}, {path: 2}), ["job", "server"])

    def test_web_changes_rebuild_web_only(self) -> None:
        path = Path(dev_stack.ROOT) / "apps" / "iroha-web" / "src" / "app.css"
        self.assertEqual(dev_watch.changed_services({path: 1}, {path: 2}), ["web"])

    def test_compose_changes_rebuild_all_services(self) -> None:
        path = dev_stack.APP_COMPOSE_FILE
        self.assertEqual(dev_watch.changed_services({path: 1}, {path: 2}), ["job", "server", "web"])


if __name__ == "__main__":
    unittest.main()
