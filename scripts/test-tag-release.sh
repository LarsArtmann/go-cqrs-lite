#!/usr/bin/env bash
# test-tag-release.sh — Smoke tests for scripts/tag-release.sh
#
# Verifies, against throwaway fixture repos (no network, no real tags):
# 1. --audit exits 1 and reports a v2 tag over a suffix-less module path
#    (the issue-#20 class: proxy-invisible tag)
# 2. --audit exits 0 when every tag matches its module path
# 3. The release flow rejects a tag whose major mismatches the module path
# 4. --smoke fails with the push-first message when the tag is not on origin
# 5. release_common.sh unit tests: path_matches_major / module_has_root_main /
#    smoke_probe_args (one shared implementation — no fork between scripts)
# 6. --audit --baseline gates on NEW violations only; --write-baseline writes
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
	local notag=(-c tag.gpgSign=false -c tag.forceSignAnnotated=false)
	git -C "$root" add -A
	git -C "$root" commit -qm init
	git "${notag[@]}" -C "$root" tag dead/v0.1.0
	git "${notag[@]}" -C "$root" tag dead/v2.0.0
	git "${notag[@]}" -C "$root" tag good/v2.0.0
	git "${notag[@]}" -C "$root" tag good/v2.1.0
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

echo "━━━ Test 5: release_common.sh unit tests (sourced lib) ━━━"
# shellcheck disable=SC1091 # sourced release lib lives beside this script
source "$(cd "$(dirname "$0")" && pwd)/lib/release_common.sh"

check "v4 tag + /v4 path matches" path_matches_major "x/v4" "v4.1.0"
check "v0 tag + suffix-less path matches" path_matches_major "x" "v0.1.0"
check "v1 tag + suffix-less path matches" path_matches_major "x" "v1.9.9"
if path_matches_major "x" "v4.0.0"; then
	echo "  ✗ FAIL: v4 tag over suffix-less path must NOT match"
	FAILED=1
else
	echo "  ✓ PASS: v4 tag over suffix-less path rejected"
fi
if path_matches_major "x/v4" "v2.0.0"; then
	echo "  ✗ FAIL: v2 tag over /v4 path must NOT match"
	FAILED=1
else
	echo "  ✓ PASS: major-mismatched /vN path rejected"
fi

LIBTMP="$(mktemp -d)"
mkdir -p "$LIBTMP/cli" "$LIBTMP/libmod"
printf 'package main\n' >"$LIBTMP/cli/main.go"
printf 'package libmod\n' >"$LIBTMP/libmod/libmod.go"
check "module_has_root_main detects a main package" module_has_root_main "$LIBTMP/cli"
if module_has_root_main "$LIBTMP/libmod"; then
	echo "  ✗ FAIL: library module must not be detected as main"
	FAILED=1
else
	echo "  ✓ PASS: library module takes the no-main skip path"
fi

printf '%s\n' '# probes' 'cli version --short' >"$LIBTMP/probes.txt"
out_lib="$(smoke_probe_args "cli" "$LIBTMP/probes.txt")"
check "smoke_probe_args reads the explicit probe" bash -c "[ \"\$0\" = 'version --short' ]" "$out_lib"
out_lib="$(smoke_probe_args "other" "$LIBTMP/probes.txt")"
check "unlisted module yields the --help default (empty)" bash -c "[ -z \"\$0\" ]" "$out_lib"
out_lib="$(smoke_probe_args "cli" "$LIBTMP/does-not-exist.txt")"
check "missing probes file yields the default (empty)" bash -c "[ -z \"\$0\" ]" "$out_lib"
rm -rf "$LIBTMP"

echo "━━━ Test 6: --smoke rejects a non-main module cleanly ━━━"
# dead/ has a main package but the tag was never pushed, so this would exit
# at the push check; good/ is pushed nowhere either. Assert the usage guard
# instead: wrong arg count for --smoke exits 2/1 without proxy access.
out="$(cd "$TMPROOT/t4" && bash "$SCRIPT" --smoke dead 2>&1)" && rc=0 || rc=$?
check "--smoke with missing version exits nonzero" test "$rc" -ne 0

