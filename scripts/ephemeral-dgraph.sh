#!/usr/bin/env bash
# ephemeral-dgraph.sh — Start an ephemeral Dgraph (Zero + Alpha) from nixpkgs.
#
# For dgraphengine integration testing with a real Dgraph instance.
# No Docker daemon, no VM — just processes from the Nix store.
#
# NOTE: the `dgraph` binary only exists on PATH inside the Nix app
# (`nix run .#integration-dgraph`). Direct `bash scripts/ephemeral-dgraph.sh`
# fails with "dgraph: command not found" unless dgraph is on your PATH.
# Prefer: nix run .#integration-dgraph
#
# Usage (when dgraph is on PATH, e.g. inside the nix app):
#   bash scripts/ephemeral-dgraph.sh                # run dgraphengine tests
#   bash scripts/ephemeral-dgraph.sh go test ./...  # arbitrary go command
#
# Environment:
#   DGRAPH_ALPHA_GRPC — override Alpha gRPC port (default: auto-select)
#   DGRAPH_ZERO_GRPC  — override Zero gRPC port (default: auto-select)
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$REPO_ROOT"

pick_free_port() {
	python3 -c 'import socket; s=socket.socket(); s.bind(("",0)); print(s.getsockname()[1]); s.close()' 2>/dev/null
}

# Pick free ports if not overridden.
if [ -z "${DGRAPH_ZERO_GRPC:-}" ]; then
	DGRAPH_ZERO_GRPC=$(pick_free_port || echo "15580")
fi
if [ -z "${DGRAPH_ALPHA_GRPC:-}" ]; then
	DGRAPH_ALPHA_GRPC=$(pick_free_port || echo "19080")
fi

# Derive port offsets from defaults (Zero: 5080, Alpha: 9080).
ZERO_OFFSET=$((DGRAPH_ZERO_GRPC - 5080))
ALPHA_OFFSET=$((DGRAPH_ALPHA_GRPC - 9080))

DGRAPH_DIR=$(mktemp -d /tmp/cqrs-dgraph-XXXXXX)
PID_FILE="/tmp/cqrs-dgraph.pid"
ZERO_PID=""
ALPHA_PID=""

# Detect and reap orphaned Dgraph processes from prior sessions.
reap_stale_dgraph() {
	if [ ! -f "$PID_FILE" ]; then
		return
	fi
	local prev_dir prev_pid
	prev_dir=""
	prev_pid=""
	while IFS=':' read -r prev_pid prev_dir; do
		if [ -n "$prev_pid" ] && kill -0 "$prev_pid" 2>/dev/null; then
			echo "==> Found orphaned Dgraph (PID $prev_pid, dir $prev_dir) — reaping"
			kill "$prev_pid" 2>/dev/null || true
			sleep 1
			kill -9 "$prev_pid" 2>/dev/null || true
			rm -rf "$prev_dir" 2>/dev/null || true
		fi
	done <"$PID_FILE"
	rm -f "$PID_FILE"
}
reap_stale_dgraph

cleanup() {
	stop_pid "$ALPHA_PID"
	stop_pid "$ZERO_PID"
	rm -rf "$DGRAPH_DIR"
	rm -f "$PID_FILE"
}

# stop_pid terminates a dgraph process WITHOUT hanging: dgraph ignores
# SIGTERM during drain, so an unconditional `wait` wedges the whole script
# (observed: 4h45m hang after a green test run). Give SIGTERM a bounded
# grace, then SIGKILL; after SIGKILL, `wait` returns immediately.
stop_pid() {
	pid="$1"
	[ -n "$pid" ] && kill -0 "$pid" 2>/dev/null || return 0
	kill "$pid" 2>/dev/null || true
	for _ in $(seq 1 10); do
		kill -0 "$pid" 2>/dev/null || break
		sleep 0.5
	done
	if kill -0 "$pid" 2>/dev/null; then
		kill -9 "$pid" 2>/dev/null || true
	fi
	wait "$pid" 2>/dev/null || true
}
trap cleanup EXIT INT TERM

echo "==> Starting Dgraph Zero (gRPC $DGRAPH_ZERO_GRPC, offset $ZERO_OFFSET)"
dgraph zero \
	--my="localhost:$DGRAPH_ZERO_GRPC" \
	--port_offset="$ZERO_OFFSET" \
	--wal "$DGRAPH_DIR/zw" \
	--logtostderr \
	>"$DGRAPH_DIR/zero.log" 2>&1 &
ZERO_PID=$!

