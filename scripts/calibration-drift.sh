#!/usr/bin/env bash
# calibration-drift.sh — compares fresh per-pattern calibration benches
# against the shipped ReadCosts constants (docs/benchmarks/calibration-2026-08-30.md).
#
# Warns (::warning annotation) when the fresh median drifts >25% from the
# constant; exits nonzero only at >100% (2x — hardware change or regression).
# Exit codes: 0 = within tolerance/warn, 1 = drift >= threshold*4 (hard fail),
#             2 = usage/environment error.
#
# Usage: scripts/calibration-drift.sh [--module DIR ...]
# Default modules: metaengine/{badger,bbolt,pebble,sqlite}engine (in-memory,
# no DSN needed). count=3 per bench, run 1 discarded, median of the rest.
set -euo pipefail

THRESHOLD_WARN=25   # percent
THRESHOLD_FAIL=100  # percent
COUNT=3

# expected: engine_module|bench_suffix|pattern_label|expected_ns_per_unit|units_per_op
# Constants mirror the committed Profile() values (ADR-0133; 2026-08-30 baseline).
EXPECTED=$(cat <<'EOT'
badgerengine|Get|point_lookup|1100|1
badgerengine|_FilteredScan|filtered_scan|650|10000
badgerengine|_CounterScan|aggregate|165|1000
badgerengine|_FullScan|scan|630|10000
bboltengine|Get|point_lookup|750|1
bboltengine|_FilteredScan|filtered_scan|620|10000
bboltengine|_CounterScan|aggregate|100|1000
bboltengine|_FullScan|scan|660|10000
pebbleengine|Get|point_lookup|700|1
pebbleengine|_FilteredScan|filtered_scan|830|10000
pebbleengine|_CounterScan|aggregate|125|1000
pebbleengine|_FullScan|scan|700|10000
sqliteengine|_PointLookup|point_lookup|3100|1
sqliteengine|_FilteredScan|filtered_scan|1080|10000
sqliteengine|_CounterScan|aggregate|530|1000
sqliteengine|_FullScan|scan|1240|10000
EOT
)

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
FAILED=0

# median of a numerically sorted list
median() {
	local -a vals=("$@")
	local n=${#vals[@]}
	if (( n % 2 == 1 )); then
		echo "${vals[n / 2]}"
	else
		awk -v a="${vals[n / 2 - 1]}" -v b="${vals[n / 2]}" 'BEGIN { printf "%.0f", (a + b) / 2 }'
	fi
}

while IFS='|' read -r mod suffix label expected units; do
	[ -z "$mod" ] && continue

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

	echo "=== $mod $label (expect ~${expected} ns/${units:+row}) ==="
	log="$(mktemp)"

	if ! (cd "$dir" && GOWORK=off go test -tags "goexperiment.jsonv2" \
		-run '^$' -bench "^${bench}" -benchmem -count "$COUNT" -timeout 20m ./... > "$log" 2>&1); then
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
	drift="$(awk -v m="$med_ns" -v u="$units" -v e="$expected" 'BEGIN { printf "%.1f", (m / u - e) * 100 / e }')"
	abs_drift="${drift%%.*}"
	abs_drift="${abs_drift#-}"

	printf '  fresh median: %s ns/op -> %s ns/unit | expected %s | drift %s%%\n' \
		"$med_ns" "$med_units" "$expected" "$drift"

	if (( abs_drift >= THRESHOLD_FAIL )); then
		echo "::error::$mod $label drifted ${drift}% (>${THRESHOLD_FAIL}%): recalibrate or fix the regression"
		FAILED=1
	elif (( abs_drift > THRESHOLD_WARN )); then
		echo "::warning::$mod $label drifted ${drift}% (>${THRESHOLD_WARN}%): consider a quiet-window recalibration"
	fi

	rm -f "$log"
done <<< "$EXPECTED"

exit "$FAILED"
