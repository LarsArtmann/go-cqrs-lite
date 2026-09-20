#!/usr/bin/env bash
# calibration-gate.sh — mechanical load gate + provenance capture for
# benchmark/calibration runs (docs/benchmarks/calibration-2026-08-30.md
# §Protocol, added 2026-09-11 after the SearchQuery run shipped an
# unverified engine-version citation and a load-ramping window).
#
# The protocol used to rely on prose ("don't calibrate during compile
# storms") on a 24-28-user shared host — this gate makes the check
# mechanical: assert the 1-min AND 5-min load are below a ceiling before
# benching (v2, 2026-09-13: load1-only let a burst-draining host — load1=4,
# load5=30 — pass while still noisy) and emit a provenance line (uptime
# samples, binary store paths, binary version output) for the measurement
# record.
#
# Usage:
#   scripts/calibration-gate.sh                        # gate only, ceiling 5
#   scripts/calibration-gate.sh --max-load 3           # stricter ceiling
#   scripts/calibration-gate.sh --provenance dgraph    # + provenance lines for
#                                                      # each named binary
#   CALIB_GATE_REQUIRED=0 scripts/calibration-gate.sh  # warn-only override
#   scripts/calibration-gate.sh --self-test            # fault-injection suite:
#                                                      # planted loadavg fixtures
#                                                      # in a temp dir (never a
#                                                      # live file) + a golden
#                                                      # pin of the FAIL-message
#                                                      # shape (scripts/testdata/)
#
# CALIB_GATE_LOADAVG_FILE points the load probe at a fixture instead of
# /proc/loadavg. Internal hook for --self-test only — do not set in normal
# use.
#
# CI (CI=true, e.g. the benchmarks.yml drift job) never aborts: shared
# runner load is not the calibration host's load; the gate exists to
# protect LOCAL baseline-producing runs.
#
# Exit codes: 0 = gate passed (or warn-only), 1 = load above ceiling, 2 = usage.
set -uo pipefail

MAX_LOAD="5"
SELFTEST_MODE=0
PROVENANCE_BINS=()

while [[ $# -gt 0 ]]; do
	case "$1" in
	--max-load)
		MAX_LOAD="$2"
		shift 2
		;;
	--provenance)
		shift
		while [[ $# -gt 0 && "$1" != --* ]]; do
			PROVENANCE_BINS+=("$1")
			shift
		done
		;;
	-h | --help)
		sed -n '2,36p' "$0"
		exit 0
		;;
	--self-test)
		SELFTEST_MODE=1
		shift
		;;
	*)
		echo "calibration-gate: unknown flag: $1 (see --help)" >&2
		exit 2
		;;
	esac
done

# Fault-injection hook for --self-test: point the load probe at a fixture
# file instead of the live /proc/loadavg. Internal — do not set in normal use.
LOADAVG_FILE="${CALIB_GATE_LOADAVG_FILE:-/proc/loadavg}"

load1() { cut -d' ' -f1 "$LOADAVG_FILE"; }
load5() { cut -d' ' -f2 "$LOADAVG_FILE"; }

LOAD1=$(load1)
LOAD5=$(load5)
LOADAVG=$(cat "$LOADAVG_FILE" 2>/dev/null || echo "unknown")
UPTIME=$(uptime 2>/dev/null || echo "unknown")
WHEN=$(date -u +%Y-%m-%dT%H:%M:%SZ)

over_ceiling() {
	# $1 = load value, $2 = label (load1/load5) for the failure message.
	if awk -v l="$1" -v m="$MAX_LOAD" 'BEGIN { exit !(l >= m) }'; then
		FAILED_LOAD="$2=$1"
		return 0
	fi
	return 1
}

# --- --self-test: fault-injection suite over temp loadavg fixtures ---

SELFTEST_SELF=""
SELFTEST_DIR=""

