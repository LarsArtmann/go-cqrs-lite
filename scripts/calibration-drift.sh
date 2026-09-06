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
# Usage: scripts/calibration-drift.sh [--module DIR ...]
# Default modules: metaengine/{badger,bbolt,pebble,sqlite}engine (in-memory,
# no DSN needed). count=3 per bench, run 1 discarded, median of the rest.
set -euo pipefail

THRESHOLD_WARN=25  # percent
THRESHOLD_FAIL=100 # percent
COUNT=3

ALL_MODULES=(badgerengine bboltengine pebbleengine sqliteengine)

MODULES=("$@")
if [ "${#MODULES[@]}" -eq 0 ]; then
	MODULES=("${ALL_MODULES[@]}")
fi

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

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

	expected="${CALIB[$mod|$label]:-}"
	if [ -z "$expected" ]; then
		echo "::error::no shipped constant for $mod $label"
		FAILED=1
		continue
	fi

	expected_ns="${expected%%|*}"
	units="${expected##*|}"

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

	echo "=== $mod $label (shipped ~${expected_ns} ns/row) ==="
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

	if ((abs_drift >= THRESHOLD_FAIL)); then
		echo "::error::$mod $label drifted ${drift}% (>${THRESHOLD_FAIL}%): recalibrate or fix the regression"
		FAILED=1
	elif ((abs_drift > THRESHOLD_WARN)); then
		echo "::warning::$mod $label drifted ${drift}% (>${THRESHOLD_WARN}%): consider a quiet-window recalibration"
	fi

	rm -f "$log"
done <<<"$ROWS"

exit "$FAILED"
