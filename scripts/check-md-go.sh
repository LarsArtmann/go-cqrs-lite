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
#   bash scripts/check-md-go.sh --self-test        # planted-fixture suite
set -euo pipefail

# ── Self-test (repo convention: CI-gating scripts ship with self-tests) ────
# Hermetic: a throwaway git repo + a PATH-stubbed md-go-validator whose
# exit code a flag file controls — never the live tree, never the real
# binary. Pins the four gate behaviors that were session-only memories:
# green pass, new-error refusal, live-path baseline refusal (the
# ARCHIVE_SEGMENT mutation leg), and the uncommitted-baseline refusal.
if [ "${1:-}" = "--self-test" ]; then
	SELF="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/$(basename "${BASH_SOURCE[0]}")"
	TMP="$(mktemp -d)"
	trap 'rm -rf "$TMP"' EXIT

	# Stub binary: exits 1 iff the fail-flag file exists (simulates "new
	# validation errors found"); answers --version like the packaged binary.
	mkdir -p "$TMP/bin"
	cat >"$TMP/bin/md-go-validator" <<'STUB'
#!/bin/sh
if [ "${1:-}" = "--version" ] || [ "${1:-}" = "-V" ]; then
	echo "md-go-validator stub-4dd9437"
	exit 0
fi
if [ -f "$(dirname "$0")/../stub-fail" ]; then
	echo "stub: planted validation error" >&2
	exit 1
fi
exit 0
STUB
	chmod +x "$TMP/bin/md-go-validator"

	# Fixture repo: live doc with a clean fence + archived doc whose errors
	# are baselined. Mirrors the layout the hardcoded relative paths expect.
	fixture() {
		rm -rf "$TMP/repo"
		mkdir -p "$TMP/repo/scripts" "$TMP/repo/docs" "$TMP/repo/docs/archived"
		git -C "$TMP/repo" init -q
		git -C "$TMP/repo" config user.email t@t && git -C "$TMP/repo" config user.name t
		printf 'walk: ["docs"]\n' >"$TMP/repo/.md-go-validator.yaml"
		printf '# t\n\n```go\nfmt.Println("ok")\n```\n' >"$TMP/repo/docs/live.md"
		printf '# old\n\n```go\nfunc broken(\n```\n' >"$TMP/repo/docs/archived/old.md"
		printf '# baseline\n%s/docs/archived/old.md:3:PARSE_ERROR\n' "$TMP/repo" \
			>"$TMP/repo/scripts/md-go-baseline.txt"
		git -C "$TMP/repo" add -A && git -C "$TMP/repo" commit -qm fixture
		rm -f "$TMP/stub-fail"
	}

	fails=0
	check() {
		if [ "$3" = "$2" ]; then
			echo "  ✓ PASS: $1"
		else
			echo "  ✗ FAIL: $1 (want rc=$2, got rc=$3)"
			fails=$((fails + 1))
		fi
	}

	fixture
	rc=0
	(cd "$TMP/repo" && PATH="$TMP/bin:$PATH" bash "$SELF") >/dev/null 2>&1 || rc=$?
	check "green: baselined archived errors pass" 0 "$rc"

	touch "$TMP/stub-fail"
	rc=0
	out=$(cd "$TMP/repo" && PATH="$TMP/bin:$PATH" bash "$SELF" 2>&1) || rc=$?
	check "new validation error fails loud" 1 "$rc"
	grep -q "New md-go-validator errors" <<<"$out" || {
		echo "  ✗ FAIL: new-error message missing"
		fails=$((fails + 1))
	}
	rm -f "$TMP/stub-fail"

	# Policy leg (the ARCHIVE_SEGMENT mutation test): a baseline row pointing
	# at a LIVE path must be refused — the baseline may never become a dumping
	# ground for live docs. The row is COMMITTED so the dirty-tree guard does
	# not mask the policy check being tested.
	fixture
	printf '%s/docs/live.md:3:PARSE_ERROR\n' "$TMP/repo" >>"$TMP/repo/scripts/md-go-baseline.txt"
	git -C "$TMP/repo" add -A && git -C "$TMP/repo" commit -qm planted-live-row
	rc=0
	out=$(cd "$TMP/repo" && PATH="$TMP/bin:$PATH" bash "$SELF" 2>&1) || rc=$?
	check "baseline row on live path fails (ARCHIVE_SEGMENT)" 1 "$rc"
	grep -q "non-archived paths" <<<"$out" || {
		echo "  ✗ FAIL: policy message missing"
		fails=$((fails + 1))
	}

	# Dirty-tree leg: an uncommitted baseline must refuse to gate.
	fixture
	printf '%s/docs/archived/other.md:3:PARSE_ERROR\n' "$TMP/repo" >>"$TMP/repo/scripts/md-go-baseline.txt"
	rc=0
	out=$(cd "$TMP/repo" && PATH="$TMP/bin:$PATH" bash "$SELF" 2>&1) || rc=$?
	check "uncommitted baseline refuses to gate" 1 "$rc"
	grep -q "uncommitted changes" <<<"$out" || {
		echo "  ✗ FAIL: dirty-tree message missing"
		fails=$((fails + 1))
	}

	if [ "$fails" -eq 0 ]; then
		echo "check-md-go self-test passed."
		exit 0
	fi
	echo "check-md-go self-test FAILED."
	exit 1
