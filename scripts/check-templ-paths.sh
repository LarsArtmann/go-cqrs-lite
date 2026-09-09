#!/usr/bin/env bash
# Tripwire: templ generate bakes the invocation cwd into _templ.go FileName
# metadata. Generated from catalog/docserver/ the names are bare filenames
# (FileName: `d2view.templ`); generated from anywhere else they carry path
# fragments, and `nix run .#check-templ` then reports permanent drift no
# matter how fresh the codegen is (AGENTS gotcha, 2026-09-08).
#
# Fails when any _templ.go FileName value contains a path separator.
set -euo pipefail

repo_root="$(cd "$(dirname "$0")/.." && pwd)"
status=0

while IFS= read -r -d '' gen; do
  if grep -nE 'FileName: `[^`]*[/\\][^`]*`' "$gen"; then
    echo "ERROR: $(basename "$gen") was generated from the wrong cwd — FileName carries a path." >&2
    echo "       Regenerate with: (cd catalog/docserver && templ generate)" >&2
    status=1
  fi
done < <(find "$repo_root" -name '*_templ.go' -not -path '*/vendor/*' -print0)

if [ "$status" -eq 0 ]; then
  echo "✓ all _templ.go FileName values are cwd-clean (bare filenames)"
fi

exit "$status"
