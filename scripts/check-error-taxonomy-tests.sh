#!/usr/bin/env bash
# check-error-taxonomy-tests.sh — a documented error code needs a test that
# names it.
#
# Why: docs/error-taxonomy.md lists ~550 codes with their family. The drift
# gate (scripts/check-error-taxonomy.sh) proves the FAMILY matches the source,
# but nothing proves the error is ever EXERCISED. A code whose family is right
# and which no test ever produces is a promise nobody has run — the wrapping
# can be broken (wrong %w, dropped context, panic on nil cause) and the table
# still reads perfectly.
#
# Detection is name-based: the code appears as a string literal in some
# _test.go. That is deliberately weak but honest — it cannot prove the assertion
# is meaningful, only that the code is reachable from a test. The strong
# detector of "a test that cannot fail" is mutation (suite T10); this gate
# stops the tail from GROWING, which is what the ratchet is for.
#
# Ratchet: the historical tail is frozen in the baseline (shrink-only). A NEW
# documented code must either be named by a test or be added deliberately via
# --update, which prints what it accepts.
#
# Usage: bash scripts/check-error-taxonomy-tests.sh [flags]
#   (default)    fail on a documented code that has neither a test nor a baseline row
#   --update     rewrite the baseline from the current tree
#   --quiet      only the verdict
#   --self-test  fault-inject the comparison against fixtures
#   --help
set -euo pipefail
export LC_ALL=C

cd "$(git rev-parse --show-toplevel)"

DOC="docs/error-taxonomy.md"
BASELINE="scripts/error-taxonomy-untested-baseline.txt"
MODE=gate
QUIET=0

die() {
	echo "::error::$1" >&2
	exit 1
}

usage() {
	sed -n '2,/^$/p' "$0" | sed 's/^# \{0,1\}//'
}

