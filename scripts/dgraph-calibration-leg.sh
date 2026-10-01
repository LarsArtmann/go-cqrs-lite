#!/usr/bin/env bash
# dgraph-calibration-leg.sh — the M20 dgraph constants re-anchor campaign as
# one quiet-window leg (publish-integrity plan; calibration-2026-08-30.md
# §Protocol). Runs the mechanical load gate WITH provenance capture first
# (protocol item 6+7), then both calibration benches against an ephemeral
# Dgraph (count=5 = median-of-4 after discard-cold, protocol item 8's
# preferred depth).
#
# Usage (normally via scripts/quiet-campaign.sh):
#   scripts/quiet-campaign.sh --legs dgraph -- scripts/dgraph-calibration-leg.sh
# Direct:
#   scripts/dgraph-calibration-leg.sh
set -uo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

cd "$REPO_ROOT" || exit 1
if ! scripts/calibration-gate.sh --provenance dgraph; then
	echo "dgraph-calibration-leg: calibration gate FAILED — not benching through load" >&2
	exit 1
fi

nix run .#ephemeral-dgraph -- bash -c \
	'cd metaengine/dgraphengine && GOWORK=off go test -run "^$" \
	 -bench "BenchmarkCalibration_DgraphScaled|BenchmarkCalibration_DgraphSearchQuery" \
	 -benchtime 20x -count 5 .'
