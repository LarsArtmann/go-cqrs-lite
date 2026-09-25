#!/usr/bin/env bash
# check-buildcache-capacity.sh — capacity monitor for the shared
# /mnt/buildcache mount (the 2026-09-21/22 100%-full incidents: ambient
# GOMODCACHE toolchain unzips died with "no space left on device" mid-gate).
#
# Policy (monitor-only by design — the mount is SHARED by several builders,
# and the 2026-09-21 18G shared-go-cache clear forced rebuilds everywhere):
#   < 80%  PASS (silent)
#  >= 80%  WARN + per-directory du breakdown (what to talk about before it
#          fills; the biggest reclaimable dirs are listed first)
#  >= 95%  FAIL (exit 1) — building now risks ENOSPC mid-gate; clear space
#          BEFORE starting a verify/bench run.
#
# Auto-cleaning is deliberately NOT implemented: any bounded `go clean`
# policy on a shared mount is an operator decision (which dirs, whose
# builds). This gate makes the state impossible to miss instead.
#
# Usage:
#   scripts/check-buildcache-capacity.sh            # monitor the real mount
#   MOUNT=/some/other/mount scripts/check-buildcache-capacity.sh
#   scripts/check-buildcache-capacity.sh --self-test
#
# Fixture hooks (self-test only): CBC_DF_FILE points the usage probe at a
# planted `df -P` fixture instead of live df; CBC_DU_DIR points the du
# breakdown at a fixture tree.
set -uo pipefail

MOUNT="${MOUNT:-/mnt/buildcache}"
WARN_PCT=80
FAIL_PCT=95

if [ "${1:-}" = "--self-test" ]; then
	tmp="$(mktemp -d)"
	trap 'rm -rf "$tmp"' EXIT
	failures=0

	run_case() {
		local name="$1" usage_pct="$2" want_rc="$3"
		local rc=0
		printf 'Filesystem 1024-blocks Used Available Capacity Mounted on\n' >"$tmp/df"
		printf '/dev/sdb1 230686720 %d %d %d%% %s\n' \
			"$((230686720 * usage_pct / 100))" \
			"$((230686720 * (100 - usage_pct) / 100))" \
			"$usage_pct" "$MOUNT" >>"$tmp/df"

		CBC_DF_FILE="$tmp/df" CBC_DU_DIR="$tmp/du" \
			bash "$0" >/dev/null 2>&1 || rc=$?

		if [ "$rc" -eq "$want_rc" ]; then
			echo "  ✓ PASS: $name (rc=$rc)"
		else
			echo "  ✗ FAIL: $name (rc=$rc, want $want_rc)" >&2
			failures=$((failures + 1))
		fi
	}

	mkdir -p "$tmp/du"

	run_case "quiet mount passes silent" 40 0
	run_case "80% warns but passes" 80 0
	run_case "95% fails loud" 95 1

	if [ "$failures" -gt 0 ]; then
		echo "check-buildcache-capacity self-test: ${failures} failure(s)" >&2
		exit 1
	fi
	echo "check-buildcache-capacity self-test: all green"
	exit 0
fi

df_line() {
	if [ -n "${CBC_DF_FILE:-}" ]; then
		cat "$CBC_DF_FILE"
	else
		df -P "$MOUNT"
	fi
}

usage_pct() {
	df_line | awk -v m="$MOUNT" '$NF == m || NR == 2 && $5 ~ /%/ {gsub(/%/, "", $5); print $5; exit}'
}

du_root() {
	du -h --max-depth=1 "${CBC_DU_DIR:-$MOUNT}" 2>/dev/null | sort -rh | head -10
}

pct="$(usage_pct)"
if [ -z "$pct" ]; then
	echo "check-buildcache-capacity: cannot read usage for $MOUNT (mount absent?) — treating as OK"
	exit 0
fi

if [ "$pct" -ge "$FAIL_PCT" ]; then
	echo "✗ /mnt/buildcache is ${pct}% full (>= ${FAIL_PCT}%) — clear space BEFORE building; ENOSPC mid-gate is the known failure mode" >&2
	du_root >&2
	exit 1
fi

if [ "$pct" -ge "$WARN_PCT" ]; then
	echo "::warning::buildcache ${pct}% full (>= ${WARN_PCT}%) — top consumers:"
	du_root
fi

exit 0
