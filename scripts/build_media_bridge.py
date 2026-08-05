#!/usr/bin/env python3
"""Build the two-hop Bangumi->MAL->AniList media bridge cache.

Writes bangumi_to_mal.json and mal_to_anilist.json to --out, in the
map[string]string shape apps/iroha-imports/media_resolution.go's
TwoHopMediaRefBridge.Lookup expects (string keys AND string values --
Bangumi/MAL/AniList external IDs are compared as strings throughout the
resolver, so int-typed values would fail every lookup silently).

Sources (community-maintained, not iroha's own data):
  Bangumi subject id -> MAL id : Rhilip/BangumiExtLinker  data/anime_map.json
  MAL id -> AniList id         : Fribb/anime-lists         anime-list-mini.json

Usage:
  uv run python scripts/build_media_bridge.py --out dist/media-bridge
"""

from __future__ import annotations

import argparse
import json
import urllib.request
from pathlib import Path

UA = "iroha/0.1 (+https://github.com/azusachino/iroha)"
BGM_MAP_URL = "https://rhilip.github.io/BangumiExtLinker/data/anime_map.json"
FRIBB_URL = "https://raw.githubusercontent.com/Fribb/anime-lists/master/anime-list-mini.json"


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


def write_json(path: Path, data: dict[str, str]) -> None:
    path.write_text(json.dumps(data, sort_keys=True, separators=(",", ":")))


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--out", default="dist/media-bridge", help="output directory")
    args = parser.parse_args()

    out_dir = Path(args.out)
    out_dir.mkdir(parents=True, exist_ok=True)

    print(f"fetching {BGM_MAP_URL}")
    bgm_to_mal = build_bangumi_to_mal(fetch_json(BGM_MAP_URL))
    print(f"fetching {FRIBB_URL}")
    mal_to_anilist = build_mal_to_anilist(fetch_json(FRIBB_URL))

    write_json(out_dir / "bangumi_to_mal.json", bgm_to_mal)
    write_json(out_dir / "mal_to_anilist.json", mal_to_anilist)

    print(f"wrote {len(bgm_to_mal)} bangumi->mal entries to {out_dir / 'bangumi_to_mal.json'}")
    print(f"wrote {len(mal_to_anilist)} mal->anilist entries to {out_dir / 'mal_to_anilist.json'}")


if __name__ == "__main__":
    main()
