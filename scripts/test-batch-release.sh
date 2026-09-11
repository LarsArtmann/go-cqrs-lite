#!/usr/bin/env bash
# test-batch-release.sh — Smoke tests for scripts/batch-release.sh
#
# Verifies, against throwaway fixture repos (no network, no real tags):
# 1. The batch release flow rejects a tag whose major mismatches the module
#    path (the issue-#20 class: proxy-invisible tag) and creates NO tags
# 2. --audit delegates to tag-release.sh and flags a proxy-invisible tag
# 3. A successful batch release creates annotated tags, strips local replaces
#    AT the tag only, and restores the working tree exactly
# 4. A module that fails the standalone build check aborts with no tags
# 5. An existing tag is rejected before anything is touched
#
# Run: bash scripts/test-batch-release.sh
set -euo pipefail

SCRIPT="$(cd "$(dirname "$0")" && pwd)/batch-release.sh"
FAILED=0

check() {
	local name="$1"
	shift
	if "$@"; then
		echo "  ✓ PASS: $name"
	else
		echo "  ✗ FAIL: $name"
		FAILED=1
	fi
}

# fixture_repo <path> writes a tiny multi-module git repo:
#   dead/   module path WITHOUT /vN suffix, tag v0.1.0 (valid) + v2.0.0 (the
#           audit's known violation)
#   good/   module path WITH /v2 suffix, go.mod carrying a LOCAL replace that
#           a batch release must strip at the tag but restore in the tree
#   broken/ module path WITH /v2 suffix whose code does not compile — the
#           standalone-build gate must abort the release
#   libx/   module path WITH /v2 suffix, NO main package — the -o build must
#           fall back to the plain build instead of refusing "no main
#           packages to build"
fixture_repo() {
	local root="$1"
	mkdir -p "$root/dead" "$root/good" "$root/broken" "$root/libx"

	cat >"$root/go.mod" <<'EOF'
module github.com/example/fixture

go 1.26.0
EOF

	cat >"$root/dead/go.mod" <<'EOF'
module github.com/example/fixture/dead

go 1.26.0
EOF

	cat >"$root/dead/main.go" <<'EOF'
package main

func main() {}
EOF

	cat >"$root/good/go.mod" <<'EOF'
module github.com/example/fixture/good/v2

go 1.26.0

replace github.com/example/fixture/dead => ../dead
EOF

	cat >"$root/good/main.go" <<'EOF'
package main

func main() {}
EOF

	cat >"$root/broken/go.mod" <<'EOF'
module github.com/example/fixture/broken/v2

go 1.26.0
EOF

	# Syntax error: the standalone-build gate must catch this offline.
	cat >"$root/broken/main.go" <<'EOF'
package main

func main( {}
EOF

	cat >"$root/libx/go.mod" <<'EOF'
module github.com/example/fixture/libx/v2

go 1.26.0
EOF

	cat >"$root/libx/libx.go" <<'EOF'
package libx

func Hello() string { return "hi" }
EOF

	git -C "$root" init -q
	# Hermetic fixture: the maintainer's global config may sign annotated
	# tags (tag.gpgSign / tag.forceSignAnnotated); disable both for the
	# repo the batch script will tag in.
	git -C "$root" config tag.gpgSign false
	git -C "$root" config tag.forceSignAnnotated false
	git -C "$root" config user.email t@example.com
	git -C "$root" config user.name t
	git -C "$root" add -A
	git -C "$root" commit -qm init
	git -C "$root" tag dead/v0.1.0
	git -C "$root" tag dead/v2.0.0
	git -C "$root" tag good/v2.0.0
}

TMPROOT="$(mktemp -d)"
trap 'rm -rf "$TMPROOT"' EXIT

echo "━━━ Test 1: batch release rejects a major-mismatched tag ━━━"
fixture_repo "$TMPROOT/t1"
out="$(cd "$TMPROOT/t1" && bash "$SCRIPT" "dead v2.0.1 x" 2>&1)" && rc=0 || rc=$?
check "release exits nonzero on mismatched path" test "$rc" -ne 0
check "error explains the /vN requirement" bash -c "printf '%s' \"\$0\" | grep -q 'end in /v2'" "$out"
check "no tag was created" bash -c "! git -C \"\$0\" tag -l dead/v2.0.1 | grep -q ." "$TMPROOT/t1"
check "tree untouched" bash -c "git -C \"\$0\" status --porcelain | wc -l | grep -qx 0" "$TMPROOT/t1"

echo "━━━ Test 1b: malformed (unquoted) triple fails with a clear error ━━━"
fixture_repo "$TMPROOT/t1b"
out="$(cd "$TMPROOT/t1b" && bash "$SCRIPT" dead v2.0.1 "x" 2>&1)" && rc=0 || rc=$?
check "unquoted triple exits nonzero" test "$rc" -ne 0
check "error names the quoted-triple form" bash -c "printf '%s' \"\$0\" | grep -q 'malformed triple'" "$out"

echo "━━━ Test 2: --audit delegates and flags the proxy-invisible tag ━━━"
fixture_repo "$TMPROOT/t2"
out="$(cd "$TMPROOT/t2" && bash "$SCRIPT" --audit 2>&1)" && rc=0 || rc=$?
check "--audit exits nonzero on violation" test "$rc" -ne 0
check "--audit names dead/v2.0.0" bash -c "printf '%s' \"\$0\" | grep -q 'FAIL  dead/v2.0.0'" "$out"

echo "━━━ Test 3: successful batch release + exact tree restore ━━━"
fixture_repo "$TMPROOT/t3"
out="$(cd "$TMPROOT/t3" && bash "$SCRIPT" "good v2.0.2 Batch cut test" "libx v2.0.1 Library cut" 2>&1)" && rc=0 || rc=$?
check "release exits 0" test "$rc" -eq 0
check "tag created (main module)" bash -c "git -C \"\$0\" tag -l good/v2.0.2 | grep -q ." "$TMPROOT/t3"
check "tag created (library module)" bash -c "git -C \"\$0\" tag -l libx/v2.0.1 | grep -q ." "$TMPROOT/t3"
check "tag is annotated" bash -c "test \"\$(git -C \"\$0\" cat-file -t good/v2.0.2)\" = tag" "$TMPROOT/t3"
check "tree fully restored" bash -c "git -C \"\$0\" status --porcelain | wc -l | grep -qx 0" "$TMPROOT/t3"
check "no build artifacts left behind" bash -c "test ! -e \"\$0/good/good\" && test ! -e \"\$0/libx/libx\"" "$TMPROOT/t3"
check "worktree go.mod keeps the local replace" bash -c "grep -q 'replace github.com/example/fixture/dead => ../dead' \"\$0/good/go.mod\"" "$TMPROOT/t3"
check "tagged go.mod has the replace stripped" bash -c "! git -C \"\$0\" show good/v2.0.2:good/go.mod | grep -q 'replace github.com/example/fixture/dead'" "$TMPROOT/t3"
check "smoke hint printed" bash -c "printf '%s' \"\$0\" | grep -q -- '--smoke good v2.0.2'" "$out"

echo "━━━ Test 4: standalone-build failure aborts with no tags ━━━"
fixture_repo "$TMPROOT/t4"
out="$(cd "$TMPROOT/t4" && bash "$SCRIPT" "broken v2.0.1 Broken code" 2>&1)" && rc=0 || rc=$?
check "release exits nonzero on build failure" test "$rc" -ne 0
check "error names the compile gate" bash -c "printf '%s' \"\$0\" | grep -q 'does not compile against its published requires'" "$out"
check "no tag was created" bash -c "! git -C \"\$0\" tag -l broken/v2.0.1 | grep -q ." "$TMPROOT/t4"
check "tree fully restored" bash -c "git -C \"\$0\" status --porcelain | wc -l | grep -qx 0" "$TMPROOT/t4"

echo "━━━ Test 5: existing tag is rejected up front ━━━"
fixture_repo "$TMPROOT/t5"
out="$(cd "$TMPROOT/t5" && bash "$SCRIPT" "good v2.0.0 Duplicate" 2>&1)" && rc=0 || rc=$?
check "release exits nonzero on duplicate tag" test "$rc" -ne 0
check "error says the tag exists" bash -c "printf '%s' \"\$0\" | grep -q 'already exists'" "$out"

if [ "$FAILED" -eq 0 ]; then
	echo ""
	echo "All batch-release smoke tests passed."
	exit 0
else
	echo ""
	echo "Some batch-release smoke tests FAILED."
	exit 1
fi
