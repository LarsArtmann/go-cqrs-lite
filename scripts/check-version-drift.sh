#!/usr/bin/env bash
# check-version-drift.sh — Detects when sibling modules reference different
# versions of the same internal dependency.
#
# In a multi-module workspace, all modules should reference the same version
# of each sibling. Drift causes subtle bugs when modules are consumed
# independently (GOWORK=off).
#
# Usage: bash scripts/check-version-drift.sh
# Exit: 0 if no drift, 1 if drift detected.

set -euo pipefail

cd "$(git rev-parse --show-toplevel)"

drift=$(
    grep -rh "go-cqrs-lite/.*/v2 v" */go.mod 2>/dev/null \
        | grep -oE 'go-cqrs-lite/[^ ]+/v2 v[0-9]+\.[0-9]+\.[0-9]+' \
        | sort -u \
        | awk '
            {
                mod = $1; ver = $2
                if (mod != prev && prev != "" && cnt > 1) {
                    print prev ": " versions
                }
                if (mod != prev) { versions = ""; cnt = 0 }
                if (index(versions, ver) == 0) {
                    if (versions != "") versions = versions ", "
                    versions = versions ver
                    cnt++
                }
                prev = mod
            }
            END { if (cnt > 1) print prev ": " versions }
        '
)

if [ -n "$drift" ]; then
    echo "::error::Version drift detected:"
    echo "$drift" | while IFS= read -r line; do echo "  $line"; done
    exit 1
fi

echo "✓ No version drift detected."
