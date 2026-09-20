#!/usr/bin/env bash
# verify-load-guard.sh — the in-#verify load threshold (M05, 2026-09-20).
# Verify attempts launched under load red out in the expensive phases 30+
# minutes in (the S03 arc attempts 7-9 class). This guard refuses the launch
# in ~0.1s when the host is loud, prints the one-line retry recipe, and
# honors VERIFY_FORCE=1 for deliberate burns. Delegates the probe to
# calibration-gate.sh (same load1+load5 v2 rule, same CI pass-through).
#
# Usage (called from the #verify flake app head):
#   scripts/verify-load-guard.sh                    # ceiling VERIFY_MAX_LOAD (default 10)
#   VERIFY_FORCE=1 scripts/verify-load-guard.sh     # override, prints a burn notice
#
# Fixture hook (self-test only): CALIB_GATE_LOADAVG_FILE (calibration-gate pattern).
#
# Exit codes: 0 = quiet enough (or forced, or CI), 1 = loud — do not launch.
set -uo pipefail

CEILING="${VERIFY_MAX_LOAD:-10}"

if [[ "${CI:-}" == true ]]; then
	echo "verify-load-guard: CI environment — load check informational, proceeding"
	bash "$(dirname "${BASH_SOURCE[0]}")/calibration-gate.sh" --max-load "$CEILING" || true
	exit 0
fi

if [[ "${VERIFY_FORCE:-}" == 1 ]]; then
	echo "verify-load-guard: VERIFY_FORCE=1 — overriding the load gate (a burn under load is on you)"
	bash "$(dirname "${BASH_SOURCE[0]}")/calibration-gate.sh" --max-load "$CEILING" || true
	exit 0
fi

if bash "$(dirname "${BASH_SOURCE[0]}")/calibration-gate.sh" --max-load "$CEILING"; then
	echo "verify-load-guard: load under ${CEILING} — launch allowed"
	exit 0
fi

echo "verify-load-guard: REFUSED — host load ≥ ${CEILING}. Do not burn the attempt:"
echo "  retry recipe : nix run .#can-run-composed-gate -- --wait-loop && nix run .#verify"
echo "  preflight    : bash scripts/preflight-composed.sh   (fix cheap reds first)"
echo "  override     : VERIFY_FORCE=1 nix run .#verify"
exit 1
