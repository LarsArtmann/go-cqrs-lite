#!/usr/bin/env bash
# check-flake-lock-coherence (F250b design's first leg, M39 2026-11-15).
#
# Incident class (2026-10-23): an auto-configure pass bumped flake.lock
# input revs while the consuming derivation hashes (vendorHash / src hash
# lines in the tracked .nix files) stayed stale — the breakage only
# surfaced when a derivation was actually realised, deep into a verify
# cycle. This check refuses exactly that incoherent COMMIT SHAPE:
#
#   red   : any flake.lock input rev changed, but ZERO sha256 hash lines
#           changed anywhere in tracked .nix files
#   warn  : sha256 hash lines changed, but no input rev changed (the
#           legitimate re-pin class — same rev, corrected hash — so this
#           direction only warns, never fails)
#
# Design note (the "rev↔vendorHash mapping" F250b asks for): a FULL check
# maps each hash-bearing attr in the nix files to its feeding lock input
# and requires THE OWNING attr to change when its input's rev changes.
# That mapping is attr-path-sensitive and brittle to nix restructuring;
# it stays the documented next step. The coarse shape check above catches
# the observed incident class (rev bump + zero hash movement) for the
# inputs that actually feed hash-bearing derivations — most lock inputs
# are path-consumed (the sibling use-block checkouts) and their rev bumps
# legitimately never move a sha256 line, so only HASH_PINNED_INPUTS
# enforce the red rule.
#
# Modes:
#   (default)          HEAD vs working tree (pre-commit/pre-push shape)
#   --baseline <rev>   compare flake.lock against any git rev instead
#   --self-test        fixture mutations, never touches the repo
set -uo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

lock_revs() { # $1 = git object spec of flake.lock ("-" = worktree file)
	if [ "$1" = "-" ]; then
		jq -r '.nodes | to_entries[] | select(.value.locked.rev != null) | "\(.key) \(.value.locked.rev)"' "$repo_root/flake.lock"
	else
		git -C "$repo_root" show "$1:flake.lock" |
			jq -r '.nodes | to_entries[] | select(.value.locked.rev != null) | "\(.key) \(.value.locked.rev)"'
	fi
}

hash_diff_lines() { # $1 = baseline rev ("-" = worktree diff vs HEAD)
	if [ "$1" = "-" ]; then
		git -C "$repo_root" diff HEAD -- '*.nix'
	else
		git -C "$repo_root" diff "$1" -- '*.nix'
	fi | grep -cE '^[+-][+-][+-]?.*(vendorHash|hash) = "sha256-' || true
}

# Lock inputs whose rev feeds a hash-bearing derivation (verified
# 2026-11-15: md-go-validator = mkMdGoValidatorSource src → vendorHash;
# go-finding = cqrs-lint prepared-source dep replace → cqrs-lint
# vendorHash). Extend this list when a new prepared source lands.
HASH_PINNED_INPUTS="md-go-validator go-finding"

is_hash_pinned() {
	local input="$1" pinned
	for pinned in $HASH_PINNED_INPUTS; do
		[ "$input" = "$pinned" ] && return 0
	done
	return 1
}

# Pure verdict core (self-test target): rev-change lines vs hash-change count.
# stdin: diff of old->new rev lists. $1 = hash diff line count.
verdict() {
	local hash_changes="$1" changed=0 exempt=0 name old_rev new_rev
	while read -r name old_rev new_rev; do
		[ -n "$name" ] || continue
		echo "    input $name: $old_rev -> $new_rev" >&2
		if is_hash_pinned "$name"; then
			changed=$((changed + 1))
		else
			exempt=$((exempt + 1))
		fi
	done

	if [ "$changed" -gt 0 ] && [ "$hash_changes" -eq 0 ]; then
		echo "::error::hash-pinned flake.lock input rev bump(s) with ZERO hash-line changes" >&2
		echo "in tracked .nix files — the consuming derivation hashes are stale;" >&2
		echo "re-derive them in the SAME commit." >&2
		return 1
	fi

	if [ "$exempt" -gt 0 ]; then
		echo "    ($exempt path-consumed input rev bump(s) — exempt from the hash rule)" >&2
	fi

	if [ "$changed" -eq 0 ] && [ "$exempt" -eq 0 ] && [ "$hash_changes" -gt 0 ]; then
		echo "::warning::hash lines changed with no pinned-input rev change (re-pin class — confirm deliberate)" >&2
	fi

	return 0
}

self_test() {
	local failures=0

	# Incident shape: rev bumped, no hash movement -> red.
	if echo "md-go-validator aaa bbb" | verdict 0 >/dev/null 2>&1; then
		echo "self-test FAIL: pinned-input rev bump + zero hash changes must be red" >&2
		failures=$((failures + 1))
	fi

	# Path-consumed input bump never needs a hash change -> green.
	if ! echo "cmdguard aaa bbb" | verdict 0 >/dev/null 2>&1; then
		echo "self-test FAIL: path-consumed input rev bump must stay green" >&2
		failures=$((failures + 1))
	fi

	# Coherent shape: rev bumped AND hashes moved -> green.
	if ! echo "md-go-validator aaa bbb" | verdict 2 >/dev/null 2>&1; then
		echo "self-test FAIL: rev bump + hash changes must be green" >&2
		failures=$((failures + 1))
	fi

	# Quiet tree -> green, no noise.
	if ! verdict 0 </dev/null >/dev/null 2>&1; then
		echo "self-test FAIL: quiet tree must be green" >&2
		failures=$((failures + 1))
	fi

	# Re-pin shape: hashes moved, no rev change -> green (warn class).
	if ! verdict 1 </dev/null >/dev/null 2>&1; then
		echo "self-test FAIL: re-pin class must stay green (warn-only)" >&2
		failures=$((failures + 1))
	fi

	[ "$failures" -eq 0 ] && echo "self-test: 5 coherence fixtures verified"
	[ "$failures" -eq 0 ]
}

baseline="HEAD"
while [ $# -gt 0 ]; do
	case "$1" in
	--baseline) baseline="${2:?--baseline needs a rev}" && shift 2 ;;
	--self-test) self_test && exit $? ;;
	*) echo "usage: $0 [--baseline <rev>|--self-test]" >&2 && exit 2 ;;
	esac
done

diff <(lock_revs "$baseline" | sort) <(lock_revs - | sort) |
	awk -F'[ \t]' '/^[<>]/ { key=$2; rev=$3; if ($1 == "<") old[key]=rev; else new[key]=rev }
		END { for (k in old) if (k in new && old[k] != new[k]) print k, old[k], new[k] }' |
	sort |
	verdict "$(hash_diff_lines "$baseline")"
