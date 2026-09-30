#!/usr/bin/env python3
"""Run the Playwright cockpit audit against a throwaway seeded stack.

The stack is release_candidate.py's: an isolated PostGIS container with the
release-candidate seed, the production server and a web preview, all removed
afterwards. Playwright gets the preview URL and the throwaway owner's session
cookie; extra arguments pass through to `playwright test`.
"""

import os
import subprocess
import sys

import iroha_auth
from release_candidate import WEB_DIR, isolated_stack, require_commands

# Next to Playwright's own test-results/, so one folder holds the whole run.
JSON_REPORT = WEB_DIR / "test-results" / "audit.json"


def main(args: list[str]) -> int:
    require_commands("podman", "mise")
    with isolated_stack(integration_tests=False) as stack:
        env = os.environ.copy()
        env["E2E_BASE_URL"] = stack.web_url
        env["E2E_SESSION_COOKIE"] = f"{iroha_auth.SESSION_COOKIE}={stack.owner.token}"
        env["PLAYWRIGHT_JSON_OUTPUT_NAME"] = str(JSON_REPORT)
        command = ["mise", "exec", "--", "bunx", "playwright", "test", "--project=audit"]
        command += ["--reporter=list,json", *args]
        print("+ " + " ".join(command), flush=True)
        return subprocess.run(command, cwd=WEB_DIR, env=env).returncode


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1:]))