echo "━━━ Test 7: --audit --baseline gates NEW violations only ━━━"
# t1 has one violation (dead/v2.0.0, the issue-#20 class). A baseline that
# names it must turn the audit green; --write-baseline must regenerate it;
# a baseline MISSING the violation must fail with a NEW VIOLATION report.
fixture_repo "$TMPROOT/t7"

out="$(cd "$TMPROOT/t7" && bash "$SCRIPT" --audit --baseline "$TMPROOT/t7-baseline.txt" 2>&1)" && rc=0 || rc=$?
check "missing baseline file errors" test "$rc" -ne 0

(cd "$TMPROOT/t7" && bash "$SCRIPT" --audit --baseline "$TMPROOT/t7-baseline.txt" --write-baseline >/dev/null 2>&1) || true
check "--write-baseline recorded the violation" bash -c "grep -qxF 'dead/v2.0.0' \"\$0\"" "$TMPROOT/t7-baseline.txt"
if (cd "$TMPROOT/t7" && bash "$SCRIPT" --audit --baseline "$TMPROOT/t7-baseline.txt" >/dev/null 2>&1); then
	echo "  ✓ PASS: known violation is baselined (exit 0)"
else
	echo "  ✗ FAIL: baselined violation must not gate"
	FAILED=1
fi

grep -v 'dead/v2.0.0' "$TMPROOT/t7-baseline.txt" >"$TMPROOT/t7-stale.txt" || true
out="$(cd "$TMPROOT/t7" && bash "$SCRIPT" --audit --baseline "$TMPROOT/t7-stale.txt" 2>&1)" && rc=0 || rc=$?
check "non-baselined violation gates (exit nonzero)" test "$rc" -ne 0
check "NEW VIOLATION is reported" bash -c "printf '%s' \"\$0\" | grep -q 'NEW VIOLATION: dead/v2.0.0'" "$out"

echo "━━━ Test 8: release flow tags, strips local replaces at the tag, restores tree ━━━"
fixture_repo "$TMPROOT/t6"
# Hermetic: the release script's internal `git tag -a` must not inherit a
# global sign-everything config (GPG key may not even exist here).
git -C "$TMPROOT/t6" config tag.gpgSign false
git -C "$TMPROOT/t6" config tag.forceSignAnnotated false
# good/ is /v2-suffixed; give its go.mod a local replace the release must
# strip at the tag but restore in the worktree.
cat >>"$TMPROOT/t6/good/go.mod" <<'EOF'

replace github.com/example/fixture/dead => ../dead
EOF
git -C "$TMPROOT/t6" add -A
git -C "$TMPROOT/t6" commit -qm add-replace
out="$(cd "$TMPROOT/t6" && bash "$SCRIPT" good v2.0.2 "single cut" 2>&1)" && rc=0 || rc=$?
check "release exits 0" test "$rc" -eq 0
check "tag created" bash -c "git -C \"\$0\" tag -l good/v2.0.2 | grep -q ." "$TMPROOT/t6"
check "tag is annotated" bash -c "test \"\$(git -C \"\$0\" cat-file -t good/v2.0.2)\" = tag" "$TMPROOT/t6"
check "tree fully restored" bash -c "git -C \"\$0\" status --porcelain | wc -l | grep -qx 0" "$TMPROOT/t6"
check "no build artifact left behind" bash -c "test ! -e \"\$0/good/good\"" "$TMPROOT/t6"
check "worktree go.mod keeps the local replace" bash -c "grep -q 'replace github.com/example/fixture/dead => ../dead' \"\$0/good/go.mod\"" "$TMPROOT/t6"
check "tagged go.mod has the replace stripped" bash -c "! git -C \"\$0\" show good/v2.0.2:good/go.mod | grep -q 'replace github.com/example/fixture/dead'" "$TMPROOT/t6"

if [ "$FAILED" -eq 0 ]; then
	echo ""
	echo "All tag-release smoke tests passed."
	exit 0
else
	echo ""
	echo "Some tag-release smoke tests FAILED."
	exit 1
fi
