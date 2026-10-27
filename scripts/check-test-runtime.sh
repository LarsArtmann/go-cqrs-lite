#!/usr/bin/env bash
# check-test-runtime.sh — per-package wall-clock budgets for the test suite.
#
# Why: `#test` reports a total duration nobody watches, so one package that
# quietly grew from 5s to 90s (a soak that lost its -short guard, a timeout
# that became a real sleep) is invisible until the suite is unusable. This gate
# reads `go test -json` events (GT_JSON=1 on scripts/go-test-shuffled.sh) and
# compares every PACKAGE's elapsed time against a committed budget.
#
# What it does NOT try to do: catch a 20% slowdown. Machine noise under load is
# far larger than that; the budget is factor 2.0 plus a 20s floor, i.e. it
# catches a package that changed CLASS, not one that wobbled.
#
# Usage: bash scripts/check-test-runtime.sh [flags]
#   (default)          compare the newest build/test-json/*.json to the baseline
#   --json FILE        compare this event stream (repeatable)
#   --baseline FILE    budget file (default scripts/test-runtime-baseline.txt)
#   --factor N         budget multiplier (default 2.0; TEST_RUNTIME_FACTOR)
#   --floor N          seconds of absolute slack (default 20; TEST_RUNTIME_FLOOR)
#   --update           rewrite the budget file from the measured stream
#   --quiet            only the verdict
#   --self-test        fault-inject the comparison against fixtures
#   --help
#
# Producing the input:
#   GT_JSON=1 nix run .#test          # or: nix run .#test -- -run '^$'
#
# Ratchet direction: budgets only move via --update, which prints the delta it
# accepts. A package that got FASTER is reported (STALE) but not failed —
# shrinking the budget is a separate deliberate --update, exactly like the
# coverage floor.

set -euo pipefail
export LC_ALL=C

cd "$(git rev-parse --show-toplevel)"

BASELINE="scripts/test-runtime-baseline.txt"
# The baseline is measured on THIS host, so the gate is tight here. A CI
# runner is a different machine class: it may widen via the env vars below and
# the local/nightly run stays the tight gate.
FACTOR="${TEST_RUNTIME_FACTOR:-2.0}"
FLOOR="${TEST_RUNTIME_FLOOR:-20}"
MODE=gate
QUIET=0
JSON_FILES=()

die() {
	echo "::error::$1" >&2
	exit 1
}

usage() {
	sed -n '2,/^$/p' "$0" | sed 's/^# \{0,1\}//'
}

while [ $# -gt 0 ]; do
	case "$1" in
	--json)
		shift
		[ $# -gt 0 ] || die "--json requires a file"
		JSON_FILES+=("$1")
		;;
	--baseline)
		shift
		[ $# -gt 0 ] || die "--baseline requires a file"
		BASELINE="$1"
		;;
	--factor)
		shift
		[ $# -gt 0 ] || die "--factor requires a number"
		FACTOR="$1"
		;;
	--floor)
		shift
		[ $# -gt 0 ] || die "--floor requires a number"
		FLOOR="$1"
		;;
	--update) MODE=update ;;
	--quiet) QUIET=1 ;;
	--self-test) MODE=selftest ;;
	--help | -h)
		usage
		exit 0
		;;
	*) die "unknown flag: $1 (try --help)" ;;
	esac
	shift
done

