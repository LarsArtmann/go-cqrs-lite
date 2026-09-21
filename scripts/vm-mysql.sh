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

KEEP_ALIVE=false
if [ "${1:-}" = "--keep-alive" ]; then
	KEEP_ALIVE=true
	shift
fi

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$REPO_ROOT"
# shellcheck disable=SC1091  # dynamic path; syntax-checked separately
source "${SCRIPT_DIR}/lib/shuffle-seed.sh"

HOST_PORT="${MYSQL_VM_PORT:-33070}"
DRIVER_PID=""

# set -m: each background job gets its OWN process group, so $! is the group
# leader and the EXIT trap can group-kill the driver tree (driver + QEMU +
# anything under it). Without this, a crashed test driver orphaned its QEMU,
# which kept holding the hostfwd port and poisoned every subsequent run
# (gotchas-testing.md, 2026-09-19).
set -m

cleanup() {
	if [ -n "$DRIVER_PID" ] && kill -0 "$DRIVER_PID" 2>/dev/null; then
		echo "==> Stopping test driver process group (PGID $DRIVER_PID)"
		kill -TERM -- -"$DRIVER_PID" 2>/dev/null || true
		sleep 2
		kill -KILL -- -"$DRIVER_PID" 2>/dev/null || true
		wait "$DRIVER_PID" 2>/dev/null || true
	fi
}
trap cleanup EXIT INT TERM

# Pre-flight stale-port check: a holder on the hostfwd port (orphaned QEMU
# class) makes every dial in the leg RST or hang; diagnose before booting.
port_held=false
if command -v ss >/dev/null 2>&1; then
	ss -tln 2>/dev/null | grep -q ":${HOST_PORT} " && port_held=true
elif (echo >/dev/tcp/127.0.0.1/"$HOST_PORT") 2>/dev/null; then
	port_held=true
fi
if [ "$port_held" = true ]; then
	echo "ERROR: port ${HOST_PORT} is already held — likely an orphaned QEMU from a crashed run:"
	command -v ss >/dev/null 2>&1 && ss -tlnp 2>/dev/null | grep ":${HOST_PORT} " || true
	pgrep -af qemu 2>/dev/null | sed 's/^/  /' || true
	echo "  Kill the orphan(s) first, then re-run (gotchas-testing.md: leaked-QEMU entry)."
	exit 1
fi

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
cat >"$TEST_SCRIPT" <<'PYEOF'
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

	# Check TCP port connectivity (mysqladmin may not be installed on host)
	if (echo >/dev/tcp/127.0.0.1/"$HOST_PORT") 2>/dev/null; then
		echo "==> MySQL is ready (TCP port $HOST_PORT accepting connections)"
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
		SEED=$(new_shuffle_seed)
		log_shuffle_seed "mysql-manual" "$SEED"
		go test -shuffle="$SEED" "$@" -count=1 -v
	fi
else
	echo "==> Running all MySQL integration tests"
	echo ""
	echo "--- stack/mysql ---"
	SEED=$(new_shuffle_seed)
	log_shuffle_seed "stack/mysql" "$SEED"
	(
		cd stack/mysql
		CGO_ENABLED=1 GOWORK=off \
			go test -shuffle="$SEED" ./... -count=1 -v 2>&1
	)
	echo ""
	echo "--- idempotency/sqlstore ---"
	SEED=$(new_shuffle_seed)
	log_shuffle_seed "idempotency/sqlstore" "$SEED"
	(
		cd idempotency/sqlstore
		# QEMU slirp port-forwarding resets bursts of simultaneous MySQL
		# connections; 10 contenders still prove row-lock serialization.
		# POSTGRES_TEST_DSN keeps the package TestMain (pgtestcontainer) off
		# testcontainers: its unconditional postgres boot (~80s of Docker and
		# ryuk churn) races the slirp forward and resets live MySQL
		# connections; the -run filter means no PG test ever uses this DSN.
		CGO_ENABLED=1 GOWORK=off MYSQL_TEST_CONCURRENCY=10 \
			POSTGRES_TEST_DSN="postgres://mysql-vm-leg@127.0.0.1:1/no-pg-here" \
			go test -tags "integration" -shuffle="$SEED" -run TestIntegration_MySQL ./... -count=1 -v 2>&1
	)
	echo ""
	echo "--- metaengine/mysqlengine (capability conformance + ADT matrix) ---"
	SEED=$(new_shuffle_seed)
	log_shuffle_seed "metaengine/mysqlengine" "$SEED"
	(
		cd metaengine/mysqlengine
		CGO_ENABLED=1 GOWORK=off ADTTEST_CAS_RACERS=10 \
			go test -shuffle="$SEED" ./... -count=1 -v 2>&1
	)
	echo ""
	echo "--- scheduling/sqlstore (MySQL claiming via SKIP LOCKED) ---"
	SEED=$(new_shuffle_seed)
	log_shuffle_seed "scheduling/sqlstore" "$SEED"
	(
		cd scheduling/sqlstore
		CGO_ENABLED=1 GOWORK=off \
			go test -tags "integration" -shuffle="$SEED" -run TestClaimingMySQL ./... -count=1 -v 2>&1
	)
	echo ""
	echo "--- queue/mysql (work-queue conformance + ADR-0142 engine surface) ---"
	SEED=$(new_shuffle_seed)
	log_shuffle_seed "queue/mysql" "$SEED"
	(
		cd queue/mysql
		CGO_ENABLED=1 GOWORK=off ADTTEST_CAS_RACERS=10 \
			go test -shuffle="$SEED" ./... -count=1 -v 2>&1
	)
fi

echo ""
echo "✅ Integration tests passed"

if [ "$KEEP_ALIVE" = true ]; then
	echo ""
	echo "==> --keep-alive: VM is still running on port $HOST_PORT"
	echo "    DSN: $MYSQL_TEST_DSN"
	echo "    Press Ctrl+C to stop the VM and exit."
	wait "$DRIVER_PID"
fi
