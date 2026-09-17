#!/usr/bin/env bash
# fp-sweep.sh — run the cqrs-lint binary over local consumer repos and record
# the per-repo finding distribution plus the low-confidence (FP-suspect) rate.
#
# Purpose (adoption plan task C25): a baseline false-positive-rate measurement
# so rule changes can be judged by DELTA on real code, not by vibes. The
# original plan tied a second sweep to the --correlate flag; correlation was
# dropped (single-tool ecosystem, see IMPROVEMENT_IDEAS "rule-split
# convention"), so the harness reports the per-repo distribution that any
# future rule change should be compared against.
#
# Usage:
#   bash scripts/fp-sweep.sh [--out FILE] [repo-dir ...]
#   default repos: the locally-available members of the IMPROVEMENT_IDEAS
#   consumer reference table; --out writes markdown (default: stdout).
#
# The binary is built from this checkout (cmd/cqrs-lint) — sweep results are
# only comparable when produced by the same binary build. Requires jq.

set -euo pipefail

cd "$(git rev-parse --show-toplevel)"

OUT=""
REPOS=()
while [ $# -gt 0 ]; do
	case "$1" in
	--out)
		OUT="$2"
		shift 2
		;;
	-*)
		echo "unknown flag: $1" >&2
		exit 2
		;;
	*)
		REPOS+=("$1")
		shift
		;;
	esac
done

if [ ${#REPOS[@]} -eq 0 ]; then
	for n in cqrs-htmx bank-sync browser-history github-local-sync go-localsync \
		crush-daily timesheets accountability-system overview storbi \
		standard-bug-tracking-schema dnsblockd; do
		[ -d "/home/lars/projects/$n" ] && REPOS+=("/home/lars/projects/$n")
	done
fi

if [ ${#REPOS[@]} -eq 0 ]; then
	echo "no consumer repos found" >&2
	exit 1
fi

command -v jq >/dev/null 2>&1 || {
	echo "jq required (not found)" >&2
	exit 1
}

BIN=$(mktemp /tmp/cqrs-lint-fp-sweep.XXXXXX)
trap 'rm -f "$BIN"' EXIT
(cd cmd/cqrs-lint && GOWORK=off go build -tags "goexperiment.jsonv2" -o "$BIN" .)

{
	echo "# cqrs-lint FP sweep — $(date -u +%Y-%m-%dT%H:%M:%SZ)"
	echo
	echo "Binary: $(git rev-parse --short HEAD) | Repos: ${#REPOS[@]}"
	echo
	echo "| repo | findings | low-confidence (<0.5) |"
	echo "| ---- | -------- | --------------------- |"
	total=0
	suspects=0
	for repo in "${REPOS[@]}"; do
		[ -d "$repo" ] || continue
		out=$("$BIN" --quiet --format json "$repo" 2>/dev/null || true)
		n=$(printf '%s' "$out" | jq '[.findings // [] | length] | add // 0' 2>/dev/null || echo 0)
		s=$(printf '%s' "$out" | jq '[.findings // [] | map(select((.confidence // 1) < 0.5)) | length] | add // 0' 2>/dev/null || echo 0)
		total=$((total + n))
		suspects=$((suspects + s))
		printf '| %s | %s | %s |\n' "$(basename "$repo")" "$n" "$s"
	done
	echo
	echo "Total: $total finding(s), $suspects low-confidence suspect(s)"
} | tee "${OUT:-/dev/stdout}" >/dev/null
