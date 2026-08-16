import json
import os
import tempfile
import unittest
from pathlib import Path

import paypay_import
from iroha_cli import IrohaClient


class FakeResponse:
    def __init__(self, body: bytes, status: int = 200) -> None:
        self.content = body
        self.status_code = status


class FakeSession:
    def __init__(self, responses: list[FakeResponse]) -> None:
        self.responses = list(responses)
        self.calls = []

    def request(self, method, url, **kwargs):
        self.calls.append((method, url, kwargs))
        return self.responses.pop(0)


def _row(overrides: dict[str, str] | None = None) -> dict[str, str]:
    row = {
        "取引日": "2026/07/31 12:04:48",
        "出金金額（円）": "650",
        "入金金額（円）": "-",
        "取引内容": "支払い",
        "取引先": "吉野家 - 虎ノ門店",
        "取引方法": "クレジット VISA 7240",
        "取引番号": "05036463223572652041",
    }
    row.update(overrides or {})
    return row


class RowToDraftTest(unittest.TestCase):
    def test_payment_row_becomes_a_draft(self) -> None:
        draft = paypay_import.row_to_draft(_row())
        assert draft is not None
        self.assertEqual(draft.occurred_on, "2026-07-31")
        self.assertEqual(draft.amount_minor, 650)
        self.assertEqual(draft.category, "food")
        self.assertEqual(draft.merchant, "吉野家 - 虎ノ門店")
        self.assertEqual(draft.source_ref, "05036463223572652041")
        self.assertIn("支払い", draft.note)
        self.assertIn("クレジット VISA 7240", draft.note)

    def test_charge_top_up_is_excluded(self) -> None:
        row = _row({"取引内容": "チャージ", "出金金額（円）": "-", "入金金額（円）": "10,000"})
        self.assertIsNone(paypay_import.row_to_draft(row))

    def test_received_money_is_excluded(self) -> None:
        row = _row({"取引内容": "受け取った金額", "出金金額（円）": "-", "入金金額（円）": "5,000", "取引先": "Yuu"})
        self.assertIsNone(paypay_import.row_to_draft(row))

    def test_point_investment_is_excluded(self) -> None:
        row = _row({"取引内容": "投資", "取引先": "PayPayポイント運用"})
        self.assertIsNone(paypay_import.row_to_draft(row))

    def test_expired_points_are_excluded(self) -> None:
        row = _row({"取引内容": "期間限定ポイントの期限切れ"})
        self.assertIsNone(paypay_import.row_to_draft(row))

    def test_person_transfer_is_included_as_other(self) -> None:
        row = _row({"取引内容": "送った金額", "取引先": "Yuu"})
        draft = paypay_import.row_to_draft(row)
        assert draft is not None
        self.assertEqual(draft.category, "other")
        self.assertEqual(draft.merchant, "Yuu")

    def test_bill_payment_is_included(self) -> None:
        row = _row({"取引内容": "請求書払い", "取引先": "川口市上下水道局（水道料金、下水道使用料）"})
        draft = paypay_import.row_to_draft(row)
        assert draft is not None
        self.assertEqual(draft.category, "utilities")

    def test_zero_amount_is_excluded(self) -> None:
        row = _row({"出金金額（円）": "0"})
        self.assertIsNone(paypay_import.row_to_draft(row))

    def test_amount_with_thousands_separator_parses(self) -> None:
        row = _row({"出金金額（円）": "5,000"})
        draft = paypay_import.row_to_draft(row)
        assert draft is not None
        self.assertEqual(draft.amount_minor, 5000)


class CategorizeMerchantTest(unittest.TestCase):
    def test_convenience_store_is_groceries(self) -> None:
        self.assertEqual(paypay_import.categorize_merchant("セブン-イレブン - 港区虎ノ門駅南"), "groceries")

    def test_subscription_service_is_subscriptions(self) -> None:
        self.assertEqual(paypay_import.categorize_merchant("Appleサービス"), "subscriptions")

    def test_unrecognized_merchant_falls_back_to_other(self) -> None:
        self.assertEqual(paypay_import.categorize_merchant("謎の店舗"), "other")


class ReadDraftsTest(unittest.TestCase):
    def test_reads_utf8_bom_csv_and_reports_skips(self) -> None:
        header = "取引日,出金金額（円）,入金金額（円）,海外出金金額,通貨,変換レート（円）,利用国,取引内容,取引先,取引方法,支払い区分,利用者,取引番号\n"
        rows = (
            "2026/07/31 13:08:37,-,\"5,000\",-,-,-,JP,受け取った金額,Yuu,PayPay残高,-,-,111\n"
            "2026/07/31 12:04:48,650,-,-,-,-,-,支払い,吉野家 - 虎ノ門店,クレジット VISA 7240,-,-,222\n"
        )
        with tempfile.NamedTemporaryFile(
            "w", suffix=".csv", delete=False, encoding="utf-8-sig"
        ) as file:
            file.write(header + rows)
            path = Path(file.name)
        try:
            drafts, skipped = paypay_import.read_drafts(path)
        finally:
            os.unlink(path)
        self.assertEqual(len(drafts), 1)
        self.assertEqual(drafts[0].source_ref, "222")
        self.assertEqual(skipped, {"受け取った金額": 1})


class ImportDraftsTest(unittest.TestCase):
    def test_created_vs_already_imported_is_read_from_status_code(self) -> None:
        session = FakeSession([FakeResponse(b'{"id":"a"}', 201), FakeResponse(b'{"id":"a"}', 200)])
        client = IrohaClient("http://iroha.test", session=session)
        drafts = [
            paypay_import.ExpenseDraft("2026-07-31", 650, "food", "m1", "note", "1"),
            paypay_import.ExpenseDraft("2026-07-30", 500, "food", "m2", "note", "2"),
        ]
        created, already_imported, failed = paypay_import.import_drafts(client, drafts)
        self.assertEqual((created, already_imported, failed), (1, 1, 0))

    def test_failed_request_is_counted_and_does_not_raise(self) -> None:
        session = FakeSession([FakeResponse(b'{"code":"bad"}', 400)])
        client = IrohaClient("http://iroha.test", session=session)
        drafts = [paypay_import.ExpenseDraft("2026-07-31", 650, "food", "m1", "note", "1")]
        created, already_imported, failed = paypay_import.import_drafts(client, drafts)
        self.assertEqual((created, already_imported, failed), (0, 0, 1))

    def test_draft_body_encodes_source_identity(self) -> None:
        draft = paypay_import.ExpenseDraft("2026-07-31", 650, "food", "吉野家", "note", "ref-1")
        body = json.loads(paypay_import.draft_body(draft))
        self.assertEqual(body["source"], {"kind": "paypay_csv", "ref": "ref-1"})
        self.assertEqual(body["currency"], "JPY")
        self.assertEqual(body["amount_minor"], 650)


if __name__ == "__main__":
    unittest.main()
