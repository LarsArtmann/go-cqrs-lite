#!/usr/bin/env bash
# vm-pg.sh — Start a NixOS QEMU VM with PostgreSQL and run integration tests.
#
# This boots a lightweight NixOS VM (from nixpkgs, pinned by flake.lock) with
# PostgreSQL 16, forwards port 55432→5432, and runs Go integration tests
# against it on the host. No Docker, no testcontainers.
#
# The VM image is built once and cached by Nix — subsequent runs reuse it.
#
# Usage:
#   nix run .#integration-pg-vm                          # run all PG integration tests
#   nix run .#integration-pg-vm -- ./storage/...         # specific package
#   nix run .#integration-pg-vm -- -run TestPostgresEventStore  # specific test
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$REPO_ROOT"

HOST_PORT="${PG_VM_PORT:-55432}"
VM_LOG=$(mktemp /tmp/cqrs-pg-vm-XXXXXX.log)
VM_PID=""
VM_CLEANUP_LOG=true

cleanup() {
    if [ -n "$VM_PID" ] && kill -0 "$VM_PID" 2>/dev/null; then
        echo "==> Stopping VM (PID $VM_PID)"
        kill "$VM_PID" 2>/dev/null || true
        wait "$VM_PID" 2>/dev/null || true
    fi
    if [ "$VM_CLEANUP_LOG" = true ]; then
        rm -f "$VM_LOG"
    fi
}
trap cleanup EXIT INT TERM

echo "==> Building PostgreSQL VM image (cached by Nix)"
VM_PATH=$(nix build .#pg-vm --no-link --print-out-paths 2>&1 | tail -1)
if [ ! -d "$VM_PATH" ]; then
    echo "ERROR: VM build failed: $VM_PATH"
    exit 1
fi

echo "==> Starting NixOS VM (PostgreSQL on host port $HOST_PORT)"
# Headless mode + serial console for debugging + port forwarding
export QEMU_OPTS="-display none -serial file:$VM_LOG"
export QEMU_NET_OPTS="hostfwd=tcp::${HOST_PORT}-:5432"
"$VM_PATH/bin/run-nixos-vm" &
VM_PID=$!

echo "==> Waiting for PostgreSQL to accept connections..."
# Wait up to 90 seconds for the VM to boot and Postgres to start
for i in $(seq 1 90); do
    # Check if QEMU is still running
    if ! kill -0 "$VM_PID" 2>/dev/null; then
        echo "ERROR: VM exited unexpectedly"
        echo "--- VM log (last 30 lines) ---"
        tail -30 "$VM_LOG" 2>/dev/null || echo "(log not available)"
        VM_CLEANUP_LOG=false
        exit 1
    fi

    # Try to connect
    if pg_isready -h 127.0.0.1 -p "$HOST_PORT" -U cqrs -d cqrs_test 2>/dev/null; then
        echo "==> PostgreSQL is ready"
        break
    fi

    if [ "$i" -eq 90 ]; then
        echo "ERROR: PostgreSQL did not become ready within 90s"
        echo "--- VM log (last 30 lines) ---"
        tail -30 "$VM_LOG" 2>/dev/null || echo "(log not available)"
        VM_CLEANUP_LOG=false
        exit 1
    fi

    sleep 1
done

export POSTGRES_TEST_DSN="postgres://cqrs@127.0.0.1:${HOST_PORT}/cqrs_test?sslmode=disable"
export DATABASE_URL="$POSTGRES_TEST_DSN"
echo "==> DSN: $POSTGRES_TEST_DSN"

# Determine what to run
if [ $# -gt 0 ]; then
    if [ "$1" = "go" ]; then
        shift
        echo "==> Running: go $*"
        go "$@"
    elif [[ "$1" == ./...* ]] || [[ "$1" == ./* ]]; then
        echo "==> Running: go test -tags=integration $*"
        go test -tags "integration goexperiment.jsonv2" "$@" -count=1 -v
    else
        echo "==> Running integration tests matching: $*"
        go test -tags "integration goexperiment.jsonv2" \
            ./storage/... ./stack/postgres/... ./metaengine/pgengine/... ./benchkit/... \
            -count=1 -v -run "$*"
    fi
else
    echo "==> Running all PostgreSQL integration tests"
    FAILED=0
    for mod in storage stack/postgres metaengine/pgengine benchkit; do
        echo ""
        echo "--- $mod ---"
        (
            cd "$mod"
            CGO_ENABLED=1 GOWORK=off \
            go test -tags "integration goexperiment.jsonv2" ./... -count=1 -v 2>&1
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
