#!/usr/bin/env bash
set -euo pipefail

BASELINE="${1:?Usage: $0 <baseline.txt> [count]}
Default count: 5}"
COUNT="${2:-5}"

if ! command -v benchstat &>/dev/null; then
    echo "Installing benchstat..."
    go install golang.org/x/perf/cmd/benchstat@latest
fi

TAGS="-tags=goexperiment.arenas,goexperiment.goroutineleakprofile,goexperiment.runtimesecret,goexperiment.simd"
MODULES="./event/... ./command/... ./query/... ./decider/... ./id/... ./dispatcher/... ./schema/... ./snapshot/... ./memory/... ./catalog/... ./middleware/... ./integration/... ./projection/... ./signing/... ./storage/... ./watermill/... ./listing/... ./pebble/... ./codec/..."

TIMESTAMP=$(date +%Y-%m-%d_%H-%M-%S)
NEWFILE="benchmarks/${TIMESTAMP}_benchstat.txt"
mkdir -p benchmarks

echo "Running benchmarks x${COUNT}..."
echo "Output: ${NEWFILE}"

go test ${TAGS} ${MODULES} \
    -bench=. -benchmem -count="${COUNT}" \
    -timeout=60m -run='^$' \
    | tee "${NEWFILE}"

echo ""
echo "=== Comparison: ${BASELINE} vs ${NEWFILE} ==="
benchstat "${BASELINE}" "${NEWFILE}"
