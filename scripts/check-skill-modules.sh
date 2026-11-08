#!/usr/bin/env bash
# check-skill-modules.sh — every go-cqrs-lite module path claimed by the skill
# docs must resolve to a real module (or package inside one) in THIS checkout,
# unless the mentioning line explicitly marks it deleted/removed.
#
# Motivating incident (2026-10-17..11-08): transport/http and transport/grpc
# were deleted under ADR-0127 but SKILL.md kept teaching them as the importable
# answer for ~3 weeks — nothing diffed doc claims against go.work/disk. This
# gate closes that class: a doc line naming github.com/larsartmann/go-cqrs-lite/<path>
# fails when <path> does not resolve to a module/package on disk AND the line
# carries no deletion marker (delet|remov|former|no longer importable|historical).
#
# Self-test: --self-test plants a fixture repo with (a) a live module claim
# (pass), (b) a deleted-module claim WITHOUT a marker (must fail — the mutation
# proof), (c) a deleted-module claim WITH a marker (must pass).
#
# Usage: check-skill-modules.sh [--self-test]
set -uo pipefail

CSM_SELF_TEST=0
[ "${1:-}" = "--self-test" ] && CSM_SELF_TEST=1

# ── helpers ────────────────────────────────────────────────────────────────
# resolve_claim <repo_root> <path-after-domain> → 0 if the path resolves to a
# module dir or a package dir inside a module; 1 dangling.
resolve_claim() {
	local root="$1" rel="$2"
	[ -z "$rel" ] && return 0 # the root module itself

	IFS='/' read -r -a segs <<<"$rel"
	local n=${#segs[@]}
	local i j d
	for ((i = n; i >= 1; i--)); do
		d="$root"
		for ((j = 0; j < i; j++)); do d="$d/${segs[j]}"; done
		if [ -d "$d" ]; then
			# Deepest existing dir found; some ancestor (incl. itself) must own
			# a go.mod for this to be a module or a package inside one.
			for ((j = i; j >= 1; j--)); do
				d="$root"
				local k
				for ((k = 0; k < j; k++)); do d="$d/${segs[k]}"; done
				[ -f "$d/go.mod" ] && return 0
			done
			return 1
		fi
	done
	return 1
}

# check_docs <repo_root> <doc...> → prints FAIL lines, returns count via stdout
# aggregate through a temp file (bash can't return two values cleanly).
check_docs() {
	local root="$1"
	shift
	local out="${CSM_OUTFILE:-/dev/stdout}"
	: >"$out"
	local doc line rel fails=0 lineno
	local -a matches
	for doc in "$@"; do
		[ -f "$doc" ] || continue
		lineno=0
		while IFS= read -r line; do
			lineno=$((lineno + 1))
			# shellcheck disable=SC2207
			matches=($(grep -oE 'github\.com/larsartmann/go-cqrs-lite(/[A-Za-z0-9_.-]+)+' <<<"$line" || true))
			[ ${#matches[@]} -eq 0 ] && continue
			# A line that talks about the deletion is honest documentation.
			if grep -qiE 'delet|remov|former|no longer importable|historical|proxy-resolv|pinned v4 tags' <<<"$line"; then
				continue
			fi
			local m
			for m in "${matches[@]}"; do
				rel="${m#github.com/larsartmann/go-cqrs-lite}"
				rel="${rel#/}"
				# Strip a trailing major-version segment (/v4, /v5, /v3).
				rel="$(sed -E 's,/v[0-9]+$,,;' <<<"$rel")"
				if ! resolve_claim "$root" "$rel"; then
					echo "FAIL: ${doc}:${lineno}: claims $m but '$rel' resolves to no module on disk" >>"$out"
					fails=$((fails + 1))
				fi
			done
		done <"$doc"
	done
	echo "$fails" >"${CSM_COUNTFILE:-/tmp/csm-count.$$}"
}

# ── self-test ──────────────────────────────────────────────────────────────
if [ "$CSM_SELF_TEST" -eq 1 ]; then
	tmp="$(mktemp -d)"
	trap 'rm -rf "$tmp"' EXIT
	root="$tmp/repo"
	mkdir -p "$root/event/subpkg" "$root/docs"
	printf 'module github.com/larsartmann/go-cqrs-lite\n' >"$root/go.mod"
	printf 'module github.com/larsartmann/go-cqrs-lite/event/v4\n' >"$root/event/go.mod"
	touch "$root/event/subpkg/x.go"
	cat >"$root/docs/doc.md" <<'MD'
Live claim: import "github.com/larsartmann/go-cqrs-lite/event/v4" works.
Package claim: github.com/larsartmann/go-cqrs-lite/event/v4/subpkg also works.
Unmarked dangling claim: use github.com/larsartmann/go-cqrs-lite/transport/http here.
Marked deleted claim: github.com/larsartmann/go-cqrs-lite/transport/grpc was deleted 2026-10-17.
MD
	out="$tmp/out" cnt="$tmp/cnt"
	CSM_OUTFILE="$out" CSM_COUNTFILE="$cnt" check_docs "$root" "$root/docs/doc.md"
	n="$(cat "$cnt")"
	rc=0
	if [ "$n" -ne 1 ]; then
		echo "SELF-TEST FAIL: expected exactly 1 unmarked-dangling claim, got $n"
		rc=1
	fi
	if ! grep -q 'transport/http' "$out"; then
		echo "SELF-TEST FAIL: the planted transport/http claim was not reported"
		rc=1
	fi
	if grep -q 'transport/grpc' "$out"; then
		echo "SELF-TEST FAIL: the marked-deleted claim must be exempt"
		rc=1
	fi
	if grep -q 'event' "$out"; then
		echo "SELF-TEST FAIL: live module/package claims must pass"
		rc=1
	fi
	if [ "$rc" -eq 0 ]; then
		echo "check-skill-modules self-test: OK (live + package + marked-deleted pass; unmarked dangling fails)"
	fi
	exit "$rc"
fi

# ── real mode ──────────────────────────────────────────────────────────────
cd "$(git rev-parse --show-toplevel)" || exit 1
root="."
docs=(
	.agents/skills/go-cqrs-lite/SKILL.md
	.agents/skills/go-cqrs-lite/references/*.md
	.agents/skills/watermill/SKILL.md
	.agents/skills/watermill/references/*.md
)
# Expand globs into existing files.
expanded=()
for d in "${docs[@]}"; do
	[ -f "$d" ] && expanded+=("$d")
done

if [ ${#expanded[@]} -eq 0 ]; then
	echo "check-skill-modules: no doc files found (cwd mismatch?)"
	exit 1
fi

out="$(mktemp)" && trap 'rm -f "$out"' EXIT
cnt="$(mktemp)"
CSM_OUTFILE="$out" CSM_COUNTFILE="$cnt" check_docs "$root" "${expanded[@]}"
n="$(cat "$cnt")"
if [ "$n" -gt 0 ]; then
	cat "$out"
	echo "check-skill-modules: $n dangling module claim(s) — fix the doc or mark the deletion"
	exit 1
fi
echo "check-skill-modules: every claimed module path resolves on disk (docs scanned: ${#expanded[@]})"