while [ $# -gt 0 ]; do
	case "$1" in
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

# documented_codes DOC OUT — backticked dotted literals from the tables.
# Wildcards (`module.*`, `module.prefix_*`) are stems, not codes: a test cannot
# name a pattern, so they are excluded rather than counted as untested.
documented_codes() { # $1=doc $2=out
	grep -oE '`[a-z][a-zA-Z0-9_]*(\.[a-zA-Z0-9_]+)+`' "$1" |
		tr -d '`' | grep -v '\*' | sort -u >"$2"
}

# tested_codes ROOT OUT — codes named as string literals in any _test.go.
tested_codes() { # $1=root $2=out
	(cd "$1" &&
		grep -rl --include='*_test.go' -F '"' . 2>/dev/null |
		grep -v '/vendor/' | grep -v '/testdata/' | grep -v '/node_modules/' |
			xargs grep -hoE '"[a-z][a-zA-Z0-9_]*(\.[a-zA-Z0-9_]+)+"' 2>/dev/null |
			tr -d '"' | sort -u) >"$2"
}

# compare DOC_CODES TESTED BASELINE — prints rows:
#   UNTESTED  <code>   documented, no test, no baseline row   (FAIL)
#   OK        <code>   documented, no test, frozen in baseline
#   STALE     <code>   baseline row that a test now names      (info)
#   DANGLING  <code>   baseline row no longer documented       (FAIL)
compare() { # $1=doc codes $2=tested $3=baseline
	awk -v tested_file="$2" -v base_file="$3" '
		BEGIN {
			while ((getline line < tested_file) > 0) tested[line] = 1
			close(tested_file)
			while ((getline line < base_file) > 0) base[line] = 1
			close(base_file)
		}
		{ doc[$0] = 1 }
		END {
			for (c in doc) {
				if (c in tested) continue
				if (c in base) { print "OK\t" c; continue }
				print "UNTESTED\t" c
			}
			for (c in base) {
				if (!(c in doc)) { print "DANGLING\t" c; continue }
				if (c in tested) print "STALE\t" c
			}
		}
	' "$1"
}

self_test() {
	local tmp failures=0 rows
	tmp="$(mktemp -d)"
	# shellcheck disable=SC2064
	trap "rm -rf '$tmp'" EXIT

	mkdir -p "$tmp/tree/pkg"
	cat >"$tmp/doc.md" <<'EOF'
| Family | Code | Meaning |
| --- | --- | --- |
| Rejection | `alpha.covered` | tested below |
| Rejection | `alpha.frozen` | frozen in the baseline |
| Conflict | `alpha.brand_new` | nobody exercises this |
| Rejection | `alpha.pattern_*` | a wildcard stem, not a code |
EOF
	cat >"$tmp/tree/pkg/x_test.go" <<'EOF'
package pkg

import "testing"

func TestCovered(t *testing.T) {
	_ = "alpha.covered"
}
EOF
	documented_codes "$tmp/doc.md" "$tmp/doc.txt"
	tested_codes "$tmp/tree" "$tmp/tested.txt"
	printf 'alpha.frozen\n' >"$tmp/base.txt"

	if grep -q 'alpha.pattern_' "$tmp/doc.txt"; then
		echo "self-test FAIL: a wildcard stem must not be treated as a code" >&2
		failures=$((failures + 1))
	fi
	rows="$(compare "$tmp/doc.txt" "$tmp/tested.txt" "$tmp/base.txt")"
	if printf '%s\n' "$rows" | grep -q 'UNTESTED	alpha.covered'; then
		echo "self-test FAIL: a code named by a test must not be UNTESTED" >&2
		failures=$((failures + 1))
	fi
	if ! printf '%s\n' "$rows" | grep -q '^OK	alpha.frozen'; then
		echo "self-test FAIL: a frozen code must report OK" >&2
		failures=$((failures + 1))
	fi
	if ! printf '%s\n' "$rows" | grep -q '^UNTESTED	alpha.brand_new'; then
		echo "self-test FAIL: a new untested code must be UNTESTED" >&2
		failures=$((failures + 1))
	fi

	# A baseline row whose code was deleted from the doc must be DANGLING.
	printf 'alpha.frozen\nalpha.removed\n' >"$tmp/base2.txt"
	rows="$(compare "$tmp/doc.txt" "$tmp/tested.txt" "$tmp/base2.txt")"
	if ! printf '%s\n' "$rows" | grep -q '^DANGLING	alpha.removed'; then
		echo "self-test FAIL: a baseline row with no documented code must be DANGLING" >&2
		failures=$((failures + 1))
	fi

	# A baseline row that a test now names must be STALE (so --update prunes it).
	printf 'alpha.frozen\nalpha.covered\n' >"$tmp/base3.txt"
	rows="$(compare "$tmp/doc.txt" "$tmp/tested.txt" "$tmp/base3.txt")"
	if ! printf '%s\n' "$rows" | grep -q '^STALE	alpha.covered'; then
		echo "self-test FAIL: a now-tested baseline row must be STALE" >&2
		failures=$((failures + 1))
	fi

	if [ "$failures" -gt 0 ]; then
		echo "::error::check-error-taxonomy-tests self-test: $failures case(s) failed" >&2
		exit 1
	fi
	echo "✓ check-error-taxonomy-tests self-test: 6/6 fault-injection cases behave"
}

if [ "$MODE" = "selftest" ]; then
	self_test
	exit 0
fi

[ -f "$DOC" ] || die "$DOC not found"

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

documented_codes "$DOC" "$tmp/doc.txt"
tested_codes "$PWD" "$tmp/tested.txt"
[ -s "$tmp/doc.txt" ] || die "extracted no documented codes from $DOC — did the table format change?"

total="$(wc -l <"$tmp/doc.txt")"

if [ "$MODE" = "update" ]; then
	awk -v tested_file="$tmp/tested.txt" '
		BEGIN { while ((getline line < tested_file) > 0) tested[line] = 1; close(tested_file) }
		{ if (!($0 in tested)) print $0 }
	' "$tmp/doc.txt" | sort -u >"$BASELINE.new"
	{
		echo "# error-taxonomy-untested-baseline.txt — documented error codes that NO"
		echo "# test names. Ratchet: this list shrinks; a new code must bring a test."
		echo "# Generated by: bash scripts/check-error-taxonomy-tests.sh --update"
		echo "# Format: <code>   (one per line)"
		cat "$BASELINE.new"
	} >"$BASELINE"
	rm -f "$BASELINE.new"
	echo "✓ baseline written: $(grep -vc '^#' "$BASELINE") of $total documented code(s) have no test naming them"
	echo "  (frozen, shrink-only — add a test and re-run --update to remove a row)"
	exit 0
fi

[ -f "$BASELINE" ] || die "baseline $BASELINE missing — bootstrap it with --update"

grep -v '^#' "$BASELINE" | grep -c . >/dev/null || true
rows="$(compare "$tmp/doc.txt" "$tmp/tested.txt" <(grep -v '^#' "$BASELINE"))"

untested="$(printf '%s\n' "$rows" | grep -c '^UNTESTED' || true)"
dangling="$(printf '%s\n' "$rows" | grep -c '^DANGLING' || true)"
stale="$(printf '%s\n' "$rows" | grep -c '^STALE' || true)"
frozen="$(printf '%s\n' "$rows" | grep -c '^OK' || true)"

if [ "$QUIET" -eq 0 ]; then
	echo "==> $total documented error code(s): $((total - frozen - untested)) tested, $frozen frozen untested"
	if [ "$untested" -gt 0 ]; then
		echo ""
		echo "New documented code(s) with no test naming them:"
		printf '%s\n' "$rows" | grep '^UNTESTED' | awk -F'\t' '{ print "  " $2 }'
	fi
	if [ "$stale" -gt 0 ]; then
		echo ""
		echo "$stale frozen code(s) are now named by a test — prune the baseline with --update."
	fi
	if [ "$dangling" -gt 0 ]; then
		echo ""
		echo "$dangling baseline row(s) name codes no longer in the doc:"
		printf '%s\n' "$rows" | grep '^DANGLING' | awk -F'\t' '{ print "  " $2 }'
	fi
fi

if [ "$untested" -gt 0 ] || [ "$dangling" -gt 0 ]; then
	[ "$untested" -gt 0 ] && echo "::error::$untested error code(s) added without a test naming them."
	[ "$dangling" -gt 0 ] && echo "::error::$dangling baseline row(s) are stale (the code left the doc)."
	echo "::error::A code nobody exercises is a promise nobody ran. Either add a test"
	echo "::error::that asserts the error (family, wrapping, context) and re-run"
	echo "::error::with --update to prune the row, or record it deliberately."
	exit 1
fi

echo "✓ every documented error code is either named by a test or frozen in the baseline ($frozen frozen)."
