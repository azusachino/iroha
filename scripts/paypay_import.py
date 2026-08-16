#!/usr/bin/env python3
"""Import PayPay transaction-history CSV exports as canonical expenses.

PayPay's export ("取引履歴") has no category field and mixes real spending
with balance top-ups, points earned/expired, and PayPay's spare-change
auto-invest feature. Only rows that represent money actually leaving your
control become expenses; everything else (top-ups, points, investment,
received money, refunds) is intentionally skipped -- see
CATEGORY_KEYWORDS / INCLUDED_TRANSACTION_TYPES below for the exact rules.

Re-running against overlapping exports is safe: each row's PayPay
transaction number (取引番号) becomes the expense's stable source ref, so
the server's existing create-idempotency (source_kind + source_ref) skips
rows already imported.
"""

from __future__ import annotations

import argparse
import csv
import json
import sys
from dataclasses import dataclass
from pathlib import Path

from iroha_cli import CLIError, IrohaClient, TransportError

SOURCE_KIND = "paypay_csv"

# 取引内容 (transaction type) values that represent real spending -- money
# that left PayPay and didn't come back. Everything else in a PayPay export
# (チャージ/top-up, 受け取った金額/received, ポイント獲得/取消, 投資/auto-invest,
# 期間限定ポイントの期限切れ/point expiry, 返金/refund) is excluded on purpose.
INCLUDED_TRANSACTION_TYPES = {"支払い", "請求書払い", "送った金額"}

PERSON_TRANSFER_TYPE = "送った金額"

# First matching keyword wins; longer/more specific entries should stay
# above generic ones. Extend this as new merchants show up in later
# imports -- it is the only part of this script meant to be hand-tuned.
CATEGORY_KEYWORDS: list[tuple[str, list[str]]] = [
    (
        "groceries",
        [
            "コモディイイダ", "東武ストア", "まるとく新鮮市場", "マルエツ", "スーパーベルクス",
            "オーケー", "ライフ", "ローソン", "セブン-イレブン", "ファミリーマート",
            "ヤマザキショップ", "NewDays", "物産店",
        ],
    ),
    (
        "health",
        [
            "薬局", "マツモトキヨシ", "ダイコクドラッグ", "ココカラファイン", "整骨院",
            "PayPayほけん",
        ],
    ),
    ("utilities", ["水道局", "電力", "ガス"]),
    (
        "subscriptions",
        ["Google", "Appleサービス", "OPENAI", "CLAUDE.AI", "Netflix", "Spotify"],
    ),
    (
        "entertainment",
        [
            "HoYoverse", "PlayStation", "プレイステーション", "エンタテインメント", "ムビチケ",
            "ローソンチケット", "TOHOシネマズ", "アソビュー", "YostarGames", "まねきねこ",
            "快活CLUB",
        ],
    ),
    (
        "transport",
        ["Ｓｕｉｃａ", "Suica", "NEXCO", "SA・PA", "WILLER", "Peach Aviation", "AGODA", "Agoda"],
    ),
    (
        "shopping",
        ["Amazon", "ユニクロ", "BOOKOFF", "＊クリプトン", "キャンドゥ", "ららテラス", "ららぽーと", "オフィシャルストア"],
    ),
    (
        "food",
        [
            "料理", "食堂", "ラーメン", "麺屋", "飯店", "酒館", "菜館", "吉野家", "マクドナルド",
            "ケンタッキー", "バーガーキング", "スターバックス", "カフェ", "CoCo壱番屋", "スパイス",
            "くら寿司", "しゃぶ葉", "南国亭", "牛肉湯", "王将", "寿司", "焼肉", "肉と米",
            "ホテル", "フェスタガーデン", "リトルマーメイド", "黄燜鶏", "XI'AN", "味仙", "あじそう",
        ],
    ),
]


@dataclass(frozen=True)
class ExpenseDraft:
    occurred_on: str
    amount_minor: int
    category: str
    merchant: str
    note: str
    source_ref: str


def categorize_merchant(merchant: str) -> str:
    for category, keywords in CATEGORY_KEYWORDS:
        if any(keyword in merchant for keyword in keywords):
            return category
    return "other"


def parse_amount(withdrawal_yen: str) -> int | None:
    value = withdrawal_yen.strip().replace(",", "")
    if not value or value == "-":
        return None
    try:
        amount = int(value)
    except ValueError:
        return None
    return amount if amount > 0 else None


