#!/usr/bin/env bash
# calibration-drift.sh — compares fresh per-pattern calibration benches
# against the SHIPPED ReadCosts constants, read LIVE from each engine's
# Profile() via TestCalibrationConstantsDump (CALIB_DUMP=1). The profile is
# the single source of truth: the gate always compares against exactly the
# values the planner routes on (the historical measurement record lives in
# docs/benchmarks/calibration-2026-08-30.md).
#
# Warns (::warning annotation) when the fresh median drifts >25% from the
# shipped constant; exits nonzero only at >100% (2x — hardware change or
# regression). Exit codes: 0 = within tolerance/warn, 1 = drift >= threshold
# (hard fail), 2 = usage/environment error.
#
# CI mode (--baseline FILE): compare against a persisted CI-baseline artifact
# (`module|label|ns_per_unit` rows recorded on the same runner class) instead
# of the absolute shipped constants — shared-runner noise routinely pushes
# rows past 100% of the shipped constant without being drift, so CI compares
# apples-to-apples with its own previous run (same pattern as benchmarks.yml's
# baseline artifact). Regenerate the artifact with --write-baseline FILE.
#
# Usage: scripts/calibration-drift.sh [--baseline FILE | --write-baseline FILE] [MODULE ...]
# Default modules: metaengine/{badger,bbolt,pebble,sqlite}engine (in-memory,
# no DSN needed). count=3 per bench, run 1 discarded, median of the rest.
set -euo pipefail

THRESHOLD_WARN=25  # percent
THRESHOLD_FAIL=100 # percent
COUNT=3

ALL_MODULES=(badgerengine bboltengine pebbleengine sqliteengine)

BASELINE_FILE=""
WRITE_BASELINE_FILE=""

MODULES=()
while [ $# -gt 0 ]; do
	case "$1" in
	--baseline)
		BASELINE_FILE="${2:?--baseline requires a file argument}"
		shift 2
		;;
	--write-baseline)
		WRITE_BASELINE_FILE="${2:?--write-baseline requires a file argument}"
		shift 2
		;;
	*)
		MODULES+=("$1")
		shift
		;;
	esac
done

if [ "${#MODULES[@]}" -eq 0 ]; then
	MODULES=("${ALL_MODULES[@]}")
fi

if [ -n "$BASELINE_FILE" ] && [ -n "$WRITE_BASELINE_FILE" ]; then
	echo "::error::--baseline and --write-baseline are mutually exclusive" >&2
	exit 2
fi

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

# Load gate (protocol §6, added 2026-09-11): drift medians are load-sensitive
# on the shared host — a compile storm skews them past the warn threshold for
# reasons that are not drift. Abort before benching; ceiling defaults wider
# than baseline runs (8 vs 5) because drift compares at 25%, and CI is exempt
# (shared-runner load is not this host's load). Override: CALIB_MAX_LOAD=N.
if ! bash "$REPO_ROOT/scripts/calibration-gate.sh" --max-load "${CALIB_MAX_LOAD:-8}"; then
	echo "::error::calibration-drift aborted by the load gate — re-run in a quiet window" >&2
	exit 2
fi

# TMPDIR filesystem gate (2026-09-15): bbolt benches need tmpfs-backed temp
# dirs — on CoW filesystems (btrfs/ZFS) the mmap+fsync workload times out
# (2026-09-04 gotcha) and poisons the medians. Refuse loudly instead of
# emitting bogus drift numbers. Override: CALIB_ALLOW_COW=1. Test hook:
# CALIB_FAKE_TMPFS_TYPE forces the detected type (harness only).
tmpfs_type="${CALIB_FAKE_TMPFS_TYPE:-$(stat -f -c %T "${TMPDIR:-/tmp}")}"
case "$tmpfs_type" in
btrfs | zfs)
	if [ "${CALIB_ALLOW_COW:-0}" != "1" ]; then
		echo "::error::TMPDIR (${TMPDIR:-/tmp}) is on $tmpfs_type (CoW) — calibration benches need tmpfs (export TMPDIR=/tmp) or CALIB_ALLOW_COW=1" >&2
		exit 2
	fi
	echo "::warning::TMPDIR on $tmpfs_type (CoW) — proceeding via CALIB_ALLOW_COW=1" >&2
	;;
