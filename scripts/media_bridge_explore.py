#!/usr/bin/env python3
"""Cross-provider bridge-coverage check for the media connectors (throwaway).

Measures the real Bangumi->MAL->AniList id-bridge hit-rate on the user's actual
anime collection, so iroha:media-sync task-7 knows how big the fuzzy/inbox tail
is before committing the resolver. Two static datasets do the hops:

  Bangumi subject id -> MAL id : Rhilip/BangumiExtLinker  data/anime_map.json
  MAL id -> AniList id         : Fribb/anime-lists         anime-list-mini.json

It also reports how many of the user's Bangumi anime resolve to a MAL id that is
ALSO in the user's AniList list -- the genuine cross-account dedup collisions the
external_refs ladder must merge onto one tb_media_items row. Anime only: books
(Bangumi subject_type 1) are not covered by these anime datasets and dedupe via
ISBN/title instead. Nothing here is canonical.

Usage:
  uv run python scripts/media_bridge_explore.py <bangumi_user> <anilist_user>
"""

from __future__ import annotations

import argparse
import json
import urllib.request

UA = "iroha/0.1 (+https://github.com/azusachino/iroha)"
BGM_MAP = "https://rhilip.github.io/BangumiExtLinker/data/anime_map.json"
FRIBB = "https://raw.githubusercontent.com/Fribb/anime-lists/master/anime-list-mini.json"
BGM_COLLECTIONS = "https://api.bgm.tv/v0/users/{u}/collections?subject_type=2&limit=50&offset={o}"
ANILIST = "https://graphql.anilist.co"


def _get_json(url: str, data: bytes | None = None, headers: dict | None = None):
    h = {"User-Agent": UA, "Accept": "application/json"}
    if headers:
        h.update(headers)
    req = urllib.request.Request(url, data=data, headers=h, method="POST" if data else "GET")
    with urllib.request.urlopen(req, timeout=60) as resp:
        return json.loads(resp.read())


def bangumi_anime(user: str) -> list[tuple[int, str]]:
    out, offset = [], 0
    while True:
        page = _get_json(BGM_COLLECTIONS.format(u=user, o=offset))
        total = page.get("total", 0)
        for r in page.get("data") or []:
            subj = r.get("subject") or {}
            out.append((r["subject_id"], subj.get("name_cn") or subj.get("name") or ""))
        offset += 50
        if offset >= total or not page.get("data"):
            return out


def anilist_mal_ids(user: str) -> set[int]:
    q = ("query($n:String){MediaListCollection(userName:$n,type:ANIME)"
         "{lists{entries{media{idMal}}}}}")
    body = json.dumps({"query": q, "variables": {"n": user}}).encode()
    data = _get_json(ANILIST, body, {"Content-Type": "application/json"})["data"]
    lists = (data.get("MediaListCollection") or {}).get("lists") or []
    return {e["media"]["idMal"] for lst in lists for e in lst["entries"]
            if (e["media"] or {}).get("idMal")}


def main() -> None:
    ap = argparse.ArgumentParser()
    ap.add_argument("bangumi_user")
    ap.add_argument("anilist_user")
    args = ap.parse_args()

    print("loading datasets...")
    bgm_to_mal = {int(e["bgm_id"]): int(e["mal_id"])
                  for e in _get_json(BGM_MAP) if e.get("mal_id")}
    mal_to_anilist = {e["mal_id"]: e["anilist_id"]
                      for e in _get_json(FRIBB)
                      if e.get("mal_id") and e.get("anilist_id")}
    print(f"  BangumiExtLinker: {len(bgm_to_mal)} bgm->mal")
    print(f"  Fribb anime-lists: {len(mal_to_anilist)} mal->anilist")

    subjects = bangumi_anime(args.bangumi_user)
    anilist_mals = anilist_mal_ids(args.anilist_user)
    n = len(subjects)

    hop1 = hop2 = collide = 0
    tail: list[str] = []
    for sid, name in subjects:
        mal = bgm_to_mal.get(sid)
        if not mal:
            tail.append(f"bgm:{sid} {name}")
            continue
        hop1 += 1
        if mal in mal_to_anilist:
            hop2 += 1
        if mal in anilist_mals:
            collide += 1

    pct = lambda x: f"{(100 * x / n):.0f}%" if n else "-"
    print(f"\nBangumi anime collection ({args.bangumi_user}): {n} subjects")
    print(f"  hop1 bgm->mal      : {hop1}/{n} ({pct(hop1)})")
    print(f"  hop2 mal->anilist  : {hop2}/{n} ({pct(hop2)})  <- full auto-bridge")
    print(f"  fuzzy/inbox tail   : {n - hop1}/{n} ({pct(n - hop1)})  (no MAL match)")
    print(f"  cross-account dedup: {collide} of these anime are ALSO in {args.anilist_user}'s"
          f" AniList list (same idMal) -> must merge onto one item")
    if tail:
        print(f"\nunmapped tail (first 15 of {len(tail)}):")
        for t in tail[:15]:
            print("  -", t)

    print("\nResolver hint: hop2 % is the share auto-merged by the two-hop bridge;"
          " the tail needs title+year -> tb_media_resolution_tasks (inbox). Books"
          " (subject_type 1) are excluded here and dedupe via ISBN/title separately.")


if __name__ == "__main__":
    main()
