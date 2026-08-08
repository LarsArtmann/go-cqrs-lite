#!/usr/bin/env bash
# profile-pg-strategies.sh — Measure startup time of PG integration strategies.
# Times the ephemeral nixpkgs approach vs Docker testcontainers.
#
# Usage:
#   bash scripts/profile-pg-strategies.sh
#   bash scripts/profile-pg-strategies.sh --ephemeral
#   bash scripts/profile-pg-strategies.sh --testcontainers
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$REPO_ROOT"

RUN_EPHEMERAL=true
RUN_TESTCONTAINERS=true

while [[ $# -gt 0 ]]; do
    case "$1" in
        --ephemeral)      RUN_TESTCONTAINERS=false; shift ;;
        --testcontainers) RUN_EPHEMERAL=false; shift ;;
        *) echo "Unknown arg: $1"; exit 1 ;;
    esac
done

EPHEMERAL_TIME=""
TC_TIME=""

# Use a real test in stack/postgres that connects to the DB. Running it
# forces full DB lifecycle: start → connect → test → stop.
TEST_MODULE="stack/postgres"
TEST_PATTERN="TestNew_ProducesWorkingBundle"

echo "=== PostgreSQL Strategy Profiling ==="
echo "Target: $TEST_MODULE (build + DB start + one test)"
echo ""

# ─── Ephemeral nixpkgs ────────────────────────────────────────────────────────

if [ "$RUN_EPHEMERAL" = true ]; then
    echo "--- Ephemeral nixpkgs ---"
    START=$(date +%s.%N)

    # ephemeral-pg.sh handles the full lifecycle: start PG, run tests, stop PG.
    # We pass test args through the "go" passthrough.
    export PGDATA_CACHE=$(mktemp -d /tmp/cqrs-pg-prof-XXXXXX)
    bash "$SCRIPT_DIR/ephemeral-pg.sh" go test \
        -tags "integration goexperiment.jsonv2" \
        -run "$TEST_PATTERN" -count=1 -v \
        "./$TEST_MODULE/..." 2>&1 | tail -5
    rm -rf "$PGDATA_CACHE"
    unset PGDATA_CACHE

    END=$(date +%s.%N)
    EPHEMERAL_TIME=$(awk "BEGIN{print $END - $START}")
    printf "Ephemeral: %.1fs\n\n" "$EPHEMERAL_TIME"
fi

# ─── Docker testcontainers ────────────────────────────────────────────────────

if [ "$RUN_TESTCONTAINERS" = true ]; then
    if ! docker info &>/dev/null 2>&1; then
        echo "--- Docker testcontainers: SKIPPED (Docker not available) ---"
    else
        echo "--- Docker testcontainers ---"
        START=$(date +%s.%N)

        # testcontainers starts the DB inside the test binary via TestMain.
        (
            cd "$TEST_MODULE"
            CGO_ENABLED=1 GOWORK=off \
                go test -tags "integration goexperiment.jsonv2" \
                -run "$TEST_PATTERN" -count=1 -v ./... 2>&1 | tail -5
        ) || true

        END=$(date +%s.%N)
        TC_TIME=$(awk "BEGIN{print $END - $START}")
        printf "Testcontainers: %.1fs\n\n" "$TC_TIME"
    fi
fi

# ─── Comparison ───────────────────────────────────────────────────────────────

echo "=== Results ==="
if [ -n "$EPHEMERAL_TIME" ] && [ -n "$TC_TIME" ]; then
    SPEEDUP=$(awk "BEGIN{printf \"%.1f\", $TC_TIME / $EPHEMERAL_TIME}")
    echo "| Strategy       | Time     |"
    echo "|----------------|----------|"
    printf "| Ephemeral      | %7.1fs |\n" "$EPHEMERAL_TIME"
    printf "| Testcontainers | %7.1fs |\n" "$TC_TIME"
    echo ""
    echo "Ephemeral is ${SPEEDUP}x faster than testcontainers."
elif [ -n "$EPHEMERAL_TIME" ]; then
    printf "Ephemeral: %.1fs\n" "$EPHEMERAL_TIME"
elif [ -n "$TC_TIME" ]; then
    printf "Testcontainers: %.1fs\n" "$TC_TIME"
fi
