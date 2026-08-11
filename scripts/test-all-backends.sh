#!/usr/bin/env bash
# test-all-backends.sh — Run tests across ALL storage backends in one command.
#
# Covers: SQLite, Pebble, bbolt (embedded), DuckDB (CGo), PostgreSQL + MySQL
# (external, via test-integration.sh auto-detection), Dgraph (graph DB).
#
# Usage:
#   bash scripts/test-all-backends.sh                # all backends
#   bash scripts/test-all-backends.sh --embedded-only  # SQLite + Pebble + bbolt only
#   bash scripts/test-all-backends.sh --external-only  # PG + MySQL only
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$REPO_ROOT"

RUN_EMBEDDED=true
RUN_EXTERNAL=true

while [[ $# -gt 0 ]]; do
	case "$1" in
	--embedded-only)
		RUN_EXTERNAL=false
		shift
		;;
	--external-only)
		RUN_EMBEDDED=false
		shift
		;;
	*)
		echo "Unknown arg: $1"
		exit 1
		;;
	esac
done

EMBEDDED_MODULES=(
	"storage"
	"storage/pebble"
	"storage/bbolt"
	"stack/sqlite"
	"stack/pebble"
	"stack/bbolt"
)

CGO_MODULES=(
	"stack/duckdb"
)

OVERALL_FAILED=0

run_module() {
	local mod="$1"
	local tags="$2"
	local timeout="${TEST_TIMEOUT:-300}"
	echo ""
	echo "--- $mod (timeout ${timeout}s) ---"
	(
		cd "$mod"
		CGO_ENABLED=1 GOWORK=off \
			timeout "$timeout" \
			go test -tags "$tags" ./... \
			-count=1 2>&1
	) || return 1
}

run_dgraph() {
	local timeout="${TEST_TIMEOUT:-300}"
	echo ""
	echo "--- metaengine/dgraphengine (timeout ${timeout}s) ---"
	bash "$SCRIPT_DIR/ephemeral-dgraph.sh" \
		bash -c "cd metaengine/dgraphengine && \
            CGO_ENABLED=1 GOWORK=off \
            timeout '$timeout' \
            go test -tags 'goexperiment.jsonv2' ./... \
            -count=1 2>&1" || return 1
}

echo "============================================"
echo "  Cross-Backend Test Suite"
echo "============================================"

if [ "$RUN_EMBEDDED" = true ]; then
	echo ""
	echo ">>> Phase 1: Embedded backends (SQLite, Pebble, bbolt)"
	for mod in "${EMBEDDED_MODULES[@]}"; do
		if ! run_module "$mod" "goexperiment.jsonv2"; then
			OVERALL_FAILED=1
		fi
	done

	echo ""
	echo ">>> Phase 2: CGo backends (DuckDB)"
	for mod in "${CGO_MODULES[@]}"; do
		if ! run_module "$mod" "cgo goexperiment.jsonv2"; then
			OVERALL_FAILED=1
		fi
	done
fi

if [ "$RUN_EXTERNAL" = true ]; then
	echo ""
	echo ">>> Phase 3: External backends (PostgreSQL, MySQL)"
	if ! bash "$SCRIPT_DIR/test-integration.sh"; then
		OVERALL_FAILED=1
	fi

	echo ""
	echo ">>> Phase 4: External graph backend (Dgraph)"
	if ! run_dgraph; then
		OVERALL_FAILED=1
	fi
fi

echo ""
echo "============================================"
if [ "$OVERALL_FAILED" -ne 0 ]; then
	echo "  FAILED: Some backend tests failed"
	echo "============================================"
	exit 1
fi

echo "  All backend tests passed"
echo "============================================"
