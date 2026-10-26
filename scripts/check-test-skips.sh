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
#   --quiet      print only the summary (no route table, no per-site detail)
#   --self-test  fault-inject the scanner against fixtures
#   --help

set -euo pipefail
export LC_ALL=C

cd "$(git rev-parse --show-toplevel)"

MODE=gate
REPORT=0
QUIET=0
# Deliberate local-only decisions: <ENV_VAR><TAB><reason>. Empty by design —
# every row is a suite that never runs CI, so the reason must carry a decision.
ROUTES_FILE="scripts/test-skip-routes.txt"

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

# scan FILE OUT — "<file>\t<line>\t<status>" for every skip site, where status
# is `ok` or `noreason`.
scan_file() { # $1=file $2=out
	awk -v file="$1" '
		function flush() {
			body = buf
			accepted = (body ~ /\([ \t]*"[^"]/) || (marker != "" && marker ~ /\/\/go-skip:accept[ \t]+[^ \t]/)
			printf "%s\t%d\t%s\n", file, startline, accepted ? "ok" : "noreason"
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

# apps_for_script FLAKE SCRIPT — names of every flake app whose body references
# SCRIPT (an app is a `mkApp "name" [...] ''...''` block; several apps may wrap
# the same script, e.g. ephemeral-X and integration-X).
apps_for_script() { # $1=flake.nix $2=script path fragment
	awk -v needle="$2" '
		/mkApp "[^"]+"/ {
			match($0, /mkApp "[^"]+"/)
			name = substr($0, RSTART + 7, RLENGTH - 8)
			inapp = 1; found = 0
		}
		inapp && index($0, needle) > 0 { found = 1 }
		inapp && /^\.*'"'"';?[[:space:]]*$/ { } # keep awk happy on odd quoting
		inapp && found && /'"'"';[[:space:]]*$/ { print name; inapp = 0; found = 0 }
	' "$1" | sort -u
}

route_report() { # $1=root
	local v class note
	printf '%-34s %-12s %s\n' ENV_VAR ROUTE NOTE
	printf '%-34s %-12s %s\n' ------- ----- -----
	for v in $(skip_env_vars "$1"); do
		IFS=$'\t' read -r class note <<<"$(route_for "$1" "$v")"
		printf '%-34s %-12s %s\n' "$v" "$class" "$note"
	done
}

# skip_env_vars ROOT — only vars that actually gate a skip are routes; the
# rest are knobs (UPDATE_GOLDEN and friends).
skip_env_vars() { # $1=root
	(cd "$1" && grep -rl --include='*_test.go' -E '(^|[^A-Za-z0-9_])(t|tb|rt)\.Skip' . |
		grep -v vendor | grep -v '/testdata/' |
		xargs grep -hoE 'os\.Getenv\("[A-Z0-9_]+"\)' 2>/dev/null |
		sed 's/.*"\(.*\)".*/\1/' | sort -u)
}

# route_for ROOT VAR — print "<class>\t<note>", class being one of
# knob | local-only | workflow | script | app | NONE.
#
# The hops matter: CI rarely sets an env var inline. It calls an app, the app
# wraps a script, and the script exports the var. A route is only real when
# every hop resolves, which is why this walks them instead of grepping one
# file.
route_for() { # $1=root $2=var
	local root="$1" v="$2" wf script call app reason
	case "$v" in
	SOAK_SKIP_* | UPDATE_* | HOME | *_DIR | *_CONCURRENCY | *_GOLDEN | CQRS_* | TURSO_IVM_*)
		printf 'knob\ttest control, not a route that runs a skipped test\n'
		return 0
		;;
	esac
	# Documented local-only decisions come first, so a deliberate choice is
	# not reported as an accident.
	if [ -f "$root/$ROUTES_FILE" ]; then
		reason="$(awk -F'\t' -v v="$v" '$1 == v { print $2; exit }' "$root/$ROUTES_FILE")"
		if [ -n "$reason" ]; then
			printf 'local-only\t%s\n' "$reason"
			return 0
		fi
	fi
	for wf in "$root"/.github/workflows/*.yml; do
		[ -e "$wf" ] || continue
		if grep -q "$v" "$wf"; then
			printf 'workflow\t%s\n' "$(basename "$wf")"
			return 0
		fi
	done
	# Two-hop: the harness script that exports the var must itself be
	# invoked by a workflow (CI integration legs call scripts, not vars).
	for script in "$root"/scripts/*.sh; do
		[ -e "$script" ] || continue
		grep -q "$v" "$script" 2>/dev/null || continue
		call="$(basename "$script")"
		for wf in "$root"/.github/workflows/*.yml; do
			[ -e "$wf" ] || continue
			if grep -q "$call" "$wf"; then
				printf 'script\t%s via %s\n' "$(basename "$wf")" "$call"
				return 0
			fi
		done
	done
	# Third hop: CI calls flake apps, which wrap the scripts. Resolve the app
	# that references the script and require a workflow to invoke it.
	for script in "$root"/scripts/*.sh; do
		[ -e "$script" ] || continue
		grep -q "$v" "$script" 2>/dev/null || continue
		call="$(basename "$script")"
		while IFS= read -r app; do
			[ -n "$app" ] || continue
			for wf in "$root"/.github/workflows/*.yml; do
				[ -e "$wf" ] || continue
				if grep -q "\.#$app" "$wf"; then
					printf 'app\t%s via .#%s\n' "$(basename "$wf")" "$app"
					return 0
				fi
			done
		done < <(apps_for_script "$root/flake.nix" "scripts/$call")
	done
	printf 'NONE\tno workflow, script or flake app runs this\n'
}

# list_unrouted ROOT — "<var>\t<note>" for every env-gated skip with no route.
list_unrouted() { # $1=root
	local v class note
	for v in $(skip_env_vars "$1"); do
		IFS=$'\t' read -r class note <<<"$(route_for "$1" "$v")"
		if [ "$class" = "NONE" ]; then
			printf '%s\t%s\n' "$v" "$note"
		fi
	done
	return 0
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
	if ! grep -q 'a_test.go	10	noreason' "$tmp/rows.txt"; then
		echo "self-test FAIL: TestWithoutReason (line 10) must be flagged" >&2
		failures=$((failures + 1))
	fi
	if grep -q 'a_test.go	6	noreason' "$tmp/rows.txt"; then
		echo "self-test FAIL: a reason string must clear the skip" >&2
		failures=$((failures + 1))
	fi
	if grep -q 'a_test.go	14	noreason' "$tmp/rows.txt"; then
		echo "self-test FAIL: a multi-line reason must clear the skip" >&2
		failures=$((failures + 1))
	fi
	if grep -q 'a_test.go	21	noreason' "$tmp/rows.txt"; then
		echo "self-test FAIL: a non-empty //go-skip:accept must clear the skip" >&2
		failures=$((failures + 1))
	fi
	if ! grep -q '	26	noreason' "$tmp/rows.txt"; then
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

if [ "$QUIET" -eq 1 ] && [ "$bad" -eq 0 ]; then
	echo "✓ $total skip sites, every one carries a reason."
	exit 0
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
