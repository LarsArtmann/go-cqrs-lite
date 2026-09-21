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
#   ./scripts/benchmark-regression.sh --force-save ...             # save even after a noise-gate failure
#   ./scripts/benchmark-regression.sh --noise-only                     # run only the benchkit noise gate
#
# Options:
#   --baseline FILE   baseline results file (default: benchmarks/benchmark-baseline.txt)
#   --current FILE    pre-computed `go test -bench` output (skips running benchmarks)
#   --save FILE       also write the raw current results to FILE (baseline
#                     refresh). Refused when this run's noise gate FAILED
#                     (non-decision-grade) unless --force-save is passed.
#   --force-save      override the noise-gate save refusal (intentional
#                     re-baselines after a deliberate perf change)
#   --threshold PCT   allowed median regression in percent (default: 25)
#   --bench REGEX     go test -bench pattern (default: the CI gate set; overrides
#                     the default gate sets with a single stack/bench run)
#   --dir PATH        package dir to benchmark in (default: stack/bench; same
#                     single-set override semantics as --bench)
#   --count N         benchmark repetitions feeding the median (default: 5)
#   --benchtime T     go test -benchtime value (default: 100x — short samples
#                     of microsecond benchmarks skew badly under CPU steal from
#                     concurrent builds/LSP; CI passes its own 10x)
#   --max-load N      load-gate ceiling for load1/load5 (default: CPU count)
#   --skip-load-gate  do not abort on an oversubscribed machine
#   --noise-only      run only the benchkit noise gate, skip the go-test gate sets
#   --skip-noise-gate do not run the benchkit noise gate in live mode
#   --noise-current FILE
#                     pre-computed `cqrs-bench run --format json` result for the
#                     noise gate (skips running the noise benchmark)
#   --noise-backend B backend for the noise benchmark (default: sqlite)
#   --noise-profile P profile for the noise benchmark (default: dev)
#   --noise-repeat N  repeats feeding the noise gate's CoV (default: 5)
#   --noise-threshold PCT
#                     cross-run CoV percent above which a metric is noisy
#                     (default: 10 — benchkit.VariationThreshold)
#   --noise-headline LIST
#                     space-separated metric names whose noise FAILS the gate
#                     (default: "write_throughput write_p50_ns load_p50_ns";
#                     non-headline noisy metrics only warn). write_p99_ns was
#                     DEMOTED 2026-09-20: across 5 gate runs its CoV ranged
#                     11.5-54% — including a deep-quiet window (load 2.89)
#                     and --noise-repeat 7 — while the three stable metrics
#                     stayed under 10% in every run. Tail quantiles from
#                     ~100-iteration runs are not gateable; the median
#                     compare itself routes contention through throughput
#                     and p50 first.
#
# Known-unstable benchmarks (KNOWN_UNSTABLE table in-script): entries
# "NAME|reason|expiry" suppress >threshold flags as UNSTABLE-KNOWN (advisory,
# non-failing) until the expiry date. Used for bimodal samplers whose
# median-of-N flips modes on a quiet machine — the noise gate certifies the
# machine, not the benchmark's sampling adequacy.
#
# Baselines are hardware- and load-specific: only compare numbers from the
# same machine or runner class, measured on a quiet machine. CI compares
# against its own `benchmark-baseline` artifact; the committed file is for
# local runs on this machine only.
#
# Re-pin protocol (fold the load gate in, 2026-09-21): a local --save re-pin
# is only valid from a run where EVERY gate passed —
#   1. scripts/calibration-gate.sh PASS (quiet window),
#   2. this script in live mode: the built-in load gate (load1 AND load5 vs
#      CPU count) and the noise gate run automatically BEFORE --save,
#   3. then --save — which refuses to write when the noise gate FAILED
#      (non-decision-grade) unless --force-save overrides for a deliberate
#      perf change.
# A --save from --current/--noise-current compare-only input skips the live
# gates; treat those saves as unverified unless the producing run was green.

set -euo pipefail

