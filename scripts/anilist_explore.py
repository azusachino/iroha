#!/usr/bin/env python3
"""Exploratory dump of an AniList user's anime/manga list (throwaway, pre-schema).

Pulls MediaListCollection (ANIME + MANGA) for a public username via the AniList
GraphQL API and prints, per media type: entry/list-status distribution, score
coverage + the user's scoreFormat, idMal coverage (the MAL bridge key for
cross-provider dedup), media-format distribution, progress/date/repeat coverage,
and how many entries expose relation edges. The point is to decide the
observations.Media field mapping and confirm the AniList->MAL bridge coverage
before the Go connector (iroha:media-sync task-5) is written.

Public lists need no auth; pass --token (an AniList OAuth access token) only for
a private list. Nothing here is canonical; the Go connector re-derives from raw
snapshots.

Usage:
  uv run python scripts/anilist_explore.py <username> [--token TOKEN]
"""

from __future__ import annotations

import argparse
import json
import urllib.request
from collections import Counter

API = "https://graphql.anilist.co"
UA = "iroha/0.1 (+https://github.com/azusachino/iroha)"

# One query per media type. relations are fetched (edge relationType only) so we
# can count how many entries carry graph edges without a second round-trip.
QUERY = """
query($n:String,$t:MediaType){
  MediaListCollection(userName:$n,type:$t){
    lists{
      name
      status
      entries{
        status score progress progressVolumes repeat private
        notes
        startedAt{year} completedAt{year}
        media{
          id idMal type format
          episodes chapters volumes
          title{romaji english native}
          relations{edges{relationType}}
        }
      }
    }
  }
}
"""


def post(query: str, variables: dict, token: str | None) -> dict:
    body = json.dumps({"query": query, "variables": variables}).encode()
    headers = {"Content-Type": "application/json", "Accept": "application/json", "User-Agent": UA}
    if token:
        headers["Authorization"] = f"Bearer {token}"
    req = urllib.request.Request(API, data=body, headers=headers, method="POST")
    with urllib.request.urlopen(req, timeout=30) as resp:
        payload = json.loads(resp.read())
    if payload.get("errors"):
        raise SystemExit(f"AniList error: {payload['errors']}")
    return payload["data"]


def score_format(username: str, token: str | None) -> str:
    data = post(
        "query($n:String){User(name:$n){mediaListOptions{scoreFormat}}}",
        {"n": username},
        token,
    )
    return (data.get("User") or {}).get("mediaListOptions", {}).get("scoreFormat", "?")


def dump(username: str, media_type: str, token: str | None) -> None:
    data = post(QUERY, {"n": username, "t": media_type}, token)
    lists = (data.get("MediaListCollection") or {}).get("lists") or []
    entries = [e for lst in lists for e in lst["entries"]]

    status = Counter(e["status"] for e in entries)
    fmt = Counter((e["media"] or {}).get("format") for e in entries)
    rel = Counter(
        edge["relationType"]
        for e in entries
        for edge in ((e["media"] or {}).get("relations") or {}).get("edges") or []
    )
    n = len(entries)
    with_mal = sum(1 for e in entries if (e["media"] or {}).get("idMal"))
    with_score = sum(1 for e in entries if e.get("score"))
    with_rel = sum(1 for e in entries
                   if ((e["media"] or {}).get("relations") or {}).get("edges"))
    with_start = sum(1 for e in entries if (e.get("startedAt") or {}).get("year"))
    with_repeat = sum(1 for e in entries if (e.get("repeat") or 0) > 0)
    private = sum(1 for e in entries if e.get("private"))
    pct = lambda x: f"{(100 * x / n):.0f}%" if n else "-"

    print(f"\n### {media_type}  ({n} entries across {len(lists)} lists)")
    print("  list status :", dict(status))
    print("  media format:", dict(fmt))
    print(f"  idMal cover : {with_mal}/{n} ({pct(with_mal)})  <- MAL bridge key")
    print(f"  score cover : {with_score}/{n} ({pct(with_score)})")
    print(f"  has started : {with_start}/{n} ({pct(with_start)})")
    print(f"  repeats>0   : {with_repeat}/{n}   private: {private}")
    print(f"  has relation: {with_rel}/{n} ({pct(with_rel)})")
    print("  relation types (top 8):", dict(rel.most_common(8)))


def main() -> None:
    ap = argparse.ArgumentParser()
    ap.add_argument("username")
    ap.add_argument("--token", default=None, help="AniList OAuth token (private lists only)")
    args = ap.parse_args()

    sf = score_format(args.username, args.token)
    print(f"user={args.username}  scoreFormat={sf}")
    print("(scoreFormat decides how to normalize `score`: POINT_100/10_DECIMAL/10/5/3)")
    for t in ("ANIME", "MANGA"):
        dump(args.username, t, args.token)

    print("\nMapping hints: idMal cover ~high -> AniList->MAL bridge is reliable;"
          " map media.format -> media_type (TV/MOVIE/OVA/ONA/SPECIAL/MANGA/NOVEL);"
          " title{romaji,english,native} -> 3 tb_media_titles rows; list status ->"
          " progress projection; repeat>0 -> N rewatch/reread events.")


if __name__ == "__main__":
    main()
