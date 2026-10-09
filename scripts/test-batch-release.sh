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
# 6. The dated run log (M15/e1) records guards, phases, failure tails, and a
#    one-line-per-module summary — while --dry-run writes no log at all
# 7. An untag-policy-listed train (ADR-0152) is refused before anything is
#    touched, in BOTH real and --dry-run modes
# 8. --lockstep (ADR-0152 v5 waves): one version fans to every manifest
#    train; path-vs-tag, untag policy, and no-positional-triples guards all
#    hold; --dry-run tags nothing
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
out="$(cd "$TMPROOT/t1" && BATCH_RELEASE_LOG_DIR="$TMPROOT/t1-logs" bash "$SCRIPT" "dead v2.0.1 x" 2>&1)" && rc=0 || rc=$?
check "release exits nonzero on mismatched path" test "$rc" -ne 0
check "error explains the /vN requirement" bash -c "printf '%s' \"\$0\" | grep -q 'end in /v2'" "$out"
check "no tag was created" bash -c "! git -C \"\$0\" tag -l dead/v2.0.1 | grep -q ." "$TMPROOT/t1"
check "tree untouched" bash -c "git -C \"\$0\" status --porcelain | wc -l | grep -qx 0" "$TMPROOT/t1"
check "run log records the guard rejection" bash -c "grep -q 'guard FAIL dead' \"\$0\"/batch-*.log" "$TMPROOT/t1-logs"

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
out="$(cd "$TMPROOT/t3" && BATCH_RELEASE_LOG_DIR="$TMPROOT/t3-logs" bash "$SCRIPT" "good v2.0.2 Batch cut test" "libx v2.0.1 Library cut" 2>&1)" && rc=0 || rc=$?
check "release exits 0" test "$rc" -eq 0
check "tag created (main module)" bash -c "git -C \"\$0\" tag -l good/v2.0.2 | grep -q ." "$TMPROOT/t3"
check "tag created (library module)" bash -c "git -C \"\$0\" tag -l libx/v2.0.1 | grep -q ." "$TMPROOT/t3"
check "tag is annotated" bash -c "test \"\$(git -C \"\$0\" cat-file -t good/v2.0.2)\" = tag" "$TMPROOT/t3"
check "tree fully restored" bash -c "git -C \"\$0\" status --porcelain | wc -l | grep -qx 0" "$TMPROOT/t3"
check "no build artifacts left behind" bash -c "test ! -e \"\$0/good/good\" && test ! -e \"\$0/libx/libx\"" "$TMPROOT/t3"
check "worktree go.mod keeps the local replace" bash -c "grep -q 'replace github.com/example/fixture/dead => ../dead' \"\$0/good/go.mod\"" "$TMPROOT/t3"
check "tagged go.mod has the replace stripped" bash -c "! git -C \"\$0\" show good/v2.0.2:good/go.mod | grep -q 'replace github.com/example/fixture/dead'" "$TMPROOT/t3"
check "smoke hint printed" bash -c "printf '%s' \"\$0\" | grep -q -- '--smoke good v2.0.2'" "$out"
check "run log summary (main module)" bash -c "grep -q '^SUMMARY good v2.0.2 tagged=yes verify=ok smoke=pending' \"\$0\"/batch-*.log" "$TMPROOT/t3-logs"
check "run log summary (library module)" bash -c "grep -q '^SUMMARY libx v2.0.1 tagged=yes verify=ok smoke=pending' \"\$0\"/batch-*.log" "$TMPROOT/t3-logs"
check "run log records phases + clean exit" bash -c "grep -q 'strip done good' \"\$0\"/batch-*.log && grep -q 'verify ok good' \"\$0\"/batch-*.log && grep -q 'tagged good/v2.0.2' \"\$0\"/batch-*.log && grep -q 'run ended rc=0' \"\$0\"/batch-*.log" "$TMPROOT/t3-logs"
check "log dir override keeps repo clean" bash -c "test ! -e \"\$0/build\"" "$TMPROOT/t3"

