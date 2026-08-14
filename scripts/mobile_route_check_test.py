import unittest
from unittest.mock import patch

import mobile_route_check


class MobileRouteInventoryTest(unittest.TestCase):
    def test_inventory_covers_canonical_alias_and_detail_routes(self):
        routes = mobile_route_check.route_inventory("activity-1", "sleep-1", "media-1")
        paths = [route for route, _ in routes]

        self.assertIn("/", paths)
        self.assertIn("/expenses?month=2026-08", paths)
        self.assertIn("/motion/activity-1", paths)
        self.assertIn("/activities/activity-1", paths)
        self.assertIn("/sleep/sleep-1", paths)
        self.assertIn("/media/media-1", paths)
        self.assertEqual(len(paths), len(set(paths)))

    def test_expected_url_preserves_canonical_period_contracts(self):
        self.assertEqual(
            mobile_route_check.expected_route_url("/night", "/night"),
            f"/night?year={mobile_route_check.date.today().year}",
        )
        self.assertEqual(
            mobile_route_check.expected_route_url("/expenses?month=2026-08", "/expenses"),
            "/expenses?month=2026-08",
        )
        self.assertEqual(
            mobile_route_check.expected_route_url("/motion", "/motion"),
            "/motion",
        )

    def test_parse_viewports_accepts_compact_matrix(self):
        with patch.dict(
            mobile_route_check.os.environ,
            {"VIEWPORTS": "320x844, 414x896"},
            clear=False,
        ):
            self.assertEqual(
                mobile_route_check.parse_viewports(),
                ((320, 844), (414, 896)),
            )


if __name__ == "__main__":
    unittest.main()
