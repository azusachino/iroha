"""Read-only prospective release identity checks; never creates tags or publishes images."""
from __future__ import annotations

import argparse
import json
import pathlib
import subprocess

REVISION_LABEL = "org.opencontainers.image.revision"


def git(root: pathlib.Path, *args: str) -> str:
    return subprocess.check_output(["git", "-C", str(root), *args], text=True).strip()


def verify_source(root: pathlib.Path, tag: str, expected_revision: str | None = None, expected_tag_object: str | None = None) -> dict[str, str]:
    if git(root, "status", "--porcelain"):
        raise ValueError("release source is dirty, including untracked files")
    ref = f"refs/tags/{tag}"
    subprocess.run(["git", "check-ref-format", ref], check=True, capture_output=True)
    if git(root, "cat-file", "-t", ref) != "tag":
        raise ValueError("release tag must be annotated")
    revision = git(root, "rev-parse", f"{ref}^{{commit}}")
    if revision != git(root, "rev-parse", "HEAD"):
        raise ValueError("release tag does not identify HEAD")
    if expected_revision is not None and revision != expected_revision:
        raise ValueError("release source revision changed")
    tag_object = git(root, "rev-parse", ref)
    if expected_tag_object is not None and tag_object != expected_tag_object:
        raise ValueError("annotated tag object changed")
    return {"tag": tag, "revision": revision, "tag_object": tag_object, "tree": git(root, "rev-parse", "HEAD^{tree}")}


def verify_image(image: str, revision: str) -> None:
    inspected = json.loads(subprocess.check_output(["podman", "image", "inspect", image], text=True))
    labels = inspected[0].get("Config", {}).get("Labels") or {}
    if labels.get(REVISION_LABEL) != revision:
        raise ValueError("OCI revision does not identify the checked release source")


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--root", type=pathlib.Path, default=pathlib.Path(__file__).resolve().parents[1])
    parser.add_argument("--tag", required=True)
    parser.add_argument("--expected-revision")
    parser.add_argument("--expected-tag-object")
    parser.add_argument("--image", help="Inspect an existing local image; never build or publish")
    args = parser.parse_args()
    try:
        identity = verify_source(args.root, args.tag, args.expected_revision, args.expected_tag_object)
        if args.image:
            verify_image(args.image, identity["revision"])
    except (ValueError, subprocess.CalledProcessError, OSError, IndexError, AttributeError, TypeError) as error:
        parser.exit(1, f"release identity check failed: {error}\n")
    print(json.dumps(identity, sort_keys=True))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