esac

declare -A CALIB # key: "<module>|<label>" → "<expected_ns_per_unit>|<units_per_op>"

# dump_constants runs the module's dump test and indexes CALIB|<label> lines
# into the CALIB map. Lines arrive as `...: CALIB|label|value|units`.
dump_constants() {
	local mod=$1
	local dir="$REPO_ROOT/metaengine/$mod"
	local out

	if ! out="$(cd "$dir" && CALIB_DUMP=1 GOWORK=off go test \
		-tags "goexperiment.jsonv2" -run '^TestCalibrationConstantsDump$' \
		-count=1 -v ./... 2>/dev/null | grep -o 'CALIB|[a-z_]*|[0-9.]*|[0-9]*' || true)"; then
		out=""
	fi

	if [ -z "$out" ]; then
		echo "::error::no calibration constants dumped for $mod (is TestCalibrationConstantsDump present?)"
		exit 2
	fi

	while IFS='|' read -r _ label value units; do
		CALIB["$mod|$label"]="$value|$units"
	done <<<"$out"
}

for mod in "${MODULES[@]}"; do
	dump_constants "$mod"
done

# CI-baseline artifact: rows `module|label|ns_per_unit`. In --baseline mode
# these rows replace the shipped constants as the comparison target.
declare -A BASELINE # key: "<module>|<label>" → ns_per_unit

if [ -n "$BASELINE_FILE" ]; then
	if [ ! -f "$BASELINE_FILE" ]; then
		echo "::error::baseline artifact not found: $BASELINE_FILE" >&2
		exit 2
	fi

	baseline_rows=0
	while IFS='|' read -r bmod blabel bns; do
		[ -z "$bmod" ] && continue
		BASELINE["$bmod|$blabel"]="$bns"
		baseline_rows=$((baseline_rows + 1))
	done <"$BASELINE_FILE"

	if [ "$baseline_rows" -eq 0 ]; then
		echo "::error::baseline artifact has no rows: $BASELINE_FILE" >&2
		exit 2
	fi
fi

if [ -n "$WRITE_BASELINE_FILE" ]; then
	: >"$WRITE_BASELINE_FILE"
fi

