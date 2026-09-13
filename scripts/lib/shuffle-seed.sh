#!/usr/bin/env bash
# shuffle-seed.sh — shared shuffle-seed helpers for integration scripts.
# Source from an ephemeral/vm script; do NOT execute directly.
#
# Why: `go test -shuffle=on` picks a random seed that is only visible as
# the first output line — a failed CI ordering could not be replayed
# reliably. These helpers mint an EXPLICIT positive seed, log it
# (timestamp + label), and pass `-shuffle="$SEED"` to go test, so any
# recorded run can be replayed exactly:
#
#   go test -shuffle=<seed> -tags "..." ./...
#
# Usage inside an integration script:
#   source "$(dirname "$0")/lib/shuffle-seed.sh"
#   SEED=$(new_shuffle_seed)
#   log_shuffle_seed "dgraphengine" "$SEED"
#   go test -shuffle="$SEED" ...
#
# Log: build/shuffle-seeds.log (gitignored; override: SHUFFLE_SEED_LOG).
# The default is ABSOLUTE — anchored at this lib's location (scripts/lib/
# → repo root two levels up) — so the entry lands in the same file no
# matter where the sourcing script has cd'd by log time.

_SHUFFLE_SEED_LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]:-$0}")" && pwd)"
SHUFFLE_SEED_LOG="${SHUFFLE_SEED_LOG:-"$_SHUFFLE_SEED_LIB_DIR/../../build/shuffle-seeds.log"}"

new_shuffle_seed() {
	# 48-bit positive int from /dev/urandom: comfortably inside go's int64
	# flag range, non-zero, decimal.
	local seed
	seed=$(od -An -N6 -tu8 /dev/urandom 2>/dev/null | tr -d ' \n')
	if [ -z "$seed" ] || [ "$seed" = "0" ]; then
		seed=$(( $(date +%s) * 1000 + 10#$((RANDOM % 1000)) ))
	fi
	echo "$seed"
}

log_shuffle_seed() {
	local label="$1" seed="$2"
	mkdir -p "$(dirname "$SHUFFLE_SEED_LOG")"
	printf '%s seed=%s label=%s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "$seed" "$label" >>"$SHUFFLE_SEED_LOG"
	echo "==> shuffle seed=${seed} (${label}) — logged to ${SHUFFLE_SEED_LOG}; replay with: go test -shuffle=${seed}"
}