echo "━━━ Test 4: standalone-build failure aborts with no tags ━━━"
fixture_repo "$TMPROOT/t4"
out="$(cd "$TMPROOT/t4" && BATCH_RELEASE_LOG_DIR="$TMPROOT/t4-logs" bash "$SCRIPT" "broken v2.0.1 Broken code" 2>&1)" && rc=0 || rc=$?
check "release exits nonzero on build failure" test "$rc" -ne 0
check "error names the compile gate" bash -c "printf '%s' \"\$0\" | grep -q 'does not compile against its published requires'" "$out"
check "no tag was created" bash -c "! git -C \"\$0\" tag -l broken/v2.0.1 | grep -q ." "$TMPROOT/t4"
check "tree fully restored" bash -c "git -C \"\$0\" status --porcelain | wc -l | grep -qx 0" "$TMPROOT/t4"
check "run log records verify FAIL + failure tail" bash -c "grep -q 'verify FAIL broken' \"\$0\"/batch-*.log && grep -q 'tail: verify broken' \"\$0\"/batch-*.log" "$TMPROOT/t4-logs"
echo "━━━ Test 5: existing tag is rejected up front ━━━"
fixture_repo "$TMPROOT/t5"
out="$(cd "$TMPROOT/t5" && BATCH_RELEASE_LOG_DIR="$TMPROOT/t5-logs" bash "$SCRIPT" "good v2.0.0 Duplicate" 2>&1)" && rc=0 || rc=$?
check "release exits nonzero on duplicate tag" test "$rc" -ne 0
check "error says the tag exists" bash -c "printf '%s' \"\$0\" | grep -q 'already exists'" "$out"
check "run log records the duplicate-tag guard" bash -c "grep -q 'guard FAIL good: tag good/v2.0.0 already exists' \"\$0\"/batch-*.log" "$TMPROOT/t5-logs"

echo "━━━ Test 6: --dry-run writes no run log ━━━"
fixture_repo "$TMPROOT/t6"
out="$(cd "$TMPROOT/t6" && BATCH_RELEASE_LOG_DIR="$TMPROOT/t6-logs" bash "$SCRIPT" --dry-run "good v2.0.2 Dry" 2>&1)" && rc=0 || rc=$?
check "dry-run exits 0" test "$rc" -eq 0
check "dry-run writes no run log" test ! -e "$TMPROOT/t6-logs"
check "dry-run does not tag" bash -c "! git -C \"\$0\" tag -l good/v2.0.2 | grep -q ." "$TMPROOT/t6"

echo "━━━ Test 7: --smoke-all opens a smoke log; malformed lines fail ━━━"
fixture_repo "$TMPROOT/t7"
printf 'good\n' >"$TMPROOT/t7/probes.txt"
out="$(cd "$TMPROOT/t7" && BATCH_RELEASE_LOG_DIR="$TMPROOT/t7-logs" bash "$SCRIPT" --smoke-all probes.txt 2>&1)" && rc=0 || rc=$?
check "smoke-all rejects a malformed probe line" test "$rc" -ne 0
check "smoke run log written" bash -c "ls \"\$0\"/smoke-*.log >/dev/null 2>&1" "$TMPROOT/t7-logs"
check "smoke log names the probes file" bash -c "grep -q 'probes: probes.txt' \"\$0\"/smoke-*.log" "$TMPROOT/t7-logs"

echo "━━━ Test 8: untag-policy guard refuses a listed train (ADR-0152) ━━━"
fixture_repo "$TMPROOT/t8"
printf '# fixture untag list\ngood\n' >"$TMPROOT/t8-untagged.txt"
out="$(cd "$TMPROOT/t8" && BATCH_RELEASE_LOG_DIR="$TMPROOT/t8-logs" UNTAGGED_TRAINS_FILE="$TMPROOT/t8-untagged.txt" bash "$SCRIPT" "good v2.0.2 Frozen train" "libx v2.0.1 Also fine" 2>&1)" && rc=0 || rc=$?
check "release exits nonzero on untagged train" test "$rc" -ne 0
check "error names the policy list" bash -c "printf '%s' \"\$0\" | grep -q 'untag policy list (scripts/untagged-trains.txt)'" "$out"
check "error cites ADR-0152" bash -c "printf '%s' \"\$0\" | grep -q 'ADR-0152'" "$out"
check "no tag was created" bash -c "! git -C \"\$0\" tag -l good/v2.0.2 | grep -q ." "$TMPROOT/t8"
check "tree untouched" bash -c "git -C \"\$0\" status --porcelain | wc -l | grep -qx 0" "$TMPROOT/t8"
check "run log records the untag guard" bash -c "grep -q 'guard FAIL good: on the untag policy list' \"\$0\"/batch-*.log" "$TMPROOT/t8-logs"

