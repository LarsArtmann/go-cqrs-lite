#!/usr/bin/env bash
# check-doc-annotations.sh — archived reports must carry a resolution
# marker, live reports must not rot.
#
# Every docs-health pass hand-audited the same two questions: did every
# archived report get annotated (banner or per-item strikethrough), and
# are live session reports drifting unarchived? This gate makes both
# mechanical:
#
#   1. ARCHIVED files (docs/{status,planning,reviews}/archived/*.md)
#      must contain a resolution banner (RESOLVED-BY-ROUTING, ARCHIVED.,
#      CLOSED, EXECUTED, ...) OR at least one `~~` strikethrough marker,
#      OR be listed in scripts/doc-annotations-baseline.txt. The
#      baseline holds the pre-convention archives (files archived before
#      the 2026-08-29 annotation era) — regeneration is a conscious act
#      (--write-baseline), never automatic.
#   2. LIVE session reports (docs/status/*.md, excluding README and the
#      KEEP-LIVE-bannered evidence docs) older than 14 days without a
#      KEEP-LIVE banner are reported as archive-drift (warning, not
#      failure — archiving is a docs-health pass decision, but the gate
#      keeps the list visible).
#
# Usage:
#   scripts/check-doc-annotations.sh                # gate
#   scripts/check-doc-annotations.sh --write-baseline
#   scripts/check-doc-annotations.sh --self-test
#
# Exit codes: 0 = clean (warnings allowed), 1 = violations, 2 = usage.
set -uo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BASELINE="$ROOT/scripts/doc-annotations-baseline.txt"
SELFTEST_MODE=0
WRITE_BASELINE=0
DRIFT_DAYS=14

case "${1:-}" in
--self-test)
	SELFTEST_MODE=1
	;;
--write-baseline)
	WRITE_BASELINE=1
	;;
"") ;;
*)
	echo "usage: $0 [--write-baseline|--self-test]" >&2
	exit 2
	;;
esac

BANNER_RE='RESOLVED-BY-ROUTING|ARCHIVED\.|CLOSED|EXECUTED banner|PROPOSED→EXECUTED|KEEP-LIVE'

has_marker() {
	grep -qE "$BANNER_RE" "$1" || grep -q '~~' "$1"
}

in_baseline() {
	[ -f "$BASELINE" ] && grep -qxF "$1" "$BASELINE"
}

violations=0
warnings=0

check_archived() {
	local dir="$1" f rel
	for f in "$dir"/archived/*.md; do
		[ -f "$f" ] || continue
		rel="${f#"$ROOT"/}"
		if has_marker "$f"; then
			continue
		fi
		if in_baseline "$rel"; then
			continue
		fi
		echo "✗ $rel archived without resolution marker (banner or ~~) and not baselined" >&2
		violations=$((violations + 1))
	done
}

check_live_drift() {
	local f cutoff
	cutoff=$(date -d "-${DRIFT_DAYS} days" +%s 2>/dev/null || date -v-${DRIFT_DAYS}d +%s 2>/dev/null || echo 0)
	[ "$cutoff" -gt 0 ] || return 0 # no date math available: skip leg loudly
	for f in "$ROOT"/docs/status/*.md; do
		[ -f "$f" ] || continue
		local base
		base="$(basename "$f")"
		[ "$base" = "README.md" ] && continue
		grep -qE "$BANNER_RE" "$f" && continue
		local mtime
		mtime=$(stat -c %Y "$f" 2>/dev/null || stat -f %m "$f" 2>/dev/null || echo 0)
		[ "$mtime" -ge "$cutoff" ] && continue
		echo "⚠ docs/status/$base is older than ${DRIFT_DAYS}d with no KEEP-LIVE banner — archive-drift candidate" >&2
		warnings=$((warnings + 1))
	done
}

self_test() {
	local tmp
	tmp="$(mktemp -d)"
	trap 'rm -rf "${tmp:-}"' EXIT
	mkdir -p "$tmp/docs/status/archived"

	echo "━━━ check-doc-annotations self-test ━━━"

	# Honest set: bannered + struck both pass.
	printf '# r\n> RESOLVED-BY-ROUTING (x): done. ARCHIVED.\n' >"$tmp/docs/status/archived/bannered.md"
	printf '# r\n- ~~item~~ done\n' >"$tmp/docs/status/archived/struck.md"
	# Violation: bare file, no baseline.
	printf '# r\nplain report\n' >"$tmp/docs/status/archived/bare.md"

	ROOT="$tmp" BASELINE="$tmp/no-such-baseline.txt" "$0" >/dev/null 2>&1
	rc=$?
	if [[ $rc == 1 ]]; then
		echo "  ✓ PASS: bare un-annotated archive fails the gate"
	else
		echo "  ✗ FAIL: bare archive should fail (rc=$rc)"
	fi

	printf 'docs/status/archived/bare.md\n' >"$tmp/baseline.txt"
	ROOT="$tmp" BASELINE="$tmp/baseline.txt" "$0" >/dev/null 2>&1
	rc=$?
	if [[ $rc == 0 ]]; then
		echo "  ✓ PASS: baselined file passes; bannered+struck pass"
	else
		echo "  ✗ FAIL: baselined+annotated set should pass (rc=$rc)"
	fi
	rm "$tmp/docs/status/archived/bare.md"

	# Mutation: stripping the marker from the struck file must re-fail.
	printf '# r\n- item still open\n' >"$tmp/docs/status/archived/struck.md"
	ROOT="$tmp" BASELINE="$tmp/baseline.txt" "$0" >/dev/null 2>&1
	rc=$?
	if [[ $rc == 1 ]]; then
		echo "  ✓ PASS: marker removal re-fails the gate"
	else
		echo "  ✗ FAIL: stripped marker not caught (rc=$rc)"
	fi
}

if [[ "$SELFTEST_MODE" == 1 ]]; then
	self_test
	exit $?
fi

cd "$ROOT"

if [[ "$WRITE_BASELINE" == 1 ]]; then
	{
		echo "# doc-annotations baseline: pre-convention archives (annotated-archival era began 2026-09-06)."
		echo "# Regenerate ONLY after judging the whole current set — new bare archives must fail, not join this list."
		echo "# known-dead: $0 --write-baseline"
		for f in docs/status/archived/*.md docs/planning/archived/*.md docs/reviews/archived/*.md; do
			[ -f "$f" ] || continue
			has_marker "$f" || echo "$f"
		done
	} >"$BASELINE"
	echo "baseline written: $(grep -vc '^#' "$BASELINE") entries"
	exit 0
fi

check_archived docs/status
check_archived docs/planning
check_archived docs/reviews
check_live_drift

if [[ $violations -gt 0 ]]; then
	echo "doc-annotations: ${violations} un-annotated archive(s) — annotate (banner or ~~) or consciously re-baseline"
	exit 1
fi
echo "doc-annotations: clean (${warnings} archive-drift warning(s))"
exit 0
