#!/usr/bin/env bash
# check-workspace-sync.sh — verify go.work modules ↔ flake.nix testModules are in sync
# Exits 1 if any module in go.work's `use` block is missing from flake.nix testModules.
set -euo pipefail

cd "$(git rev-parse --show-toplevel)"

# Modules that are test helpers (tested transitively by their parent module)
EXCLUDE=(
  "event/v4/eventtest"  # tested by event/ tests
)

is_excluded() {
  local mod="$1"
  for exc in "${EXCLUDE[@]}"; do
    [ "$mod" = "$exc" ] && return 0
  done
  return 1
}

# Extract modules from go.work (strip leading ./, trailing comments)
go_work_modules=$(grep -oE '\./[a-zA-Z0-9/_-]+' go.work | sed 's|^\./||' | sort -u)

# Extract modules from flake.nix testModules array
flake_modules=$(awk '/testModules = \[/,/\]/' flake.nix \
  | grep -oE '"[a-zA-Z0-9/_-]+"' | tr -d '"' | sort -u)

# Also get example modules (always tested via examplePaths)
example_modules=$(grep '"\./example/' flake.nix | sed 's|.*"\./\(example/[a-zA-Z0-9/_-]*\)/\.\.\.".*|\1|' | sort -u)

# Combine flake + example modules as "tested"
tested_modules=$(printf '%s\n%s\n' "$flake_modules" "$example_modules" | sort -u)

# Find go.work modules not in tested set
missing=""
for mod in $go_work_modules; do
  # Skip root module
  [ "$mod" = "." ] && continue
  # Skip cmd modules (not tested, only built)
  [[ "$mod" == cmd/* ]] && continue
  # Skip test helper packages
  is_excluded "$mod" && continue
  # Check if in tested set
  if ! echo "$tested_modules" | grep -qxF "$mod"; then
    missing="$missing\n  $mod"
  fi
done

if [ -n "$missing" ]; then
  echo "FAIL: modules in go.work but NOT in flake.nix testModules or examplePaths:"
  echo -e "$missing"
  echo ""
  echo "Add them to flake.nix testModules array."
  exit 1
fi

echo "OK: go.work ↔ flake.nix are in sync"
