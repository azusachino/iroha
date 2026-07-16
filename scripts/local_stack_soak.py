#!/usr/bin/env python3
"""Run non-mutating HTTP checks against the complete local stack."""
import argparse
import time
from urllib.error import URLError
from urllib.request import urlopen


def probe(url: str) -> None:
    with urlopen(url, timeout=5) as response:
        if response.status != 200:
            raise RuntimeError(f"{url} returned HTTP {response.status}")


def main() -> int:
    parser = argparse.ArgumentParser(description="soak the local Podman Compose HTTP boundary")
    parser.add_argument("--api-base", default="http://127.0.0.1:8080")
    parser.add_argument("--duration-s", type=int, default=60)
    parser.add_argument("--interval-s", type=float, default=2)
    args = parser.parse_args()

    urls = [f"{args.api_base}/healthz", f"{args.api_base}/api/v1/activities?limit=1"]
    deadline = time.monotonic() + args.duration_s
    probes = 0
    while time.monotonic() < deadline:
        try:
            for url in urls:
                probe(url)
                probes += 1
        except (RuntimeError, URLError) as exc:
            print(f"soak failed after {probes} probes: {exc}")
            return 1
        time.sleep(args.interval_s)
    print(f"soak passed: {probes} HTTP probes in {args.duration_s}s")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