self_run() {
	# $1 = fixture loadavg line; remaining args = gate flags. Echoes the
	# gate's combined output; the caller owns the exit code. CI is forced
	# false so the gate's CI pass-through (GH runners set CI=true) cannot
	# flip the failure legs green on the runner (the 12/12-local vs red-CI
	# divergence); the dedicated CI leg opts back in via SELFTEST_CI_LEG=1.
	local content="$1"
	shift
	printf '%s\n' "$content" >"$SELFTEST_DIR/loadavg"
	if [[ "${SELFTEST_CI_LEG:-}" == 1 ]]; then
		CALIB_GATE_LOADAVG_FILE="$SELFTEST_DIR/loadavg" \
			bash "$SELFTEST_SELF" "$@" 2>&1
	else
		CI=false CALIB_GATE_LOADAVG_FILE="$SELFTEST_DIR/loadavg" \
			bash "$SELFTEST_SELF" "$@" 2>&1
	fi
}

self_check() {
	# $1 name, $2 want-rc, $3 got-rc, $4 output, $5 required substring.
	local name="$1" want="$2" got="$3" out="$4" pattern="$5"
	local ok=1
	[[ "$got" == "$want" ]] || ok=0
	grep -qF -- "$pattern" <<<"$out" || ok=0
	if [[ "$ok" != 1 ]]; then
		echo "  ✗ FAIL: $name (rc=$got, want=$want; missing '$pattern') got:"
		echo "      ${out//$'\n'/$'\n'      }"
		return 1
	fi
	echo "  ✓ PASS: $name"
}

self_test_gate() {
	local out rc fails=0

	rc=0
	out=$(self_run "0.42 0.51 0.10 2/1234 5678" --max-load 5) || rc=$?
	self_check "quiet host passes" 0 "$rc" "$out" \
		"PASS — load1=0.42, load5=0.51 < 5" || fails=$((fails + 1))

	rc=0
	out=$(self_run "7.20 1.80 1.00 2/1234 5678") || rc=$?
	self_check "load1 over ceiling fails" 1 "$rc" "$out" \
		"LOAD GATE FAILED — load1=7.20" || fails=$((fails + 1))

	rc=0
	out=$(self_run "0.42 9.90 4.00 2/1234 5678") || rc=$?
	self_check "burst-draining host fails (load5, v2 rule)" 1 "$rc" "$out" \
		"LOAD GATE FAILED — load5=9.90" || fails=$((fails + 1))

	export CALIB_GATE_REQUIRED=0
	rc=0
	out=$(self_run "0.42 9.90 4.00 2/1234 5678") || rc=$?
	unset CALIB_GATE_REQUIRED
	self_check "warn-only override stays green" 0 "$rc" "$out" \
		"WARN-ONLY override" || fails=$((fails + 1))

	export SELFTEST_CI_LEG=1 CI=true
	rc=0
	out=$(self_run "7.20 9.90 4.00 2/1234 5678") || rc=$?
	unset SELFTEST_CI_LEG CI
	self_check "CI never aborts" 0 "$rc" "$out" \
		"informational only" || fails=$((fails + 1))

	return "$fails"
}

self_test_provenance() {
	local out rc

	mkdir -p "$SELFTEST_DIR/bin"
	printf '#!/bin/sh\necho "calgate-stub 1.2.3"\n' \
		>"$SELFTEST_DIR/bin/calgate-stub"
	chmod +x "$SELFTEST_DIR/bin/calgate-stub"
	printf '0.42 0.51 0.10 2/1234 5678\n' >"$SELFTEST_DIR/loadavg"

	rc=0
	out=$(CALIB_GATE_LOADAVG_FILE="$SELFTEST_DIR/loadavg" \
		PATH="$SELFTEST_DIR/bin:$PATH" \
		bash "$SELFTEST_SELF" --provenance calgate-stub 2>&1) || rc=$?
	self_check "provenance probes a stub binary" 0 "$rc" "$out" \
		'PROVENANCE binary=calgate-stub' || return 1
	self_check "provenance records the version output" 0 "$rc" "$out" \
		'version="calgate-stub 1.2.3"'
}

