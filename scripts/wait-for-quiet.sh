#!/usr/bin/env bash
# wait-for-quiet.sh — block until the shared host quiets enough for the
# composed gates (#verify, #verify-ci, load-sweep, benchmark baselines).
#
# Ask #3: sessions kept burning 30-40-minute composed attempts against load
# 18-167 (a foreign Node/V8 build + release smokes), then re-running "until
# lucky" — the 2026-09-19/20 verify sessions paid this tax three times in a
# row. This gate makes the WAIT mechanical: poll the 1-min AND 5-min load
# (mirroring calibration-gate.sh v2: load1-only lets a burst-draining host
# pass while still noisy) and return only when both are under the ceiling.
#
# Usage:
#   scripts/wait-for-quiet.sh                     # block, ceiling 10
#   scripts/wait-for-quiet.sh --max-load 5        # stricter (bench-grade)
#   scripts/wait-for-quiet.sh --timeout 7200      # give up after 2h
#   scripts/wait-for-quiet.sh --interval 60       # poll every 60s
#   scripts/wait-for-quiet.sh --self-test         # planted-fixture suite
#
# QUIET_LOADAVG_FILE points the load probe at a fixture instead of
# /proc/loadavg (the --self-test hook — same pattern as
# CALIB_GATE_LOADAVG_FILE; never point it at a tracked file).
#
# CI (CI=true) passes immediately: shared-runner load says nothing about
# the calibration host.
#
# Exit codes: 0 = quiet (or CI), 1 = timeout, 2 = usage.
set -uo pipefail

MAX_LOAD="10"
MAX_LOAD5="15"
TIMEOUT="3600"
INTERVAL="30"
SELFTEST_MODE=0

while [[ $# -gt 0 ]]; do
	case "$1" in
	--max-load)
		MAX_LOAD="$2"
		MAX_LOAD5=$((MAX_LOAD * 3 / 2))
		shift 2
		;;
	--max-load5)
		MAX_LOAD5="$2"
		shift 2
		;;
	--timeout)
		TIMEOUT="$2"
		shift 2
		;;
	--interval)
		INTERVAL="$2"
		shift 2
		;;
	-h | --help)
		sed -n '2,26p' "$0"
		exit 0
		;;
	--self-test)
		SELFTEST_MODE=1
		shift
		;;
	*)
		echo "wait-for-quiet: unknown flag: $1 (see --help)" >&2
		exit 2
		;;
	esac
done

LOADAVG_FILE="${QUIET_LOADAVG_FILE:-/proc/loadavg}"

numeric_or_9999() {
	case "$1" in
	'' | *[!0-9.]*) echo 9999 ;;
	*) echo "$1" ;;
	esac
}

load1() { numeric_or_9999 "$(cut -d' ' -f1 "$LOADAVG_FILE" 2>/dev/null)"; }
load5() { numeric_or_9999 "$(cut -d' ' -f2 "$LOADAVG_FILE" 2>/dev/null)"; }

quiet_now() {
	awk -v l1="$(load1)" -v l5="$(load5)" -v c1="$MAX_LOAD" -v c5="$MAX_LOAD5" \
		'BEGIN { exit !(l1 + 0 < c1 && l5 + 0 < c5) }'
}

wait_for_quiet() {
	local deadline=$((SECONDS + TIMEOUT))
	while ! quiet_now; do
		if ((SECONDS >= deadline)); then
			echo "wait-for-quiet: TIMEOUT after ${TIMEOUT}s — load1=$(load1) load5=$(load5) (ceilings ${MAX_LOAD}/${MAX_LOAD5})" >&2
			return 1
		fi
		echo "… load1=$(load1) load5=$(load5) — waiting for <${MAX_LOAD}/<${MAX_LOAD5} (${TIMEOUT}s budget, ${INTERVAL}s poll)"
		sleep "$INTERVAL"
	done
	echo "wait-for-quiet: QUIET — load1=$(load1) load5=$(load5) (ceilings ${MAX_LOAD}/${MAX_LOAD5})"
	return 0
}

self_test() {
	SELFTEST_TMP="$(mktemp -d)"
	trap 'rm -rf "$SELFTEST_TMP"' EXIT
	local tmp="$SELFTEST_TMP"

	local failures=0
	check() { # name expected_rc fixture max_load max_load5 timeout
		local name="$1" want="$2" fixture="$3" ml="$4" ml5="$5" tmo="$6"
		local rc=0
		out=$(QUIET_LOADAVG_FILE="$fixture" timeout 30 "$0" --max-load "$ml" --max-load5 "$ml5" \
			--timeout "$tmo" --interval 1 2>&1) || rc=$?
		if ((rc == want)); then
			echo "  ✓ PASS: $name"
		else
			echo "  ✗ FAIL: $name — expected rc=$want, got rc=$rc (output: $out)"
			failures=$((failures + 1))
		fi
	}

	printf '2.0 3.0 1.0 1/123 99999\n' >"$tmp/quiet"
	printf '40.0 55.0 30.0 1/123 99999\n' >"$tmp/loud"
	printf '4.0 30.0 20.0 1/123 99999\n' >"$tmp/draining"
	printf 'garbage\n' >"$tmp/broken"

	echo "━━━ wait-for-quiet self-test ━━━"
	check "quiet host passes" 0 "$tmp/quiet" 10 15 10
	check "loud host times out" 1 "$tmp/loud" 10 15 2
	check "burst-draining host (load1 ok, load5 high) refused" 1 "$tmp/draining" 10 15 2
	check "broken probe file never passes (fails loud, not silently)" 1 "$tmp/broken" 10 15 2

	if ((failures > 0)); then
		echo "self-test: ${failures} failure(s)"
		return 1
	fi
	echo "self-test: all green"
	return 0
}

if [[ "$SELFTEST_MODE" == 1 ]]; then
	self_test
	exit $?
fi

if [[ "${CI:-}" == true ]]; then
	echo "wait-for-quiet: CI environment — skipping (runner load is not host load)"
	exit 0
fi

wait_for_quiet
