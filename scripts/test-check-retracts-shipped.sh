#!/usr/bin/env bash
# test-check-retracts-shipped.sh — clean-dir acceptance test for
# scripts/check-retracts-shipped.sh (throwaway fixture repo; no network).
#
# 1. A retract on master that NO tag carries → gate fails (inert retract).
# 2. Tagging a release whose go.mod carries the retract → gate passes.
# 3. A module with retracts and NO tags at all → gate fails.
#
# Run: bash scripts/test-check-retracts-shipped.sh
set -euo pipefail

GATE="$(cd "$(dirname "$0")" && pwd)/check-retracts-shipped.sh"
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

# fixture_repo <path> writes a tiny module (path WITHOUT /vN — v0/v1 tags)
# whose v1.0.0 tag predates any retract.
fixture_repo() {
	local root="$1"
	mkdir -p "$root/lib"

	cat >"$root/go.mod" <<'EOF'
module github.com/example/fixture

go 1.26.0
EOF

	cat >"$root/lib/go.mod" <<'EOF'
module github.com/example/fixture/lib

go 1.26.0
EOF

	git -C "$root" init -q
	git -C "$root" config user.email t@example.com
	git -C "$root" config user.name t
	local notag=(-c tag.gpgSign=false -c tag.forceSignAnnotated=false)
	git -C "$root" add -A
	git -C "$root" commit -qm init
	git "${notag[@]}" -C "$root" tag lib/v1.0.0
}

TMPROOT="$(mktemp -d)"
trap 'rm -rf "$TMPROOT"' EXIT

echo "━━━ Test 1: retract on master, no release ships it → FAIL ━━━"
fixture_repo "$TMPROOT/t1"
cat >>"$TMPROOT/t1/lib/go.mod" <<'EOF'
retract v1.0.0 // poisoned build
EOF
out="$(cd "$TMPROOT/t1" && bash "$GATE" 2>&1)" && rc=0 || rc=$?
check "gate exits nonzero on the unshipped retract" test "$rc" -ne 0
check "gate names the newest tag lacking the retract" bash -c "printf '%s' \"\$0\" | grep -q 'missing from lib/v1.0.0'" "$out"
check "gate points at the inert retract" bash -c "printf '%s' \"\$0\" | grep -q 'inert until a release ships them'" "$out"

echo "━━━ Test 2: a tagged release carrying the retract → PASS ━━━"
fixture_repo "$TMPROOT/t2"
cat >>"$TMPROOT/t2/lib/go.mod" <<'EOF'
retract v1.0.0 // poisoned build
EOF
git -C "$TMPROOT/t2" add -A
git -C "$TMPROOT/t2" -c tag.gpgSign=false -c tag.forceSignAnnotated=false tag lib/v1.0.1
if (cd "$TMPROOT/t2" && bash "$GATE" >/dev/null 2>&1); then
	echo "  ✓ PASS: gate passes once a release ships the retract"
else
	echo "  ✗ FAIL: gate should pass when the newest tag carries the retract"
	FAILED=1
fi

echo "━━━ Test 3: retracts on a never-tagged module → FAIL ━━━"
fixture_repo "$TMPROOT/t3"
mkdir -p "$TMPROOT/t3/fresh"
cat >"$TMPROOT/t3/fresh/go.mod" <<'EOF'
module github.com/example/fixture/fresh

go 1.26.0

retract v0.9.0
EOF
git -C "$TMPROOT/t3" add -A
git -C "$TMPROOT/t3" commit -qm fresh
out="$(cd "$TMPROOT/t3" && bash "$GATE" 2>&1)" && rc=0 || rc=$?
check "gate exits nonzero for a tagless module" test "$rc" -ne 0
check "gate explains that nothing ships the retracts" bash -c "printf '%s' \"\$0\" | grep -q 'NO tags'" "$out"

if [ "$FAILED" -eq 0 ]; then
	echo ""
	echo "All check-retracts-shipped acceptance tests passed."
	exit 0
else
	echo ""
	echo "Some check-retracts-shipped acceptance tests FAILED."
	exit 1
fi