BASELINE="benchmarks/benchmark-baseline.txt"
CURRENT_INPUT=""
SAVE=""
FORCE_SAVE=0
THRESHOLD="25"
# Known-unstable benchmarks (2026-09-21): "NAME|reason|expiry (YYYY-MM-DD)".
# A >threshold flag on a listed, unexpired benchmark is demoted to an advisory
# UNSTABLE-KNOWN line that does not fail the gate. The noise gate certifies
# MACHINE QUIETNESS, not sampling adequacy: a bimodal benchmark's median-of-N
# lands on either mode regardless of load, so suppressing the flag keeps
# quiet-night runs green while the sampling fix (widening) is decided.
# Entries auto-expire — after the expiry date the flag is a REGRESSION again.
KNOWN_UNSTABLE=(
	"BenchmarkMatViewRead/agg=MIN/scale=1k/matview|bimodal median-of-5 (samples 7.9-15µs), widening decision pending|2026-10-21"
)
# The gate set is an EXPLICIT allowlist of "DIR::BENCH_REGEX[::BENCHTIME[::COUNT]]"
# entries, deliberately NOT auto-discovered:
# widening it (e.g. `.`) would pull in load-sensitive benchmarks like
# watermill's BenchmarkCatchUp_ReplayThroughput and flake the CI regression
# gate on shared runners. New gate benchmarks must be added here on purpose,
# with load-aware budgets (loadScaledCeiling/soakTestScale) inside their own
# package. `BenchmarkBenchkitSuite_Memory$` is anchored so the _Small variant
# never matches implicitly. The matview entry guards the Turso materialized
# view serving path at scale=1k (accelerated reads are O(1)/O(groups) in N,
# so 1k IS the steady-state number; ≥10k is un-benched by upstream seeding
# constraint, see the bench file).
GATE_SETS=(
	"stack/bench::BenchmarkFullPipeline_Memory|BenchmarkBenchkitSuite_Memory$"
	"stack/bench::BenchmarkBenchkitSuite_SQLite$"
	"metaengine/tursoengine::BenchmarkMatViewRead/agg=[A-Z]+/scale=1k::100x::9"
	"metaengine/claimkit::BenchmarkClaimDue_ClaimKit|BenchmarkTimerRoundTrip_ClaimKit|BenchmarkDedupCheckAndRecord_FreshKeys_ClaimKit|BenchmarkDedupCheckAndRecord_LiveWindowHit"
)
BENCH_OVERRIDE=""
DIR_OVERRIDE=""
COUNT="5"
BENCHTIME="100x"

