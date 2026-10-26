#!/usr/bin/env bash
# check-systemscenario-adoption.sh — a module that boots a real system in its
# tests should prove behavior through the system-level harness, or say why not.
#
# Why: systemscenario (ADR-0153) is the tier that boots system.New with a
# journal, a projection host and a bus, then asserts on observable outcomes.
# Nothing measured whether new composition-root tests used it, so each new
# module re-invented a bespoke wiring harness — the consumer-shaped drift the
# harness exists to prevent. This gate makes the choice explicit:
#
#   * a testModules module whose tests call system.New but never import
#     systemscenario must be on the allowlist WITH A REASON,
#   * a NEW such module fails until it either adopts the harness or records
#     why the lower-level boot is the right tool there,
#   * an allowlist row that no longer applies fails as stale (prune it).
#
# Adoption is measured, not asserted: the report prints how many systemscenario
# call sites each module has, so progress and regressions are both visible.
#
# Usage: bash scripts/check-systemscenario-adoption.sh [flags]
#   (default)    verify against the allowlist
#   --report     print the adoption table and exit 0
#   --update     rewrite the allowlist (preserving existing reasons)
#   --self-test  fault-inject the compare logic against fixtures
#   --help
#
# Environment: SYSTEMSCENARIO_ALLOWLIST (default
# scripts/systemscenario-adoption-allowlist.txt)

set -euo pipefail
export LC_ALL=C

cd "$(git rev-parse --show-toplevel)"

ALLOWLIST="${SYSTEMSCENARIO_ALLOWLIST:-scripts/systemscenario-adoption-allowlist.txt}"
MODE=gate

die() {
	echo "::error::$1" >&2
	exit 1
}

usage() {
	sed -n '2,/^$/p' "$0" | sed 's/^# \{0,1\}//'
}

while [ $# -gt 0 ]; do
	case "$1" in
	--report) MODE=report ;;
	--update) MODE=update ;;
	--self-test) MODE=selftest ;;
	--help | -h)
		usage
		exit 0
		;;
	*) die "unknown flag: $1 (try --help)" ;;
	esac
	shift
done

# adoption ROOT OUT — "<module>\t<boots>\t<scenario_sites>".
adoption() { # $1=root $2=out
	local mods mods_longest
	mods="$(cd "$1" && sed -n '/testModules = \[/,/^[[:space:]]*\];/p' flake.nix |
		grep -oE '"[^"]+"' | tr -d '"' | sort)"
	mods_longest="$(printf '%s\n' "$mods" | awk '{ print length, $0 }' | sort -rn | cut -d' ' -f2- | tr '\n' ' ')"
	[ -n "$mods_longest" ] || die "could not read testModules from flake.nix"

	{
		(cd "$1" && grep -rl --include='*_test.go' 'system\.New(' . | grep -v '^./vendor/' | sed 's|^\./||')
		(cd "$1" && grep -rl --include='*_test.go' 'systemscenario' . | grep -v '^./vendor/' | sed 's|^\./||')
	} | sort -u |
		awk -v mods="$mods_longest" '
			{
				dir = $0; sub(/\/[^\/]*$/, "", dir)
				n = split(mods, m, " "); mod = ""
				for (i = 1; i <= n; i++)
					if (dir == m[i] || index(dir, m[i] "/") == 1) { mod = m[i]; break }
				if (mod != "") print mod
			}' | sort -u >"$2.modules"

	: >"$2"
	while IFS= read -r mod; do
		local boots sites
		boots="$(cd "$1" && grep -rl --include='*_test.go' 'system\.New(' "$mod" 2>/dev/null | wc -l)"
		sites="$(cd "$1" && grep -rho --include='*_test.go' 'systemscenario\.[A-Z][A-Za-z0-9_]*' "$mod" 2>/dev/null | wc -l)"
		printf '%s\t%s\t%s\n' "$mod" "$boots" "$sites" >>"$2"
	done <"$2.modules"
}

# offenders ADOPTION OUT — modules that boot a system but never touch the harness.
offenders() { # $1=adoption $2=out
	awk -F'\t' '$2 > 0 && $3 == 0 { print $1 }' "$1" | sort >"$2"
}

allow_keys() { # file -> module names, reasons stripped
	[ -f "$1" ] || return 0
	awk -F'\t' '/^#/ || NF == 0 { next } { print $1 }' "$1" | sort -u
}

compare_sets() { # CURRENT_ALLOWED_FROM_ALLOWLIST OFFENDERS OUT
	local rc=0
	awk -v cur_file="$1" -v off_file="$2" '
		FILENAME == off_file { off[$0] = 1; next }
		FILENAME == cur_file { listed[$0] = 1; next }
		END {
			fail = 0
			for (m in off) if (!(m in listed)) { print "NEW " m; fail = 1 }
			for (m in listed) if (!(m in off)) print "STALE " m
			exit fail
		}' "$2" "$1" | sort >"$3" || rc=$?
	return "$rc"
}

