#!/usr/bin/env bash
# Loose guard: business Cache() usage must go through tenancy.CacheKey / TenantCacheKey,
# OR the file must be allowlisted (intentional global / platform / caller-prefixed keys).
# Does not forbid facades.Cache() itself (unlike Orm().Query for DB).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

CACHE_PATTERN='facades\.Cache\(\)'
KEY_PATTERN='tenancy\.CacheKey|TenantCacheKey'

DIRS=(app/services app/http app/jobs app/utils app/health)

# Intentional global / platform / caller-supplied keys (not tenant-scoped business data).
ALLOW=(
  app/services/oidc_service.go
  app/services/tenant_domain_service.go
  app/services/tenant_ops_service.go
  app/services/tenant_ops_alert.go
  app/services/tenant_health_inspect.go
  app/services/tenant_scope.go
  app/services/schedule_service.go
  app/services/slow_query_service.go
  app/health/ready_alert.go
  app/health/queue_backlog_alert.go
  app/http/controllers/admin/notification_ws_controller.go
  app/http/controllers/api/queue_test_controller.go
  app/utils/lock_helper.go
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
  if ! grep -qE "$CACHE_PATTERN" "$file" 2>/dev/null; then
    continue
  fi
  if is_allowed "$rel"; then
    continue
  fi
  if grep -qE "$KEY_PATTERN" "$file" 2>/dev/null; then
    continue
  fi
  hits+=("$rel")
done < <(find "${DIRS[@]}" -type f -name '*.go' -print0 2>/dev/null)

if ((${#hits[@]} > 0)); then
  echo "ERROR: facades.Cache() without tenancy.CacheKey / TenantCacheKey in business file(s)."
  echo "Prefix tenant keys with tenancy.CacheKey(ctx, ...) or helpers.TenantCacheKey(ctx, ...)."
  echo "For intentional global keys, add the file to ALLOW in scripts/check-tenant-cache.sh"
  echo "  and note // global on purpose (or equivalent) near the key."
  echo
  echo "Allowlist (global / platform / caller-prefixed):"
  printf '  - %s\n' "${ALLOW[@]}"
  echo
  echo "Violations:"
  printf '  - %s\n' "${hits[@]}"
  exit 1
fi

echo "OK: Cache() sites under ${DIRS[*]} use CacheKey or are allowlisted"
