#!/usr/bin/env bash
# check-depguard.sh — Detects dependencies in go.mod files that are missing
# from the depguard allow list in .golangci.yml.
#
# Problem: dependencies are only added to .golangci.yml AFTER lint fails.
# This script catches the gap proactively, before the developer hits a
# confusing depguard error.
#
# Usage: bash scripts/check-depguard.sh
# Exit: 0 if all go.mod requires are in the depguard allow list, 1 otherwise.

set -euo pipefail

cd "$(git rev-parse --show-toplevel)"

# ── Extract allowed prefixes from .golangci.yml depguard section ──────────
#
# The allow list looks like:
#   depguard:
#     rules:
#       Main:
#         allow:
#           - $gostd
#           - github.com/larsartmann/go-cqrs-lite
#           - ...
#
# We extract everything after "- " in the allow block, skipping $gostd
# (handled separately — stdlib packages have no domain prefix).

ALLOW_FILE=$(mktemp)
trap 'rm -f "$ALLOW_FILE"' EXIT

# Extract the depguard allow list. Use awk to grab lines between
# "allow:" and the next dedented line, stripping the "- " prefix.
awk '
  /depguard:/ { in_depguard = 1 }
  in_depguard && /allow:/ { in_allow = 1; next }
  in_allow {
    if (/^            - /) {
      gsub(/^            - /, "")
      if ($0 != "$gostd") print $0
    } else if (/^          [^ ]/) {
      in_allow = 0
    }
  }
' .golangci.yml > "$ALLOW_FILE"

if [ ! -s "$ALLOW_FILE" ]; then
  echo "ERROR: could not extract depguard allow list from .golangci.yml" >&2
  exit 1
fi

echo "Allowed prefixes ($(wc -l < "$ALLOW_FILE") entries):"
sed 's/^/  /' "$ALLOW_FILE"
echo ""

# ── Collect all direct (non-indirect) requires from every go.mod ──────────
#
# Direct requires look like:
#   github.com/foo/bar v1.2.3
# Indirect ones have " // indirect" suffix — we skip those.

REQUIRES_FILE=$(mktemp)
trap 'rm -f "$ALLOW_FILE" "$REQUIRES_FILE"' EXIT

find . -name go.mod -not -path './vendor/*' -print0 |
  while IFS= read -r -d '' modfile; do
    # Extract require block entries (not indirect).
    awk '
      /^require \(/ { in_block = 1; next }
      in_block && /^\)/ { in_block = 0; next }
      in_block && !/\/\/ indirect/ && /^\t/ { print $1 }
      /^require / && !/\/\// && !/\(/ { print $2 }
    ' "$modfile"
  done | sort -u > "$REQUIRES_FILE"

# ── Check each require against the allow list ─────────────────────────────
#
# A require matches if it starts with any allowed prefix (prefix match,
# same as depguard's behavior). Stdlib packages (no dot in first path
# segment) are always allowed.

MISSING=0
while IFS= read -r dep; do
  [ -z "$dep" ] && continue

  # Skip stdlib (no dot in first segment = no domain).
  first_seg="${dep%%/*}"
  if [[ "$first_seg" != *.* ]]; then
    continue
  fi

  # Check against allow list.
  if ! grep -qF "$dep" "$ALLOW_FILE" 2>/dev/null; then
    # Try prefix match: does any allowed entry match as a prefix?
    matched=0
    while IFS= read -r prefix; do
      if [[ "$dep" == "$prefix"* ]]; then
        matched=1
        break
      fi
    done < "$ALLOW_FILE"

    if [ "$matched" -eq 0 ]; then
      echo "MISSING from depguard allow list: $dep"
      MISSING=$((MISSING + 1))
    fi
  fi
done < "$REQUIRES_FILE"

echo ""
if [ "$MISSING" -gt 0 ]; then
  echo "FAIL: $MISSING dependencies in go.mod files are not in the depguard allow list."
  echo "Fix: add them to .golangci.yml under depguard.rules.Main.allow"
  exit 1
fi

TOTAL_REQUIRES=$(wc -l < "$REQUIRES_FILE")
echo "OK: all $TOTAL_REQUIRES unique direct dependencies are covered by the depguard allow list."
