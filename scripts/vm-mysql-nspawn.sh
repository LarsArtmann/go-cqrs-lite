#!/usr/bin/env bash
# vm-mysql-nspawn.sh — Start a systemd-nspawn container with MySQL/MariaDB and run integration tests.
#
# ~10x faster than the QEMU VM (~15s startup vs ~131s) because nspawn shares
# the host kernel — no full VM boot. The NixOS test driver uses NspawnMachine
# instead of QEMUMachine.
#
# REQUIREMENTS (host must support nspawn):
#   nix.settings = {
#     auto-allocate-uids = true;
#     system-features = [ "uid-range" "nixos-test" "kvm" ];
#     experimental-features = [ "flakes" "nix-command" "cgroups" ];
#   };
#   systemd is required (nspawn needs cgroups).
#   Interactive runs need root (systemd-nspawn requires CAP_SYS_ADMIN).
#
# If nspawn is not available, this script falls back to the QEMU VM script.
#
# Usage:
#   sudo nix run .#integration-mysql-nspawn                    # run all MySQL tests
#   sudo nix run .#integration-mysql-nspawn -- ./stack/mysql/...
#   nix run .#integration-mysql-nspawn -- --check-only          # just verify nspawn works
set -euo pipefail

CHECK_ONLY=false
if [ "${1:-}" = "--check-only" ]; then
    CHECK_ONLY=true
    shift
fi

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$REPO_ROOT"

HOST_PORT="${MYSQL_VM_PORT:-3306}"

# --- Check if nspawn is supported on this host -------------------------------

check_nspawn_support() {
    local errors=()

    # systemd-nspawn binary
    if ! command -v systemd-nspawn &>/dev/null; then
        errors+=("systemd-nspawn binary not found")
    fi

    # uid-range system feature (required by the test driver derivation)
    local sys_features
    sys_features=$(nix show-config 2>/dev/null | grep '^system-features' | sed 's/.*= //' || echo "")
    if ! echo "$sys_features" | grep -qw uid-range; then
        errors+=("uid-range not in system-features (need: nix.settings.system-features = [ \"uid-range\" ... ])")
    fi

    # auto-allocate-uids
    local auto_uids
    auto_uids=$(nix show-config 2>/dev/null | grep '^auto-allocate-uids' | sed 's/.*= //' || echo "")
    if [ "$auto_uids" != "true" ]; then
        errors+=("auto-allocate-uids is not enabled (need: nix.settings.auto-allocate-uids = true)")
    fi

    # Root privileges (interactive nspawn requires CAP_SYS_ADMIN)
    if [ "$(id -u)" -ne 0 ]; then
        errors+=("not running as root (interactive nspawn requires sudo)")
    fi

    if [ ${#errors[@]} -gt 0 ]; then
        echo "ERROR: nspawn not supported on this host:" >&2
        for e in "${errors[@]}"; do
            echo "  - $e" >&2
        done
        echo "" >&2
        echo "To enable nspawn, add to your NixOS configuration:" >&2
        echo "  nix.settings = {" >&2
        echo "    auto-allocate-uids = true;" >&2
        echo '    system-features = [ "uid-range" "nixos-test" "kvm" "benchmark" "big-parallel" ];' >&2
        echo '    experimental-features = [ "flakes" "nix-command" "cgroups" ];' >&2
        echo "  };" >&2
        echo "Then rebuild and run with: sudo nix run .#integration-mysql-nspawn" >&2
        return 1
    fi
    return 0
}

# --- Fall back to QEMU VM script ---------------------------------------------

fallback_to_qemu() {
    echo "==> Falling back to QEMU VM (scripts/vm-mysql.sh)"
    exec bash "$SCRIPT_DIR/vm-mysql.sh" "$@"
}

# --- Check nspawn support, fall back if unavailable -------------------------

if ! check_nspawn_support 2>/dev/null; then
    echo "⚠️  nspawn not available — falling back to QEMU VM"
    fallback_to_qemu "$@"
fi

echo "==> nspawn support detected"

# --- Check-only mode: just build and run the check --------------------------

if [ "$CHECK_ONLY" = true ]; then
    echo "==> Building and running nspawn MySQL check"
    nix build .#checks.x86_64-linux.mysql-nspawn -L
    echo "✅ nspawn MySQL check passed"
    exit 0
fi

# --- Integration test mode: keep container alive, run Go tests --------------

DRIVER_PID=""
CONTAINER_IP=""

cleanup() {
    if [ -n "$DRIVER_PID" ] && kill -0 "$DRIVER_PID" 2>/dev/null; then
        echo "==> Stopping nspawn test driver (PID $DRIVER_PID)"
        kill "$DRIVER_PID" 2>/dev/null || true
        wait "$DRIVER_PID" 2>/dev/null || true
    fi
}
trap cleanup EXIT INT TERM

echo "==> Building nspawn MySQL test driver (cached by Nix)"
DRIVER=$(nix build .#checks.x86_64-linux.mysql-nspawn.driver --no-link --print-out-paths 2>&1 | tail -1)
if [ ! -x "$DRIVER/bin/nixos-test-driver" ]; then
    echo "ERROR: Driver build failed: $DRIVER"
    exit 1
fi

# Custom test script: boot container, wait for MySQL, set up TCP user, keep alive.
# The nspawn container shares the host network namespace (no vlans configured),
# so MySQL binds to the host's 0.0.0.0:3306 — accessible from 127.0.0.1.
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

echo "==> Starting nspawn test driver (MySQL on host port $HOST_PORT)"

# Feed the custom test script. The driver process stays alive (time.sleep).
"$DRIVER/bin/nixos-test-driver" --test-script "$TEST_SCRIPT" &
DRIVER_PID=$!

echo "==> Waiting for MySQL to become ready..."
for i in $(seq 1 60); do
    if ! kill -0 "$DRIVER_PID" 2>/dev/null; then
        echo "ERROR: Driver exited unexpectedly"
        exit 1
    fi

    # Check TCP port connectivity (nspawn container shares host network)
    if (echo > /dev/tcp/127.0.0.1/"$HOST_PORT") 2>/dev/null; then
        echo "==> MySQL is ready (TCP port $HOST_PORT accepting connections)"
        break
    fi

    if [ "$i" -eq 60 ]; then
        echo "ERROR: MySQL did not become ready within 60s"
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
echo "✅ Integration tests passed (nspawn)"
