#!/usr/bin/env bash
# can-run-composed-gate.sh — one-shot precondition assert for the composed
# gates (#verify, #verify-ci). The 2026-09-19/20 sessions each burned a
# 30-40-minute composed attempt on a condition a 5-second check would have
# caught: a live release smoke mutating testdata mid-run (verify2), or a
# dirty tree that the daemon absorbed mid-phase. git state alone is NOT
# enough — `ps` is part of the gate.
#
# Asserts, then exits 0 ONLY if all hold:
#   1. no release/smoke scripts alive (batch-release, tag-release,
#      module go installs — the classes that mutate the tree or caches)
#   2. tree stable: HEAD + status digest identical across two samples
#      60s apart (the auto-commit daemon absorbs fast; identical digests
#      across a minute means nothing is actively editing)
#   3. load under the ceiling (1-min AND 5-min, wait-for-quiet v2 rule)
#
# Usage:
#   scripts/can-run-composed-gate.sh            # assert now (ceiling 10)
#   scripts/can-run-composed-gate.sh --wait     # block via wait-for-quiet
#                                               # first, then assert
#   scripts/can-run-composed-gate.sh --self-test
#
# Fixture hooks (self-test only): QUIET_LOADAVG_FILE for the load probe,
# CANARY_PROCS (override pgrep patterns), TREE_STABLE_DELAY (shrink the
# stability window).
#
# Exit codes: 0 = safe to launch, 1 = precondition failed (reason printed),
# 2 = usage.
set -uo pipefail

CEILING="${VERIFY_MAX_LOAD:-10}"
WAIT_MODE=0
SELFTEST_MODE=0

while [[ $# -gt 0 ]]; do
	case "$1" in
	--wait)
		WAIT_MODE=1
		shift
		;;
	--max-load)
		CEILING="$2"
		shift 2
		;;
	-h | --help)
		sed -n '2,24p' "$0"
		exit 0
		;;
	--self-test)
		SELFTEST_MODE=1
		shift
		;;
	*)
		echo "can-run-composed-gate: unknown flag: $1 (see --help)" >&2
		exit 2
		;;
	esac
done

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LOADAVG_FILE="${QUIET_LOADAVG_FILE:-/proc/loadavg}"
STABLE_DELAY="${TREE_STABLE_DELAY:-60}"
RELEASE_PROCS_PATTERN="${CANARY_PROCS:-batch-release[.]sh|tag-release[.]sh|go install github[.]com/larsartmann/go-cqrs-lite}"

fail() {
	echo "can-run-composed-gate: $1" >&2
	exit 1
}

tree_digest() {
	git rev-parse HEAD 2>/dev/null
	git status --porcelain 2>/dev/null | sha256sum | cut -d' ' -f1
}

check_release_procs() {
	if pgrep -f "$RELEASE_PROCS_PATTERN" >/dev/null 2>&1; then
		echo "release/smoke processes alive:" >&2
		pgrep -af "$RELEASE_PROCS_PATTERN" >&2
		return 1
	fi
	return 0
}

check_load() {
	local l1 l5
	l1=$(cut -d' ' -f1 "$LOADAVG_FILE" 2>/dev/null || echo 9999)
	l5=$(cut -d' ' -f2 "$LOADAVG_FILE" 2>/dev/null || echo 9999)
	awk -v l1="$l1" -v l5="$l5" -v c="$CEILING" \
		'BEGIN {
			if (l1 !~ /^[0-9.]+$/ || l5 !~ /^[0-9.]+$/) exit 1
			exit !(l1 + 0 < c && l5 + 0 < c * 1.5)
		}'
}

self_test() {
	local tmp failures=0
	tmp="$(mktemp -d)"
	trap 'rm -rf "${tmp:-}"' EXIT

	printf '2.0 3.0 1.0 1/1 1\n' >"$tmp/quiet"
	printf '40.0 55.0 1.0 1/1 1\n' >"$tmp/loud"

	echo "━━━ can-run-composed-gate self-test ━━━"

	# 1. load probe: quiet passes, loud fails (procs+tree hooks bypassed by
	#    impossible patterns / tiny delay; only the load leg is under test).
	if QUIET_LOADAVG_FILE="$tmp/quiet" CANARY_PROCS='impossible-pattern-xyz' \
		TREE_STABLE_DELAY=0 "$0" >/dev/null 2>&1; then
		echo "  ✓ PASS: quiet load + idle tree passes"
	else
		echo "  ✗ FAIL: quiet load + idle tree should pass"
		failures=$((failures + 1))
	fi

	if QUIET_LOADAVG_FILE="$tmp/loud" CANARY_PROCS='impossible-pattern-xyz' \
		TREE_STABLE_DELAY=0 "$0" >/dev/null 2>&1; then
		echo "  ✗ FAIL: loud load must be refused"
		failures=$((failures + 1))
	else
		echo "  ✓ PASS: loud load refused"
	fi

	# 2. canary: a planted process matching the release pattern must fail the gate.
	sleep 300 &
	local planted=$!
	if QUIET_LOADAVG_FILE="$tmp/quiet" TREE_STABLE_DELAY=0 \
		CANARY_PROCS="sleep[ ]300" "$0" >/dev/null 2>&1; then
		echo "  ✗ FAIL: live release-class process must fail the gate"
		failures=$((failures + 1))
	else
		echo "  ✓ PASS: live release-class process refused"
	fi
	kill -9 "$planted" 2>/dev/null || true

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
	echo "can-run-composed-gate: CI environment — tree/procs checks skipped, gate passes"
	exit 0
fi

if [[ "$WAIT_MODE" == 1 ]]; then
	"$SCRIPT_DIR/wait-for-quiet.sh" --max-load "$CEILING" || fail "host never quieted (wait-for-quiet rc=$?)"
fi

check_release_procs || fail "refusing: release session alive (their sweeps mutate fixtures/pins mid-gate)"

d1="$(tree_digest)"
sleep "$STABLE_DELAY"
d2="$(tree_digest)"
if [[ "$d1" != "$d2" ]]; then
	fail "refusing: tree changed during the ${STABLE_DELAY}s stability window (daemon or concurrent session mid-edit)"
fi

if ! check_load; then
	fail "refusing: load above ceiling ${CEILING} (use --wait to block first)"
fi

echo "can-run-composed-gate: GREEN — no release procs, tree stable ${STABLE_DELAY}s, load under ${CEILING}"
exit 0
