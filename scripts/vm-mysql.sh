#!/usr/bin/env bash
# vm-mysql.sh — Start a NixOS QEMU VM with MySQL/MariaDB and run integration tests.
#
# Uses the runNixOSTest driver (not eval-config.nix) for reliable service
# management. The driver boots the VM, waits for MySQL to be ready,
# then keeps the VM alive while Go tests run on the host against the
# port-forwarded database.
#
# Usage:
#   nix run .#integration-mysql-vm                    # run all MySQL integration tests
#   nix run .#integration-mysql-vm -- ./stack/mysql/...
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$REPO_ROOT"

HOST_PORT="${MYSQL_VM_PORT:-33070}"
DRIVER_PID=""

cleanup() {
    if [ -n "$DRIVER_PID" ] && kill -0 "$DRIVER_PID" 2>/dev/null; then
        echo "==> Stopping test driver (PID $DRIVER_PID)"
        kill "$DRIVER_PID" 2>/dev/null || true
        wait "$DRIVER_PID" 2>/dev/null || true
    fi
}
trap cleanup EXIT INT TERM

# Warn if KVM is not available (10-50x slowdown without it)
if [ ! -e /dev/kvm ]; then
    echo "WARNING: /dev/kvm not found — QEMU will use software emulation (10-50x slower)"
fi

echo "==> Building MySQL test driver (cached by Nix)"
DRIVER=$(nix build .#checks.x86_64-linux.mysql-vm.driver --no-link --print-out-paths 2>&1 | tail -1)
if [ ! -x "$DRIVER/bin/nixos-test-driver" ]; then
    echo "ERROR: Driver build failed: $DRIVER"
    exit 1
fi

# Custom test script: boot VM, wait for MySQL, set up TCP user, keep alive
TEST_SCRIPT=$(mktemp /tmp/cqrs-mysql-test-XXXXXX.py)
cat > "$TEST_SCRIPT" <<'PYEOF'
machine.start()
machine.wait_for_unit("mysql.service")
machine.succeed("mysql -u root -e \"CREATE USER IF NOT EXISTS 'cqrs'@'%' IDENTIFIED BY 'cqrs'; GRANT ALL PRIVILEGES ON *.* TO 'cqrs'@'%'; FLUSH PRIVILEGES;\"")
print("MYSQL_READY", flush=True)
import time
time.sleep(999999)
PYEOF

echo "==> Starting NixOS test driver (MySQL on host port $HOST_PORT)"
export QEMU_NET_OPTS="hostfwd=tcp::${HOST_PORT}-:3306"

# Feed a custom test script that boots the VM, waits for MySQL, then sleeps forever.
"$DRIVER/bin/nixos-test-driver" --test-script "$TEST_SCRIPT" &
DRIVER_PID=$!

echo "==> Waiting for MySQL to become ready..."
for i in $(seq 1 180); do
    if ! kill -0 "$DRIVER_PID" 2>/dev/null; then
        echo "ERROR: Driver exited unexpectedly"
        exit 1
    fi

    if mysqladmin ping -h 127.0.0.1 -P "$HOST_PORT" -u cqrs --password=cqrs 2>/dev/null | grep -q "alive"; then
        echo "==> MySQL is ready"
        break
    fi

    if [ "$i" -eq 180 ]; then
        echo "ERROR: MySQL did not become ready within 180s"
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
