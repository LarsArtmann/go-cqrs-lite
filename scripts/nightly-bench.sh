#!/usr/bin/env bash
# nightly-bench.sh — scheduled local benchmark-gate run: verification vs the
# committed baseline inside a quiet window (the gate's own noise/save guards
# apply; a REGRESSION keeps the baseline and leaves the log for triage).
#
# Driver: systemd user timer (scripts/nightly/go-cqrs-nightly-bench.timer) or
# manual: nix run .#nightly-bench
# Logs: /var/tmp/cqrs-nightly/<UTC-date>.log
set -uo pipefail

LOGDIR=/var/tmp/cqrs-nightly
mkdir -p "$LOGDIR"
LOG="$LOGDIR/$(date -u +%Y-%m-%d).log"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO="$(git -C "$SCRIPT_DIR" rev-parse --show-toplevel 2>/dev/null || echo "$SCRIPT_DIR/..")"
cd "$REPO" || exit 2

echo "[$(date -u +%Y-%m-%dT%H:%M:%SZ)] nightly bench start (repo=$REPO)" >>"$LOG"
scripts/quiet-window-run.sh --deadline 10800 --attempts 2 \
	--log "$LOG" \
	-- ./scripts/benchmark-regression.sh >>"$LOG" 2>&1
rc=$?
echo "[$(date -u +%Y-%m-%dT%H:%M:%SZ)] nightly bench exit=$rc" >>"$LOG"
exit "$rc"
