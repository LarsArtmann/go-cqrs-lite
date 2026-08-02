#!/usr/bin/env bash
# check-coverage.sh — Detects when actual module coverage drifts from the
# numbers documented in AGENTS.md.
#
# This script exists because coverage claims in AGENTS.md drifted from reality
# across 4 consecutive sessions (a repeated "coverage-verification gap"). By
# running this as part of docs-health VERIFY (or CI), drift is caught
# mechanically instead of relying on a human remembering to spot-check.
#
# Usage: bash scripts/check-coverage.sh [--update]
#   --update  recompute and print the AGENTS.md-ready coverage line, then exit 0
# Exit: 0 if all modules within tolerance, 1 if any drifted beyond tolerance.

set -euo pipefail

cd "$(git rev-parse --show-toplevel)"

# Tolerance: coverage changes below this are noise (minor refactors). Anything
# larger is real drift that should update AGENTS.md.
TOLERANCE=2.0

# Module → documented coverage (verified 2027-07-27). Keep in sync with AGENTS.md.
declare -A EXPECTED=(
    [decider]=96.1
    [storage/memory]=96.9
    [snapshot]=91.9
    [schema]=89.9
    [command]=88.3
    [event]=88.2
    [id]=86.4
    [metaengine]=76.3
    [query]=83.0
    [dispatcher]=81.5
    [kv]=71.9
    [codec]=70.2
)

# Modules are measured in workspace mode (go.work replacements) with the
# goexperiment.jsonv2 build tag, matching the verify gate.
TAGS="goexperiment.jsonv2"

compute_coverage() {
    local mod="$1"
    # Take the primary package's coverage line (first "coverage:" occurrence).
    go test -tags "$TAGS" -cover "./$mod/..." 2>/dev/null \
        | grep -oE 'coverage: [0-9]+\.[0-9]+%' \
        | head -1 \
        | grep -oE '[0-9]+\.[0-9]+'
}

if [ "${1:-}" = "--update" ]; then
    echo "# Recomputed coverage ($(date +%Y-%m-%d)), workspace mode, $TAGS tag:"
    for mod in "${!EXPECTED[@]}"; do
        printf "  %-18s %s%%\n" "$mod" "$(compute_coverage "$mod")"
    done | sort -t% -k1 -r
    echo ""
    echo "Update the EXPECTED map in this script AND the AGENTS.md coverage line."
    exit 0
fi

drifted=0
printf "%-18s %8s %10s %8s %s\n" "MODULE" "DOC" "ACTUAL" "DELTA" "STATUS"
printf "%-18s %8s %10s %8s %s\n" "------" "---" "------" "-----" "------"

for mod in $(echo "${!EXPECTED[@]}" | tr ' ' '\n' | sort); do
    expected="${EXPECTED[$mod]}"
    actual="$(compute_coverage "$mod" || echo '0.0')"
    delta=$(awk -v a="$actual" -v e="$expected" 'BEGIN { printf "%.1f", a - e }')
    abs_delta=$(awk -v d="$delta" 'BEGIN { if (d < 0) d = -d; printf "%.1f", d }')
    if awk -v ad="$abs_delta" -v t="$TOLERANCE" 'BEGIN { exit !(ad > t) }'; then
        status="DRIFT"
        drifted=$((drifted + 1))
    else
        status="ok"
    fi
    printf "%-18s %7s%% %9s%% %7s%% %s\n" "$mod" "$expected" "$actual" "$delta" "$status"
done

echo ""
if [ "$drifted" -gt 0 ]; then
    echo "::error::$drifted module(s) drifted beyond ±${TOLERANCE}% tolerance."
    echo "Run: bash scripts/check-coverage.sh --update"
    echo "Then update AGENTS.md coverage line to match."
    exit 1
fi

echo "✓ All coverage claims within ±${TOLERANCE}% tolerance."