self_test_golden() {
	local golden rc err

	golden="$(dirname "$SELFTEST_SELF")/testdata/calibration-gate-fail-message.golden"
	if [[ ! -f "$golden" ]]; then
		echo "  ✗ FAIL: golden missing: $golden"
		return 1
	fi

	printf '0.42 9.90 4.00 2/1234 5678\n' >"$SELFTEST_DIR/loadavg"
	rc=0
	err=$(CALIB_GATE_LOADAVG_FILE="$SELFTEST_DIR/loadavg" \
		bash "$SELFTEST_SELF" 2>&1 1>/dev/null) || rc=$?
	if [[ "$rc" != 1 ]]; then
		echo "  ✗ FAIL: golden run expected rc=1, got $rc"
		return 1
	fi

	local normalized
	normalized=$(printf '%s\n' "$err" | sed -E '/^  \(load1=/! s/^  .*/  <UPTIME>/')
	if ! diff -u "$golden" <(printf '%s\n' "$normalized"); then
		echo "  ✗ FAIL: FAIL-message shape drifted from the golden"
		return 1
	fi
	echo "  ✓ PASS: FAIL-message shape matches the golden"
}

self_test() {
	SELFTEST_SELF="$(cd "$(dirname "$0")" && pwd)/$(basename "$0")"
	SELFTEST_DIR="$(mktemp -d)"
	trap 'rm -rf "$SELFTEST_DIR"' RETURN

	local fails=0
	self_test_gate || fails=$((fails + 1))
	self_test_provenance || fails=$((fails + 1))
	self_test_golden || fails=$((fails + 1))

	return "$fails"
}

if [[ "$SELFTEST_MODE" == "1" ]]; then
	if self_test; then
		echo "calibration-gate self-test passed."
		exit 0
	fi
	echo "calibration-gate self-test FAILED."
	exit 1
fi

if [[ "${CI:-}" == "true" ]]; then
	echo "calibration-gate: CI environment — load gate is informational only (load1=${LOAD1}, load5=${LOAD5})"
elif [[ "${CALIB_GATE_REQUIRED:-1}" == "0" ]]; then
	echo "calibration-gate: WARN-ONLY override (CALIB_GATE_REQUIRED=0) — load1=${LOAD1}, load5=${LOAD5}, ceiling=${MAX_LOAD}"
else
	FAILED_LOAD=""
	if over_ceiling "$LOAD1" load1 || over_ceiling "$LOAD5" load5; then
		cat >&2 <<EOF
calibration-gate: LOAD GATE FAILED — ${FAILED_LOAD} >= ceiling ${MAX_LOAD}
  (load1=${LOAD1}, load5=${LOAD5}).
  ${UPTIME}
Benching now would produce load-skewed numbers (the 2026-09-11 SearchQuery
failure mode). load5 over ceiling = a burst is still draining; wait for a
quiet window and re-run, or set an explicit, justified ceiling: --max-load N.
To inspect without gating:
CALIB_GATE_REQUIRED=0 scripts/calibration-gate.sh.
EOF
		exit 1
	fi

	echo "calibration-gate: PASS — load1=${LOAD1}, load5=${LOAD5} < ${MAX_LOAD}"
fi

# Provenance line(s) for the measurement record. Binary store paths pin the
# exact nix derivation in use; the version probe output pins what the binary
# claims about itself (the 09-11 run cited a version from prior docs instead).
echo "PROVENANCE ${WHEN} loadavg=\"${LOADAVG}\""
for bin in "${PROVENANCE_BINS[@]}"; do
	path=$(command -v "$bin" 2>/dev/null || echo "not-on-PATH")
	ver=$("$bin" version 2>/dev/null | head -1 || true)
	if [[ -z "$ver" ]]; then
		ver=$("$bin" --version 2>/dev/null | head -1 || echo "version-probe-failed")
	fi
	echo "PROVENANCE binary=${bin} path=\"${path}\" version=\"${ver}\""
done

exit 0
