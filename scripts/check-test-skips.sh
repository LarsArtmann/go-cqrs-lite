#!/usr/bin/env bash
# check-test-skips.sh — a skipped test must say WHY, and an environment-gated
# skip must have a route that actually runs it.
#
# Why: 217 `t.Skip*` sites existed at the 2026-10-26 census. A skip without a
# reason is indistinguishable from abandonment — nobody can tell whether the
# test waits for a database, covers a platform we do not ship, or papers over a
# bug. A skip WITH a reason can still be a silent loss: if the only gate that
# sets the env var was deleted, the suite is green because the feature is never
# exercised.
#
# Rules enforced (hard):
#   * every t.Skip/t.Skipf/t.SkipNow/tb.Skip/rt.Skip carries a non-empty
#     reason string, or an annotation `//go-skip:accept <reason>` on the call
#     line or the line above (rapid's rt.Skip takes no argument).
# Report (advisory, --report):
#   * env vars that gate a skip and appear in NO workflow file — those tests
#     only run when a human remembers, which is how coverage quietly leaves.
#
# Usage: bash scripts/check-test-skips.sh [flags]
#   (default)    fail on a reasonless skip
#   --report     also print the env-gated-skip route table
#   --self-test  fault-inject the scanner against fixtures
#   --help

set -euo pipefail
export LC_ALL=C

cd "$(git rev-parse --show-toplevel)"

MODE=gate
REPORT=0

die() {
	echo "::error::$1" >&2
	exit 1
}

usage() {
	sed -n '2,/^$/p' "$0" | sed 's/^# \{0,1\}//'
}

while [ $# -gt 0 ]; do
	case "$1" in
	--report) REPORT=1 ;;
	--self-test) MODE=selftest ;;
	--help | -h)
		usage
		exit 0
		;;
	*) die "unknown flag: $1 (try --help)" ;;
	esac
	shift
done

# scan FILE OUT — "<file>\t<line>\t<status>" for every skip site, where status
# is `ok` or `noreason`.
scan_file() { # $1=file $2=out
	awk -v file="$1" '
		function flush() {
			body = buf
			accepted = (body ~ /\([ \t]*"[^"]/) || (marker != "" && marker ~ /\/\/go-skip:accept[ \t]+[^ \t]/)
			if (!accepted) printf "%s\t%d\tnoreason\n", file, startline
			buf = ""; pending = 0; marker = ""
		}
		{
			if (pending) {
				buf = buf " " $0
				if ($0 ~ /\)/) flush()
				next
			}
			if ($0 ~ /(^|[^A-Za-z0-9_])(t|tb|rt)\.Skip(f|Now)?\(/) {
				buf = $0; startline = FNR
				if ($0 ~ /\)/) { flush() } else { pending = 1; curmarker = marker }
				next
			}
			marker = $0
		}
		END { if (pending) flush() }' "$1" >>"$2"
}

scan_tree() { # $1=root $2=out
	: >"$2"
	while IFS= read -r file; do
		scan_file "$1/$file" "$2"
	done < <(cd "$1" && grep -rl --include='*_test.go' -E '(^|[^A-Za-z0-9_])(t|tb|rt)\.Skip' . |
		grep -v '^./vendor/' | grep -v '/testdata/' | grep -v '^./node_modules/' |
		sed 's|^\./||' | sort)
}

route_report() { # $1=root
	local envs v wf hit
	envs="$(cd "$1" && grep -rhoE 'os\.Getenv\("[A-Z0-9_]+"\)' --include='*_test.go' . |
		grep -v vendor | sed 's/.*"\(.*\)".*/\1/' | sort -u)"
	printf '%-32s %-10s %s\n' ENV_VAR WORKFLOW OWNER_HINT
	printf '%-32s %-10s %s\n' ------- -------- ----------
	for v in $envs; do
		hit=""
		for wf in "$1"/.github/workflows/*.yml; do
			if grep -q "$v" "$wf"; then
				hit="$(basename "$wf")"
				break
			fi
		done
		if [ -z "$hit" ]; then
			printf '%-32s %-10s %s\n' "$v" "NONE" "no workflow sets this — runs only on a human's box"
		else
			printf '%-32s %-10s %s\n' "$v" "yes" "$hit"
		fi
	done
}

self_test() {
	local tmp failures=0 rc out
	tmp="$(mktemp -d)"
	# shellcheck disable=SC2064
	trap "rm -rf '$tmp'" EXIT

	mkdir -p "$tmp/tree"
	cat >"$tmp/tree/a_test.go" <<'EOF'
package a

import "testing"

func TestWithReason(t *testing.T) {
	t.Skip("needs a Postgres container")
}

func TestWithoutReason(t *testing.T) {
	t.Skip()
}

func TestMultiLine(t *testing.T) {
	t.Skipf(
		"needs Redis: %v", "none",
	)
}

func TestAccepted(t *testing.T) {
	//go-skip:accept rapid owns the skip API; no reason argument exists
	rt.Skip()
}

func TestEmptyMarker(t *testing.T) {
	//go-skip:accept
	t.Skip()
}
EOF
	scan_tree "$tmp/tree" "$tmp/rows.txt"
	if [ "$(grep -c noreason "$tmp/rows.txt")" -ne 2 ]; then
		echo "self-test FAIL: expected exactly 2 reasonless skips, got:" >&2
		cat "$tmp/rows.txt" >&2
		failures=$((failures + 1))
	fi
	if ! grep -q 'a_test.go	11	noreason' "$tmp/rows.txt"; then
		echo "self-test FAIL: TestWithoutReason (line 11) must be flagged" >&2
		failures=$((failures + 1))
	fi
	if grep -q 'a_test.go	7	noreason' "$tmp/rows.txt"; then
		echo "self-test FAIL: a reason string must clear the skip" >&2
		failures=$((failures + 1))
	fi
	if grep -q 'a_test.go	16	noreason' "$tmp/rows.txt"; then
		echo "self-test FAIL: a multi-line reason must clear the skip" >&2
		failures=$((failures + 1))
	fi
	if grep -q 'a_test.go	25	noreason' "$tmp/rows.txt"; then
		echo "self-test FAIL: a non-empty //go-skip:accept must clear the skip" >&2
		failures=$((failures + 1))
	fi
	if ! grep -q '	24	noreason' "$tmp/rows.txt"; then
		echo "self-test FAIL: an EMPTY //go-skip:accept must not clear the skip" >&2
		failures=$((failures + 1))
	fi

	if [ "$failures" -gt 0 ]; then
		echo "::error::check-test-skips self-test: $failures case(s) failed" >&2
		exit 1
	fi
	echo "✓ check-test-skips self-test: 6/6 fault-injection cases behave"
}

if [ "$MODE" = "selftest" ]; then
	self_test
	exit 0
fi

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
scan_tree "$PWD" "$tmp/rows.txt"

total="$(grep -c . "$tmp/rows.txt" || true)"
bad="$(grep -c noreason "$tmp/rows.txt" || true)"

if [ "$REPORT" -eq 1 ]; then
	route_report "$PWD"
	echo ""
fi

if [ "$bad" -gt 0 ]; then
	echo "::error::$bad of $total skip site(s) carry no reason:"
	sed 's/^/  /' <(grep noreason "$tmp/rows.txt")
	echo ""
	echo 'Add a reason: t.Skip("needs <what> so that <why>")'
	echo 'or annotate: //go-skip:accept <reason> (for APIs with no reason argument)'
	exit 1
fi
echo "✓ $total skip sites, every one carries a reason (or an //go-skip:accept annotation)."
