import unittest

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


if __name__ == "__main__":
    unittest.main()
