#!/usr/bin/env bash
# check-api-stability-sync.sh — verify go.work modules ↔ api-stability tracked modules
# Exits 1 if any tracked module is missing or stale.
set -euo pipefail

cd "$(git rev-parse --show-toplevel)"

# Modules tracked by cmd/api-stability/main.go
tracked_modules=$(grep -oE '"[a-z][a-z0-9/_-]*"' cmd/api-stability/main.go \
  | tr -d '"' | grep -v '^\.' | sort -u)

# Modules in go.work that should be tracked (exclude test helpers, cmd, examples)
go_work_modules=$(grep -oE '\./[a-zA-Z0-9/_-]+' go.work | sed 's|^\./||' | sort -u)

# Modules intentionally excluded from API stability tracking
EXCLUDE=(
  "event/v4/eventtest"
  "query/querytest"
  "id/idtest"
  "kv/viewstoretest"
  "stack/contracttest"
  "stack/sqlopt"
  "graph/graphtest"
  "example/getting-started"
  "example/taskmanager"
  "stack/bench"
  "integration"
)

is_excluded() {
  local mod="$1"
  for exc in "${EXCLUDE[@]}"; do
    [ "$mod" = "$exc" ] && return 0
  done
  return 1
}

missing=""
for mod in $go_work_modules; do
  [ "$mod" = "." ] && continue
  [[ "$mod" == cmd/* ]] && continue
  [[ "$mod" == example/* ]] && continue
  is_excluded "$mod" && continue
  if ! echo "$tracked_modules" | grep -qxF "$mod"; then
    missing="$missing\n  $mod"
  fi
done

if [ -n "$missing" ]; then
  echo "WARN: modules in go.work but NOT tracked by cmd/api-stability/main.go:"
  echo -e "$missing"
  echo ""
  echo "Add them to the modules slice in cmd/api-stability/main.go."
  echo "(Some modules may be intentionally untracked — review before adding.)"
  exit 1
fi

echo "OK: go.work ↔ api-stability tracking are in sync"