self_test() {
	local tmp failures=0 rc out
	tmp="$(mktemp -d)"
	# shellcheck disable=SC2064
	trap "rm -rf '$tmp'" EXIT

	printf 'a\t3\t0\nb\t2\t7\nc\t0\t4\n' >"$tmp/adoption.txt"
	offenders "$tmp/adoption.txt" "$tmp/off.txt"
	if [ "$(grep -c . "$tmp/off.txt")" -ne 1 ] || ! grep -qx a "$tmp/off.txt"; then
		echo "self-test FAIL: only modules that boot without the harness are offenders" >&2
		failures=$((failures + 1))
	fi

	printf '# header\na\tsystem-level boot predates the harness\n' >"$tmp/allow.txt"
	allow_keys "$tmp/allow.txt" >"$tmp/keys.txt"
	if ! compare_sets "$tmp/keys.txt" "$tmp/off.txt" "$tmp/rows.txt" >/dev/null; then
		echo "self-test FAIL: a listed offender must be green" >&2
		failures=$((failures + 1))
	fi

	printf '# header\n' >"$tmp/allow.txt"
	allow_keys "$tmp/allow.txt" >"$tmp/keys.txt"
	rc=0
	compare_sets "$tmp/keys.txt" "$tmp/off.txt" "$tmp/rows.txt" || rc=$?
	if [ "$rc" -eq 0 ] || ! grep -q "NEW a" "$tmp/rows.txt"; then
		echo "self-test FAIL: an unlisted offender must be red and named" >&2
		failures=$((failures + 1))
	fi

	printf '# header\nzzz\tno longer boots a system\n' >"$tmp/allow.txt"
	allow_keys "$tmp/allow.txt" >"$tmp/keys.txt"
	rc=0
	compare_sets "$tmp/keys.txt" "$tmp/off.txt" "$tmp/rows.txt" || rc=$?
	if [ "$rc" -eq 0 ] || ! grep -q "STALE zzz" "$tmp/rows.txt"; then
		echo "self-test FAIL: a stale allowlist row must be red and named" >&2
		failures=$((failures + 1))
	fi

	if [ "$failures" -gt 0 ]; then
		echo "::error::check-systemscenario-adoption self-test: $failures case(s) failed" >&2
		exit 1
	fi
	echo "✓ check-systemscenario-adoption self-test: 4/4 fault-injection cases behave"
}

if [ "$MODE" = "selftest" ]; then
	self_test
	exit 0
fi

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
adoption "$PWD" "$tmp/adoption.txt"
offenders "$tmp/adoption.txt" "$tmp/offenders.txt"

if [ "$MODE" = "report" ]; then
	printf '%-26s %-10s %s\n' MODULE BOOTS HARNESS_CALLS
	printf '%-26s %-10s %s\n' ------ ----- -------------
	awk -F'\t' '{ printf "%-26s %-10s %s\n", $1, $2, $3 }' "$tmp/adoption.txt"
	exit 0
fi

if [ "$MODE" = "update" ]; then
	{
		printf '%s\n' \
			"# Modules that boot a real system in tests WITHOUT using systemscenario." \
			"# Generated by: bash scripts/check-systemscenario-adoption.sh --update" \
			"#" \
			"# Every row is a deliberate choice, so every row needs a REASON:" \
			"#   <module><TAB><why the lower-level boot is the right tool here>" \
			"# A module that adopts the harness disappears from this list; --update" \
			"# preserves the reason on rows that survive."
		if [ -f "$ALLOWLIST" ]; then
			awk -F'\t' '
				NR == FNR {
					if ($0 ~ /^[[:space:]]*#/ || $0 == "") next
					reason[$1] = substr($0, length($1) + 2)
					next
				}
				{ print $1 "\t" (($1 in reason) ? reason[$1] : "<TODO: reason>") }' \
				"$ALLOWLIST" "$tmp/offenders.txt"
		else
			awk '{ print $1 "\t<TODO: reason>" }' "$tmp/offenders.txt"
		fi
	} >"$ALLOWLIST"
	echo "✓ Wrote $(grep -vc '^#' "$ALLOWLIST") allowlist rows to $ALLOWLIST"
	echo "  Fill in any '<TODO: reason>' row — the gate fails on an empty reason."
	exit 0
fi

[ -f "$ALLOWLIST" ] || die "allowlist '$ALLOWLIST' is missing — run: bash scripts/check-systemscenario-adoption.sh --update"

# A reason is mandatory: `<module>\t<reason>` with real text.
if grep -q '<TODO: reason>' "$ALLOWLIST"; then
	echo "::error::allowlist rows still carry '<TODO: reason>':"
	grep -n '<TODO: reason>' "$ALLOWLIST" | sed 's/^/  /'
	exit 1
fi
if awk -F'\t' '$1 !~ /^#/ && NF > 0 && (NF < 2 || $2 == "") { found = 1 } END { exit !found }' "$ALLOWLIST"; then
	die "every allowlist row needs a reason column (module<TAB>reason)"
fi

allow_keys "$ALLOWLIST" >"$tmp/keys.txt"
rc=0
compare_sets "$tmp/keys.txt" "$tmp/offenders.txt" "$tmp/rows.txt" || rc=$?
if [ "$rc" -eq 0 ]; then
	echo "✓ systemscenario adoption: $(grep -c . "$tmp/adoption.txt") modules boot or use the harness; $(grep -c . "$tmp/offenders.txt") allowlisted with reasons."
	exit 0
fi
echo "::error::systemscenario adoption drift:"
sed 's/^/  /' "$tmp/rows.txt"
echo ""
echo "Adopt the harness (systemscenario.New/Given/When/Then) or record why not:"
echo "  add '<module>\\t<reason>' to $ALLOWLIST"
exit 1