fi

cd "$(git rev-parse --show-toplevel)"

BASELINE="scripts/md-go-baseline.txt"
CONFIG=".md-go-validator.yaml"
ARCHIVE_SEGMENT='(^|/)archive(d)?/'

baseline_entries() {
	grep -cv '^#' "$BASELINE"
}

if ! command -v md-go-validator >/dev/null 2>&1; then
	echo "SKIP: md-go-validator not on PATH (use: nix run .#check-md-go — packaged binary)"
	echo "NOTE: a host-installed binary that lags the flake pin can silently change"
	echo "gate semantics — compare 'md-go-validator --version' against the pinned rev"
	echo "in flake.lock (node md-go-validator)."
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
# The baseline may only SHRINK: entries whose file no longer exists on disk
# are ghosts (the doc was trashed/renamed) and fail the gate — prune them
# with --update-baseline (baseline-bump ritual, 2026-09-13 audit §b).
check_policy() {
	local bad
	bad=$(grep -v '^#' "$BASELINE" | cut -d: -f1 | sort -u | grep -Ev "$ARCHIVE_SEGMENT" || true)
	if [ -n "$bad" ]; then
		echo "::error::$BASELINE references non-archived paths — live docs must carry"
		echo "::error::'// skip-validate' annotations (or real fixes), not baseline entries:"
		echo "$bad" | sed 's/^/  - /'
		return 1
	fi

	local ghosts
	ghosts=$(grep -v '^#' "$BASELINE" | cut -d: -f1 | sort -u | while IFS= read -r p; do
		[ -e "$p" ] || echo "$p"
	done)
	if [ -n "$ghosts" ]; then
		echo "::error::$BASELINE references files that no longer exist (ghost entries; the"
		echo "::error::baseline may only shrink) — regenerate: bash scripts/check-md-go.sh --update-baseline"
		echo "$ghosts" | sed 's/^/  - /'
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
echo "    binary: $(md-go-validator --version 2>/dev/null || echo 'version unknown')"
if ! md-go-validator . --baseline "$tmpbase" -q; then
	echo "::error::New md-go-validator errors (not in $BASELINE)."
	echo "Fix the fence, or annotate intentional pseudo-code with '// skip-validate'"
	echo "(inside the go fence). Genuinely new ARCHIVED history may be baselined:"
	echo "  bash scripts/check-md-go.sh --update-baseline"
	exit 1
fi
echo "✅ md-go-validator: no new errors ($(baseline_entries) baselined archived error(s))."
