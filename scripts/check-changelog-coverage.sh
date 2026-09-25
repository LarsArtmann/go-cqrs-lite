#!/usr/bin/env bash
# check-changelog-coverage.sh — the REVERSE changelog gate: consumer-visible
# surface that CHANGELOG never mentions must not ship silently.
#
# check-changelog-symbols.sh answers "does everything the CHANGELOG cites
# exist?"; this answers "does the CHANGELOG know about every module and every
# headline export?". The blind-spot case (2026-09-24): the mesh-demo, E019,
# and docserver DataProducts all shipped green with zero CHANGELOG mention —
# nothing cited, nothing to check, no gate fired.
#
# Legs:
#   1. module coverage — every module alias with exports in
#      docs/api_surface.txt must be named in CHANGELOG.md at least once.
#   2. headline-export coverage — `func`/`type` exports whose name starts
#      with an uppercase letter AND carries domain vocabulary (heuristic:
#      NOT one of the small common-primitive words below) must appear in
#      CHANGELOG.md OR scripts/changelog-coverage-baseline.txt (report-only
#      allowlist; prune with --prune-baseline).
#
# Leg 2 is REPORT-ONLY for pre-baselined symbols (--fail-on-new makes NEW
# unmentioned headline exports fail — wire that once the baseline is trusted).
#
# Usage:
#   bash scripts/check-changelog-coverage.sh                # gate (modules hard, headline report)
#   bash scripts/check-changelog-coverage.sh --fail-on-new  # + fail on NEW unmentioned exports
#   bash scripts/check-changelog-coverage.sh --prune-baseline
#   bash scripts/check-changelog-coverage.sh --self-test
set -uo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
GOLDEN="$ROOT/docs/api_surface.txt"
CHANGELOG="$ROOT/CHANGELOG.md"
BASELINE="$ROOT/scripts/changelog-coverage-baseline.txt"

# Common-primitive export names that a changelog need not call out
# individually (they ride module-level entries). Extend freely.
COMMON_NAMES='^(New|Default|Config|Options?|Version|String|Error|Err|ID|Key|Name|Type|Kind|Result|Status|Store|Engine|Client|Server|Manager|Handler|Context|Nil|Zero|Test|Must|Wrap|Clone|Equal|Is|Has|Get|Set|List|Map|Slice|SliceOf|Ptr|Value|Of|For|From|To|With|Without|Len|Size|Cap|Empty|Valid|Parse|Format|Marshal|Unmarshal|Encode|Decode|Read|Write|Open|Close|Flush|Init|Run|Start|Stop|Reset|Copy|Add|Remove|Delete|Update|Create|Apply|Load|Save|Find|First|Last|Next|Prev|Min|Max|Sum|Count|Range|Each|All|Any|Filter|Sort|Group|Join|Split|Trim|Contains|HasPrefix|HasSuffix|Index|Replace|Repeat|Upper|Lower|Title)$'

failures=0
MODE="gate"

