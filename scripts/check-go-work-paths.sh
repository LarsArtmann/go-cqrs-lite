#!/usr/bin/env bash
# scripts/check-go-work-paths.sh
# Guard: every go.work `use` path's basename must equal the last segment of
# the module path declared by the go.mod in that directory.
#
# Why: the repo's BuildFlow hygiene check `workspace/go-work-paths` warns when
# a /v4-suffixed module (github.com/…/X/v4) lives at its parent directory X.
# The convention is that module X/v4 lives at directory X/v4, so go.work use
# paths align with module names. This script pins that rule so a new module
# dropped into a parent directory fails fast here instead of surfacing as a
# BuildFlow WARN later.
#
# Usage:
#   scripts/check-go-work-paths.sh            # gate: exit 1 on any mismatch
#   scripts/check-go-work-paths.sh --self-test
#                                             # scratch-dir fixture: one
#                                             # aligned + one misaligned
#                                             # module; asserts both detection
#                                             # and the pass path
# Exit code: 0 = all use-paths aligned (or self-test passed), 1 = mismatches.
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

# mismatches DIR — print one "MISMATCH <use-path> (<reason>)" line per
# unaligned entry in DIR/go.work; prints nothing when clean.
mismatches() {
	local root="$1"
	awk '
		/^use \(/ { inuse = 1; next }
		inuse && /^\)/ { inuse = 0; next }
		inuse {
			gsub(/^[[:space:]]+|[[:space:]]+$/, "")
			if ($0 == "" || $0 == ".") next
			path = $0
			sub(/^\.\//, "", path)
			print path
		}
	' "$root/go.work" | while IFS= read -r dir; do
		if [ ! -f "${root}/${dir}/go.mod" ]; then
			echo "MISMATCH ${dir} (no go.mod — stale use entry or wrong path)"
			continue
		fi
		local modpath base lastseg
		modpath="$(awk '/^module /{print $2; exit}' "${root}/${dir}/go.mod")"
		base="$(basename "$dir")"
		lastseg="${modpath##*/}"
		if [ "$base" != "$lastseg" ]; then
			echo "MISMATCH ${dir} (use basename '${base}' != module last segment '${lastseg}')"
		fi
	done
}

run_gate() {
	cd "$REPO_ROOT"
	if [ ! -f go.work ]; then
		echo "::error::go.work missing" >&2
		exit 1
	fi
	local out
	out="$(mismatches "$REPO_ROOT")"
	if [ -n "$out" ]; then
		echo "::error::go.work use-paths misaligned with module names (dir basename must equal module path's last segment):" >&2
		echo "$out" >&2
		exit 1
	fi
	echo "✓ go.work use-paths aligned with module names ($(grep -cE '^[[:space:]]+\.' go.work) use entries)"
}

FIXTURE_DIR=""

run_self_test() {
	local tmp
	tmp="$(mktemp -d /tmp/go-work-paths-selftest.XXXXXX)"
	FIXTURE_DIR="$tmp"
	mkdir -p "$tmp/repo/alpha/v4" "$tmp/repo/beta/v4"
	printf 'module example.com/alpha/v4\n\ngo 1.27\n' >"$tmp/repo/alpha/v4/go.mod"
	printf 'module example.com/beta/v4\n\ngo 1.27\n' >"$tmp/repo/beta/v4/go.mod"

	# Case 1: aligned (beta/v4 at beta/v4) — must pass.
	{
		echo "go 1.27"
		echo ""
		echo "use ("
		echo "	./alpha/v4"
		echo "	./beta/v4"
		echo ")"
	} >"$tmp/repo/go.work"
	local out
	out="$(mismatches "$tmp/repo")"
	if [ -n "$out" ]; then
		echo "✗ self-test FAIL: aligned fixture flagged: $out" >&2
		exit 1
	fi

	# Case 2: misaligned (beta/v4 module at parent dir beta) — must detect.
	mkdir -p "$tmp/repo/beta"
	mv "$tmp/repo/beta/v4/go.mod" "$tmp/repo/beta/go.mod"
	{
		echo "go 1.27"
		echo ""
		echo "use ("
		echo "	./alpha/v4"
		echo "	./beta"
		echo ")"
	} >"$tmp/repo/go.work"
	out="$(mismatches "$tmp/repo")"
	if ! echo "$out" | grep -q "MISMATCH beta"; then
		echo "✗ self-test FAIL: misaligned fixture not detected (got: '$out')" >&2
		exit 1
	fi

	echo "✓ check-go-work-paths self-test passed (aligned fixture passes, misaligned fixture detected)"
}

trap '[ -n "$FIXTURE_DIR" ] && rm -rf "$FIXTURE_DIR"' EXIT

case "${1:-}" in
--self-test) run_self_test ;;
*) run_gate ;;
esac
