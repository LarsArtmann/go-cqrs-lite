#!/usr/bin/env bash
# verify-parallel.sh — Run module tests in parallel batches to cut verify time.
#
# Splits the module list into N batches (default: CPU count) and runs
# `go test -race` for each batch concurrently. Collects results and exits
# non-zero if any batch fails.
#
# Usage: scripts/verify-parallel.sh [extra go test flags]
set -euo pipefail
cd "$(git rev-parse --show-toplevel)"

export CGO_ENABLED=1
TAGS="-tags=goexperiment.jsonv2"
BATCHES="${BATCHES:-$(nproc 2>/dev/null || echo 4)}"
EXTRA_ARGS="$*"

# Read module list from flake.nix (same list as #test).
mapfile -t MODULES < <(find . -maxdepth 3 -name go.mod -not -path './vendor/*' \
  -not -path './.git/*' -not -path './example/*' -not -name 'go.work' \
  -exec dirname {} \; | sed 's|^\./||' | sort -u)

echo "=== Parallel test: ${#MODULES[@]} modules in $BATCHES batches (race detector ON) ==="

FAILED=0
PIDS=()
RESULTS_DIR=$(mktemp -d)
trap 'rm -rf "$RESULTS_DIR"' EXIT

batch_size=$(( (${#MODULES[@]} + BATCHES - 1) / BATCHES ))

batch_idx=0
for ((i=0; i<${#MODULES[@]}; i+=batch_size)); do
  batch=("${MODULES[@]:i:batch_size}")
  batch_idx=$((batch_idx + 1))
  result_file="$RESULTS_DIR/batch_${batch_idx}.log"

  (
    paths=""
    for m in "${batch[@]}"; do
      paths+="./${m}/... "
    done
    if go test $TAGS -race -count=1 $EXTRA_ARGS $paths > "$result_file" 2>&1; then
      echo "✅ Batch $batch_idx passed" >> "$result_file"
    else
      echo "❌ Batch $batch_idx FAILED" >> "$result_file"
    fi
  ) &
  PIDS+=($!)
done

# Wait for all batches
for pid in "${PIDS[@]}"; do
  wait "$pid" || FAILED=1
done

# Print results
for result_file in "$RESULTS_DIR"/batch_*.log; do
  if grep -q "FAILED" "$result_file" 2>/dev/null; then
    cat "$result_file"
  fi
done

if [ "$FAILED" -eq 0 ]; then
  echo "✅ All parallel test batches passed"
else
  echo "❌ One or more test batches failed"
  exit 1
fi
