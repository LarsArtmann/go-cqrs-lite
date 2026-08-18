#!/usr/bin/env bash
# ephemeral-pg.sh — Start an ephemeral PostgreSQL instance from nixpkgs.
#
# Replaces Docker/testcontainers for local integration testing.
# No Docker daemon, no VM, no network — just a process from the Nix store.
#
# Portability: Linux (NixOS) verified. macOS: static review 2026-08-16 found
# no Linux-isms — nixpkgs provides the same pg_ctl/initdb/createdb binaries,
# mktemp/python3/pgrep\ -u\ -f/pkill\ -f behave identically on Darwin, and the
# /dev/kvm check is uname-guarded. NOT yet exercised on real Mac hardware;
# if it breaks there, check `unix_socket_directories` permissions first.
#
# Usage:
#   nix run .#integration-pg                           # run all PG integration tests
#   nix run .#integration-pg -- -run TestPostgresEventStore_CRUD  # specific test
#   nix run .#integration-pg -- go test ./storage/...  # arbitrary go command
#
# Environment:
#   PG_PORT       — override the port (default: auto-select free port)
#   PGDATA_CACHE  — cache dir for PG data (default: none; fresh mktemp each run).
#                   When set, initdb runs only once; subsequent starts reuse the
#                   existing cluster (~2s saved per run). Example:
#                     export PGDATA_CACHE=/tmp/cqrs-pg-cache
set -euo pipefail

SOCKDIR=""
CACHE_MODE=false

# Use cache dir if provided and valid; otherwise fall back to fresh mktemp.
if [ -n "${PGDATA_CACHE:-}" ]; then
	PGDATA="$PGDATA_CACHE"
	CACHE_MODE=true
	mkdir -p "$PGDATA"
else
	PGDATA=$(mktemp -d /tmp/cqrs-pg-XXXXXX)
fi

cleanup() {
	if [ -f "$PGDATA/postmaster.pid" ]; then
		pg_ctl -D "$PGDATA" -m fast stop -w 2>/dev/null || true
	fi
	if [ "$CACHE_MODE" = false ]; then
		rm -rf "$PGDATA"
	fi
	rm -rf "$SOCKDIR"
	# Verify no orphan postgres processes
	if pgrep -u "$USER" -f "$PGDATA" >/dev/null 2>&1; then
		echo "WARNING: orphan postgres processes detected, killing"
		pkill -f "$PGDATA" 2>/dev/null || true
	fi
}
trap cleanup EXIT INT TERM

# Warn if KVM is not available (affects nix build performance)
if [ ! -e /dev/kvm ] && [ "$(uname)" = "Linux" ]; then
	echo "NOTE: /dev/kvm not found — builds may be slower"
fi

# Pick a free port if not overridden
if [ -z "${PG_PORT:-}" ]; then
	PG_PORT=$(python3 -c 'import socket; s=socket.socket(); s.bind(("",0)); print(s.getsockname()[1]); s.close()' 2>/dev/null ||
		echo "55432")
fi

# Initialize only if the data directory lacks PG_VERSION (cache miss or fresh run).
if [ ! -f "$PGDATA/PG_VERSION" ]; then
	echo "==> Initializing PostgreSQL data dir (port $PG_PORT, data $PGDATA)"
	initdb -D "$PGDATA" -A trust --no-locale --username=cqrs 2>&1 | tail -3
else
	echo "==> Reusing cached PostgreSQL data dir ($PGDATA)"
fi

# NixOS puts /run/postgresql as the default socket dir, which requires root.
# Override to a temp directory.
SOCKDIR=$(mktemp -d /tmp/cqrs-pg-socks-XXXXXX)

echo "==> Starting PostgreSQL"
pg_ctl -D "$PGDATA" \
	-o "-c listen_addresses='127.0.0.1' -c port=$PG_PORT -c log_min_messages=WARNING -c unix_socket_directories='$SOCKDIR'" \
	-l "$PGDATA/pg.log" start -w

# Create the test database
createdb -h 127.0.0.1 -p "$PG_PORT" -U cqrs cqrs_test 2>/dev/null || true

export POSTGRES_TEST_DSN="postgres://cqrs@127.0.0.1:$PG_PORT/cqrs_test?sslmode=disable"
export DATABASE_URL="$POSTGRES_TEST_DSN"

echo "==> PostgreSQL ready: $POSTGRES_TEST_DSN"

# Determine what to run.
# Per-module GOWORK=off is required because the multi-module workspace
# doesn't resolve integration build tags correctly in workspace mode.
PG_MODULES="storage stack/postgres metaengine/pgengine projectionhost scheduling/sqlstore idempotency/sqlstore benchkit"

if [ $# -gt 0 ] && [ "$1" = "go" ]; then
	shift
	echo "==> Running: go $*"
	go "$@"
else
	EXTRA_ARGS=("$@")
	FAILED=0
	TEST_TIMEOUT="${TEST_TIMEOUT:-300}"
	for mod in $PG_MODULES; do
		echo ""
		echo "--- $mod (timeout ${TEST_TIMEOUT}s) ---"
		(
			cd "$mod"
			CGO_ENABLED=1 GOWORK=off \
				timeout "$TEST_TIMEOUT" \
				go test -tags "integration goexperiment.jsonv2" ./... \
				-count=1 -v "${EXTRA_ARGS[@]}" 2>&1
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
