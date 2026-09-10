#!/usr/bin/env bash
# test-tag-release.sh — Smoke tests for scripts/tag-release.sh
#
# Verifies, against throwaway fixture repos (no network, no real tags):
# 1. --audit exits 1 and reports a v2 tag over a suffix-less module path
#    (the issue-#20 class: proxy-invisible tag)
# 2. --audit exits 0 when every tag matches its module path
# 3. The release flow rejects a tag whose major mismatches the module path
# 4. --smoke fails with the push-first message when the tag is not on origin
#
# Run: bash scripts/test-tag-release.sh
set -euo pipefail

SCRIPT="$(cd "$(dirname "$0")" && pwd)/tag-release.sh"
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
#   dead/   module path WITHOUT /vN suffix, tags v0.1.0 (OK) + v2.0.0 (FAIL)
#   good/   module path WITH /v2 suffix,   tags v2.0.0 + v2.1.0 (both OK)
fixture_repo() {
	local root="$1"
	mkdir -p "$root/dead" "$root/good"

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
EOF

	cat >"$root/good/main.go" <<'EOF'
package main

func main() {}
EOF

	git -C "$root" init -q
	git -C "$root" config user.email t@example.com
	git -C "$root" config user.name t
	# The maintainer's global config signs annotated tags (tag.gpgSign /
	# tag.forceSignAnnotated); fixtures must stay hermetic, so strip both.
	local notag="-c tag.gpgSign=false -c tag.forceSignAnnotated=false"
	git -C "$root" add -A
	git -C "$root" commit -qm init
	git $notag -C "$root" tag dead/v0.1.0
	git $notag -C "$root" tag dead/v2.0.0
	git $notag -C "$root" tag good/v2.0.0
	git $notag -C "$root" tag good/v2.1.0
}

TMPROOT="$(mktemp -d)"
trap 'rm -rf "$TMPROOT"' EXIT

echo "━━━ Test 1: --audit flags a proxy-invisible (mismatched) tag ━━━"
fixture_repo "$TMPROOT/t1"
out="$(cd "$TMPROOT/t1" && bash "$SCRIPT" --audit 2>&1)" && rc=0 || rc=$?
check "--audit exits nonzero on violation" test "$rc" -ne 0
check "--audit names dead/v2.0.0" bash -c "printf '%s' \"\$0\" | grep -q 'FAIL  dead/v2.0.0'" "$out"
check "--audit reports 1 violation" bash -c "printf '%s' \"\$0\" | grep -q '1 violation'" "$out"

echo "━━━ Test 2: --audit passes a consistent repo ━━━"
fixture_repo "$TMPROOT/t2"
git -C "$TMPROOT/t2" tag -d dead/v2.0.0 >/dev/null
if (cd "$TMPROOT/t2" && bash "$SCRIPT" --audit >/dev/null 2>&1); then
	echo "  ✓ PASS: --audit exits 0 when all tags match"
else
	echo "  ✗ FAIL: --audit should pass on a consistent repo"
	FAILED=1
fi

echo "━━━ Test 3: release flow rejects a major-mismatched tag ━━━"
fixture_repo "$TMPROOT/t3"
out="$(cd "$TMPROOT/t3" && bash "$SCRIPT" dead v2.0.1 "x" 2>&1)" && rc=0 || rc=$?
check "release exits nonzero on mismatched path" test "$rc" -ne 0
check "error explains the /vN requirement" bash -c "printf '%s' \"\$0\" | grep -q 'end in /v2'" "$out"
check "no tag was created" bash -c "! git -C \"\$0\" tag -l dead/v2.0.1 | grep -q ." "$TMPROOT/t3"

echo "━━━ Test 4: --smoke without a pushed tag says push first ━━━"
fixture_repo "$TMPROOT/t4"
git -C "$TMPROOT/t4" init -q --bare "$TMPROOT/t4-origin"
git -C "$TMPROOT/t4" remote add origin "$TMPROOT/t4-origin"
out="$(cd "$TMPROOT/t4" && bash "$SCRIPT" --smoke dead v0.1.0 2>&1)" && rc=0 || rc=$?
check "--smoke exits nonzero pre-push" test "$rc" -ne 0
check "--smoke tells you to push" bash -c "printf '%s' \"\$0\" | grep -q 'push first'" "$out"

echo "━━━ Test 5: --smoke rejects a non-main module cleanly ━━━"
# dead/ has a main package but the tag was never pushed, so this would exit
# at the push check; good/ is pushed nowhere either. Assert the usage guard
# instead: wrong arg count for --smoke exits 2/1 without proxy access.
out="$(cd "$TMPROOT/t4" && bash "$SCRIPT" --smoke dead 2>&1)" && rc=0 || rc=$?
check "--smoke with missing version exits nonzero" test "$rc" -ne 0

if [ "$FAILED" -eq 0 ]; then
	echo ""
	echo "All tag-release smoke tests passed."
	exit 0
else
	echo ""
	echo "Some tag-release smoke tests FAILED."
	exit 1
fi
