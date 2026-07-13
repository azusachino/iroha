#!/usr/bin/env python3
"""Exploratory dump of a Bangumi.tv user's collection (throwaway, pre-schema).

Pages through /v0/users/{username}/collections and prints subject-type and
collection-type distributions, name_cn coverage (the Chinese-title source that
complements AniList's romaji/english), ep/vol progress ranges, rate coverage,
and the updated_at span. The point is to decide (a) whether v1 covers non-anime
Bangumi subject types (books/manga/games/music), and (b) the observations.Media
field mapping before the Go connector (iroha:media-sync task-6) is written.

Public collections need no auth; pass --token (a Bangumi personal access token)
only for a private collection. A descriptive User-Agent is sent as good practice
(reads work without one). Nothing here is canonical.

Usage:
  uv run python scripts/bangumi_explore.py <username> [--token TOKEN]
"""

from __future__ import annotations

import argparse
import json
import urllib.request
from collections import Counter

API = "https://api.bgm.tv/v0/users/{user}/collections"
UA = "iroha/0.1 (+https://github.com/azusachino/iroha)"

SUBJECT_TYPE = {1: "book", 2: "anime", 3: "music", 4: "game", 6: "real"}
# Bangumi collection type (user's state on the subject).
COLLECTION_TYPE = {1: "wish", 2: "collect", 3: "doing", 4: "on_hold", 5: "dropped"}


def get(user: str, limit: int, offset: int, token: str | None) -> dict:
    url = API.format(user=user) + f"?limit={limit}&offset={offset}"
    headers = {"Accept": "application/json", "User-Agent": UA}
    if token:
        headers["Authorization"] = f"Bearer {token}"
    req = urllib.request.Request(url, headers=headers, method="GET")
    with urllib.request.urlopen(req, timeout=30) as resp:
        return json.loads(resp.read())


def main() -> None:
    ap = argparse.ArgumentParser()
    ap.add_argument("username")
    ap.add_argument("--token", default=None, help="Bangumi PAT (private collections only)")
    args = ap.parse_args()

    rows: list[dict] = []
    offset, limit = 0, 50
    total = None
    while True:
        page = get(args.username, limit, offset, args.token)
        if total is None:
            total = page.get("total", 0)
            print(f"user={args.username}  total collections={total}")
        data = page.get("data") or []
        rows.extend(data)
        offset += limit
        if offset >= total or not data:
            break

    n = len(rows)
    subj = Counter(SUBJECT_TYPE.get(r.get("subject_type"), r.get("subject_type")) for r in rows)
    coll = Counter(COLLECTION_TYPE.get(r.get("type"), r.get("type")) for r in rows)
    with_cn = sum(1 for r in rows if (r.get("subject") or {}).get("name_cn"))
    with_rate = sum(1 for r in rows if (r.get("rate") or 0) > 0)
    with_ep = sum(1 for r in rows if (r.get("ep_status") or 0) > 0)
    with_vol = sum(1 for r in rows if (r.get("vol_status") or 0) > 0)
    private = sum(1 for r in rows if r.get("private"))
    updated = sorted(r["updated_at"] for r in rows if r.get("updated_at"))
    pct = lambda x: f"{(100 * x / n):.0f}%" if n else "-"

    print(f"\nfetched {n} rows")
    print("  subject types   :", dict(subj), " <- decides non-anime v1 scope")
    print("  collection types:", dict(coll))
    print(f"  name_cn cover   : {with_cn}/{n} ({pct(with_cn)})  <- zh title source")
    print(f"  rate>0          : {with_rate}/{n} ({pct(with_rate)})")
    print(f"  ep_status>0     : {with_ep}/{n}   vol_status>0: {with_vol}")
    print(f"  private         : {private}")
    if updated:
        print(f"  updated_at span : {updated[0]} .. {updated[-1]}")

    anime = subj.get("anime", 0)
    print(f"\nScope hint: anime={anime}, non-anime={n - anime}. If non-anime is large,"
          " v1 should cover book/manga too (map subject_type 1->manga/light_novel via"
          " platform, 2->anime). collection type -> progress status; ep_status/vol_status"
          " -> progress position; name+name_cn -> original + zh-Hans tb_media_titles rows.")


if __name__ == "__main__":
    main()
