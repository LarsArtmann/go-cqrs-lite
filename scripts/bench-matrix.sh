#!/usr/bin/env bash
# bench-matrix.sh — runs the full-pipeline benchmark across all valid
# backend × codec × payload-size combinations. Outputs structured JSON
# suitable for benchstat comparison.
#
# Usage:
#   bash scripts/bench-matrix.sh [--quick] [--output results.json]
#
# --quick: run with -benchtime=1x (fast smoke test)
# --output: write JSON results to file (default: stdout)

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$REPO_ROOT"

QUICK=""
OUTPUT=""
BENCHTIME="3x"
COUNT="3"

while [[ $# -gt 0 ]]; do
	case "$1" in
		--quick)
			QUICK="1"
			BENCHTIME="1x"
			COUNT="1"
			shift
			;;
		--output)
			OUTPUT="$2"
			shift 2
			;;
		--benchtime)
			BENCHTIME="$2"
			shift 2
			;;
		--count)
			COUNT="$2"
			shift 2
			;;
		*)
			echo "Unknown flag: $1" >&2
			exit 1
			;;
	esac
done

export GOEXPERIMENT=jsonv2
export CGO_ENABLED=1
TAGS="goexperiment.jsonv2"

echo "=== Benchmark Matrix Runner ==="
echo "Config: benchtime=$BENCHTIME count=$COUNT"
echo ""

# ─── Full Pipeline Backends ───
echo "--- Full Pipeline Backend Comparison ---"
GOEXPERIMENT=jsonv2 CGO_ENABLED=1 go test -tags "$TAGS" \
	-run='^$' \
	-bench='BenchmarkFullPipeline_Backends' \
	-benchtime="$BENCHTIME" -count="$COUNT" -benchmem \
	-timeout 30m \
	./stack/bench/... 2>&1 | tee "${OUTPUT:+/tmp/matrix_pipeline.txt}"

# ─── Durability Tiers ───
echo ""
echo "--- Durability Tier Comparison ---"
GOEXPERIMENT=jsonv2 CGO_ENABLED=1 go test -tags "$TAGS" \
	-run='^$' \
	-bench='BenchmarkDurabilityTiers_SQLite' \
	-benchtime="$BENCHTIME" -count="$COUNT" -benchmem \
	-timeout 30m \
	./stack/bench/... 2>&1 | tee "${OUTPUT:+/tmp/matrix_durability.txt}"

# ─── Codec Pipeline ───
echo ""
echo "--- Codec Pipeline Comparison ---"
GOEXPERIMENT=jsonv2 CGO_ENABLED=1 go test -tags "$TAGS" \
	-run='^$' \
	-bench='BenchmarkCodecPipeline_WriteRead' \
	-benchtime="$BENCHTIME" -count="$COUNT" -benchmem \
	-timeout 30m \
	./stack/bench/... 2>&1 | tee "${OUTPUT:+/tmp/matrix_codec.txt}"

# ─── Batch Size Sweep ───
echo ""
echo "--- Batch Size Sweep ---"
GOEXPERIMENT=jsonv2 CGO_ENABLED=1 go test -tags "$TAGS" \
	-run='^$' \
	-bench='BenchmarkBatchSizeSweep_SQLite' \
	-benchtime="$BENCHTIME" -count="$COUNT" -benchmem \
	-timeout 30m \
	./stack/bench/... 2>&1 | tee "${OUTPUT:+/tmp/matrix_batch.txt}"

# ─── Metaengine Promise ───
echo ""
echo "--- Metaengine Promise Benchmarks ---"
GOEXPERIMENT=jsonv2 CGO_ENABLED=1 go test -tags "$TAGS" \
	-run='^$' \
	-bench='BenchmarkMultiQuery_EventFanOut|BenchmarkMultiQuery_ReadMix|BenchmarkPlanner_PlanLatency' \
	-benchtime="$BENCHTIME" -count="$COUNT" -benchmem \
	-timeout 30m \
	./metaengine/... 2>&1 | tee "${OUTPUT:+/tmp/matrix_metaengine.txt}"

if [[ -n "$OUTPUT" ]]; then
	# Combine all results into a single output file.
	cat /tmp/matrix_pipeline.txt /tmp/matrix_durability.txt \
		/tmp/matrix_codec.txt /tmp/matrix_batch.txt \
		/tmp/matrix_metaengine.txt > "$OUTPUT"
	echo ""
	echo "Combined results written to: $OUTPUT"
fi

echo ""
echo "=== Matrix Complete ==="
