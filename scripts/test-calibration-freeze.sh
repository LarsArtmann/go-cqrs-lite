#!/usr/bin/env bash
# test-calibration-freeze.sh — structural self-test for the calibration
# baseline's freeze-on-fail + cold-establishment contract in
# .github/workflows/benchmarks.yml (A2/F013-F014, 2026-11-09).
#
# The contract has three load-bearing guards, none of which a local bench
# can exercise (they live in GitHub Actions semantics):
#   F1 freeze-on-fail: the "Refresh calibration baseline" step must carry
#      `if: success()` — a red drift gate must NOT refresh the baseline, or
#      night N+1 compares against the regression itself and goes green.
#   F2 cold-path: the drift step must handle a MISSING baseline file by
#      establishing one (comparison starts next run), not failing the job.
#   F3 evidence: the drift-log upload must be `if-no-files-found: error` —
#      a green run may not pass with silently absent evidence.
#
# This script pins all three textually, then mutation-proves itself by
# running the same checks against corrupted copies (guard stripped) and
# requiring each corruption to FAIL.
#
# Run: bash scripts/test-calibration-freeze.sh
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
WORKFLOW="$REPO_ROOT/.github/workflows/benchmarks.yml"

fail() {
	echo "test-calibration-freeze: $*" >&2
	exit 1
}

# check Guards <file> — prints FAIL lines, returns count.
check_guards() {
	local f="$1" bad=0

	if ! grep -A3 'name: Refresh calibration baseline for the next run' "$f" |
		grep -q 'if: success()'; then
		echo "F1: refresh step lost its if: success() freeze guard"
		bad=$((bad + 1))
	fi

	if ! grep -q 'no runner-class baseline yet — this run establishes one' "$f"; then
		echo "F2: cold-path (missing baseline) guard absent from the drift step"
		bad=$((bad + 1))
	fi

	if ! grep -A12 'name: Upload drift log' "$f" | grep -q 'if-no-files-found: error'; then
		echo "F3: drift-log upload is not evidence-class (if-no-files-found: error)"
		bad=$((bad + 1))
	fi

	echo "$bad"
}

echo "━━━ Contract pins against the live workflow ━━━"
violations="$(check_guards "$WORKFLOW" | tail -1)"
check_guards "$WORKFLOW" >/dev/null || true
if [ "$violations" -ne 0 ]; then
	check_guards "$WORKFLOW" >&2 || true
	fail "$violations guard(s) missing — freeze/cold/evidence contract broken"
fi
echo "  ✓ F1 freeze-on-fail, F2 cold-path, F3 evidence all pinned"

echo "━━━ Mutation proofs (each corruption MUST fail) ━━━"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

# Mutation F1: strip the freeze guard.
sed 's/^\(\s*\)if: success()$/\1if: always()/' "$WORKFLOW" >"$tmp/m1.yml"
n="$(check_guards "$tmp/m1.yml" | tail -1)"
[ "$n" -ne 0 ] || fail "mutation F1 (if: always()) was NOT caught"
echo "  ✓ F1 mutation caught"

# Mutation F2: break the cold path.
sed 's/no runner-class baseline yet — this run establishes one/COLD PATH REMOVED/' "$WORKFLOW" >"$tmp/m2.yml"
n="$(check_guards "$tmp/m2.yml" | tail -1)"
[ "$n" -ne 0 ] || fail "mutation F2 (cold path removed) was NOT caught"
echo "  ✓ F2 mutation caught"

# Mutation F3: demote the evidence policy.
sed 's/^\(\s*\)if-no-files-found: error$/\1if-no-files-found: warn/' "$WORKFLOW" >"$tmp/m3.yml"
n="$(check_guards "$tmp/m3.yml" | tail -1)"
[ "$n" -ne 0 ] || fail "mutation F3 (warn downgrade) was NOT caught"
echo "  ✓ F3 mutation caught"

echo "test-calibration-freeze: OK (3 guards pinned, 3 mutations caught)"
