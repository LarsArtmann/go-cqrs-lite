#!/usr/bin/env bash
# vm-mysql-nspawn.sh — Start a systemd-nspawn container with MySQL/MariaDB and
# run integration tests. ~10x faster than QEMU (~15s vs ~131s) because nspawn
# shares the host kernel — no full VM boot.
#
# HOW IT WORKS:
#   1. Builds the nspawn test DRIVER (no uid-range needed — just a regular build)
#   2. Runs the driver binary with sudo (systemd-nspawn needs root)
#   3. The driver boots the container, starts MySQL, then sleeps
#   4. Go tests run on the host against the container's MySQL
#
# For the CHECK path (nix build .#checks.x86_64-linux.mysql-nspawn), the host
# needs `uid-range` system feature + `auto-allocate-uids`. Run
# scripts/enable-nspawn-support.sh once to enable.
#
# Usage:
#   sudo bash scripts/vm-mysql-nspawn.sh                     # run all MySQL tests
#   sudo bash scripts/vm-mysql-nspawn.sh -- ./stack/mysql/...
#   sudo nix run .#integration-mysql-nspawn                   # same via nix
#
# If nspawn is not available, falls back to the QEMU VM script.
set -euo pipefail

KEEP_ALIVE=false
if [ "${1:-}" = "--keep-alive" ]; then
    KEEP_ALIVE=true
    shift
fi

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$REPO_ROOT"

# The nspawn container gets VLAN 1, IP 192.168.1.1 (first machine).
# The bridge br1 is on the host, so we connect directly — no port forwarding.
CONTAINER_IP="192.168.1.1"
CONTAINER_PORT="${MYSQL_VM_PORT:-3306}"

# --- Check if nspawn is viable -----------------------------------------------

check_nspawn() {
    local errors=()

    # systemd-nspawn binary
    if ! command -v systemd-nspawn &>/dev/null; then
        errors+=("systemd-nspawn binary not found")
    fi

    # Root privileges (systemd-nspawn requires CAP_SYS_ADMIN)
    if [ "$(id -u)" -ne 0 ]; then
        errors+=("not running as root (use: sudo bash scripts/vm-mysql-nspawn.sh)")
    fi

    if [ ${#errors[@]} -gt 0 ]; then
        echo "⚠️  nspawn not available:" >&2
        for e in "${errors[@]}"; do
            echo "   - $e" >&2
        done
        return 1
    fi
    return 0
}

# --- Fall back to QEMU VM script ---------------------------------------------

fallback_to_qemu() {
    echo "==> Falling back to QEMU VM (scripts/vm-mysql.sh)"
    exec bash "$SCRIPT_DIR/vm-mysql.sh" "$@"
}

# --- Main --------------------------------------------------------------------

if ! check_nspawn 2>/dev/null; then
    fallback_to_qemu "$@"
fi

echo "==> nspawn support detected (running as root)"

DRIVER_PID=""

cleanup() {
    if [ -n "$DRIVER_PID" ] && kill -0 "$DRIVER_PID" 2>/dev/null; then
        echo "==> Stopping nspawn test driver (PID $DRIVER_PID)"
        kill "$DRIVER_PID" 2>/dev/null || true
        wait "$DRIVER_PID" 2>/dev/null || true
    fi
    # Clean up any leftover nspawn containers and bridges
    machinectl poweroff machine 2>/dev/null || true
    ip link delete br1 2>/dev/null || true
    ip netns delete nixos-nspawn-machine 2>/dev/null || true
}
trap cleanup EXIT INT TERM

echo "==> Building nspawn MySQL test driver (cached by Nix)"
DRIVER=$(nix build .#checks.x86_64-linux.mysql-nspawn.driver --no-link --print-out-paths 2>&1 | tail -1)
if [ ! -x "$DRIVER/bin/nixos-test-driver" ]; then
    echo "ERROR: Driver build failed: $DRIVER"
    exit 1
fi

# Custom test script: boot container, wait for MySQL, set up TCP user, keep alive.
# The nspawn container gets VLAN 1 → IP 192.168.1.1. MySQL binds to 0.0.0.0:3306
# inside the container, reachable from the host via the bridge.
TEST_SCRIPT=$(mktemp /tmp/cqrs-mysql-nspawn-XXXXXX.py)
cat > "$TEST_SCRIPT" <<'PYEOF'
machine.start()
machine.wait_for_unit("mysql.service")
machine.wait_for_open_port(3306)
machine.succeed("mysql -u root -e \"CREATE USER IF NOT EXISTS 'cqrs'@'%' IDENTIFIED BY 'cqrs'; GRANT ALL PRIVILEGES ON *.* TO 'cqrs'@'%'; FLUSH PRIVILEGES;\"")
print("MYSQL_READY", flush=True)
import time
time.sleep(999999)
PYEOF

echo "==> Starting nspawn container (MySQL at ${CONTAINER_IP}:${CONTAINER_PORT})"

"$DRIVER/bin/nixos-test-driver" --test-script "$TEST_SCRIPT" &
DRIVER_PID=$!

echo "==> Waiting for MySQL to become ready at ${CONTAINER_IP}:${CONTAINER_PORT}..."
for i in $(seq 1 60); do
    if ! kill -0 "$DRIVER_PID" 2>/dev/null; then
        echo "ERROR: Driver exited unexpectedly"
        exit 1
    fi

    # Check TCP connectivity to the container's MySQL via the bridge
    if (echo > /dev/tcp/"$CONTAINER_IP"/"$CONTAINER_PORT") 2>/dev/null; then
        echo "==> MySQL is ready (${CONTAINER_IP}:${CONTAINER_PORT} accepting connections)"
        break
    fi

    if [ "$i" -eq 60 ]; then
        echo "ERROR: MySQL did not become ready within 60s"
        echo "   The nspawn container may still be booting. Try:"
        echo "   sudo nix build .#checks.x86_64-linux.mysql-nspawn -L"
        exit 1
    fi

    sleep 1
done

export MYSQL_TEST_DSN="cqrs:cqrs@tcp(${CONTAINER_IP}:${CONTAINER_PORT})/cqrs_test?parseTime=true&multiStatements=true"
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
echo "✅ Integration tests passed (nspawn)"

if [ "$KEEP_ALIVE" = true ]; then
    echo ""
    echo "==> --keep-alive: container is still running at ${CONTAINER_IP}:${CONTAINER_PORT}"
    echo "    DSN: $MYSQL_TEST_DSN"
    echo "    Press Ctrl+C to stop the container and exit."
    wait "$DRIVER_PID"
fi