# ── benchkit per-metric noise gate (2026-09-16 decision: per-metric CI
# gating over A/B-by-revision benchstat — the median compare's known
# weakness is a loud machine, not a missing A/B workflow) ──
# A median compare cannot tell a real regression from scheduler wait. The
# noise gate runs ONE short cqrs-bench benchmark with --repeat N and fails
# when a HEADLINE metric's cross-run CoV reaches the threshold, so a green
# gate asserts the medians were measured on a quiet-enough machine.
NOISE_BACKEND="sqlite"
NOISE_PROFILE="dev"
NOISE_REPEAT="5"
NOISE_THRESHOLD="10"
# 2026-09-20: write_p99_ns demoted from the headline list — see the usage
# block above for the evidence. p99-class metrics (and max, worse still)
# measure estimator variance on this host, not machine loudness.
NOISE_HEADLINE="write_throughput write_p50_ns load_p50_ns"
NOISE_CURRENT=""
NOISE_ONLY=0
SKIP_NOISE_GATE=0
SKIP_LOAD_GATE=0
MAX_LOAD=""
# Fault-injection hook for the fixture tests: point the load probe at a
# fixture file instead of /proc/loadavg (calibration-gate.sh semantics).
LOADAVG_FILE="${BENCH_GATE_LOADAVG_FILE:-/proc/loadavg}"
# Fault-injection hook for the fixture tests: point the rename guards at a
# planted temp source tree instead of the repo (guards never mutate and never
# depend on live tracked files for their own tests).
GUARD_ROOT="${BENCH_GATE_GUARD_ROOT:-}"

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
	--force-save)
		FORCE_SAVE=1
		shift
		;;
	--threshold)
		THRESHOLD="$2"
		shift 2
		;;
	--bench)
		BENCH_OVERRIDE="$2"
		shift 2
		;;
	--dir)
		DIR_OVERRIDE="$2"
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
	--max-load)
		MAX_LOAD="$2"
		shift 2
		;;
	--skip-load-gate)
		SKIP_LOAD_GATE=1
		shift
		;;
	--noise-only)
		NOISE_ONLY=1
		shift
		;;
	--skip-noise-gate)
		SKIP_NOISE_GATE=1
		shift
		;;
	--noise-current)
		NOISE_CURRENT="$2"
		shift 2
		;;
	--noise-backend)
		NOISE_BACKEND="$2"
		shift 2
		;;
	--noise-profile)
		NOISE_PROFILE="$2"
		shift 2
		;;
	--noise-repeat)
		NOISE_REPEAT="$2"
		shift 2
		;;
	--noise-threshold)
		NOISE_THRESHOLD="$2"
		shift 2
		;;
	--noise-headline)
		NOISE_HEADLINE="$2"
		shift 2
		;;
	*)
		echo "Unknown flag: $1" >&2
		exit 1
		;;
	esac
done

# --bench/--dir collapse the gate to a single legacy-style set (backwards
# compatible with the pre-matview invocation shape).
if [[ -n "$BENCH_OVERRIDE" || -n "$DIR_OVERRIDE" ]]; then
	GATE_SETS=(
		"${DIR_OVERRIDE:-stack/bench}::${BENCH_OVERRIDE:-BenchmarkFullPipeline_Memory|BenchmarkBenchkitSuite_Memory$}"
	)
fi

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

# guard_root resolves the source tree the rename guards grep: the repo, or
# the fixture-injection root.
guard_root() {
	printf '%s' "${GUARD_ROOT:-$REPO_ROOT}"
}