def row_to_draft(row: dict[str, str]) -> ExpenseDraft | None:
    """Pure, DB- and network-free: decides whether a CSV row becomes an
    expense and what its fields are. Returns None to skip the row."""
    transaction_type = row.get("取引内容", "").strip()
    if transaction_type not in INCLUDED_TRANSACTION_TYPES:
        return None
    amount = parse_amount(row.get("出金金額（円）", ""))
    if amount is None:
        return None
    date_part = row.get("取引日", "").strip().split(" ")[0].replace("/", "-")
    txn_id = row.get("取引番号", "").strip()
    if not date_part or not txn_id:
        return None
    merchant = row.get("取引先", "").strip()
    method = row.get("取引方法", "").strip()
    category = "other" if transaction_type == PERSON_TRANSFER_TYPE else categorize_merchant(merchant)
    note = f"PayPay {transaction_type}" + (f" via {method}" if method and method != "-" else "")
    return ExpenseDraft(
        occurred_on=date_part,
        amount_minor=amount,
        category=category,
        merchant=merchant,
        note=note,
        source_ref=txn_id,
    )


def read_drafts(path: Path) -> tuple[list[ExpenseDraft], dict[str, int]]:
    drafts: list[ExpenseDraft] = []
    skipped_by_type: dict[str, int] = {}
    with path.open(encoding="utf-8-sig", newline="") as file:
        for row in csv.DictReader(file):
            draft = row_to_draft(row)
            if draft is not None:
                drafts.append(draft)
                continue
            transaction_type = row.get("取引内容", "").strip() or "(unrecognized row)"
            skipped_by_type[transaction_type] = skipped_by_type.get(transaction_type, 0) + 1
    return drafts, skipped_by_type


def draft_body(draft: ExpenseDraft) -> bytes:
    return json.dumps(
        {
            "occurred_on": draft.occurred_on,
            "currency": "JPY",
            "amount_minor": draft.amount_minor,
            "category": draft.category,
            "merchant": draft.merchant,
            "note": draft.note,
            "items": [],
            "source": {"kind": SOURCE_KIND, "ref": draft.source_ref},
        },
        ensure_ascii=False,
    ).encode()


def import_drafts(client: IrohaClient, drafts: list[ExpenseDraft]) -> tuple[int, int, int]:
    """Returns (created, already_imported, failed)."""
    created = already_imported = failed = 0
    for draft in drafts:
        try:
            status, _ = client.request_with_status(
                "POST", "/api/v1/expenses", draft_body(draft)
            )
        except (CLIError, TransportError) as error:
            failed += 1
            print(
                f"  FAILED {draft.occurred_on} {draft.merchant!r} ({draft.source_ref}): {error}",
                file=sys.stderr,
            )
            continue
        # handleCreateExpense returns 201 for a freshly inserted row and 200
        # for an idempotent retry against an already-imported source ref.
        if status == 201:
            created += 1
        else:
            already_imported += 1
    return created, already_imported, failed


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("csv_files", nargs="+", type=Path, help="PayPay transaction CSV export(s)")
    parser.add_argument("--api-base", default=None, help="Iroha API base URL (defaults to IROHA_API_BASE)")
    parser.add_argument(
        "--dry-run", action="store_true", help="parse and categorize only; do not call the API"
    )
    args = parser.parse_args(argv)

    all_drafts: list[ExpenseDraft] = []
    all_skipped: dict[str, int] = {}
    for csv_path in args.csv_files:
        try:
            drafts, skipped = read_drafts(csv_path)
        except OSError as error:
            print(f"iroha-paypay-import: cannot read {csv_path}: {error}", file=sys.stderr)
            return 1
        all_drafts.extend(drafts)
        for key, count in skipped.items():
            all_skipped[key] = all_skipped.get(key, 0) + count

    by_category: dict[str, int] = {}
    for draft in all_drafts:
        by_category[draft.category] = by_category.get(draft.category, 0) + 1

    print(f"{len(all_drafts)} rows to import across {len(args.csv_files)} file(s):")
    for category, count in sorted(by_category.items(), key=lambda item: -item[1]):
        print(f"  {category:<15} {count}")
    if all_skipped:
        print("skipped (not spending, or unparseable):")
        for transaction_type, count in sorted(all_skipped.items(), key=lambda item: -item[1]):
            print(f"  {transaction_type:<15} {count}")

    if args.dry_run:
        print("dry run -- nothing was sent")
        return 0

    client = IrohaClient(args.api_base)
    created, already_imported, failed = import_drafts(client, all_drafts)
    print(f"created={created} already_imported={already_imported} failed={failed}")
    return 1 if failed else 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except CLIError as error:
        print(f"iroha-paypay-import: {error}", file=sys.stderr)
        raise SystemExit(1) from error