# Wait for Zero to become leader.
for i in $(seq 1 60); do
	if (echo >/dev/tcp/127.0.0.1/"$DGRAPH_ZERO_GRPC") 2>/dev/null; then
		echo "==> Zero ready"
		break
	fi
	if [ "$i" -eq 60 ]; then
		echo "ERROR: Zero did not become ready within 30s"
		cat "$DGRAPH_DIR/zero.log" 2>/dev/null || true
		exit 1
	fi
	sleep 0.5
done

ALPHA_INTERNAL=$((7080 + ALPHA_OFFSET))
echo "==> Starting Dgraph Alpha (gRPC $DGRAPH_ALPHA_GRPC, internal $ALPHA_INTERNAL)"
dgraph alpha \
	--my="localhost:$ALPHA_INTERNAL" \
	--zero="localhost:$DGRAPH_ZERO_GRPC" \
	--port_offset="$ALPHA_OFFSET" \
	--postings "$DGRAPH_DIR/p" \
	--wal "$DGRAPH_DIR/w" \
	--security "whitelist=0.0.0.0/0" \
	--logtostderr \
	>"$DGRAPH_DIR/alpha.log" 2>&1 &
ALPHA_PID=$!

# Wait for Alpha to accept gRPC connections.
for i in $(seq 1 60); do
	if (echo >/dev/tcp/127.0.0.1/"$DGRAPH_ALPHA_GRPC") 2>/dev/null; then
		echo "==> Alpha ready"
		break
	fi
	if [ "$i" -eq 60 ]; then
		echo "ERROR: Alpha did not become ready within 30s"
		cat "$DGRAPH_DIR/alpha.log" 2>/dev/null || true
		cat "$DGRAPH_DIR/zero.log" 2>/dev/null || true
		exit 1
	fi
	sleep 0.5
done

# Wait for Alpha to become fully ready (not just port open — Dgraph needs
# to sync with Zero and load posting lists). Poll the HTTP health endpoint.
ALPHA_HTTP=$((8080 + ALPHA_OFFSET))
# Total health-wait budget in seconds (each poll is 2s HTTP timeout + 0.5s
# sleep, so 120 iterations = 60s). Override for slow cold starts:
#   DGRAPH_HEALTH_TIMEOUT=120 ./scripts/ephemeral-dgraph.sh ...
HEALTH_TIMEOUT="${DGRAPH_HEALTH_TIMEOUT:-60}"
HEALTH_ITERS=$((HEALTH_TIMEOUT * 2))
echo "==> Waiting for Alpha health endpoint (HTTP $ALPHA_HTTP, budget ${HEALTH_TIMEOUT}s)..."
for i in $(seq 1 "$HEALTH_ITERS"); do
	HEALTH=$(python3 -c "
import urllib.request, sys
try:
    r = urllib.request.urlopen('http://127.0.0.1:$ALPHA_HTTP/health', timeout=2)
    print(r.read().decode())
except Exception as e:
    print('ERR:' + str(e))
" 2>/dev/null || echo "ERR:python")
	if echo "$HEALTH" | grep -q '"healthy"'; then
		echo "==> Alpha healthy"
		break
	fi
	if [ "$i" -eq "$HEALTH_ITERS" ]; then
		echo "ERROR: Alpha did not become healthy within ${HEALTH_TIMEOUT}s"
		echo "--- alpha.log (last 30 lines) ---"
		tail -30 "$DGRAPH_DIR/alpha.log" 2>/dev/null || true
		echo "--- zero.log (last 30 lines) ---"
		tail -30 "$DGRAPH_DIR/zero.log" 2>/dev/null || true
		exit 1
	fi
	sleep 0.5
done

export DGRAPH_ADDR="localhost:$DGRAPH_ALPHA_GRPC"
echo "$ALPHA_PID:$DGRAPH_DIR" >"$PID_FILE"
echo "==> DGRAPH_ADDR=$DGRAPH_ADDR"
echo "==> Logs in $DGRAPH_DIR"

# Run the requested command, or default to running dgraphengine tests.
if [ $# -gt 0 ] && [ "$1" = "go" ]; then
	shift
	echo "==> Running: go $*"
	go "$@"
elif [ $# -gt 0 ]; then
	echo "==> Running: $*"
	"$@"
else
	echo "==> Running dgraphengine integration tests"
	TEST_TIMEOUT="${TEST_TIMEOUT:-600}"
	(
		cd metaengine/dgraphengine
		CGO_ENABLED=1 GOWORK=off \
			timeout -k 15 "$TEST_TIMEOUT" \
			go test -tags "goexperiment.jsonv2" ${TEST_ARGS:-} .\
			-count=1 -v -timeout="${TEST_TIMEOUT}s" ${TEST_ARGS2:-} 2>&1
	)
fi
