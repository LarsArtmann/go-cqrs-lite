#!/usr/bin/env bash
# test-calibration-drift.sh — Smoke test for scripts/calibration-drift.sh's
# validation paths (no benches executed: every case exits before benching).
#
# Verifies:
# 1. --baseline + --write-baseline together is a usage error (exit 2)
# 2. A missing baseline artifact is a usage error (exit 2)
# 3. An empty baseline artifact is a usage error (exit 2)
# 4. CoW TMPDIR (faked) refuses to run without CALIB_ALLOW_COW (exit 2)
# 5. CALIB_ALLOW_COW=1 proceeds past the CoW refusal (warning emitted)
#
# Run: bash scripts/test-calibration-drift.sh
set -euo pipefail

SCRIPT="$(cd "$(dirname "$0")" && pwd)/calibration-drift.sh"
FAILED=0

echo "━━━ Test 1: --baseline and --write-baseline are mutually exclusive ━━━"
if bash "$SCRIPT" --baseline /tmp/a --write-baseline /tmp/b >/dev/null 2>&1; then
	echo "  ✗ FAIL: should exit 2 on mutually exclusive flags"
	FAILED=1
else
	echo "  ✓ PASS: usage error rejected"
fi

echo "━━━ Test 2: missing baseline artifact ━━━"
out="$(CALIB_MAX_LOAD=1000 bash "$SCRIPT" --baseline /tmp/does-not-exist-calib 2>&1 || true)"
if echo "$out" | grep -q "baseline artifact not found"; then
	echo "  ✓ PASS: missing artifact reported"
else
	echo "  ✗ FAIL: missing-artifact message absent"
	FAILED=1
fi

echo "━━━ Test 3: empty baseline artifact ━━━"
empty=$(mktemp)
out="$(CALIB_MAX_LOAD=1000 bash "$SCRIPT" --baseline "$empty" 2>&1 || true)"
rm -f "$empty"
if echo "$out" | grep -q "baseline artifact has no rows"; then
	echo "  ✓ PASS: empty artifact reported"
else
	echo "  ✗ FAIL: empty-artifact message absent"
	FAILED=1
fi

echo "━━━ Test 4: CoW TMPDIR refuses to run ━━━"
out="$(CALIB_MAX_LOAD=1000 CALIB_FAKE_TMPFS_TYPE=btrfs bash "$SCRIPT" 2>&1 || true)"
if echo "$out" | grep -q "is on btrfs (CoW)"; then
	echo "  ✓ PASS: CoW refusal emitted"
else
	echo "  ✗ FAIL: CoW refusal absent"
	FAILED=1
fi

echo "━━━ Test 5: CALIB_ALLOW_COW=1 proceeds past the CoW gate ━━━"
out="$(CALIB_MAX_LOAD=1000 CALIB_FAKE_TMPFS_TYPE=zfs CALIB_ALLOW_COW=1 bash "$SCRIPT" --baseline /tmp/does-not-exist-calib 2>&1 || true)"
if echo "$out" | grep -q "proceeding via CALIB_ALLOW_COW"; then
	echo "  ✓ PASS: override proceeds past the CoW gate"
else
	echo "  ✗ FAIL: override did not proceed past the CoW gate"
	FAILED=1
fi

if [ "$FAILED" -ne 0 ]; then
	echo "❌ test-calibration-drift: FAILURES above"
	exit 1
fi

echo "✅ test-calibration-drift: all checks passed"
