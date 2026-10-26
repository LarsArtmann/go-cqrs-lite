#!/usr/bin/env bash
# rerun-shuffle-seed.sh — replay one recorded shuffle order exactly.
#
# Counterpart of go-test-shuffled.sh: the runner prints a seed on every run and
# again on failure, and this script turns that seed back into the same `go test`
# invocation without any guessing.
#
# Usage:
#   bash scripts/rerun-shuffle-seed.sh <seed>                  # whole suite
#   bash scripts/rerun-shuffle-seed.sh <seed> ./event/...      # one module
#   bash scripts/rerun-shuffle-seed.sh <seed> -race            # race mode
#   GT_LABEL=test-race ... (label only affects the seed log)
#
# Environment: same knobs as go-test-shuffled.sh (GT_PATHS, GT_TIMEOUT,
# GT_COUNT, GT_CPU, GT_RACE).

set -euo pipefail

cd "$(git rev-parse --show-toplevel)"

if [ $# -lt 1 ] || [ -z "${1:-}" ]; then
	echo "usage: rerun-shuffle-seed.sh <seed> [paths...|-race]" >&2
	echo "       find recorded seeds in build/shuffle-seeds.log" >&2
	exit 2
fi

seed="$1"
shift

export SHUFFLE_SEED="$seed"
export GT_LABEL="${GT_LABEL:-rerun}"

# Extra args: paths replace the suite; flags are forwarded as-is.
extra=()
for arg in "$@"; do
	case "$arg" in
	-race) export GT_RACE=1 ;;
	*) extra+=("$arg") ;;
	esac
done

if [ "${#extra[@]}" -gt 0 ]; then
	export GT_PATHS="${extra[*]}"
	extra=()
fi

exec bash "$PWD/scripts/go-test-shuffled.sh" "${extra[@]}"