# gate_set_guard fails loudly when a GATE_SETS entry's regex would match zero
# `func Benchmark` declarations in its package: a silently renamed benchmark
# empties the gate set, and an empty gate compares nothing and passes forever.
# Sub-benchmark paths (after '/') and '$' anchors never appear in func
# declarations, so the top-level name is what gets grepped.
gate_set_guard() {
	local root
	root="$(guard_root)"

	local failed=0 set dir bench alt name names pattern
	for set in "${GATE_SETS[@]}"; do
		dir="${set%%::*}"
		bench="${set#*::}"
		bench="${bench%%::*}" # drop a ::benchtime::count suffix

		names=()
		IFS='|' read -ra alts <<<"$bench"
		for alt in "${alts[@]}"; do
			name="${alt%\$}"   # drop the $ anchor
			name="${name%%/*}" # drop the sub-benchmark path
			[[ -n "$name" ]] && names+=("$name")
		done

		[[ ${#names[@]} -eq 0 ]] && continue

		pattern=""
		for name in "${names[@]}"; do
			pattern="${pattern:+$pattern|}$name"
		done
		pattern="func ($pattern)\\("

		if ! grep -rEq -- "$pattern" "$root/$dir" --include='*_test.go' 2>/dev/null; then
			echo "GATE SET GUARD FAILED — no 'func Benchmark' under $dir matches gate regex '$bench'"
			echo "  A renamed benchmark silently empties the gate set (the gate would compare"
			echo "  nothing and pass forever). Fix GATE_SETS in scripts/benchmark-regression.sh."
			failed=1
		fi
	done

	return "$failed"
}

# noise_target_guard asserts the noise gate's target still exists in the
# sources: the cqrs-bench 'run' subcommand, the backend name in the factory
# switch, and the profile name in benchkit. A rename would otherwise surface
# later as an opaque 'cqrs-bench run exited non-zero'.
noise_target_guard() {
	local root
	root="$(guard_root)"

	local failed=0

	if ! grep -q -- '"run"' "$root/cmd/cqrs-bench/main.go" 2>/dev/null; then
		echo "NOISE TARGET GUARD FAILED — cqrs-bench 'run' subcommand not found in cmd/cqrs-bench/main.go"
		failed=1
	fi

	if ! grep -rEq -- "\"$NOISE_BACKEND\"" "$root/cmd/cqrs-bench" --include='*.go' 2>/dev/null; then
		echo "NOISE TARGET GUARD FAILED — backend '$NOISE_BACKEND' not found in cmd/cqrs-bench sources"
		failed=1
	fi

	if ! grep -rEq -- "\"$NOISE_PROFILE\"" "$root/benchkit" --include='*.go' 2>/dev/null; then
		echo "NOISE TARGET GUARD FAILED — profile '$NOISE_PROFILE' not found in benchkit sources"
		failed=1
	fi

	if [[ "$failed" == 1 ]]; then
		echo "  The noise benchmark targets these names; a rename must update the NOISE_*"
		echo "  defaults in scripts/benchmark-regression.sh in the same change."
	fi

	return "$failed"
}

# load_gate aborts before any benchmarking when the machine is already
# oversubscribed (calibration-gate.sh v2 rule: load1 AND load5 must sit
# under the ceiling — load1-only let a burst-draining host pass while still
# noisy). Default ceiling is the CPU count: benchkit flags runs at the same
# line, so the gate refuses exactly the runs benchkit would stamp NOISY.
load_gate() {
	if [[ "$SKIP_LOAD_GATE" == 1 ]]; then
		echo "==> Load gate: skipped (--skip-load-gate)"

		return 0
	fi

	local ceiling="${MAX_LOAD:-$(nproc 2>/dev/null || echo 1)}"
	local l1 l5
	l1="$(cut -d' ' -f1 "$LOADAVG_FILE" 2>/dev/null || echo unknown)"
	l5="$(cut -d' ' -f2 "$LOADAVG_FILE" 2>/dev/null || echo unknown)"

	if [[ "$l1" == unknown || "$l5" == unknown ]]; then
		echo "WARN: load average unavailable (probe: $LOADAVG_FILE) — load gate skipped"

		return 0
	fi

	if awk -v a="$l1" -v b="$l5" -v c="$ceiling" 'BEGIN { exit !(a > c || b > c) }'; then
		echo "LOAD GATE FAILED — load1=$l1 load5=$l5 exceeds ceiling=$ceiling"
		echo "  An oversubscribed machine measures scheduler wait, not the backend:"
		echo "  any regression verdict from this run would be noise. Re-run on a"
		echo "  quiet machine, or override with --max-load / --skip-load-gate."

		return 1
	fi

	echo "==> Load gate: load1=$l1 load5=$l5 < ceiling=$ceiling"

	return 0
}

# noise_gate checks a cqrs-bench `--format json` result for headline metrics
# whose cross-run CoV reached the threshold. Metrics recorded by the run but
# not on the headline list only warn — the headline list is what median
# regression verdicts are actually built from.
noise_gate() {
	local file="$1"

	if [[ ! -s "$file" ]]; then
		echo "NOISE GATE FAILED — result file '$file' is empty or missing"
		echo "  The noise gate needs 'cqrs-bench run --repeat N --format json' output."

		return 1
	fi

	local total
	total="$(jq -r '(.metricVariation // []) | length' "$file" 2>/dev/null)" || total=""

	if [[ -z "$total" ]]; then
		echo "NOISE GATE FAILED — could not parse cqrs-bench result JSON: $file"
		echo "  Regenerate with: cqrs-bench run --backend $NOISE_BACKEND --profile $NOISE_PROFILE --repeat $NOISE_REPEAT --format json"

		return 1
	fi

	if [[ "$total" -eq 0 ]]; then
		echo "NOISE GATE FAILED — result carries no metricVariation data"
		echo "  Cross-run dispersion needs --repeat >= 2; the gate cannot assess"
		echo "  reliability without it. Re-run with --repeat $NOISE_REPEAT."

		return 1
	fi

	local noisy
	noisy="$(jq -r --argjson t "$NOISE_THRESHOLD" \
		'(.metricVariation // [])[] | select(.cov * 100 >= $t) | "\(.name) \(.cov)"' "$file")"

	local noisy_count=0

	if [[ -n "$noisy" ]]; then
		noisy_count="$(printf '%s\n' "$noisy" | grep -c .)"
	fi

	echo "==> Noise gate: $((total - noisy_count))/$total metrics stable (CoV < ${NOISE_THRESHOLD}%)"

	local failed=0

	while read -r name cov; do
		[[ -z "$name" ]] && continue

		local pct
		pct="$(awk -v c="$cov" 'BEGIN { printf "%.1f", c * 100 }')"

		if printf '%s\n' "$NOISE_HEADLINE" | grep -qw -- "$name"; then
			echo "NOISY HEADLINE  $name  CoV=${pct}%  (headline metrics must stay under ${NOISE_THRESHOLD}%)"
			failed=1
		else
			echo "  noisy (non-headline, advisory): $name CoV=${pct}%"
		fi
	done <<<"$noisy"

	if [[ "$failed" == 1 ]]; then
		echo "NOISE GATE FAILED — headline metrics moved too much between runs;"
		echo "  median comparisons from this run are not decision-grade. Re-run on"
		echo "  a quiet machine or increase --noise-repeat, or override with"
		echo "  --skip-noise-gate / --noise-headline."

		return 1
	fi

	return 0
}

# run_noise_benchmark produces a fresh result JSON for the noise gate.
run_noise_benchmark() {
	local out="$1"

	echo "==> Building cqrs-bench for the noise gate"

	if ! (cd "$REPO_ROOT" && GOTOOLCHAIN=auto go build -o "$out" ./cmd/cqrs-bench); then
		echo "NOISE GATE FAILED — could not build ./cmd/cqrs-bench"

		return 1
	fi

	echo "==> Noise benchmark: backend=$NOISE_BACKEND profile=$NOISE_PROFILE repeat=$NOISE_REPEAT"

	if ! "$out" run --backend "$NOISE_BACKEND" --profile "$NOISE_PROFILE" \
		--repeat "$NOISE_REPEAT" --format json --quiet --output "${out}.json"; then
		echo "NOISE GATE FAILED — cqrs-bench run exited non-zero"

		return 1
	fi
}

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
noise_result=""
trap 'rm -f "$current_file" ${noise_result:+"$noise_result" "${noise_result}.json"}' EXIT

# Rename guards FIRST — pure greps, the cheapest possible failure point. A
# renamed gate benchmark or noise target must fail with an actionable message
# here, not as a confusing empty-pass or opaque non-zero exit after minutes of
# benchmarking. --current compare-only mode skips them: those benchmarks and
# artifacts were produced by an earlier invocation.
if [[ -z "$CURRENT_INPUT" ]] && ! gate_set_guard; then
	exit 1
fi

if [[ "$SKIP_NOISE_GATE" == 0 && (-z "$CURRENT_INPUT" || "$NOISE_ONLY" == 1) ]] && ! noise_target_guard; then
	exit 1
fi

# Load gate next: refusing loud machines BEFORE benching saves the full
# gate run on a box whose numbers would be worthless anyway. Skipped in
# --current compare-only mode — those benchmarks already ran elsewhere —
# but --noise-only always benches, so it always pays the load gate.
if [[ -z "$CURRENT_INPUT" || "$NOISE_ONLY" == 1 ]] && ! load_gate; then
	exit 1
fi

# Captured at start so a --save header records load AT BENCH TIME, not at
# save time (benchmarks may run for many minutes before --save fires).
UPTIME_AT_START=$(uptime 2>/dev/null || echo 'uptime unknown')

# Snapshot the baseline BEFORE --save can overwrite it — otherwise a
# save+compare run compares current against itself and always passes.
had_baseline=false
if [[ -f "$BASELINE" ]] && [[ -n "$(medians "$BASELINE")" ]]; then
	base_medians=$(medians "$BASELINE" | LC_ALL=C sort)
	had_baseline=true
fi

# Superseded-baseline archive dir (owner ruling 2026-09-21: dated series
# under docs/benchmarks/baselines/). Fixture-injectable so test runs never
# drop fake archives into the real dated series.
ARCHIVE_DIR="${BENCH_GATE_ARCHIVE_DIR:-$REPO_ROOT/docs/benchmarks/baselines}"

if [[ -n "$CURRENT_INPUT" ]]; then
	cp "$CURRENT_INPUT" "$current_file"
elif [[ "$NOISE_ONLY" == 1 ]]; then
	echo "==> Noise-only mode: skipping go-test gate sets"
else
	for set in "${GATE_SETS[@]}"; do
		set_dir="${set%%::*}"
		set_bench="${set#*::}"
		set_bt="$BENCHTIME"
		set_cnt="$COUNT"
		if [[ "$set_bench" == *"::"* ]]; then
			set_opts="${set_bench#*::}"
			set_bench="${set_bench%%::*}"
			set_bt="${set_opts%%::*}"
			set_cnt="${set_opts#*::}"
		fi
		echo "==> Running gate benchmarks ($set_dir, bench=$set_bench, count=$set_cnt, benchtime=$set_bt)"
		(
			cd "$set_dir"
			GOTOOLCHAIN=auto go test \
				-run='^$' -bench="$set_bench" -benchmem \
				-benchtime="$set_bt" -count="$set_cnt" -timeout 30m 2>&1
		) | tee -a "$current_file"
	done
fi

compare_status=0
noise_status=0

# Noise gate runs whenever this invocation benches (live mode), or when a
# pre-computed result was handed over via --noise-current. In --current
# (compare-only) mode without --noise-current there is nothing to check: the
# gate benchmarks ran in a different step/process.
if [[ -n "$NOISE_CURRENT" ]]; then
	noise_gate "$NOISE_CURRENT" || noise_status=$?
elif [[ "$SKIP_NOISE_GATE" == 0 && (-z "$CURRENT_INPUT" || "$NOISE_ONLY" == 1) ]]; then
	noise_result="$(mktemp)"
	run_noise_benchmark "$noise_result" || noise_status=$?

	if [[ "$noise_status" == 0 ]]; then
		noise_gate "${noise_result}.json" || noise_status=$?
	fi
fi

if [[ "$NOISE_ONLY" == 1 ]]; then
	if [[ $noise_status -ne 0 ]]; then
		echo "FAIL: noise gate (see above)"
		exit 1
	fi

	echo "PASS: noise gate"

	exit 0
fi

if [[ "$had_baseline" == true ]]; then
	cur_medians=$(medians "$current_file" | LC_ALL=C sort)

	# Benchmarks that vanished or appeared are informational, never a failure.
	# LC_ALL=C on comm matches the sort above — the ambient UTF-8 locale
	# orders "/" and "=" differently and comm misreports the input as unsorted.
	LC_ALL=C comm -23 <(printf '%s\n' "$base_medians" | awk '{print $1}') \
		<(printf '%s\n' "$cur_medians" | awk '{print $1}') |
		grep '^.' | sed 's/^/  removed from current: /' || true
	LC_ALL=C comm -13 <(printf '%s\n' "$base_medians" | awk '{print $1}') \
		<(printf '%s\n' "$cur_medians" | awk '{print $1}') |
		grep '^.' | sed 's/^/  new in current: /' || true

	echo ""
	echo "==> Comparing medians (threshold: ${THRESHOLD}%)"

	known_names="$(printf '%s\n' "${KNOWN_UNSTABLE[@]}" | awk -F'|' -v today="$(date -u +%Y-%m-%d)" '$3 >= today {print $1}')"

	LC_ALL=C join <(printf '%s\n' "$base_medians") <(printf '%s\n' "$cur_medians") |
		KNOWN_SET="$known_names" awk -v t="$THRESHOLD" '
			BEGIN {
				known_n = split(ENVIRON["KNOWN_SET"], ka, "\n")
				for (i = 1; i <= known_n; i++) known[ka[i]] = 1
			}
			{
				name = $1
				base = $2 + 0
				cur = $4 + 0
				if (base <= 0 || cur <= 0) next
				pct = (cur - base) * 100 / base
				if (pct > t) {
					if (name in known) {
						printf "UNSTABLE-KNOWN  %-49s %12.1f → %12.1f ns/op  (+%.1f%%)  [suppressed: known bimodal sampler]\n", name, base, cur, pct
						unstable_known++
					} else {
						printf "REGRESSION  %-55s %12.1f → %12.1f ns/op  (+%.1f%%)\n", name, base, cur, pct
						regressions++
					}
				} else if (pct < -5) {
					improvements++
				} else {
					stable++
				}
			}
			END {
				printf "\nSummary: %d regression(s), %d improvement(s), %d stable, %d known-unstable(suppressed)\n", regressions + 0, improvements + 0, stable + 0, unstable_known + 0
				exit (regressions > 0) ? 1 : 0
			}' || compare_status=$?
else
	echo "WARN: no parseable baseline at $BASELINE — skipping comparison (save-only run)."
fi

# --save runs AFTER the comparison and regardless of its outcome: re-baselining
# after an intentional perf change must overwrite even a "regressed" baseline.
if [[ -n "$SAVE" ]]; then
	if [[ -f "$SAVE" && -s "$SAVE" ]]; then
		mkdir -p "$ARCHIVE_DIR"
		archive="$ARCHIVE_DIR/benchmark-baseline-$(date -u +%Y%m%dT%H%M%SZ).txt"
		cp "$SAVE" "$archive"
		echo "==> Superseded baseline archived to $archive (owner ruling 2026-09-21: dated series under docs/benchmarks/baselines/)"
	fi
	if [[ $noise_status -ne 0 && "$FORCE_SAVE" != 1 ]]; then
		echo "REFUSING --save: the noise gate FAILED this run (non-decision-grade)."
		echo "  A baseline from a noise-failed run makes every future compare lie."
		echo "  Re-run on a quieter machine, or pass --force-save to override for"
		echo "  an intentional re-baseline after a deliberate perf change."
		exit 1
	fi
	mkdir -p "$(dirname "$SAVE")"
	# Titled re-pin (2026-09-11 protocol): a baseline without provenance is
	# unreviewable — the 02:40 refresh landed during a load-ramp and nothing
	# in the file said so. Header lines start with '#' and are ignored by the
	# medians parser. Local re-baselines must follow a
	# scripts/calibration-gate.sh PASS first; CI saves are exempt.
	{
		echo "# benchmark baseline — re-pinned $(date -u +%Y-%m-%dT%H:%M:%SZ)"
		echo "# ${UPTIME_AT_START:-$(uptime 2>/dev/null || echo 'uptime unknown')}"
		echo "# gate: scripts/calibration-gate.sh must PASS before a local re-pin; CI saves are exempt"
		echo "# re-pin protocol: calibration PASS -> live gate run (load+noise gates) -> --save (refuses on noise fail)"
		cat "$current_file"
	} >"$SAVE"
	echo "==> Current results saved to $SAVE (with provenance header)"
fi

if [[ $compare_status -ne 0 ]]; then
	echo "FAIL: benchmark regression above ${THRESHOLD}% threshold"
	exit 1
fi

if [[ $noise_status -ne 0 ]]; then
	echo "FAIL: benchkit noise gate (see above)"
	exit 1
fi

echo "PASS: no regression above ${THRESHOLD}% threshold"