out="$(cd "$TMPROOT/t8" && UNTAGGED_TRAINS_FILE="$TMPROOT/t8-untagged.txt" bash "$SCRIPT" --dry-run "good v2.0.2 Frozen train" 2>&1)" && rc=0 || rc=$?
check "dry-run also refuses an untagged train" test "$rc" -ne 0
check "dry-run refusal names the policy list" bash -c "printf '%s' \"\$0\" | grep -q 'untag policy list'" "$out"

echo "━━━ Test 9: --lockstep fans one version to every manifest train ━━━"
fixture_repo "$TMPROOT/t9"
printf '# v5-style wave manifest (dependency order, no versions)\ngood Core of the wave\nlibx\n' >"$TMPROOT/t9-wave.txt"
out="$(cd "$TMPROOT/t9" && BATCH_RELEASE_LOG_DIR="$TMPROOT/t9-logs" bash "$SCRIPT" --lockstep v2.0.9 --from-manifest "$TMPROOT/t9-wave.txt" 2>&1)" && rc=0 || rc=$?
check "lockstep release exits 0" test "$rc" -eq 0
check "lockstep tag created (described train)" bash -c "git -C \"\$0\" tag -l good/v2.0.9 | grep -q ." "$TMPROOT/t9"
check "lockstep tag created (bare train)" bash -c "git -C \"\$0\" tag -l libx/v2.0.9 | grep -q ." "$TMPROOT/t9"
check "lockstep tag carries the manifest description" bash -c "git -C \"\$0\" tag -l --format='%(contents:subject)' libx/v2.0.9 | grep -q 'Lockstep wave v2.0.9'" "$TMPROOT/t9"
check "tree fully restored" bash -c "git -C \"\$0\" status --porcelain | wc -l | grep -qx 0" "$TMPROOT/t9"

echo "━━━ Test 10: --lockstep refuses a wrong-major train (path-vs-tag) ━━━"
fixture_repo "$TMPROOT/t10"
printf 'good Wrong-major train\n' >"$TMPROOT/t10-wave.txt"
out="$(cd "$TMPROOT/t10" && bash "$SCRIPT" --lockstep v3.0.0 --from-manifest "$TMPROOT/t10-wave.txt" 2>&1)" && rc=0 || rc=$?
check "lockstep wrong-major exits nonzero" test "$rc" -ne 0
check "path-vs-tag guard fired" bash -c "printf '%s' \"\$0\" | grep -q 'end in /v3'" "$out"
check "no tag was created" bash -c "! git -C \"\$0\" tag -l good/v3.0.0 | grep -q ." "$TMPROOT/t10"

echo "━━━ Test 11: --lockstep + untag policy + malformed input ━━━"
fixture_repo "$TMPROOT/t11"
printf '# fixture untag list\nlibx\n' >"$TMPROOT/t11-untagged.txt"
printf 'good Fine\nlibx Frozen\n' >"$TMPROOT/t11-wave.txt"
out="$(cd "$TMPROOT/t11" && UNTAGGED_TRAINS_FILE="$TMPROOT/t11-untagged.txt" bash "$SCRIPT" --lockstep v2.0.10 --from-manifest "$TMPROOT/t11-wave.txt" 2>&1)" && rc=0 || rc=$?
check "lockstep refuses an untagged train" test "$rc" -ne 0
check "untag policy error shown" bash -c "printf '%s' \"\$0\" | grep -q 'untag policy list'" "$out"
check "no tag was created" bash -c "! git -C \"\$0\" tag -l '*/v2.0.10' | grep -q ." "$TMPROOT/t11"
out="$(cd "$TMPROOT/t11" && bash "$SCRIPT" --lockstep v2.0.11 --from-manifest "$TMPROOT/t11-wave.txt" \"good stray triple\" 2>&1)" && rc=0 || rc=$?
check "lockstep rejects positional triples" test "$rc" -ne 0
check "lockstep positional error is actionable" bash -c "printf '%s' \"\$0\" | grep -q 'takes the module list from --from-manifest'" "$out"
out="$(cd "$TMPROOT/t11" && bash "$SCRIPT" --lockstep v2.0.12 --from-manifest "$TMPROOT/t11-wave.txt" --dry-run 2>&1)" && rc=0 || rc=$?
check "lockstep dry-run exits 0 without tagging" test "$rc" -eq 0
check "lockstep dry-run creates no tags" bash -c "! git -C \"\$0\" tag -l '*/v2.0.12' | grep -q ." "$TMPROOT/t11"

if [ "$FAILED" -eq 0 ]; then
	echo ""
	echo "All batch-release smoke tests passed."
	exit 0
else
	echo ""
	echo "Some batch-release smoke tests FAILED."
	exit 1
fi
