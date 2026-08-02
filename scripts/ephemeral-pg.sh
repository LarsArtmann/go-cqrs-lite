#!/usr/bin/env bash
# ephemeral-pg.sh — Start an ephemeral PostgreSQL instance from nixpkgs.
#
# Replaces Docker/testcontainers for local integration testing.
# No Docker daemon, no VM, no network — just a process from the Nix store.
#
# Usage:
#   nix run .#integration-pg                           # run all PG integration tests
#   nix run .#integration-pg -- ./storage/... -run TestPostgresEventStore  # specific test
#   nix run .#integration-pg -- go test -tags=integration -v ./storage/...
#
# Environment:
#   PG_PORT    — override the port (default: auto-select free port)
#   PG_VERSION — override the postgresql package (default: whatever nixpkgs provides)
set -euo pipefail

PGDATA=$(mktemp -d /tmp/cqrs-pg-XXXXXX)

cleanup() {
    if [ -f "$PGDATA/postmaster.pid" ]; then
        pg_ctl -D "$PGDATA" -m fast stop -w 2>/dev/null || true
    fi
    rm -rf "$PGDATA"
}
trap cleanup EXIT INT TERM

# Pick a free port if not overridden
if [ -z "${PG_PORT:-}" ]; then
    PG_PORT=$(python3 -c 'import socket; s=socket.socket(); s.bind(("",0)); print(s.getsockname()[1]); s.close()' 2>/dev/null \
        || echo "55432")
fi

echo "==> Initializing ephemeral PostgreSQL (port $PG_PORT, data $PGDATA)"
initdb -D "$PGDATA" -A trust --no-locale --username=cqrs 2>&1 | tail -3

echo "==> Starting PostgreSQL"
pg_ctl -D "$PGDATA" \
    -o "-c listen_addresses='127.0.0.1' -c port=$PG_PORT -c log_min_messages=WARNING" \
    -l "$PGDATA/pg.log" start -w

# Create the test database
createdb -h 127.0.0.1 -p "$PG_PORT" -U cqrs cqrs_test 2>/dev/null || true

export POSTGRES_TEST_DSN="postgres://cqrs@127.0.0.1:$PG_PORT/cqrs_test?sslmode=disable"
export DATABASE_URL="$POSTGRES_TEST_DSN"

echo "==> PostgreSQL ready: $POSTGRES_TEST_DSN"

# Determine what to run
if [ $# -gt 0 ]; then
    # If args start with "go", assume the caller wants to run go directly
    if [ "$1" = "go" ]; then
        shift
        echo "==> Running: go $*"
        go "$@"
    elif [[ "$1" == ./...* ]] || [[ "$1" == ./* ]]; then
        # Package paths passed directly
        echo "==> Running: go test -tags=integration $*"
        go test -tags "integration goexperiment.jsonv2" "$@" -count=1 -v
    else
        # Assume it's a test pattern or flag
        echo "==> Running integration tests: $*"
        go test -tags "integration goexperiment.jsonv2" ./storage/... ./stack/postgres/... ./metaengine/pgengine/... ./benchkit/... -count=1 -v -run "$*"
    fi
else
    # Default: run all Postgres integration tests across all modules
    echo "==> Running all PostgreSQL integration tests"

    FAILED=0

    for mod in storage stack/postgres metaengine/pgengine benchkit; do
        echo ""
        echo "--- $mod ---"
        (
            cd "$mod"
            CGO_ENABLED=1 GOWORK=off go test -tags "integration goexperiment.jsonv2" ./... -count=1 -v 2>&1
        ) || FAILED=1
    done

    if [ "$FAILED" -ne 0 ]; then
        echo ""
        echo "❌ Some integration tests failed"
        exit 1
    fi
fi

echo ""
echo "✅ Integration tests passed"
