#!/usr/bin/env bash
#
# check-module-isolation.sh — verify every Go module builds under GOWORK=off.
#
# The go.work workspace resolves all sibling modules transitively, hiding
# missing require/replace directives in individual go.mod files. This script
# builds each module in true isolation (GOWORK=off) to catch that class of bug.
#
# Usage: bash scripts/check-module-isolation.sh [--test]
#   --test  run `go test` instead of `go build`
#
# Exit 0 = all pass, exit 1 = at least one module fails.

set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

MODE="build"
if [ "${1:-}" = "--test" ]; then
  MODE="test"
fi

TAGS="-tags=goexperiment.arenas,goexperiment.jsonv2"
failures=0
total=0
failed_modules=""

echo "=== Module Isolation Check ($MODE) ==="

for mod in $(find . -name go.mod -not -path './vendor/*' | sed 's|/go.mod||' | sort); do
  total=$((total + 1))
  mod_name="${mod#./}"

  if [ "$MODE" = "test" ]; then
    result=$(cd "$mod" && GOWORK=off go test $TAGS ./... 2>&1)
  else
    result=$(cd "$mod" && GOWORK=off go build $TAGS ./... 2>&1)
  fi

  rc=$?
  if [ $rc -ne 0 ]; then
    failures=$((failures + 1))
    failed_modules="$failed_modules\n  $mod_name"
    echo "  FAIL: $mod_name"
    echo "$result" | head -3 | sed 's/^/    /'
  else
    echo "  OK:   $mod_name"
  fi
done

echo ""
echo "=== Result: $failures / $total modules failed ==="

if [ $failures -gt 0 ]; then
  echo -e "Failed modules:$failed_modules"
  exit 1
fi

echo "All modules build in isolation."
exit 0
