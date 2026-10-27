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
#   GT_LOG_OUT   full-output log path (default build/test-logs/<label>-<ts>-<seed>.log)
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
# -count is ALWAYS 1: Ginkgo v2 refuses any other value ("Ginkgo does not
# support using go test -count to rerun suites"), which made the entire
# repeat leg red on its 2026-10-27 first real execution — decider, event,
# command, query and metaengine are Ginkgo suites. Repetition is done by
# run_once (below) invoking the binary N times, which is what -count does
# anyway, just without Ginkgo's veto.
args+=(-count=1 -timeout="$TIMEOUT" -p "$P")

# -cpu takes ONE value per invocation: Ginkgo v2 rejects a LIST ("RunSpecs more
# than once"), so `-cpu=1,4` broke the same five suites that `-count=5` broke.
# The repeat leg cycles the values one per iteration instead, giving both
# dimensions without the veto. A single GT_CPU value behaves as before.
cpu_values=()
if [ -n "${GT_CPU:-}" ]; then
	IFS=',' read -r -a cpu_values <<<"$GT_CPU"
fi

if [ "${SHUFFLE:-on}" = "off" ]; then
	echo "==> shuffle DISABLED (SHUFFLE=off) — order-dependent failures cannot be caught in this run" >&2
else
	seed="${SHUFFLE_SEED:-}"
	if [ -z "$seed" ]; then
		seed="$(new_shuffle_seed)"
	fi
	log_shuffle_seed "$LABEL" "$seed"
fi

# The single-invocation path (COUNT=1) uses this too, so the log name carries
# the real seed instead of "noshuffle".
seed_for_iteration="$seed"

# run_args ITER — the full go test argument list for one invocation.
run_args() { # $1=iteration (1-based)
	local iter="$1"
	local -a out=("${args[@]}")

	if [ "${#cpu_values[@]}" -gt 0 ]; then
		local idx=$(((iter - 1) % ${#cpu_values[@]}))
		out+=(-cpu="${cpu_values[$idx]}")
	fi

	if [ "${SHUFFLE:-on}" != "off" ]; then
		out+=(-shuffle="$seed_for_iteration")
	fi

	printf '%s\n' "${out[@]}"
}

# Always keep the evidence. A red run used to survive only as whatever
# scrolled past, which is why the 2026-10-27 failure needed a full replay to
# even name the package — and the replay passed, so the log would have been the
# only record that it ever failed.
STAMP="$(date -u +%Y%m%dT%H%M%SZ)"

if [ "${GT_JSON:-0}" = "1" ] && [ "$COUNT" != "1" ]; then
	echo "::error::GT_JSON=1 with GT_COUNT=$COUNT is not supported (a JSON stream must describe one invocation)" >&2
	exit 2
fi

print_replay() { # $1=seed
	{
		echo ""
		echo "::error::${LABEL} failed under shuffle seed ${1}."
		echo "Replay exactly this order:"
		echo "  nix run .#rerun-shuffle-seed -- ${1}"
		echo "Narrow it to one package (the failing one is named above):"
		echo "  nix run .#rerun-shuffle-seed -- ${1} ./<module>/..."
	} >&2
}

if [ "${GT_JSON:-0}" = "1" ]; then
	# -json for the runtime budget (T12): one stream, tee'd so a human still
	# sees progress while scripts/check-test-runtime.sh gets per-package
	# Elapsed values. PIPESTATUS keeps go test's exit code authoritative.
	json_out="${GT_JSON_OUT:-build/test-json/${LABEL}-${STAMP}.json}"
	mkdir -p "$(dirname "$json_out")"
	status=0
	seed_for_iteration="$seed"
	mapfile -t json_args < <(run_args 1)
	CGO_ENABLED=1 go test "${json_args[@]}" -json "${PATHS[@]}" "$@" | tee "$json_out" || status=${PIPESTATUS[0]}
	echo "==> JSON event stream: $json_out" >&2

	if [ "$status" -ne 0 ] && [ "${SHUFFLE:-on}" != "off" ]; then
		print_replay "$seed"
	fi

	exit "$status"
fi

# run_once ITER TAGS_EXTRA... — one go test invocation, logged, status returned.
run_once() { # $1=iteration number (1-based)
	local iter="$1"
	shift
	local log_out
	log_out="${GT_LOG_OUT:-build/test-logs/${LABEL}-${STAMP}-i${iter}-${seed_for_iteration:-noshuffle}.log}"
	mkdir -p "$(dirname "$log_out")"
	local -a iter_args
	mapfile -t iter_args < <(run_args "$iter")
	if CGO_ENABLED=1 go test "${iter_args[@]}" "${PATHS[@]}" "$@" 2>&1 | tee "$log_out"; then
		echo "==> Full output: $log_out" >&2
		return 0
	fi
	echo "==> Full output: $log_out" >&2
	return 1
}

status=0
if [ "$COUNT" = "1" ]; then
	run_once 1 "$@" || status=1
else
	echo "==> Repeating ${COUNT}x, one process per iteration (-count and multi-value -cpu are rejected by Ginkgo)" >&2
	for ((iter = 1; iter <= COUNT; iter++)); do
		if [ -n "${SHUFFLE_SEED:-}" ]; then
			# A caller that pinned the seed asked for one specific order.
			seed_for_iteration="$SHUFFLE_SEED"
		elif [ "$iter" -eq 1 ] || [ "${SHUFFLE:-on}" = "off" ]; then
			seed_for_iteration="$seed"
		else
			# Fresh order per repetition is the point of a repeat leg.
			seed_for_iteration="$(new_shuffle_seed)"
			log_shuffle_seed "${LABEL}-i${iter}" "$seed_for_iteration"
		fi
		echo "==> Iteration ${iter}/${COUNT} (seed ${seed_for_iteration})" >&2
		if ! run_once "$iter" "$@"; then
			status=1
			if [ "${SHUFFLE:-on}" != "off" ]; then
				print_replay "$seed_for_iteration"
			fi
			break
		fi
	done
fi

exit "$status"
