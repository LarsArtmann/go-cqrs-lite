#!/usr/bin/env bash
# vm-mysql.sh — Start a NixOS QEMU VM with MySQL/MariaDB and run integration tests.
#
# This boots a lightweight NixOS VM (from nixpkgs, pinned by flake.lock) with
# MariaDB (MySQL-compatible), forwards port 33070→3306, and runs Go integration
# tests against it on the host. No Docker, no testcontainers.
#
# The VM image is built once and cached by Nix — subsequent runs reuse it.
#
# Usage:
#   nix run .#integration-mysql-vm                    # run all MySQL integration tests
#   nix run .#integration-mysql-vm -- ./stack/mysql/...
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$REPO_ROOT"

HOST_PORT="${MYSQL_VM_PORT:-33070}"
VM_LOG=$(mktemp /tmp/cqrs-mysql-vm-XXXXXX.log)
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

echo "==> Building MySQL VM image (cached by Nix)"
VM_PATH=$(nix build .#mysql-vm --no-link --print-out-paths 2>&1 | tail -1)
if [ ! -d "$VM_PATH" ]; then
    echo "ERROR: VM build failed: $VM_PATH"
    exit 1
fi

echo "==> Starting NixOS VM (MySQL on host port $HOST_PORT)"
# Headless mode + serial console for debugging + port forwarding
export QEMU_OPTS="-display none -serial file:$VM_LOG"
export QEMU_NET_OPTS="hostfwd=tcp::${HOST_PORT}-:3306"
"$VM_PATH/bin/run-nixos-vm" &
VM_PID=$!

echo "==> Waiting for MySQL to accept connections..."
for i in $(seq 1 120); do
    if ! kill -0 "$VM_PID" 2>/dev/null; then
        echo "ERROR: VM exited unexpectedly"
        echo "--- VM log (last 30 lines) ---"
        tail -30 "$VM_LOG" 2>/dev/null || echo "(log not available)"
        VM_CLEANUP_LOG=false
        exit 1
    fi

    # Check if MySQL responds to ping
    if mysqladmin ping -h 127.0.0.1 -P "$HOST_PORT" -u cqrs --password=cqrs 2>/dev/null | grep -q "alive"; then
        echo "==> MySQL is ready"
        break
    fi

    if [ "$i" -eq 120 ]; then
        echo "ERROR: MySQL did not become ready within 120s"
        echo "--- VM log (last 30 lines) ---"
        tail -30 "$VM_LOG" 2>/dev/null || echo "(log not available)"
        VM_CLEANUP_LOG=false
        exit 1
    fi

    sleep 1
done

export MYSQL_TEST_DSN="cqrs:cqrs@tcp(127.0.0.1:${HOST_PORT})/cqrs_test?parseTime=true&multiStatements=true"
echo "==> DSN: $MYSQL_TEST_DSN"

if [ $# -gt 0 ]; then
    if [ "$1" = "go" ]; then
        shift
        echo "==> Running: go $*"
        go "$@"
    else
        echo "==> Running: go test $*"
        go test -tags "goexperiment.jsonv2" "$@" -count=1 -v
    fi
else
    echo "==> Running all MySQL integration tests"
    echo ""
    echo "--- stack/mysql ---"
    (
        cd stack/mysql
        CGO_ENABLED=1 GOWORK=off \
        go test -tags "goexperiment.jsonv2" ./... -count=1 -v 2>&1
    )
fi

echo ""
echo "✅ Integration tests passed"
