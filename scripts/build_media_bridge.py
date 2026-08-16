#!/usr/bin/env python3
"""Refresh the two-hop Bangumi->MAL->AniList media bridge in Postgres.

Upserts tb_media_ref_bridge (migration 00012), which
apps/iroha-imports/media_resolution.go's LoadTwoHopMediaRefBridgeFromDB loads
into an in-memory map at iroha-job startup, same shape as before this moved
out of ConfigMap-mounted JSON files.

Sources (community-maintained, not iroha's own data):
  Bangumi subject id -> MAL id : Rhilip/BangumiExtLinker  data/anime_map.json
  MAL id -> AniList id         : Fribb/anime-lists         anime-list-mini.json

Usage:
  uv run python scripts/build_media_bridge.py
"""

from __future__ import annotations

import argparse
import json
import os
import sys
import urllib.request

import psycopg

UA = "iroha/0.1 (+https://github.com/azusachino/iroha)"
BGM_MAP_URL = "https://rhilip.github.io/BangumiExtLinker/data/anime_map.json"
FRIBB_URL = "https://raw.githubusercontent.com/Fribb/anime-lists/master/anime-list-mini.json"

HOP_BANGUMI_TO_MAL = "bangumi_to_mal"
HOP_MAL_TO_ANILIST = "mal_to_anilist"

UPSERT_SQL = """
    insert into tb_media_ref_bridge (hop, source_id, target_id, updated_at)
    values (%s, %s, %s, now())
    on conflict (hop, source_id) do update
      set target_id = excluded.target_id, updated_at = excluded.updated_at
"""


def fetch_json(url: str) -> object:
    req = urllib.request.Request(url, headers={"User-Agent": UA, "Accept": "application/json"})
    with urllib.request.urlopen(req, timeout=60) as resp:
        return json.loads(resp.read())


def build_bangumi_to_mal(records: list[dict]) -> dict[str, str]:
    """bgm_id -> mal_id, skipping entries with no MAL match."""
    out: dict[str, str] = {}
    for record in records:
        bgm_id = record.get("bgm_id")
        mal_id = record.get("mal_id")
        if bgm_id and mal_id:
            out[str(bgm_id)] = str(mal_id)
    return out


def build_mal_to_anilist(records: list[dict]) -> dict[str, str]:
    """mal_id -> anilist_id, skipping entries with no AniList match."""
    out: dict[str, str] = {}
    for record in records:
        mal_id = record.get("mal_id")
        anilist_id = record.get("anilist_id")
        if mal_id and anilist_id:
            out[str(mal_id)] = str(anilist_id)
    return out


def upsert_hop(conn: psycopg.Connection, hop: str, mapping: dict[str, str]) -> None:
    with conn.cursor() as cur:
        cur.executemany(UPSERT_SQL, [(hop, source_id, target_id) for source_id, target_id in mapping.items()])
    conn.commit()


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.parse_args()

    database_url = os.environ.get("DATABASE_URL") or os.environ.get("IROHA_DATABASE_URL")
    if not database_url:
        print("DATABASE_URL or IROHA_DATABASE_URL is required", file=sys.stderr)
        return 2

    print(f"fetching {BGM_MAP_URL}")
    bgm_to_mal = build_bangumi_to_mal(fetch_json(BGM_MAP_URL))
    print(f"fetching {FRIBB_URL}")
    mal_to_anilist = build_mal_to_anilist(fetch_json(FRIBB_URL))

    with psycopg.connect(database_url) as conn:
        upsert_hop(conn, HOP_BANGUMI_TO_MAL, bgm_to_mal)
        upsert_hop(conn, HOP_MAL_TO_ANILIST, mal_to_anilist)

    print(f"upserted {len(bgm_to_mal)} bangumi->mal rows")
    print(f"upserted {len(mal_to_anilist)} mal->anilist rows")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
