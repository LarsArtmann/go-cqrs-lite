#!/usr/bin/env bash
# check-todo-stale-premises.sh — flag TODO_LIST rows whose BLOCKING premise has
# already shipped.
#
# The motivating rot: the "Post-tag adoption tail" row stayed "blocked ONLY on
# the next schema/decider/snapshot tag wave" for three weeks AFTER its tags
# shipped (2026-10-12 finding). This detector parses rows that claim to be
# blocked on a tag and checks every cited vX.Y.Z against the local tag list
# (mirrors origin via the versions-manifest nightly gate): a cited tag that
# EXISTS while the row still says "blocked" is a STALE PREMISE.
#
# Default mode REPORTS (exit 0) — semantic false positives are possible (a row
# may cite a shipped tag as history, not as the blocker); --strict fails.
#
# Self-test (--self-test) plants a fixture TODO + fake tag list: a
# blocked-on-existing-tag row is flagged (the mutation proof), a
# blocked-on-missing-tag row passes, and a historical citation on an open row
# passes.
#
# Usage: check-todo-stale-premises.sh [--self-test] [--strict]
set -uo pipefail

STRICT=0
SELF_TEST=0
for arg in "$@"; do
	[ "$arg" = "--strict" ] && STRICT=1
	[ "$arg" = "--self-test" ] && SELF_TEST=1
done

# detect <todo-file> <tagsfile> <outfile> → echoes stale-premise count
detect() {
	local file="$1" tags="$2" out="$3"
	local lineno=0 line stale=0 tag
	while IFS= read -r line; do
		lineno=$((lineno + 1))
		grep -qiE '\bblocked\b' <<<"$line" || continue
		grep -qiE '\btag' <<<"$line" || continue
		for tag in $(grep -oE 'v[0-9]+\.[0-9]+\.[0-9]+' <<<"$line" || true); do
			if grep -qxF "$tag" "$tags"; then
				echo "STALE-PREMISE: ${file}:${lineno}: row says blocked, but tag ${tag} EXISTS — re-check the premise: ${line:0:110}" >>"$out"
				stale=$((stale + 1))
			fi
		done
	done <"$file"
	echo "$stale"
}

if [ "$SELF_TEST" -eq 1 ]; then
	tmp="$(mktemp -d)"
	trap 'rm -rf "$tmp"' EXIT
	cat >"$tmp/tags.txt" <<'TAGS'
v4.7.5
v4.12.0
TAGS
	cat >"$tmp/TODO_LIST.md" <<'MD'
- [ ] [BLOCKED] post-tag tail — blocked ONLY on the next schema/decider/snapshot tag wave (v4.12.0 cited as the expected tag)
- [ ] blocked on a tag that has NOT shipped (v9.9.9)
- [ ] open row citing v4.7.5 as historical context only — skip
MD
	out="$tmp/out"
	n="$(detect "$tmp/TODO_LIST.md" "$tmp/tags.txt" "$out")"
	rc=0
	[ "$n" -eq 1 ] || {
		echo "SELF-TEST FAIL: expected 1 stale premise, got $n"
		cat "$out"
		rc=1
	}
	grep -q 'v4.12.0' "$out" || {
		echo "SELF-TEST FAIL: the shipped-tag premise was not flagged"
		rc=1
	}
	grep -q 'v9.9.9' "$out" && {
		echo "SELF-TEST FAIL: unshipped tag must not flag"
		rc=1
	}
	[ "$rc" -eq 0 ] && echo "check-todo-stale-premises self-test: OK (shipped-tag premise flags; unshipped + non-blocked pass)"
	exit "$rc"
fi

cd "$(git rev-parse --show-toplevel)" || exit 1
todo="${1:-TODO_LIST.md}"
[ -f "$todo" ] || {
	echo "check-todo-stale-premises: $todo not found"
	exit 1
}

tags="$(mktemp)"
out="$(mktemp)"
trap 'rm -f "$tags" "$out"' EXIT
git tag -l >"$tags"

n="$(detect "$todo" "$tags" "$out")"
if [ "$n" -gt 0 ]; then
	cat "$out"
	echo "check-todo-stale-premises: $n stale premise(s) in $todo (report mode; --strict fails)"
	[ "$STRICT" -eq 1 ] && exit 1
	exit 0
fi
echo "check-todo-stale-premises: no stale blocking premises in $todo"
