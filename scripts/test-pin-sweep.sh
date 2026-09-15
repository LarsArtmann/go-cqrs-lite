#!/usr/bin/env bash
# test-pin-sweep.sh — Smoke test for scripts/pin-sweep.sh against a fixture
# git repo (fake module dirs + fake tags; no Go toolchain needed because
# --check/--dry-run never invoke go).
#
# Verifies:
# 1. --check exits 1 and names a stale pin
# 2. --check exits 0 when every pin is at its latest local tag
# 3. --dry-run reports the planned bump and mutates nothing
# 4. --check --remote catches a tag that only exists on origin — the
#    local-refs blind spot where plain --check stays green
#
# Run: bash scripts/test-pin-sweep.sh
set -euo pipefail

SCRIPT="$(cd "$(dirname "$0")" && pwd)/pin-sweep.sh"
FAILED=0

GIT="git -c user.email=test@example.com -c user.name=test -c init.defaultBranch=main"
export GIT_AUTHOR_NAME=test GIT_AUTHOR_EMAIL=test@example.com
export GIT_COMMITTER_NAME=test GIT_COMMITTER_EMAIL=test@example.com

FIXTURE=$(mktemp -d)
trap 'rm -rf "$FIXTURE"' EXIT

# fixture_repo <consumer-pin> — one go.mod pinning alpha at <consumer-pin>.
fixture_repo() {
	local pin="$1"
	rm -rf "$FIXTURE/repo"
	mkdir -p "$FIXTURE/repo"
	cd "$FIXTURE/repo"
	$GIT init -q
	mkdir -p alpha
	cat >alpha/go.mod <<EOF
module github.com/larsartmann/go-cqrs-lite/alpha/v4

go 1.26
EOF
	cat >go.mod <<EOF
module example.com/consumer

go 1.26

require (
	github.com/larsartmann/go-cqrs-lite/alpha/v4 $pin
)
EOF
	$GIT add -A
	$GIT commit -q -m init
	$GIT tag -a -m v4.2.0 alpha/v4.2.0
}

echo "━━━ Test 1: --check detects a stale pin ━━━"
fixture_repo "v4.1.0"
if bash "$SCRIPT" --check >"$FIXTURE/out.log" 2>&1; then
	echo "  ✗ FAIL: --check should exit 1 on a stale pin"
	FAILED=1
elif ! grep -q "stale pin" "$FIXTURE/out.log"; then
	echo "  ✗ FAIL: --check output does not name the stale pin"
	FAILED=1
else
	echo "  ✓ PASS: stale pin reported, exit 1"
fi

echo "━━━ Test 2: --check green at latest local tag ━━━"
fixture_repo "v4.2.0"
if bash "$SCRIPT" --check >/dev/null 2>&1; then
	echo "  ✓ PASS: --check exits 0 when pins are current"
else
	echo "  ✗ FAIL: --check should pass on a current tree"
	FAILED=1
fi

echo "━━━ Test 3: --dry-run reports without mutating ━━━"
fixture_repo "v4.1.0"
if ! bash "$SCRIPT" --dry-run >"$FIXTURE/out.log" 2>&1; then
	echo "  ✗ FAIL: --dry-run should exit 0"
	FAILED=1
elif ! grep -q "would bump" "$FIXTURE/out.log"; then
	echo "  ✗ FAIL: --dry-run output does not preview the bump"
	FAILED=1
elif [ -n "$(git status --porcelain)" ]; then
	echo "  ✗ FAIL: --dry-run mutated the tree"
	FAILED=1
else
	echo "  ✓ PASS: preview printed, tree untouched"
fi

echo "━━━ Test 4: --remote catches tags the local clone lacks ━━━"
fixture_repo "v4.2.0"
git clone -q --bare "$FIXTURE/repo" "$FIXTURE/origin.git"
git remote add origin "$FIXTURE/origin.git"
git --git-dir="$FIXTURE/origin.git" tag -a -m v4.3.0 alpha/v4.3.0

if ! bash "$SCRIPT" --check >/dev/null 2>&1; then
	echo "  ✗ FAIL: precondition — plain --check should be green (local refs only)"
	FAILED=1
elif bash "$SCRIPT" --check --remote >/dev/null 2>&1; then
	echo "  ✗ FAIL: --check --remote should flag the origin-only tag"
	FAILED=1
else
	echo "  ✓ PASS: remote refs expose the blind spot (exit 1)"
fi

if [ "$FAILED" -ne 0 ]; then
	echo "❌ test-pin-sweep: FAILURES above"
	exit 1
fi

echo "✅ test-pin-sweep: all checks passed"
