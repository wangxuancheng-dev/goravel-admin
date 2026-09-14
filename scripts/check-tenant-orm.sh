#!/usr/bin/env bash
# Fail if business code calls facades.Orm().Query() (use appfacades.OrmQuery(ctx)).
# Seeders / migrations / tests are out of scope. Tenant maintenance helpers are allowlisted.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

PATTERN='facades\.Orm\(\)\.Query\('
DIRS=(app/services app/http app/jobs app/console)
ALLOW=(
  app/services/tenant_connection_service.go
  app/services/tenant_platform_extra.go
)

is_allowed() {
  local f="$1"
  local a
  for a in "${ALLOW[@]}"; do
    if [[ "$f" == "$a" ]]; then
      return 0
    fi
  done
  return 1
}

hits=()
while IFS= read -r -d '' file; do
  rel="${file#./}"
  case "$rel" in
    *_test.go) continue ;;
  esac
  if is_allowed "$rel"; then
    continue
  fi
  # Skip pure comment lines (still catch code + trailing comments).
  while IFS= read -r line; do
    if [[ "$line" =~ ^[0-9]+:[[:space:]]*// ]]; then
      continue
    fi
    hits+=("${rel}:${line}")
  done < <(grep -nE "$PATTERN" "$file" 2>/dev/null || true)
done < <(find "${DIRS[@]}" -type f -name '*.go' -print0 2>/dev/null)

if ((${#hits[@]} > 0)); then
  echo "ERROR: facades.Orm().Query() is forbidden in business dirs."
  echo "Use appfacades.OrmQuery(ctx) (tenant) or PlatformOrmQuery(ctx) (platform)."
  echo "Allowlist (tenant maintenance only):"
  printf '  - %s\n' "${ALLOW[@]}"
  echo
  printf '%s\n' "${hits[@]}"
  exit 1
fi

echo "OK: no forbidden Orm().Query() under ${DIRS[*]}"
