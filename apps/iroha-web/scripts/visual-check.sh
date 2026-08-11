#!/usr/bin/env bash
set -euo pipefail

base="${BASE:-http://127.0.0.1:5173}"
theme="${THEME:-field-journal}"
routes="${ROUTES:-overview}"
out="${OUT:-.visual-check}"
session="${AGENT_BROWSER_SESSION:-iroha-visual}"

command -v agent-browser >/dev/null || {
  echo "agent-browser is required" >&2
  exit 1
}

mkdir -p "$out"
cleanup() {
  agent-browser --session "$session" open about:blank >/dev/null 2>&1 || true
  agent-browser --session "$session" close >/dev/null 2>&1 || true
}
trap cleanup EXIT

agent-browser --session "$session" open "$base/"
agent-browser --session "$session" storage local set iroha-design-language "$theme"

IFS=',' read -r -a route_list <<< "$routes"
for route in "${route_list[@]}"; do
  route="${route#/}"
  agent-browser --session "$session" open "$base/$route"
  agent-browser --session "$session" wait 800
  agent-browser --session "$session" errors
  agent-browser --session "$session" screenshot --full "$out/$theme-${route//\//-}.png"
  echo "checked: $route"
done
