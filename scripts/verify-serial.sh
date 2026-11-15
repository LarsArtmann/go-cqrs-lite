#!/usr/bin/env bash
# verify-serial — the #verify-serial fallback tier (:1343, M39).
#
# The composed gate refuses to launch unless load1 AND load5 < 10 plus a
# 60s stable tree — a precondition that never held on this 32-core shared
# host at sustained load 15-87 (20 attempts / 3600s, zero launches). This
# tier trades wall clock for launchability: SAME gate list as #verify (it
# literally runs it), package parallelism pinned to 1, NO aggregate load
# precondition, bounded by an outer wall clock.
#
# Timeout honesty: #verify's go legs already carry PER-PACKAGE timeouts
# (test 10m, race 12m) and every ratchet gate is seconds-fast; this wrapper
# adds the outer bound that catches a hang anywhere else. TRUE per-gate
# timeouts would require lifting the gate list out of flake.nix into a
# shared script — deliberately not done: a second copy of the gate list is
# a drift factory, and the outer bound covers the residual risk.
#
# Usage:
#   nix run .#verify-serial                # loud-box composed proof
#   VERIFY_SERIAL_MAX=8h nix run .#verify-serial
#   nix run .#verify-serial -- --self-test # plumbing check, never runs #verify
#
# Exclusivity still applies: never run concurrent with #verify or the
# integration suites — serial is slower, not concurrent-safe-er.
set -uo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

# shellcheck source=scripts/go-env.sh
source "$repo_root/scripts/go-env.sh"

if [ "${1:-}" = "--self-test" ]; then
	failures=0
	# The escape hatch this tier depends on must keep existing.
	if ! grep -q 'VERIFY_FORCE' "$repo_root/scripts/verify-load-guard.sh"; then
		echo "self-test FAIL: verify-load-guard.sh no longer honors VERIFY_FORCE" >&2
		failures=$((failures + 1))
	fi
	# The parallelism knob must stay wired into both go legs.
	if ! grep -q 'VERIFY_TEST_P' "$repo_root/flake.nix"; then
		echo "self-test FAIL: flake.nix no longer reads VERIFY_TEST_P" >&2
		failures=$((failures + 1))
	fi
	command -v timeout >/dev/null 2>&1 || {
		echo "self-test FAIL: coreutils timeout not on PATH" >&2
		failures=$((failures + 1))
	}
	[ "$failures" -eq 0 ] && echo "self-test: serial-tier plumbing verified (VERIFY_FORCE, VERIFY_TEST_P, timeout)"
	[ "$failures" -eq 0 ]
	exit $?
fi

export VERIFY_FORCE=1
export VERIFY_TEST_P=1
export VERIFY_SERIAL_MAX="${VERIFY_SERIAL_MAX:-6h}"

echo "verify-serial: loud-box tier — VERIFY_FORCE=1, VERIFY_TEST_P=1, outer bound ${VERIFY_SERIAL_MAX}"
exec timeout --preserve-status --kill-after=5m "$VERIFY_SERIAL_MAX" nix run .#verify "$@"
