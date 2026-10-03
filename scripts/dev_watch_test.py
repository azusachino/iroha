import tempfile
import unittest
from unittest.mock import patch
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

    def test_workspace_and_shared_changes_rebuild_web_only(self) -> None:
        for relative in ("bun.lock", "package.json",
                         "packages/iroha-shared/src/Theme.svelte",
                         "apps/iroha-public-site/package.json"):
            path = dev_stack.ROOT / relative
            with self.subTest(path=relative):
                self.assertEqual(dev_watch.changed_services({path: 1}, {path: 2}), ["web"])
                self.assertEqual(dev_watch.changed_services({path: 1}, {}), ["web"])

    def test_snapshot_includes_workspace_but_ignores_generated_files(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            source = root / "shared/src/Theme.svelte"
            generated = root / "shared/.svelte-kit/generated.ts"
            dependency = root / "shared/node_modules/library/index.ts"
            lock = root / "bun.lock"
            for path in (source, generated, dependency, lock):
                path.parent.mkdir(parents=True, exist_ok=True)
                path.write_text("fixture")
            with patch.object(dev_watch, "WEB_ROOTS", (root / "shared",)), \
                 patch.object(dev_watch, "GO_ROOTS", ()), \
                 patch.object(dev_watch, "WORKSPACE_FILES", (lock,)):
                result = dev_watch.snapshot()
                self.assertIn(source, result)
                self.assertIn(lock, result)
                self.assertNotIn(generated, result)
                self.assertNotIn(dependency, result)

    def test_compose_changes_rebuild_all_services(self) -> None:
        path = dev_stack.APP_COMPOSE_FILE
        self.assertEqual(dev_watch.changed_services({path: 1}, {path: 2}), ["job", "server", "web"])


if __name__ == "__main__":
    unittest.main()
