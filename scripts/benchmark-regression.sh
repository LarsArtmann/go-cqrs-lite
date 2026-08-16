#!/usr/bin/env bash
# benchmark-regression.sh — the ONE benchmark regression gate.
#
# Compares the median ns/op per benchmark name between a baseline and the
# current results and exits 1 when any benchmark regresses more than the
# threshold. `go test -bench -count=N` emits N result lines per benchmark;
# the median absorbs the noise so a single slow sample cannot fail (or hide)
# a regression — per-line comparison breaks on -count>1 output.
#
# Usage:
#   ./scripts/benchmark-regression.sh                                  # run gate set, compare vs committed baseline
#   ./scripts/benchmark-regression.sh --current results.txt            # compare pre-computed output (CI)
#   ./scripts/benchmark-regression.sh --save benchmarks/benchmark-baseline.txt  # refresh local baseline
#
# Options:
#   --baseline FILE   baseline results file (default: benchmarks/benchmark-baseline.txt)
#   --current FILE    pre-computed `go test -bench` output (skips running benchmarks)
#   --save FILE       also write the raw current results to FILE (baseline refresh)
#   --threshold PCT   allowed median regression in percent (default: 25)
#   --bench REGEX     go test -bench pattern (default: the CI gate set)
#   --dir PATH        package dir to benchmark in (default: stack/bench)
#   --count N         benchmark repetitions feeding the median (default: 5)
#   --benchtime T     go test -benchtime value (default: 10x)
#
# Baselines are hardware-specific: only compare numbers from the same machine
# or runner class. CI compares against its own `benchmark-baseline` artifact;
# the committed file is for local runs on this machine only.

set -euo pipefail

BASELINE="benchmarks/benchmark-baseline.txt"
CURRENT_INPUT=""
SAVE=""
THRESHOLD="25"
BENCH='BenchmarkFullPipeline_Memory|BenchmarkBenchkitSuite_Memory$'
BENCH_DIR="stack/bench"
COUNT="5"
BENCHTIME="10x"

while [[ $# -gt 0 ]]; do
	case "$1" in
	--baseline)
		BASELINE="$2"
		shift 2
		;;
	--current)
		CURRENT_INPUT="$2"
		shift 2
		;;
	--save)
		SAVE="$2"
		shift 2
		;;
	--threshold)
		THRESHOLD="$2"
		shift 2
		;;
	--bench)
		BENCH="$2"
		shift 2
		;;
	--dir)
		BENCH_DIR="$2"
		shift 2
		;;
	--count)
		COUNT="$2"
		shift 2
		;;
	--benchtime)
		BENCHTIME="$2"
		shift 2
		;;
	*)
		echo "Unknown flag: $1" >&2
		exit 1
		;;
	esac
done

# medians FILE — prints "<name> <median_ns/op> <samples>" per benchmark.
# Accepts raw `go test -bench` output; extra lines (PASS, ok, benchmarks
# without an ns/op column) are ignored.
medians() {
	awk '
	function medianOf(name, cnt,   i, j, v, mid, s) {
		for (i = 1; i <= cnt; i++) s[i] = vals[name, i]
		for (i = 2; i <= cnt; i++) {
			v = s[i]
			for (j = i - 1; j >= 1 && s[j] > v; j--) s[j + 1] = s[j]
			s[j + 1] = v
		}
		mid = int((cnt + 1) / 2)
		return (cnt % 2) ? s[mid] : (s[mid] + s[mid + 1]) / 2
	}
	$1 ~ /^Benchmark/ && $1 ~ /-[0-9]+$/ && $2 ~ /^[0-9]+$/ && $3 ~ /^[0-9.]+$/ && $4 == "ns/op" {
		name = $1
		sub(/-[0-9]+$/, "", name)
		cnt[name]++
		vals[name, cnt[name]] = $3 + 0
	}
	END {
		for (name in cnt) printf "%s %.1f %d\n", name, medianOf(name, cnt[name]), cnt[name]
	}
	' "$1"
}

current_file=$(mktemp)
trap 'rm -f "$current_file"' EXIT

# Snapshot the baseline BEFORE --save can overwrite it — otherwise a
# save+compare run compares current against itself and always passes.
had_baseline=false
if [[ -f "$BASELINE" ]] && [[ -n "$(medians "$BASELINE")" ]]; then
	base_medians=$(medians "$BASELINE" | LC_ALL=C sort)
	had_baseline=true
fi

if [[ -n "$CURRENT_INPUT" ]]; then
	cp "$CURRENT_INPUT" "$current_file"
else
	echo "==> Running gate benchmarks ($BENCH_DIR, count=$COUNT, benchtime=$BENCHTIME)"
	(
		cd "$BENCH_DIR"
		GOTOOLCHAIN=auto GOEXPERIMENT=jsonv2 go test -tags goexperiment.jsonv2 \
			-run='^$' -bench="$BENCH" -benchmem \
			-benchtime="$BENCHTIME" -count="$COUNT" -timeout 10m 2>&1
	) | tee "$current_file"
fi

compare_status=0
if [[ "$had_baseline" == true ]]; then
	cur_medians=$(medians "$current_file" | LC_ALL=C sort)

	# Benchmarks that vanished or appeared are informational, never a failure.
	comm -23 <(printf '%s\n' "$base_medians" | awk '{print $1}') \
		<(printf '%s\n' "$cur_medians" | awk '{print $1}') |
		grep '^.' | sed 's/^/  removed from current: /' || true
	comm -13 <(printf '%s\n' "$base_medians" | awk '{print $1}') \
		<(printf '%s\n' "$cur_medians" | awk '{print $1}') |
		grep '^.' | sed 's/^/  new in current: /' || true

	echo ""
	echo "==> Comparing medians (threshold: ${THRESHOLD}%)"

	LC_ALL=C join <(printf '%s\n' "$base_medians") <(printf '%s\n' "$cur_medians") |
		awk -v t="$THRESHOLD" '{
			name = $1
			base = $2 + 0
			cur = $4 + 0
			if (base <= 0 || cur <= 0) next
			pct = (cur - base) * 100 / base
			if (pct > t) {
				printf "REGRESSION  %-55s %12.1f → %12.1f ns/op  (+%.1f%%)\n", name, base, cur, pct
				regressions++
			} else if (pct < -5) {
				improvements++
			} else {
				stable++
			}
		}
		END {
			printf "\nSummary: %d regression(s), %d improvement(s), %d stable\n", regressions + 0, improvements + 0, stable + 0
			exit (regressions > 0) ? 1 : 0
		}' || compare_status=$?
else
	echo "WARN: no parseable baseline at $BASELINE — skipping comparison (save-only run)."
fi

# --save runs AFTER the comparison and regardless of its outcome: re-baselining
# after an intentional perf change must overwrite even a "regressed" baseline.
if [[ -n "$SAVE" ]]; then
	mkdir -p "$(dirname "$SAVE")"
	cp "$current_file" "$SAVE"
	echo "==> Current results saved to $SAVE"
fi

if [[ $compare_status -ne 0 ]]; then
	echo "FAIL: benchmark regression above ${THRESHOLD}% threshold"
	exit 1
fi

echo "PASS: no regression above ${THRESHOLD}% threshold"