# elapsed_by_package OUT FILES... — "package<TAB>seconds" for every package
# that reported a terminal event. go test -json emits one such event per
# package; Test events carry a "Test" key and are excluded by construction.
elapsed_by_package() { # $1=out, rest=json files
	local out="$1"
	shift
	awk '
		/"Action":"(pass|fail|skip)"/ && !/"Test":/ {
			if (!match($0, /"Package":"[^"]*"/)) next
			pkg = substr($0, RSTART + 11, RLENGTH - 12)
			if (!match($0, /"Elapsed":[0-9.eE+-]+/)) next
			el = substr($0, RSTART + 10, RLENGTH - 10)
			sum[pkg] = el
		}
		END { for (p in sum) printf "%s\t%s\n", p, sum[p] }
	' "$@" | sort >"$out"
}

# read_baseline FILE OUT — "package<TAB>seconds" from a budget file.
read_baseline() { # $1=file $2=out
	awk -F'\t' '!/^#/ && NF >= 2 && $1 != "" { printf "%s\t%s\n", $1, $2 }' "$1" | sort >"$2"
}

compare() { # $1=measured $2=budget ; prints rows, returns 1 on breach
	local measured="$1" budget="$2" breaches=0
	awk -F'\t' -v factor="$FACTOR" -v floor="$FLOOR" '
		NR == FNR { base[$1] = $2; next }
		{
			pkg = $1; el = $2 + 0
			if (!(pkg in base)) { printf "NEW\t%s\t%.1f\t-\t-", pkg, el; print ""; next }
			limit = base[pkg] * factor + floor
			if (el > limit) {
				printf "BREACH\t%s\t%.1f\t%.1f\t%.1f\n", pkg, el, base[pkg], limit
				breaches++
			} else if (el * 2 < base[pkg] && base[pkg] >= 5) {
				printf "STALE\t%s\t%.1f\t%.1f\t%.1f\n", pkg, el, base[pkg], limit
			}
		}
		END { exit (breaches > 0 ? 3 : 0) }
	' "$budget" "$measured"
}

report_new_dangling() { # $1=measured $2=budget
	awk -F'\t' 'NR == FNR { seen[$1] = 1; next } !($1 in seen) { print "DANGLING\t" $1 }' "$1" "$2"
}

self_test() {
	local tmp failures=0 out rc
	tmp="$(mktemp -d)"
	# shellcheck disable=SC2064
	trap "rm -rf '$tmp'" EXIT

	cat >"$tmp/budget.txt" <<'EOF'
# fixture budget
example/a	10
example/b	100
example/c	2
EOF
	cat >"$tmp/run.json" <<'EOF'
{"Time":"2026-10-27T00:00:00Z","Action":"start","Package":"example/a"}
{"Time":"2026-10-27T00:00:05Z","Action":"pass","Package":"example/a","Test":"TestOne","Elapsed":5}
{"Time":"2026-10-27T00:00:12Z","Action":"pass","Package":"example/a","Elapsed":12}
{"Time":"2026-10-27T00:01:40Z","Action":"pass","Package":"example/b","Elapsed":100}
{"Time":"2026-10-27T00:03:45Z","Action":"fail","Package":"example/c","Elapsed":225}
{"Time":"2026-10-27T00:03:50Z","Action":"pass","Package":"example/new","Elapsed":3}
EOF
	elapsed_by_package "$tmp/measured.txt" "$tmp/run.json"
	read_baseline "$tmp/budget.txt" "$tmp/base.txt"

	if [ "$(wc -l <"$tmp/measured.txt")" -ne 4 ]; then
		echo "self-test FAIL: expected 4 packages (Test events excluded), got:" >&2
		cat "$tmp/measured.txt" >&2
		failures=$((failures + 1))
	fi
	# example/a improved (12s vs 10s budget) → within floor, not a breach.
	set +e
	out="$(compare "$tmp/measured.txt" "$tmp/base.txt")"
	rc=$?
	set -e
	if [ "$rc" -eq 0 ]; then
		echo "self-test FAIL: example/c at 225s vs a 220s limit must breach (rc=$rc)" >&2
		failures=$((failures + 1))
	fi
	if ! printf '%s\n' "$out" | grep -q '^BREACH	example/c	'; then
		echo "self-test FAIL: example/c must be the breach row" >&2
		printf '%s\n' "$out" >&2
		failures=$((failures + 1))
	fi
	if printf '%s\n' "$out" | grep -q '^BREACH	example/b'; then
		echo "self-test FAIL: example/b at exactly its limit (2x100+20) must pass" >&2
		failures=$((failures + 1))
	fi
	if ! printf '%s\n' "$out" | grep -q '^NEW	example/new'; then
		echo "self-test FAIL: an unbudgeted package must be reported NEW" >&2
		failures=$((failures + 1))
	fi
	# The floor is what keeps tiny packages from tripping on jitter.
	cat >"$tmp/budget2.txt" <<'EOF'
example/tiny	1
EOF
	cat >"$tmp/run2.json" <<'EOF'
{"Time":"2026-10-27T00:00:00Z","Action":"pass","Package":"example/tiny","Elapsed":5}
EOF
	elapsed_by_package "$tmp/measured2.txt" "$tmp/run2.json"
	read_baseline "$tmp/budget2.txt" "$tmp/base2.txt"
	set +e
	compare "$tmp/measured2.txt" "$tmp/base2.txt" >/dev/null
	rc=$?
	set -e
	if [ "$rc" -ne 0 ]; then
		echo "self-test FAIL: 5s against a 1s budget must pass on the 20s floor" >&2
		failures=$((failures + 1))
	fi
	# A faster package is STALE, never a failure.
	cat >"$tmp/budget3.txt" <<'EOF'
example/faster	100
EOF
	cat >"$tmp/run3.json" <<'EOF'
{"Time":"2026-10-27T00:00:00Z","Action":"pass","Package":"example/faster","Elapsed":20}
EOF
	elapsed_by_package "$tmp/measured3.txt" "$tmp/run3.json"
	read_baseline "$tmp/budget3.txt" "$tmp/base3.txt"
	set +e
	out="$(compare "$tmp/measured3.txt" "$tmp/base3.txt")"
	rc=$?
	set -e
	if [ "$rc" -ne 0 ] || ! printf '%s\n' "$out" | grep -q '^STALE	example/faster'; then
		echo "self-test FAIL: a 5x faster package must be STALE and rc=0" >&2
		failures=$((failures + 1))
	fi

	if [ "$failures" -gt 0 ]; then
		echo "::error::check-test-runtime self-test: $failures case(s) failed" >&2
		exit 1
	fi
	echo "✓ check-test-runtime self-test: 6/6 fault-injection cases behave"
}

if [ "$MODE" = "selftest" ]; then
	self_test
	exit 0
fi

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

if [ "${#JSON_FILES[@]}" -eq 0 ]; then
	newest="$(ls -1t build/test-json/*.json 2>/dev/null | head -1 || true)"
	[ -n "$newest" ] || die "no event stream found — run GT_JSON=1 nix run .#test first, or pass --json FILE"
	JSON_FILES=("$newest")
fi
for f in "${JSON_FILES[@]}"; do
	[ -f "$f" ] || die "--json '$f' does not exist"
done

[ "$MODE" = "update" ] || [ -f "$BASELINE" ] || die "baseline '$BASELINE' missing — bootstrap it with --update"
[ "$MODE" = "update" ] || [ -s "$BASELINE" ] || die "baseline '$BASELINE' is empty — bootstrap it with --update"

elapsed_by_package "$tmp/measured.txt" "${JSON_FILES[@]}"
[ -s "$tmp/measured.txt" ] || die "no package events parsed from ${JSON_FILES[*]} — was the run killed before any package finished?"

packages="$(wc -l <"$tmp/measured.txt")"

if [ "$MODE" = "update" ]; then
	{
		echo "# test-runtime-baseline.txt — per-package wall-clock budgets (T12)."
		echo "#"
		echo "# Generated by scripts/check-test-runtime.sh --update from:"
		for f in "${JSON_FILES[@]}"; do
			echo "#   $f"
		done
		echo "# Tolerance at gate time: factor ${FACTOR}x + ${FLOOR}s absolute floor."
		echo "# Format: <package><TAB><seconds>"
		echo "#"
		echo "# A package that legitimately got slower needs a fresh --update, which"
		echo "# prints every accepted delta — never hand-edit a number."
		cat "$tmp/measured.txt"
	} >"$tmp/baseline.new"
	mv "$tmp/baseline.new" "$BASELINE"
	echo "✓ budget written for $packages package(s) → $BASELINE"
	exit 0
fi

read_baseline "$BASELINE" "$tmp/base.txt"

set +e
rows="$(compare "$tmp/measured.txt" "$tmp/base.txt")"
rc=$?
set -e

breaches="$(printf '%s\n' "$rows" | grep -c '^BREACH' || true)"
news="$(printf '%s\n' "$rows" | grep -c '^NEW' || true)"
stales="$(printf '%s\n' "$rows" | grep -c '^STALE' || true)"
dangling="$(report_new_dangling "$tmp/measured.txt" "$tmp/base.txt" | grep -c . || true)"

if [ "$QUIET" -eq 0 ]; then
	echo "==> $packages package(s) measured; budget: ${FACTOR}x + ${FLOOR}s"
	echo ""
	printf '%-9s %-58s %8s %8s %8s\n' STATUS PACKAGE ELAPSED BASE LIMIT
	printf '%-9s %-58s %8s %8s %8s\n' ------- ------- ------- ---- -----
	# No rows is the good case; awk handles an empty input without a pipeline
	# status to worry about (grep returns 1 on no match, which pipefail+errexit
	# would turn into a silent early exit).
	printf '%s\n' "$rows" | awk -F'\t' 'NF > 1 { printf "%-9s %-58s %8s %8s %8s\n", $1, $2, $3, $4, $5 }'
	echo ""
	echo "Slowest 10 packages in this run:"
	sort -t"$(printf '\t')" -k2,2gr "$tmp/measured.txt" | head -10 |
		awk -F'\t' '{ printf "  %8.1fs  %s\n", $2, $1 }'
	echo ""
fi

if [ "$stales" -gt 0 ]; then
	echo "note: $stales package(s) are more than 2x FASTER than their budget (STALE)."
	echo "      Budgets shrink only through --update, never automatically."
fi
if [ "$dangling" -gt 0 ]; then
	echo "note: $dangling budget row(s) name packages absent from this run (DANGLING)."
	report_new_dangling "$tmp/measured.txt" "$tmp/base.txt" | cut -f2 | sed 's/^/  /'
fi

if [ "$breaches" -gt 0 ] || [ "$news" -gt 0 ] || [ "$rc" -ne 0 ]; then
	[ "$breaches" -gt 0 ] && echo "::error::$breaches package(s) over budget:"
	printf '%s\n' "$rows" | grep '^BREACH' | awk -F'\t' '{ printf "::error::  %s: %.1fs vs %.1fs budget (limit %.1fs)\n", $2, $3, $4, $5 }'
	[ "$news" -gt 0 ] && {
		echo "::error::$news package(s) have no budget row:"
		printf '%s\n' "$rows" | grep '^NEW' | cut -f2 | sed 's/^/::error::  /'
		echo "::error::Regenerate the budget: bash scripts/check-test-runtime.sh --update"
	}
	exit 1
fi

echo "✓ all $packages package(s) within budget (and every package has one)."
