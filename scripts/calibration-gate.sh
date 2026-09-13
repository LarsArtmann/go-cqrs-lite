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
#
# CI (CI=true, e.g. the benchmarks.yml drift job) never aborts: shared
# runner load is not the calibration host's load; the gate exists to
# protect LOCAL baseline-producing runs.
#
# Exit codes: 0 = gate passed (or warn-only), 1 = load above ceiling, 2 = usage.
set -uo pipefail

MAX_LOAD="5"
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
		sed -n '2,22p' "$0"
		exit 0
		;;
	*)
		echo "calibration-gate: unknown flag: $1 (see --help)" >&2
		exit 2
		;;
	esac
done

load1() { cut -d' ' -f1 /proc/loadavg; }
load5() { cut -d' ' -f2 /proc/loadavg; }

LOAD1=$(load1)
LOAD5=$(load5)
LOADAVG=$(cat /proc/loadavg 2>/dev/null || echo "unknown")
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
