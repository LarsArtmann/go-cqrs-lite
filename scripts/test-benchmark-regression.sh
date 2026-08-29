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

echo ""
if [[ $failures -gt 0 ]]; then
	echo "FAIL: $failures fixture test(s) failed"
	exit 1
fi
echo "PASS: all benchmark-regression fixture tests"
