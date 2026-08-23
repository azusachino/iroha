import unittest
from unittest.mock import patch

import mobile_route_check


class MobileRouteInventoryTest(unittest.TestCase):
    def test_inventory_covers_canonical_alias_and_detail_routes(self):
        routes = mobile_route_check.route_inventory("activity-1", "sleep-1", "media-1")
        paths = [route for route, _ in routes]

        self.assertIn("/", paths)
        self.assertIn(f"/?date={mobile_route_check.date.today().isoformat()}", paths)
        self.assertIn("/expenses?month=2026-08", paths)
        self.assertIn("/motion/activity-1", paths)
        self.assertIn("/activities/activity-1", paths)
        self.assertIn("/sleep/sleep-1", paths)
        self.assertIn("/media/media-1", paths)
        self.assertEqual(len(paths), len(set(paths)))

    def test_expected_url_preserves_canonical_period_contracts(self):
        self.assertEqual(
            mobile_route_check.expected_route_url("/night", "/night"),
            f"/night?date={mobile_route_check.date.today().year}",
        )
        self.assertEqual(
            mobile_route_check.expected_route_url("/sleep", "/night"),
            f"/night?date={mobile_route_check.date.today().year}",
        )
        self.assertEqual(
            mobile_route_check.expected_route_url("/motion", "/motion"),
            f"/motion?date={mobile_route_check.date.today().year}",
        )
        self.assertEqual(
            mobile_route_check.expected_route_url(
                "/metrics?metric=health.steps&month=2026-08", "/metrics"
            ),
            "/metrics?metric=health.steps&date=2026-08",
        )
        self.assertEqual(
            mobile_route_check.expected_route_url("/expenses?month=2026-08", "/expenses"),
            "/expenses?month=2026-08",
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

    def test_accessibility_failures_report_verified_compact_defects(self):
        state = {
            "skipLink": {"exists": False, "targetExists": False},
            "mainCount": 1,
            "footerInMain": False,
            "h1Count": 1,
            "firstHeading": "H2",
            "focusOrderMismatch": True,
            "smallTargetCount": 1,
            "mouseOnlyRows": 0,
        }

        self.assertEqual(
            mobile_route_check.accessibility_failures(state, (375, 844)),
            [
                "missing or invalid skip link",
                "H1 is not the single first heading",
                "compact focus order differs from visual order",
                "1 standalone controls are smaller than 24x24px",
            ],
        )

    def test_accessibility_failures_accept_conforming_desktop_state(self):
        state = {
            "skipLink": {"exists": True, "targetExists": True},
            "mainCount": 1,
            "footerInMain": False,
            "h1Count": 1,
            "firstHeading": "H1",
            "focusOrderMismatch": True,
            "smallTargetCount": 0,
            "mouseOnlyRows": 0,
        }

        self.assertEqual(
            mobile_route_check.accessibility_failures(state, (1440, 900)),
            [],
        )

    def test_accessibility_failures_reject_nested_main_and_footer(self):
        state = {
            "skipLink": {"exists": True, "targetExists": True},
            "mainCount": 2,
            "footerInMain": True,
            "h1Count": 1,
            "firstHeading": "H1",
            "focusOrderMismatch": False,
            "smallTargetCount": 0,
            "mouseOnlyRows": 1,
        }

        self.assertEqual(
            mobile_route_check.accessibility_failures(state, (375, 844)),
            [
                "expected exactly one main landmark",
                "theme footer is inside the main landmark",
                "mouse-only clickable table rows: 1",
            ],
        )

    def test_report_route_redacts_detail_identifiers(self):
        self.assertEqual(
            mobile_route_check.report_route("/motion/private-activity-id"),
            "/motion/:id",
        )
        self.assertEqual(
            mobile_route_check.report_route("/metrics?metric=health.steps"),
            "/metrics?metric=health.steps",
        )


if __name__ == "__main__":
    unittest.main()
