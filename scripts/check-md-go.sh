#!/usr/bin/env bash
# check-md-go.sh — md-go-validator gate, BASELINE + ANNOTATION form.
#
# Policy (2026-09-21, M23 — the md-go-validator CI-integration TODO):
#   Every ```go fence in live (non-archived) docs must parse as Go or carry
#   an explicit `// skip-validate` annotation inside the fence; frozen
#   history is baselined in scripts/md-go-baseline.txt instead. The gate
#   fails when:
#     - a NEW validation error appears anywhere (not in the baseline), or
#     - the baseline references a non-archived path (it must never become
#       a dumping ground for live docs — annotate or fix instead).
#   Shrinking is always allowed: fixed archived errors leave inert entries
#   that can be pruned with --update-baseline. New errors in archived trees
#   also fail — regenerate the baseline only for genuinely new history.
#
# Path portability: md-go-validator keys baselines on ABSOLUTE paths (the
# input path is absolutized before the walk). The committed baseline is
# repo-relative; this script re-absolutizes it at runtime, so the artifact
# is portable across machines and CI.
#
# Usage:
#   bash scripts/check-md-go.sh                    # gate (default)
#   bash scripts/check-md-go.sh --update-baseline  # regenerate baseline
set -euo pipefail

cd "$(git rev-parse --show-toplevel)"

BASELINE="scripts/md-go-baseline.txt"
CONFIG=".md-go-validator.yaml"
ARCHIVE_SEGMENT='(^|/)archive(d)?/'

baseline_entries() {
	grep -cv '^#' "$BASELINE"
}

if ! command -v md-go-validator >/dev/null 2>&1; then
	echo "SKIP: md-go-validator not on PATH (use: nix run .#check-md-go — packaged binary)"
	exit 0
fi

if [ ! -f "$CONFIG" ]; then
	echo "::error::$CONFIG missing — run 'md-go-validator --init' and commit it."
	exit 1
fi

if [ ! -f "$BASELINE" ]; then
	echo "::error::$BASELINE missing — run 'bash scripts/check-md-go.sh --update-baseline' and commit it."
	exit 1
fi

# Policy: the baseline is for frozen history only. Live docs must carry
# // skip-validate annotations (or real fence fixes) — never baseline rows.
check_policy() {
	local bad
	bad=$(grep -v '^#' "$BASELINE" | cut -d: -f1 | sort -u | grep -Ev "$ARCHIVE_SEGMENT" || true)
	if [ -n "$bad" ]; then
		echo "::error::$BASELINE references non-archived paths — live docs must carry"
		echo "::error://'// skip-validate' annotations (or real fixes), not baseline entries:"
		echo "$bad" | sed 's/^/  - /'
		return 1
	fi
	return 0
}

if [ "${1:-}" = "--update-baseline" ]; then
	tmp=$(mktemp)
	trap 'rm -f "$tmp"' EXIT
	# The tool exits 1 whenever errors exist — including when saving a
	# baseline of them. Its non-zero exit here is expected, not a failure.
	md-go-validator . -q --save-baseline "$tmp" >/dev/null || true
	{
		echo "# md-go-validator ratchet baseline (policy in scripts/check-md-go.sh)."
		echo "# Format: <repo-relative-file>:<line>:<errorCode> — only errors NOT listed here fail the gate."
		echo "# Entries may ONLY reference archived/history paths (*/archive*/); live docs use"
		echo "# // skip-validate annotations or real fixes. Regenerate:"
		echo "#   md-go-validator . -q --save-baseline /tmp/base && \\"
		echo "#   { head -5 scripts/md-go-baseline.txt; sed \"s|^$PWD/||\" /tmp/base | sort -u; } > scripts/md-go-baseline.txt"
		sed "s|^$PWD/||" "$tmp" | grep -v '^#' | sort -u
	} >"$BASELINE"
	echo "Baseline written: $(baseline_entries) known error(s)."
	# The file is written for inspection, but a baseline that grew into live
	# docs is exactly the abuse this policy forbids — fail loud on it.
	check_policy || {
		echo "::error::--update-baseline captured LIVE-doc errors. Annotate or fix them, then re-run."
		exit 1
	}
	exit 0
fi

# Dirty-tree guard (mirrors check-duplication): the gate must run against a
# COMMITTED baseline. Re-pinning while the baseline is uncommitted validates
# against in-flight state and invites pinning foreign half-done code.
if ! git diff --quiet -- "$BASELINE" 2>/dev/null ||
	! git diff --cached --quiet -- "$BASELINE" 2>/dev/null; then
	echo "ERROR: $BASELINE has uncommitted changes."
	echo "Commit the baseline first, or restore it: git restore $BASELINE"
	exit 1
fi

check_policy

tmpbase=$(mktemp)
trap 'rm -f "$tmpbase"' EXIT
grep -v '^#' "$BASELINE" | sed "s|^|$PWD/|" >"$tmpbase"

echo "==> md-go-validator gate ($(baseline_entries) baselined archived error(s))"
if ! md-go-validator . --baseline "$tmpbase" -q; then
	echo "::error::New md-go-validator errors (not in $BASELINE)."
	echo "Fix the fence, or annotate intentional pseudo-code with '// skip-validate'"
	echo "(inside the go fence). Genuinely new ARCHIVED history may be baselined:"
	echo "  bash scripts/check-md-go.sh --update-baseline"
	exit 1
fi
echo "✅ md-go-validator: no new errors ($(baseline_entries) baselined archived error(s))."
