import tempfile
import unittest
from pathlib import Path

import check_responsive_contract


class ResponsiveContractTest(unittest.TestCase):
    def test_accepts_the_canonical_breakpoints(self):
        root = Path(tempfile.mkdtemp())
        source = root / "apps/iroha-web/src/routes/app.css"
        source.parent.mkdir(parents=True)
        source.write_text(
            "@media (max-width: 640px) {}\n"
            "@media (max-width: 768px) {}\n"
            "@media (max-width: 1024px) {}\n"
        )

        self.assertEqual(check_responsive_contract.find_violations(root), [])

    def test_rejects_a_route_local_breakpoint(self):
        root = Path(tempfile.mkdtemp())
        source = root / "packages/iroha-shared/src/Widget.svelte"
        source.parent.mkdir(parents=True)
        source.write_text("@media (max-width: 700px) {}\n")

        self.assertEqual(
            check_responsive_contract.find_violations(root),
            [
                "packages/iroha-shared/src/Widget.svelte:1: max-width 700px is not in [640, 768, 1024]"
            ],
        )


if __name__ == "__main__":
    unittest.main()
