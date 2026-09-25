#!/usr/bin/env bash
# nightly-bench.sh — scheduled local benchmark-gate run: verification vs the
# committed baseline inside a quiet window (the gate's own noise/save guards
# apply; a REGRESSION keeps the baseline and leaves the log for triage).
#
# Driver: systemd user timer (scripts/nightly/go-cqrs-nightly-bench.timer) or
# manual: nix run .#nightly-bench
# Logs: /var/tmp/cqrs-nightly/<UTC-date>.log
#
# --self-test: composition checks only (no benches, no /var/tmp writes) —
# syntax, the Sunday-only guard on the load-sweep leg, the delegation lines,
# and the timer unit files. The engine (quiet-window-run) and the gate
# (test-benchmark-regression) carry their own self-tests, wired separately
# into #check-release-scripts / #check-bench-gate.
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

if [ "${1:-}" = "--self-test" ]; then
	failures=0

	bash -n "$SCRIPT_DIR/nightly-bench.sh" || failures=$((failures + 1))

	# The Sunday-only guard and both delegation lines must stay in the real
	# run path (rot here silently kills the weekly leg or the quiet window).
	for pattern in 'date -u +%u" = "7' 'quiet-window-run.sh' 'benchmark-regression.sh'; do
		if ! grep -qF -- "$pattern" "$SCRIPT_DIR/nightly-bench.sh"; then
			echo "  ✗ FAIL: nightly-bench lost contract line: $pattern" >&2
			failures=$((failures + 1))
		fi
	done

	# The systemd timer units must exist and drive this script.
	for unit in go-cqrs-nightly-bench.service go-cqrs-nightly-bench.timer; do
		if ! [ -f "$SCRIPT_DIR/nightly/$unit" ]; then
			echo "  ✗ FAIL: missing timer unit scripts/nightly/$unit" >&2
			failures=$((failures + 1))
		fi
	done
	if ! grep -q 'nightly-bench.sh' "$SCRIPT_DIR/nightly/go-cqrs-nightly-bench.service" 2>/dev/null; then
		echo "  ✗ FAIL: timer service does not invoke nightly-bench.sh" >&2
		failures=$((failures + 1))
	fi

	if [ "$failures" -gt 0 ]; then
		echo "nightly-bench self-test: ${failures} failure(s)" >&2
		exit 1
	fi
	echo "nightly-bench self-test: all green"
	exit 0
fi

LOGDIR=/var/tmp/cqrs-nightly
mkdir -p "$LOGDIR"
LOG="$LOGDIR/$(date -u +%Y-%m-%d).log"
REPO="$(git -C "$SCRIPT_DIR" rev-parse --show-toplevel 2>/dev/null || echo "$SCRIPT_DIR/..")"
cd "$REPO" || exit 2

echo "[$(date -u +%Y-%m-%dT%H:%M:%SZ)] nightly bench start (repo=$REPO)" >>"$LOG"
scripts/quiet-window-run.sh --deadline 10800 --attempts 2 \
	--log "$LOG" \
	-- ./scripts/benchmark-regression.sh >>"$LOG" 2>&1
rc=$?

# Weekly load-sweep leg (owner ruling 2026-09-21): Sundays only — timing
# tests under deliberate CPU soakers. Never fails the nightly (its flakes
# are findings, not gate breaches); runs after the gate so the two never
# share a measurement window.
if [ "$(date -u +%u)" = "7" ]; then
	echo "[$(date -u +%Y-%m-%dT%H:%M:%SZ)] weekly load-sweep leg (Sundays)" >>"$LOG"
	scripts/load-sweep.sh >>"$LOG" 2>&1 || true
fi

echo "[$(date -u +%Y-%m-%dT%H:%M:%SZ)] nightly bench exit=$rc" >>"$LOG"
exit "$rc"