self_test() {
	# No `local`: the EXIT trap below runs at SCRIPT scope where a local
	# would already be out of scope (unbound under set -u).
	tmp="$(mktemp -d)"
	trap 'rm -rf "$tmp"' EXIT
	mkdir -p "$tmp/scripts" "$tmp/docs"
	printf 'mod/func DomainThing\nmod/func NewThing\n' >"$tmp/docs/api_surface.txt"
	printf '# Unreleased\n\n- mod: shipped DomainThing\n' >"$tmp/CHANGELOG.md"
	touch "$tmp/scripts/changelog-coverage-baseline.txt"

	echo "━━━ check-changelog-coverage self-test ━━━"

	# Leg 1 positive + leg 2: NewThing rides the common-name exemption.
	GOLDEN="$tmp/docs/api_surface.txt" CHANGELOG="$tmp/CHANGELOG.md" BASELINE="$tmp/scripts/changelog-coverage-baseline.txt" \
		bash "$0" >/dev/null 2>&1
	local rc=$?
	if [ $rc -eq 0 ]; then
		echo "  ✓ PASS: covered module + common-name exemption pass"
	else
		echo "  ✗ FAIL: covered module should pass"
		return 1
	fi

	# Leg 1 mutation: module never mentioned.
	printf 'other/func DomainThing\n' >>"$tmp/docs/api_surface.txt"
	GOLDEN="$tmp/docs/api_surface.txt" CHANGELOG="$tmp/CHANGELOG.md" BASELINE="$tmp/scripts/changelog-coverage-baseline.txt" \
		bash "$0" >/dev/null 2>&1
	rc=$?
	[ $rc -eq 1 ] && echo "  ✓ PASS: unmentioned module fails leg 1" ||
		{ echo "  ✗ FAIL: unmentioned module not caught"; return 1; }
	printf 'mod/func DomainThing\nmod/func NewThing\n' >"$tmp/docs/api_surface.txt"

	# Leg 2 strict: unmentioned domain export fails under --fail-on-new.
	GOLDEN="$tmp/docs/api_surface.txt" CHANGELOG="$tmp/CHANGELOG.md" BASELINE="$tmp/scripts/changelog-coverage-baseline.txt" \
		bash "$0" --fail-on-new >/dev/null 2>&1
	rc=$?
	[ $rc -eq 1 ] && echo "  ✓ PASS: unmentioned domain export fails under --fail-on-new" ||
		{ echo "  ✗ FAIL: strict leg not caught"; return 1; }

	echo "self-test: all legs behave"
	return 0
}

for arg in "$@"; do
	case "$arg" in
	--fail-on-new) MODE="strict" ;;
	--prune-baseline) MODE="prune" ;;
	--self-test)
		self_test
		exit $?
		;;
	*)
		echo "usage: $0 [--fail-on-new|--prune-baseline|--self-test]" >&2
		exit 2
		;;
	esac
done

[ -f "$GOLDEN" ] || { echo "::error::missing $GOLDEN" >&2; exit 1; }
[ -f "$CHANGELOG" ] || { echo "::error::missing $CHANGELOG" >&2; exit 1; }
[ -f "$BASELINE" ] || : >"$BASELINE"

# Leg 1: module aliases vs CHANGELOG mentions.
mods="$(cut -d' ' -f1 "$GOLDEN" | cut -d'/' -f1 | sort -u)"
for mod in $mods; do
	if ! grep -qF "$mod" "$CHANGELOG"; then
		echo "✗ module '$mod' has exports in the api golden but is NEVER mentioned in CHANGELOG.md" >&2
		failures=$((failures + 1))
	fi
done

# Leg 2: headline exports never mentioned (report; strict fails on new ones).
missing=()
while IFS= read -r line; do
	sym="${line##* }"
	kind_seg="${line#*/}"
	kind="${kind_seg%% *}"

	[ "$kind" = "func" ] || [ "$kind" = "type" ] || continue
	case "$sym" in
	[[:upper:]]*) ;;
	*) continue ;;
	esac
	grep -qE "^$COMMON_NAMES$" <<<"$sym" && continue
	grep -qF "$sym" "$CHANGELOG" && continue
	grep -qxF "$line" "$BASELINE" && continue
	missing+=("$line")
done < <(sort -u "$GOLDEN")

if [ "${#missing[@]}" -gt 0 ]; then
	echo "::warning::${#missing[@]} headline export(s) never mentioned in CHANGELOG (full list):"
	printf '  %s\n' "${missing[@]}"

	if [ "$MODE" = "strict" ]; then
		echo "::error::--fail-on-new: unmentioned, unbaselined headline exports above" >&2
		failures=$((failures + 1))
	elif [ "$MODE" = "prune" ]; then
		printf '  %s\n' "${missing[@]}" >>"$BASELINE"
		sort -u "$BASELINE" -o "$BASELINE"
		echo "baseline appended: scripts/changelog-coverage-baseline.txt ($(wc -l <"$BASELINE") entries) — review the diff before committing"
	fi
fi

echo "changelog-coverage: modules=$([ "$failures" -eq 0 ] && echo OK || echo DRIFT), unmentioned headline exports=${#missing[@]}"

[ "$failures" -eq 0 ] || exit 1
exit 0
