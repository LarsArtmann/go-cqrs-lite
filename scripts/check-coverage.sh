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

# Module → documented coverage (verified 2026-08-27). Keep in sync with AGENTS.md.
# Keys are plain module paths; codec was deleted (ADR-0128) and retry/
# flightrecorder/idempotency shims never had coverage claims.
# 2026-08-16: event 87.3→90.0, query 84.5→89.9 re-baselined UPWARD — the T9
# defect-fix regression tests (06e046c2f) raised both; drift was stale docs, not lost tests.
declare -A EXPECTED=(
	[decider]=96.1
	["storage/memory"]=95.7
	[snapshot]=91.9
	[schema]=92.2
	[command]=88.5
	[event]=90.0
	[id]=86.5
	[metaengine]=83.3
	[query]=89.9
	[dispatcher]=87.7
	[kv]=71.9
)

# Meta-check: every EXPECTED key must resolve to a real module dir. A dangling
# key (codec-dangle class — codec was deleted by ADR-0128) computes as 0.0%
# and is misdiagnosed as coverage DRIFT instead of a stale key. Fail fast with
# the precise diagnosis instead.
for mod in "${!EXPECTED[@]}"; do
	path="${mod// /}"
	if [ ! -f "$path/go.mod" ]; then
		echo "::error::EXPECTED key '$mod' does not resolve to a module (missing $path/go.mod)."
		echo "Remove the stale key or fix the path."
		exit 1
	fi
done

# Modules are measured in workspace mode (go.work replacements) with the
# goexperiment.jsonv2 build tag, matching the verify gate.
TAGS="goexperiment.jsonv2"

compute_coverage() {
	local mod="$1"
	# Defensive: strip any spaces if someone reintroduces spaced display keys.
	local path="${mod// /}"
	# Take the primary package's coverage line (first "coverage:" occurrence).
	go test -tags "$TAGS" -cover "./$path/..." 2>/dev/null |
		grep -oE 'coverage: [0-9]+\.[0-9]+%' |
		head -1 |
		grep -oE '[0-9]+\.[0-9]+'
}

if [ "${1:-}" = "--update" ]; then
	# Auto-stamp the "verified" date in the EXPECTED header comment so the
	# marker cannot go stale when numbers are recomputed. (GNU sed; this repo
	# is Nix/Linux-first.)
	sed -i "s|(verified [0-9]\{4\}-[0-9]\{2\}-[0-9]\{2\})|(verified $(date +%Y-%m-%d))|" "$0"
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

rows=""
for mod in "${!EXPECTED[@]}"; do
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
	rows+="$(printf "%-18s %7s%% %9s%% %7s%% %s" "$mod" "$expected" "$actual" "$delta" "$status")"$'\n'
done
printf '%s' "$rows" | sort

echo ""
if [ "$drifted" -gt 0 ]; then
	echo "::error::$drifted module(s) drifted beyond ±${TOLERANCE}% tolerance."
	echo "Run: bash scripts/check-coverage.sh --update"
	echo "Then update AGENTS.md coverage line to match."
	exit 1
fi

echo "✓ All coverage claims within ±${TOLERANCE}% tolerance."