# median of a numerically sorted list
median() {
	local -a vals=("$@")
	local n=${#vals[@]}
	if ((n % 2 == 1)); then
		echo "${vals[n / 2]}"
	else
		awk -v a="${vals[n / 2 - 1]}" -v b="${vals[n / 2]}" 'BEGIN { printf "%.0f", (a + b) / 2 }'
	fi
}

# bench row: module|bench_suffix|pattern_label (bench suffix "" = Get)
ROWS=$(
	cat <<'EOT'
badgerengine|Get|point_lookup
badgerengine|_FilteredScan|filtered_scan
badgerengine|_CounterScan|aggregate
badgerengine|_FullScan|scan
bboltengine|Get|point_lookup
bboltengine|_FilteredScan|filtered_scan
bboltengine|_CounterScan|aggregate
bboltengine|_FullScan|scan
pebbleengine|Get|point_lookup
pebbleengine|_FilteredScan|filtered_scan
pebbleengine|_CounterScan|aggregate
pebbleengine|_FullScan|scan
sqliteengine|_PointLookup|point_lookup
sqliteengine|_FilteredScan|filtered_scan
sqliteengine|_CounterScan|aggregate
sqliteengine|_FullScan|scan
EOT
)

FAILED=0

while IFS='|' read -r mod suffix label; do
	[ -z "$mod" ] && continue

	skip=true
	for m in "${MODULES[@]}"; do
		[ "$m" = "$mod" ] && skip=false
	done

	$skip && continue

	pair="${CALIB["$mod|$label"]:-}"
	if [ -z "$pair" ]; then
		echo "::error::no shipped constant for $mod $label"
		FAILED=1
		continue
	fi

	expected_ns="${pair%%|*}"
	units="${pair##*|}"

	if [ -n "$BASELINE_FILE" ]; then
		baseline_ns="${BASELINE["$mod|$label"]:-}"
		if [ -z "$baseline_ns" ]; then
			echo "::error::no baseline row for $mod $label — regenerate the artifact with --write-baseline"
			FAILED=1
			continue
		fi
		echo "=== $mod $label (baseline ${baseline_ns} ns/unit, shipped ~${expected_ns}) ==="
	else
		echo "=== $mod $label (shipped ~${expected_ns} ns/row) ==="
	fi

	dir="$REPO_ROOT/metaengine/$mod"
	# Engine-name casing fixups (Badger/Bbolt/Pebble/SQLite bench prefixes).
	case "$mod" in
	badgerengine) bench="BenchmarkCalibration_Badger" ;;
	bboltengine) bench="BenchmarkCalibration_Bbolt" ;;
	pebbleengine) bench="BenchmarkCalibration_Pebble" ;;
	sqliteengine) bench="BenchmarkCalibration_SQLite" ;;
	esac

	if [ "$suffix" != "Get" ] && [ -n "$suffix" ]; then
		bench="${bench}${suffix}\$"
	else
		bench="${bench}Get\$"
	fi

	log="$(mktemp)"

	if ! (cd "$dir" && GOWORK=off go test -tags "goexperiment.jsonv2" \
		-run '^$' -bench "^${bench}" -benchmem -count "$COUNT" -timeout 20m ./... >"$log" 2>&1); then
		echo "::error::calibration bench failed for $mod $label (see $log)"
		FAILED=1
		continue
	fi

	# Collect ns/op values (col 3), sort numerically, drop run 1, take median.
	mapfile -t ns < <(grep -E '^Benchmark' "$log" | awk '{print $3}' | sort -n)
	if [ "${#ns[@]}" -lt 2 ]; then
		echo "::error::no benchmark output for $mod $label"
		FAILED=1
		continue
	fi

	med_ns="$(median "${ns[@]:1}")" # discard run 1 (cold)
	med_units="$(awk -v m="$med_ns" -v u="$units" 'BEGIN { printf "%.0f", m / u }')"
	drift="$(awk -v m="$med_ns" -v u="$units" -v e="$expected_ns" 'BEGIN { printf "%.1f", (m / u - e) * 100 / e }')"
	abs_drift="${drift%%.*}"
	abs_drift="${abs_drift#-}"

	printf '  fresh median: %s ns/op -> %s ns/unit | shipped %s | drift %s%%\n' \
		"$med_ns" "$med_units" "$expected_ns" "$drift"

	if [ -n "$WRITE_BASELINE_FILE" ]; then
		printf '%s|%s|%s\n' "$mod" "$label" "$med_units" >>"$WRITE_BASELINE_FILE"
		continue
	fi

	if [ -n "$BASELINE_FILE" ]; then
		base_drift="$(awk -v m="$med_units" -v e="$baseline_ns" 'BEGIN { printf "%.1f", (m - e) * 100 / e }')"
		base_abs="${base_drift%%.*}"
		base_abs="${base_abs#-}"
		printf '  baseline median: %s ns/unit | drift vs baseline %s%%\n' "$baseline_ns" "$base_drift"

		if ((base_abs >= THRESHOLD_FAIL)); then
			echo "::error::$mod $label drifted ${base_drift}% vs the CI baseline (>${THRESHOLD_FAIL}%): investigate this runner class"
			FAILED=1
		elif ((base_abs > THRESHOLD_WARN)); then
			echo "::warning::$mod $label drifted ${base_drift}% vs the CI baseline (>${THRESHOLD_WARN}%)"
		fi
	elif ((abs_drift >= THRESHOLD_FAIL)); then
		echo "::error::$mod $label drifted ${drift}% (>${THRESHOLD_FAIL}%): recalibrate or fix the regression"
		FAILED=1
	elif ((abs_drift > THRESHOLD_WARN)); then
		echo "::warning::$mod $label drifted ${drift}% (>${THRESHOLD_WARN}%): consider a quiet-window recalibration"
	fi

	rm -f "$log"
done <<<"$ROWS"

exit "$FAILED"
