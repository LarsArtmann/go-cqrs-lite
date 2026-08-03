#!/usr/bin/env bash
# ephemeral-mysql.sh — Start an ephemeral MySQL/MariaDB instance from nixpkgs.
#
# Replaces Docker/testcontainers for local integration testing.
# No Docker daemon, no VM, no network — just a process from the Nix store.
#
# Usage:
#   nix run .#integration-mysql                        # run all MySQL integration tests
#   nix run .#integration-mysql -- -run TestMultidb    # specific test
#   nix run .#integration-mysql -- go test ./stack/mysql/...
#
# Environment:
#   MYSQL_PORT — override the port (default: auto-select free port)
set -euo pipefail

MYSQLDATA=$(mktemp -d /tmp/cqrs-mysql-XXXXXX)
SOCKET_DIR=$(mktemp -d /tmp/cqrs-mysql-sock-XXXXXX)

cleanup() {
    if pgrep -f "datadir=$MYSQLDATA" >/dev/null 2>&1; then
        mysqladmin --socket="$SOCKET_DIR/mysql.sock" shutdown 2>/dev/null || true
        sleep 0.5
        pkill -f "datadir=$MYSQLDATA" 2>/dev/null || true
    fi
    rm -rf "$MYSQLDATA" "$SOCKET_DIR"
}
trap cleanup EXIT INT TERM

# Pick a free port if not overridden
if [ -z "${MYSQL_PORT:-}" ]; then
    MYSQL_PORT=$(python3 -c 'import socket; s=socket.socket(); s.bind(("",0)); print(s.getsockname()[1]); s.close()' 2>/dev/null \
        || echo "33070")
fi

# Detect whether we have MariaDB or MySQL 8.0 binaries
if command -v mariadbd >/dev/null 2>&1; then
    MYSQLD_BIN="mariadbd"
    INSTALL_DB="mariadb-install-db"
    CLIENT="mariadb"
    ADMIN="mysqladmin"
    FLAVOR="MariaDB"
elif command -v mysqld >/dev/null 2>&1; then
    MYSQLD_BIN="mysqld"
    INSTALL_DB=""
    CLIENT="mysql"
    ADMIN="mysqladmin"
    FLAVOR="MySQL"
else
    echo "ERROR: Neither mariadbd nor mysqld found. Add pkgs.mariadb or pkgs.mysql80 to PATH."
    exit 1
fi

echo "==> Initializing ephemeral $FLAVOR (port $MYSQL_PORT, data $MYSQLDATA)"

if [ "$FLAVOR" = "MariaDB" ]; then
    "$INSTALL_DB" \
        --auth-root-authentication=normal \
        --datadir="$MYSQLDATA" \
        --user="$(whoami)" 2>&1 | tail -3
else
    # MySQL 8.0 uses --initialize-insecure
    "$MYSQLD_BIN" --initialize-insecure --datadir="$MYSQLDATA"
fi

echo "==> Starting $FLAVOR"
"$MYSQLD_BIN" \
    --datadir="$MYSQLDATA" \
    --socket="$SOCKET_DIR/mysql.sock" \
    --port="$MYSQL_PORT" \
    --pid-file="$MYSQLDATA/mysql.pid" \
    --skip-networking=false \
    --log-error="$MYSQLDATA/mysql.err" &

MYSQLD_PID=$!

# Wait for MySQL to be ready
echo "==> Waiting for $FLAVOR to accept connections..."
for i in $(seq 1 30); do
    if "$ADMIN" --socket="$SOCKET_DIR/mysql.sock" ping 2>/dev/null | grep -q "alive"; then
        break
    fi
    if [ "$i" -eq 30 ]; then
        echo "ERROR: $FLAVOR did not start within 30s"
        cat "$MYSQLDATA/mysql.err" 2>/dev/null | tail -20
        exit 1
    fi
    sleep 1
done

# Create test database and user
"$CLIENT" --socket="$SOCKET_DIR/mysql.sock" -u root <<SQL
CREATE DATABASE IF NOT EXISTS cqrs_test;
CREATE USER IF NOT EXISTS 'cqrs'@'%' IDENTIFIED BY 'cqrs';
GRANT ALL PRIVILEGES ON *.* TO 'cqrs'@'%' WITH GRANT OPTION;
FLUSH PRIVILEGES;
SQL

export MYSQL_TEST_DSN="cqrs:cqrs@tcp(127.0.0.1:$MYSQL_PORT)/cqrs_test?parseTime=true&multiStatements=true"

echo "==> $FLAVOR ready: $MYSQL_TEST_DSN"

# Per-module GOWORK=off is required because the multi-module workspace
# doesn't resolve integration build tags correctly in workspace mode.
MYSQL_MODULES="stack/mysql"

if [ $# -gt 0 ] && [ "$1" = "go" ]; then
    shift
    echo "==> Running: go $*"
    go "$@"
else
    EXTRA_ARGS="$*"
    FAILED=0
    for mod in $MYSQL_MODULES; do
        echo ""
        echo "--- $mod ---"
        (
            cd "$mod"
            CGO_ENABLED=1 GOWORK=off \
            go test -tags "goexperiment.jsonv2" ./... \
                -count=1 -v $EXTRA_ARGS 2>&1
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
