#!/usr/bin/env bash
# Fail if govulncheck reports a reachable vulnerability that is not allowlisted.
# Every allowlisted ID needs a reason; remove it as soon as the upstream fix can be taken.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

ALLOW=(
  # excelize: fixed only in unreleased commits after v2.11.0 — bump once v2.11.1 is tagged.
  GO-2026-6684
  GO-2026-6688
  # otel log v0.21 breaks goravel/framework v1.18.0 (telemetry/instrumentation/log) — bump with the framework.
  GO-2026-6508
  GO-2026-6615
)

GOVULNCHECK="${GOVULNCHECK:-govulncheck}"

set +e
out="$("$GOVULNCHECK" ./... 2>&1)"
status=$?
set -e

# 0 = clean, 3 = vulnerabilities found; anything else is a tool/build failure.
if [[ $status -ne 0 && $status -ne 3 ]]; then
  echo "$out"
  echo "ERROR: govulncheck failed (exit $status)"
  exit "$status"
fi

mapfile -t found < <(tr -d '\r' <<<"$out" | grep -oE '^Vulnerability #[0-9]+: GO-[0-9]{4}-[0-9]+' | awk '{print $3}' | sort -u)

is_allowed() {
  local id="$1"
  local a
  for a in "${ALLOW[@]}"; do
    if [[ "$id" == "$a" ]]; then
      return 0
    fi
  done
  return 1
}

unexpected=()
for id in "${found[@]}"; do
  if ! is_allowed "$id"; then
    unexpected+=("$id")
  fi
done

for a in "${ALLOW[@]}"; do
  if ! printf '%s\n' "${found[@]}" | grep -qx "$a"; then
    echo "NOTE: allowlisted $a is no longer reported — remove it from ${BASH_SOURCE[0]##*/}."
  fi
done

if ((${#unexpected[@]} > 0)); then
  echo "$out"
  echo
  echo "ERROR: reachable vulnerabilities not in the allowlist:"
  printf '  - %s\n' "${unexpected[@]}"
  exit 1
fi

echo "OK: no unexpected reachable vulnerabilities (${#found[@]} allowlisted)"
