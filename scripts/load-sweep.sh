#!/usr/bin/env bash
# load-sweep.sh — run timing-assertion tests under deliberate CPU load.
#
# The 12:39 session burned two full verify-gate cycles (~20 min each)
# discovering load-sensitive flakes one at a time. This script front-loads
# that discovery: it starts CPU soakers (all cores minus one), runs the
# timing-assertion suites (-run 'Latency|Timer|Deadline'), and reports.
# Run it BEFORE `nix run .#verify` when a session touched timing paths.
#
# Usage: nix run .#load-sweep   (or: bash scripts/load-sweep.sh)
# Env:   SOAKER_CORES — override soaker count (default: nproc - 1, min 1)
#        EXTRA_TEST_ARGS — extra flags for go test (e.g. "-race")
# Exit: 0 = all timing tests survived load, 1 = flake found under load.

set -euo pipefail

cd "$(git rev-parse --show-toplevel)"

TAGS="goexperiment.jsonv2"
RUN_PATTERN='Latency|Timer|Deadline'
# Modules containing timing-assertion tests (Test.*(Latency|Timer|Deadline)).
TIMING_MODULES=(
	benchkit
	cmd/cqrs-lint
	decider
	event
	metaengine
	middleware
	scheduling
	storage
)

nproc_cmd="$(command -v nproc || echo true)"
total_cores=$("$nproc_cmd" 2>/dev/null || echo 2)
SOAKER_CORES="${SOAKER_CORES:-$((total_cores > 2 ? total_cores - 1 : 1))}"

echo "==> Starting $SOAKER_CORES CPU soaker(s) ($total_cores cores total)"
soaker_pids=()
for _ in $(seq 1 "$SOAKER_CORES"); do
	# shellcheck disable=SC2016 # busy-loop must not expand anything
	bash -c 'while :; do :; done' &
	soaker_pids+=($!)
done

cleanup() {
	for pid in "${soaker_pids[@]}"; do
		kill "$pid" 2>/dev/null || true
	done
}
trap cleanup EXIT INT TERM

# Let the soakers ramp up before measuring.
sleep 2

failed=0
for mod in "${TIMING_MODULES[@]}"; do
	echo ""
	echo "--- $mod (-run '$RUN_PATTERN' under load) ---"
	if ! (cd "$mod" && GOWORK=off go test -tags "$TAGS" -run "$RUN_PATTERN" \
		-count=1 -timeout=10m ${EXTRA_TEST_ARGS:-} ./... 2>&1 | tee "/tmp/load-sweep-${mod//\//-}.log"); then
		echo "FAIL: $mod timing tests flaked under load (see /tmp/load-sweep-${mod//\//-}.log)"
		failed=1
	fi
done

echo ""
if [ "$failed" -ne 0 ]; then
	echo "❌ Load-sweep found flaky timing test(s) — fix before running #verify."
	exit 1
fi

echo "✅ All timing-assertion tests survived CPU load."
