#!/usr/bin/env bash
# test-benchmark-regression.sh — fixture tests for benchmark-regression.sh.
#
# Pins the gate's behavior with pre-computed `go test -bench` fixtures
# (no real benchmarks): median computation over -count>1 samples, the
# regression threshold, the save-after-compare ordering (a --save run must
# compare against the PRE-save baseline, never against itself), and the
# informational (non-failing) handling of vanished/new benchmarks.
#
# Run: scripts/test-benchmark-regression.sh   (exits non-zero on failure)

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
GATE="$SCRIPT_DIR/benchmark-regression.sh"

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

failures=0
check() {
	local name="$1" expected="$2" actual="$3"
	if [[ "$actual" == "$expected" ]]; then
		echo "PASS: $name"
	else
		echo "FAIL: $name (expected exit $expected, got $actual)"
		failures=$((failures + 1))
	fi
}

# Fixture line: BenchmarkX-8  <iters>  <ns/op> ns/op
fixture() { # fixture FILE NS_PER_OP...
	local file="$1"
	shift

	local n
	for n in "$@"; do
		printf 'BenchmarkGate-8  1000000  %d ns/op\n' "$n" >>"$file"
	done
}

# --- 1. medians: -count=3 odd sample count picks the middle value ---
: >"$tmp/base1"
fixture "$tmp/base1" 100 100 100
: >"$tmp/cur1"
fixture "$tmp/cur1" 100 110 100
"$GATE" --baseline "$tmp/base1" --current "$tmp/cur1" >/dev/null 2>&1
check "median ignores a single slow sample (3 samples)" 0 $?

# --- 2. medians: -count=4 even sample count averages the middle pair ---
: >"$tmp/base2"
fixture "$tmp/base2" 100 100 100 100
: >"$tmp/cur2"
fixture "$tmp/cur2" 100 100 150 200
"$GATE" --baseline "$tmp/base2" --current "$tmp/cur2" >/dev/null 2>&1
# median current = (100+150)/2 = 125 vs base 100 = +25% — exactly at the
# default threshold is NOT a regression (strict >)
check "even sample median, +25% exactly is not a regression" 0 $?

# --- 3. threshold: regression above the threshold fails ---
: >"$tmp/cur3"
fixture "$tmp/cur3" 150 150 150
"$GATE" --baseline "$tmp/base2" --current "$tmp/cur3" >/dev/null 2>&1
check "median +50% regresses past 25% threshold" 1 $?

# --- 4. custom threshold widens the pass band ---
"$GATE" --baseline "$tmp/base2" --current "$tmp/cur3" --threshold 60 >/dev/null 2>&1
check "+50% passes under a 60% threshold" 0 $?

# --- 5. --save compares against the PRE-save baseline, then overwrites it ---
: >"$tmp/base5"
fixture "$tmp/base5" 100 100
: >"$tmp/cur5"
fixture "$tmp/cur5" 300 300
"$GATE" --baseline "$tmp/save-target" --current "$tmp/cur5" --save "$tmp/save-target" >/dev/null 2>&1
# First run against a MISSING baseline: save-only, must exit 0.
check "save against missing baseline is save-only and passes" 0 $?
[[ -f "$tmp/save-target" ]] || true
if [[ -f "$tmp/save-target" ]] && grep -q "300 ns/op" "$tmp/save-target"; then
	echo "PASS: --save wrote the raw current results"
else
	echo "FAIL: --save did not write the raw current results"
	failures=$((failures + 1))
fi

# --- 6. the save+compare self-comparison trap: baseline snapshot before save ---
: >"$tmp/base6"
fixture "$tmp/base6" 100 100 100
: >"$tmp/cur6"
fixture "$tmp/cur6" 300 300 300
"$GATE" --baseline "$tmp/base6" --current "$tmp/cur6" --save "$tmp/base6" >/dev/null 2>&1
# A +200% regression must still FAIL even though --save overwrote the baseline.
check "save does not mask a regression (compare-before-save)" 1 $?

# --- 7. vanished and new benchmarks are informational only ---
: >"$tmp/base7"
fixture "$tmp/base7" 100 100
printf 'BenchmarkGone-8  1000000  100 ns/op\n' >>"$tmp/base7"
: >"$tmp/cur7"
fixture "$tmp/cur7" 100 100
printf 'BenchmarkNew-8  1000000  100 ns/op\n' >>"$tmp/cur7"
"$GATE" --baseline "$tmp/base7" --current "$tmp/cur7" >/dev/null 2>&1
check "vanished/new benchmark names never fail the gate" 0 $?

# --- 8. non-benchmark noise lines are ignored ---
: >"$tmp/cur8"
{
	echo "goos: linux"
	echo "PASS"
	echo "ok  	stack/bench	1.2s"
	fixture "$tmp/cur8" 100 100
} >/dev/null
"$GATE" --baseline "$tmp/base7" --current "$tmp/cur8" >/dev/null 2>&1
check "go test noise lines do not break parsing" 0 $?

