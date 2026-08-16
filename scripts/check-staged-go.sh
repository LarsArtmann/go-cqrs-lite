#!/usr/bin/env bash
# check-staged-go.sh — staged .go syntax + formatting gate.
#
# Blocks concurrent-session mid-write corruption from entering the git index
# (2026-08-16: "func (w *workor)" and "fojection." both landed staged, twice).
# gofmt -e reports parse errors, -l lists files whose formatting drifted;
# the pre-commit formatters run BEFORE this gate, so ANY output here is
# genuine corruption or drift.
#
# Wired from: .git/hooks/pre-commit (post-BuildFlow appended block — must be
# re-appended after `buildflow precommit install`, like the api-stability
# block), scripts/pre-commit.sh, and scripts/install-hooks.sh.
#
# Usage: bash scripts/check-staged-go.sh   (from anywhere in the repo)
# Exit: 0 = staged .go files parse and are gofmt-clean, 1 = otherwise.

set -euo pipefail

cd "$(git rev-parse --show-toplevel)"

GOFMT="$(command -v gofmt || echo "$(go env GOROOT)/bin/gofmt")"

staged=$(git diff --cached --name-only --diff-filter=ACMR |
	grep '\.go$' |
	grep -v '_templ\.go$' |
	grep -v '\.pb\.go$' |
	grep -v '\.gen\.go$' |
	grep -v '/testdata/' || true)

if [ -z "$staged" ]; then
	echo "✓ No staged .go files — syntax gate skipped"
	exit 0
fi

# shellcheck disable=SC2086 # repo Go paths contain no spaces
# gofmt exits 2 on parse errors — `|| true` keeps the message for the check
# below instead of tripping set -e at the assignment.
out=$("$GOFMT" -e -l $staged 2>&1 || true)
if [ -n "$out" ]; then
	echo "ERROR: staged .go files failed the gofmt gate (parse error or formatting drift):"
	echo "$out"
	echo "Fix the syntax error, or run gofmt / nix fmt and re-stage."
	exit 1
fi

echo "✓ Staged .go files parse and are gofmt-clean"
