#!/usr/bin/env bash
# check-todo-list-hygiene.sh — structural honesty gates for TODO_LIST.md.
#
# Encodes the 2026-11-03 sweep's debt classes (114 stale done rows, one
# byte-identical duplicate block that survived three docs-health passes, two
# hand-fixed dangling pointers) as mechanical failures:
#
#   1. DONE rows must be dated: every `- [x]` / `> - [x]` checkbox must carry
#      a YYYY-MM-DD date. Undated done rows fail; done rows older than
#      STALE_DAYS (default 14) fail — the header policy deletes finished
#      rows at docs-health passes, so ancient [x] rows are rot.
#   2. Byte-identical duplicate list items (whitespace-normalized) fail.
# "the X row above/below" pointers are deliberately NOT gated — descriptive
# pointer phrases rarely repeat literally and the naive rule false-positived
# on first run (2026-11-08); they stay a docs-health human-audit item.
#
# Self-test (--self-test) plants a fixture TODO file exercising every rule:
# dated-recent pass, undated fail, stale fail, duplicate fail.
#
# Usage: check-todo-list-hygiene.sh [--self-test]
set -uo pipefail

STALE_DAYS="${TODO_HYGIENE_STALE_DAYS:-14}"
TODAY="${TODO_HYGIENE_TODAY:-$(date -u +%Y-%m-%d)}"

# check_file <todo-file> <outfile> → appends FAIL lines, echoes failure count
check_file() {
	local file="$1" out="$2"
	local fails=0 line lineno normalized
	local -A seen=()

	# Rule 1: dated done rows.
	lineno=0
	while IFS= read -r line; do
		lineno=$((lineno + 1))
		grep -qE '^\s*>?\s*-\s*\[x\]' <<<"$line" || continue
		if ! grep -qE '20[0-9]{2}-[0-9]{2}-[0-9]{2}' <<<"$line"; then
			echo "FAIL: ${file}:${lineno}: done checkbox without a YYYY-MM-DD date: ${line:0:100}" >>"$out"
			fails=$((fails + 1))
			continue
		fi
		# Closure date = a date within ~40 chars after a closure keyword; the
		# LAST bare date is only the fallback — receipts often END with older
		# historical citations (e.g. "the documented 2026-10-01 lesson").
		local d
		d="$(grep -oiE '(DONE|ANSWERED|RESOLVED|shipped|RULED|VERIFIED|FIXED|CLOSED|PARKED)[^0-9]{0,40}20[0-9]{2}-[0-9]{2}-[0-9]{2}' <<<"$line" | grep -oE '20[0-9]{2}-[0-9]{2}-[0-9]{2}' | head -1)"
		[ -z "$d" ] && d="$(grep -oE '20[0-9]{2}-[0-9]{2}-[0-9]{2}' <<<"$line" | tail -1)"
		if [[ "$d" < "$(date -u -d "$TODAY -$STALE_DAYS days" +%Y-%m-%d 2>/dev/null || echo 0000-00-00)" ]]; then
			echo "FAIL: ${file}:${lineno}: done checkbox dated $d is older than ${STALE_DAYS}d — delete the row (header policy)" >>"$out"
			fails=$((fails + 1))
		fi
	done <"$file"

	# Rule 2: byte-identical duplicate list items (whitespace-normalized).
	lineno=0
	while IFS= read -r line; do
		lineno=$((lineno + 1))
		grep -qE '^\s*- ' <<<"$line" || continue
		normalized="$(tr -s '[:space:]' ' ' <<<"$line")"
		[ -z "${normalized// /}" ] && continue
		if [ -n "${seen["$normalized"]:-}" ]; then
			echo "FAIL: ${file}:${lineno}: byte-identical duplicate of line ${seen["$normalized"]}: ${normalized:0:100}" >>"$out"
			fails=$((fails + 1))
		else
			seen["$normalized"]="$lineno"
		fi
	done <"$file"

	echo "$fails"
}

if [ "${1:-}" = "--self-test" ]; then
	tmp="$(mktemp -d)"
	trap 'rm -rf "$tmp"' EXIT
	fixture="$tmp/TODO_LIST.md"
	recent="$(date -u -d "$TODAY -2 days" +%Y-%m-%d)"
	old="$(date -u -d "$TODAY -30 days" +%Y-%m-%d)"
	cat >"$fixture" <<MD
# TODO List
- [ ] open row stays open
- [x] dated done row — DONE ${recent} receipt here
- [x] undated done row with no date at all
- [x] stale done row — DONE ${old} should have been deleted
- [ ] alpha row with unique text
- [ ] alpha row with unique text
MD
	out="$tmp/out"
	fails="$(check_file "$fixture" "$out")"
	rc=0
	[ "$fails" -eq 3 ] || {
		echo "SELF-TEST FAIL: expected 3 failures (undated, stale, duplicate), got $fails"
		cat "$out"
		rc=1
	}
	grep -q 'without a YYYY-MM-DD' "$out" || {
		echo "SELF-TEST FAIL: undated rule did not fire"
		rc=1
	}
	grep -q 'older than' "$out" || {
		echo "SELF-TEST FAIL: stale rule did not fire"
		rc=1
	}
	grep -q 'byte-identical duplicate' "$out" || {
		echo "SELF-TEST FAIL: duplicate rule did not fire"
		rc=1
	}
	[ "$rc" -eq 0 ] && echo "check-todo-list-hygiene self-test: OK (dated pass; undated/stale/duplicate fail)"
	exit "$rc"
fi

cd "$(git rev-parse --show-toplevel)" || exit 1
target="${1:-TODO_LIST.md}"
[ -f "$target" ] || {
	echo "check-todo-list-hygiene: $target not found"
	exit 1
}

out="$(mktemp)" && trap 'rm -f "$out"' EXIT
fails="$(check_file "$target" "$out")"
if [ "$fails" -gt 0 ]; then
	cat "$out"
	echo "check-todo-list-hygiene: $fails violation(s) in $target"
	exit 1
fi
echo "check-todo-list-hygiene: $target clean (done rows dated + fresh, no duplicate items)"