# --- 9. --save writes a titled provenance header the parser still ignores ---
# (2026-09-11 protocol: the 02:40 baseline refresh landed during a load-ramp
# with nothing in the file saying so — re-pins must be titled.)
"$GATE" --baseline "$tmp/base7" --current "$tmp/cur8" --save "$tmp/save-titled" >/dev/null 2>&1
check "titled save run passes" 0 $?
if grep -q '^# benchmark baseline — re-pinned' "$tmp/save-titled" &&
	grep -q '^# .*load average' "$tmp/save-titled" &&
	grep -q '100 ns/op' "$tmp/save-titled"; then
	echo "PASS: --save wrote the provenance header plus raw results"
else
	echo "FAIL: --save header missing or results lost"
	failures=$((failures + 1))
fi
# The header must not break a subsequent comparison against the saved file.
"$GATE" --baseline "$tmp/save-titled" --current "$tmp/cur8" >/dev/null 2>&1
check "header-prefixed baseline still parses" 0 $?

# --- 10-16. benchkit noise gate + load gate (2026-09-16 per-metric CI gating) ---
# noise_json writes a cqrs-bench `--format json` result fixture; entries are
# "name:cov" pairs (cov as a fraction, as MetricVariation serializes it).
noise_json() { # noise_json FILE [name:cov]...
	local file="$1"
	shift

	local entries="" e name cov
	for e in "$@"; do
		name="${e%%:*}"
		cov="${e#*:}"
		entries+="${entries:+,}{\"name\":\"$name\",\"unit\":\"ns/op\",\"cov\":$cov,\"reliable\":false}"
	done

	printf '{"backend":"sqlite","metricVariation":[%s]}' "$entries" >"$file"
}

noise_json "$tmp/nj-noisy-headline" "write_throughput:0.14" "load_p50_ns:0.03"
"$GATE" --baseline "$tmp/base1" --current "$tmp/cur1" --noise-current "$tmp/nj-noisy-headline" >/dev/null 2>&1
check "noisy HEADLINE metric fails the noise gate" 1 $?

noise_json "$tmp/nj-noisy-tail" "gc_total_pause_ns:0.40" "write_throughput:0.02"
"$GATE" --baseline "$tmp/base1" --current "$tmp/cur1" --noise-current "$tmp/nj-noisy-tail" >/dev/null 2>&1
check "noisy non-headline metric only warns (gate passes)" 0 $?

noise_json "$tmp/nj-stable" "write_throughput:0.02" "load_p50_ns:0.03"
"$GATE" --baseline "$tmp/base1" --current "$tmp/cur1" --noise-current "$tmp/nj-stable" >/dev/null 2>&1
check "all-stable metrics pass the noise gate" 0 $?

printf 'this is not json' >"$tmp/nj-broken"
"$GATE" --baseline "$tmp/base1" --current "$tmp/cur1" --noise-current "$tmp/nj-broken" >/dev/null 2>&1
check "unparseable noise result fails loudly" 1 $?

printf '{"backend":"sqlite"}' >"$tmp/nj-novariation"
"$GATE" --baseline "$tmp/base1" --current "$tmp/cur1" --noise-current "$tmp/nj-novariation" >/dev/null 2>&1
check "result without metricVariation (repeat < 2) fails" 1 $?

noise_json "$tmp/nj-twelve" "write_throughput:0.12"
"$GATE" --baseline "$tmp/base1" --current "$tmp/cur1" --noise-current "$tmp/nj-twelve" >/dev/null 2>&1
check "CoV 12% is noisy at the default 10% threshold" 1 $?

"$GATE" --baseline "$tmp/base1" --current "$tmp/cur1" --noise-current "$tmp/nj-twelve" \
	--noise-threshold 15 >/dev/null 2>&1
check "CoV 12% passes under a custom 15% threshold" 0 $?

# Load gate (calibration-gate semantics, fixture-injectable probe): a loud
# machine aborts BEFORE any benchmarking; --skip-load-gate overrides.
LOUD_LOADAVG="$tmp/loadavg-loud"
printf '99.0 88.0 1.00 2/1234 5678\n' >"$LOUD_LOADAVG"
QUIET_LOADAVG="$tmp/loadavg-quiet"
printf '0.50 0.40 1.00 2/1234 5678\n' >"$QUIET_LOADAVG"

BENCH_GATE_LOADAVG_FILE="$LOUD_LOADAVG" "$GATE" --noise-only \
	--noise-current "$tmp/nj-stable" >/dev/null 2>&1
check "load gate aborts an oversubscribed machine (noise-only)" 1 $?

BENCH_GATE_LOADAVG_FILE="$LOUD_LOADAVG" "$GATE" --noise-only \
	--noise-current "$tmp/nj-stable" --skip-load-gate >/dev/null 2>&1
check "--skip-load-gate overrides the loud machine abort" 0 $?

BENCH_GATE_LOADAVG_FILE="$QUIET_LOADAVG" "$GATE" --noise-only \
	--noise-current "$tmp/nj-stable" >/dev/null 2>&1
check "quiet machine runs the noise gate through (noise-only)" 0 $?

BENCH_GATE_LOADAVG_FILE="$LOUD_LOADAVG" "$GATE" --noise-only \
	--noise-current "$tmp/nj-noisy-headline" --skip-load-gate >/dev/null 2>&1
check "noise-only still fails on a noisy headline" 1 $?

echo ""
if [[ $failures -gt 0 ]]; then
	echo "FAIL: $failures fixture test(s) failed"
	exit 1
fi
echo "PASS: all benchmark-regression fixture tests"
