#!/usr/bin/env bash
# vm-pg.sh — Start a NixOS QEMU VM with PostgreSQL and run integration tests.
#
# Uses the runNixOSTest driver (not eval-config.nix) for reliable service
# management. The driver boots the VM, waits for PostgreSQL to be ready,
# then keeps the VM alive while Go tests run on the host against the
# port-forwarded database.
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
DRIVER_PID=""
DRIVER_LOG=$(mktemp /tmp/cqrs-pg-driver-XXXXXX.log)

cleanup() {
    if [ -n "$DRIVER_PID" ] && kill -0 "$DRIVER_PID" 2>/dev/null; then
        echo "==> Stopping test driver (PID $DRIVER_PID)"
        kill "$DRIVER_PID" 2>/dev/null || true
        wait "$DRIVER_PID" 2>/dev/null || true
    fi
    rm -f "$DRIVER_LOG"
}
trap cleanup EXIT INT TERM

echo "==> Building PostgreSQL test driver (cached by Nix)"
DRIVER=$(nix build .#checks.x86_64-linux.postgres-vm.driver --no-link --print-out-paths 2>&1 | tail -1)
if [ ! -x "$DRIVER/bin/nixos-test-driver" ]; then
    echo "ERROR: Driver build failed: $DRIVER"
    exit 1
fi

# Custom test script: boot VM, wait for PG, keep alive for external tests
TEST_SCRIPT=$(mktemp /tmp/cqrs-pg-test-XXXXXX.py)
cat > "$TEST_SCRIPT" <<'PYEOF'
machine.start()
machine.wait_for_unit("postgresql.service")
print("POSTGRESQL_READY", flush=True)
import time
time.sleep(999999)
PYEOF

echo "==> Starting NixOS test driver (PostgreSQL on host port $HOST_PORT)"
export QEMU_NET_OPTS="hostfwd=tcp::${HOST_PORT}-:5432"

# Feed a custom test script that boots the VM, waits for PG, then sleeps forever.
# The driver handles VM lifecycle, port forwarding, and service readiness.
"$DRIVER/bin/nixos-test-driver" --test-script "$TEST_SCRIPT" &
DRIVER_PID=$!

echo "==> Waiting for PostgreSQL to become ready..."
for i in $(seq 1 120); do
    if ! kill -0 "$DRIVER_PID" 2>/dev/null; then
        echo "ERROR: Driver exited unexpectedly"
        cat "$DRIVER_LOG" 2>/dev/null | tail -30
        exit 1
    fi

    if pg_isready -h 127.0.0.1 -p "$HOST_PORT" -U cqrs -d cqrs_test 2>/dev/null; then
        echo "==> PostgreSQL is ready"
        break
    fi

    if [ "$i" -eq 120 ]; then
        echo "ERROR: PostgreSQL did not become ready within 120s"
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
