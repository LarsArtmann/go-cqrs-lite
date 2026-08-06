#!/usr/bin/env bash
#
# bench-all.sh — Run benchmarks across EVERY module in the go-cqrs-lite workspace.
#
# No cherry-picking. No curated groups. Every module with a func Benchmark gets run.
# Reports: total benchmarks, ran, skipped, failed, wall time.
#
# Usage:
#   scripts/bench-all.sh              # run all benchmarks
#   scripts/bench-all.sh --quick      # skip slow modules (duckdb, integration)
#   scripts/bench-all.sh --module event  # run a single module
#
# Environment:
#   GOEXPERIMENT=jsonv2   (required — set automatically)
#   CGO_ENABLED=1         (required for DuckDB/SQLite-CGo — set automatically)
#   MYSQL_BENCH_DSN       (optional — enables MySQL benchmarks)
#   POSTGRES_BENCH_DSN    (optional — enables Postgres benchmarks)
#   BENCH_TIMEOUT         (default: 20m per module group)
#
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$ROOT_DIR"

export GOEXPERIMENT=jsonv2
export CGO_ENABLED=1
export GOFLAGS="-tags=goexperiment.jsonv2"

TIMEOUT="${BENCH_TIMEOUT:-20m}"
QUICK=false
MODULE_FILTER=""

while [[ $# -gt 0 ]]; do
    case "$1" in
        --quick) QUICK=true; shift ;;
        --module) MODULE_FILTER="$2"; shift 2 ;;
        *) echo "Unknown option: $1"; exit 1 ;;
    esac
done

# Modules that have benchmark files (auto-discovered).
# We search for func Benchmark in _test.go files.
mapfile -t MODULES < <(
    find . -name '*_test.go' -not -path './vendor/*' -not -path './.git/*' \
        -exec grep -l 'func Benchmark' {} \; \
    | xargs -I{} dirname {} \
    | sort -u
)

# Slow modules skipped in --quick mode
SLOW_MODULES="./metaengine/duckdbengine/... ./metaengine/pgengine/... ./integration/... ./stack/bench/..."

echo "============================================"
echo "  bench-all.sh — Full Benchmark Suite"
echo "============================================"
echo "  Date:     $(date '+%Y-%m-%d %H:%M:%S')"
echo "  Go:       $(go version)"
echo "  CGO:      $CGO_ENABLED"
echo "  Timeout:  $TIMEOUT per module"
echo "  Quick:    $QUICK"
echo "  Modules:  ${#MODULES[@]} with benchmarks"
echo "============================================"
echo ""

TOTAL_RAN=0
TOTAL_PASSED=0
TOTAL_FAILED=0
TOTAL_SKIPPED=0
FAILED_MODULES=()
START_TIME=$SECONDS

for mod in "${MODULES[@]}"; do
    mod_name="${mod#./}"

    # Apply module filter
    if [[ -n "$MODULE_FILTER" && "$mod_name" != *"$MODULE_FILTER"* ]]; then
        continue
    fi

    # Quick mode: skip slow modules
    if $QUICK; then
        skip=false
        for slow in $SLOW_MODULES; do
            if [[ "$mod" == *$(echo "$slow" | sed 's|/\.\.\.||')* ]]; then
                skip=true
                break
            fi
        done
        if $skip; then
            echo "  SKIP   $mod_name (slow module, --quick)"
            ((TOTAL_SKIPPED++)) || true
            continue
        fi
    fi

    echo "  RUN    $mod_name"

    # Count benchmarks in this module
    bench_count=$(
        grep -rh 'func Benchmark' "$mod"/*_test.go 2>/dev/null \
        | grep -c 'func Benchmark' || true
    )

    output=$(timeout "$TIMEOUT" go test \
        -tags "goexperiment.jsonv2" \
        -run='^$' \
        -bench=. \
        -benchmem \
        -count=1 \
        -timeout "$TIMEOUT" \
        "./${mod_name}/..." 2>&1) || true

    # Check for failures
    if echo "$output" | grep -qE '(FAIL|panic:)'; then
        echo "  FAIL   $mod_name ($bench_count benchmarks)"
        echo "$output" | grep -E '(FAIL|panic:|--- FAIL)' | head -5 | sed 's/^/         /'
        ((TOTAL_FAILED++)) || true
        FAILED_MODULES+=("$mod_name")
    elif echo "$output" | grep -qE '(--- SKIP|no benchmarks|no Go files)'; then
        echo "  SKIP   $mod_name (skipped/no-output)"
        ((TOTAL_SKIPPED++)) || true
    elif echo "$output" | grep -qE '^ok'; then
        duration=$(echo "$output" | grep '^ok' | sed 's/.*\t//' || echo "?")
        echo "  PASS   $mod_name ($bench_count benchmarks, $duration)"
        ((TOTAL_PASSED++)) || true
    else
        echo "  ???    $mod_name (unexpected output)"
        ((TOTAL_FAILED++)) || true
        FAILED_MODULES+=("$mod_name")
    fi
    ((TOTAL_RAN++)) || true
    echo ""
done

ELAPSED=$((SECONDS - START_TIME))
ELAPSED_MIN=$((ELAPSED / 60))
ELAPSED_SEC=$((ELAPSED % 60))

echo ""
echo "============================================"
echo "  RESULTS"
echo "============================================"
echo "  Modules run:     $TOTAL_RAN"
echo "  Modules passed:  $TOTAL_PASSED"
echo "  Modules failed:  $TOTAL_FAILED"
echo "  Modules skipped: $TOTAL_SKIPPED"
echo "  Wall time:       ${ELAPSED_MIN}m ${ELAPSED_SEC}s"
if [[ ${#FAILED_MODULES[@]} -gt 0 ]]; then
    echo ""
    echo "  FAILED MODULES:"
    for m in "${FAILED_MODULES[@]}"; do
        echo "    - $m"
    done
fi
echo "============================================"

if [[ $TOTAL_FAILED -gt 0 ]]; then
    exit 1
fi
