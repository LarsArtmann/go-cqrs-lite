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
#   3. Dangling "the <X> row above/below" pointers fail when <X> appears
#      nowhere else in the file (the definition vanished; the pointer stayed).
#
# Self-test (--self-test) plants a fixture TODO file exercising every rule:
# dated-recent pass, undated fail, stale fail, duplicate fail, dangling
# pointer fail, resolvable pointer pass.
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
		local d
		d="$(grep -oE '20[0-9]{2}-[0-9]{2}-[0-9]{2}' <<<"$line" | head -1)"
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

	# Rule 3: dangling "the <X> row above/below" pointers.
	lineno=0
	while IFS= read -r line; do
		lineno=$((lineno + 1))
		grep -qiE 'the .+ row (above|below)' <<<"$line" || continue
		local phrase
		phrase="$(sed -nE 's/.*[Tt]he (.+) row (above|below).*/\1/p' <<<"$line" | head -1)"
		[ -z "$phrase" ] && continue
		# The pointer must reference a phrase that exists OUTSIDE this line.
		if [ "$(grep -cF "$phrase" "$file")" -le 1 ]; then
			echo "FAIL: ${file}:${lineno}: 'the ${phrase} row' points at nothing else in the file — dangling pointer" >>"$out"
			fails=$((fails + 1))
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
The M07 wave row above carries the receipts.
MD
	out="$tmp/out"
	fails="$(check_file "$fixture" "$out")"
	rc=0
	[ "$fails" -eq 3 ] || { echo "SELF-TEST FAIL: expected 3 failures (undated, stale, duplicate), got $fails"; cat "$out"; rc=1; }
	grep -q 'without a YYYY-MM-DD' "$out" || { echo "SELF-TEST FAIL: undated rule did not fire"; rc=1; }
	grep -q 'older than' "$out" || { echo "SELF-TEST FAIL: stale rule did not fire"; rc=1; }
	grep -q 'byte-identical duplicate' "$out" || { echo "SELF-TEST FAIL: duplicate rule did not fire"; rc=1; }
	grep -q 'dangling' "$out" && { echo "SELF-TEST FAIL: resolvable pointer (M07 wave exists twice) must pass"; rc=1; }

	# Mutation leg: same fixture with a pointer to a vanished row.
	echo "The ZZZ vanished row above." >>"$fixture"
	out2="$tmp/out2"
	fails2="$(check_file "$fixture" "$out2")"
	[ "$fails2" -eq 4 ] || { echo "SELF-TEST FAIL: dangling-pointer rule did not fire (expected 4 failures, got $fails2)"; rc=1; }

	[ "$rc" -eq 0 ] && echo "check-todo-list-hygiene self-test: OK (dated pass; undated/stale/duplicate/dangling fail)"
	exit "$rc"
fi

cd "$(git rev-parse --show-toplevel)" || exit 1
target="${1:-TODO_LIST.md}"
[ -f "$target" ] || { echo "check-todo-list-hygiene: $target not found"; exit 1; }

out="$(mktemp)" && trap 'rm -f "$out"' EXIT
fails="$(check_file "$target" "$out")"
if [ "$fails" -gt 0 ]; then
	cat "$out"
	echo "check-todo-list-hygiene: $fails violation(s) in $target"
	exit 1
fi
echo "check-todo-list-hygiene: $target clean (dated done rows, no duplicates, no dangling pointers)"
