#!/usr/bin/env bash
# go-test-shuffled.sh — the one `go test` runner behind #test and #test-race.
#
# Why: `-shuffle=on` prints its seed only on the FIRST line of output, so a
# failure caused by test order was effectively unreproducible unless someone
# scrolled back through CI logs. This runner mints an EXPLICIT seed, logs it
# (build/shuffle-seeds.log, same helper the integration scripts use) and prints
# the exact replay command, so any red run is reproducible by construction.
#
# It also refuses the silent-random case: with no seed in the environment the
# seed exists as a number in the log before go test starts, not as a hope.
#
# Usage: bash scripts/go-test-shuffled.sh [extra go test args...]
#   GT_LABEL     label for the seed log (default "test")
#   GT_PATHS     space-separated package patterns (required)
#   GT_RACE=1    add -race
#   GT_TIMEOUT   per-package timeout (default 30m)
#   GT_COUNT     -count value (default 1)
#   GT_CPU       -cpu value (optional, e.g. 1,4)
#   GT_P         how many PACKAGES run in parallel (default 4). Without a cap
#                go test uses NumCPU and the heaviest packages (duckdbengine
#                ~218s, tursoengine ~173s of CGo work) starve everything else
#                into false timeouts — observed 2026-10-27: four systemscenario
#                awaits and benchkit's journey budget failed purely because the
#                host was oversubscribed. #verify has capped at 4 for the same
#                reason since 2026-09-19.
#   GT_JSON=1    emit `go test -json` events to build/test-json/<label>-<ts>.json
#                (feeds scripts/check-test-runtime.sh); GT_JSON_OUT overrides
#   SHUFFLE=off  disable shuffling (escape hatch for a bisect)
#   SHUFFLE_SEED explicit seed (replay); otherwise a fresh one is minted
#
# Replay: nix run .#rerun-shuffle-seed -- <seed>

set -euo pipefail

cd "$(git rev-parse --show-toplevel)"

# shellcheck source=lib/shuffle-seed.sh
source "$PWD/scripts/lib/shuffle-seed.sh"
# shellcheck source=go-env.sh
source "$PWD/scripts/go-env.sh"

LABEL="${GT_LABEL:-test}"
TIMEOUT="${GT_TIMEOUT:-30m}"
COUNT="${GT_COUNT:-1}"
P="${GT_P:-4}"
[ -n "${GT_PATHS:-}" ] || {
	echo "::error::GT_PATHS is empty — run this through nix run .#test / .#test-race" >&2
	exit 2
}
read -r -a PATHS <<<"$GT_PATHS"

args=()
if [ -n "${GT_TAGS:-}" ]; then
	args+=(-tags="$GT_TAGS")
fi
if [ "${GT_RACE:-0}" = "1" ]; then
	args+=(-race)
fi
args+=(-count="$COUNT" -timeout="$TIMEOUT" -p "$P")
if [ -n "${GT_CPU:-}" ]; then
	args+=(-cpu="$GT_CPU")
fi
if [ "${SHUFFLE:-on}" = "off" ]; then
	echo "==> shuffle DISABLED (SHUFFLE=off) — order-dependent failures cannot be caught in this run" >&2
else
	seed="${SHUFFLE_SEED:-}"
	if [ -z "$seed" ]; then
		seed="$(new_shuffle_seed)"
	fi
	log_shuffle_seed "$LABEL" "$seed"
	args+=(-shuffle="$seed")
fi

status=0
if [ "${GT_JSON:-0}" = "1" ]; then
	# -json for the runtime budget (T12): one stream, tee'd so a human still
	# sees progress while scripts/check-test-runtime.sh gets per-package
	# Elapsed values. PIPESTATUS keeps go test's exit code authoritative.
	json_out="${GT_JSON_OUT:-build/test-json/${LABEL}-$(date -u +%Y%m%dT%H%M%SZ).json}"
	mkdir -p "$(dirname "$json_out")"
	CGO_ENABLED=1 go test "${args[@]}" -json "${PATHS[@]}" "$@" | tee "$json_out" || status=${PIPESTATUS[0]}
	echo "==> JSON event stream: $json_out" >&2
else
	CGO_ENABLED=1 go test "${args[@]}" "${PATHS[@]}" "$@" || status=$?
fi

if [ "$status" -ne 0 ] && [ "${SHUFFLE:-on}" != "off" ]; then
	{
		echo ""
		echo "::error::${LABEL} failed under shuffle seed ${seed}."
		echo "Replay exactly this order:"
		echo "  nix run .#rerun-shuffle-seed -- ${seed}"
		echo "Narrow it to one package (the failing one is named above):"
		echo "  nix run .#rerun-shuffle-seed -- ${seed} ./<module>/..."
	} >&2
fi

exit "$status"
