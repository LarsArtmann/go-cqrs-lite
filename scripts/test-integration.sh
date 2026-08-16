#!/usr/bin/env bash
# test-integration.sh — one-command integration test runner.
#
# Auto-detects the best available strategy for each database and runs all
# integration tests. Works in any environment: Nix devShell, plain Docker,
# CI with external databases, or bare Linux with KVM.
#
# Strategy priority (fastest first):
#   PostgreSQL: external DSN → ephemeral nixpkgs → testcontainers → QEMU VM
#   MySQL:      external DSN → systemd-nspawn   → testcontainers → QEMU VM
#
# Usage:
#   bash scripts/test-integration.sh                      # auto-detect, all DBs
#   bash scripts/test-integration.sh --pg-only            # PostgreSQL only
#   bash scripts/test-integration.sh --mysql-only         # MySQL only
#   bash scripts/test-integration.sh --strategy=vm        # force QEMU VM
#   bash scripts/test-integration.sh --strategy=testcontainers  # force Docker
#   bash scripts/test-integration.sh --list               # dry-run: show detection
#   bash scripts/test-integration.sh -- -run TestName     # pass-through to go test
#
# Environment overrides (skip auto-detection, use an existing database):
#   POSTGRES_TEST_DSN / DATABASE_URL  — point PG tests at your database
#   MYSQL_TEST_DSN                    — point MySQL tests at your database
#   RESET_DB=0                        — skip the pre-run reset of external
#                                        test databases (default: reset)
set -euo pipefail

# ─── Constants ────────────────────────────────────────────────────────────────

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$REPO_ROOT"

PG_MODULES="storage stack/postgres metaengine/pgengine projectionhost scheduling/sqlstore benchkit"
MYSQL_MODULES="stack/mysql"

# ─── Argument Parsing ─────────────────────────────────────────────────────────

STRATEGY="auto"
RUN_PG=true
RUN_MYSQL=true
PG_ONLY_SET=false
MYSQL_ONLY_SET=false
LIST_ONLY=false
EXTRA_ARGS=()

while [[ $# -gt 0 ]]; do
	case "$1" in
	--pg-only)
		RUN_PG=true
		RUN_MYSQL=false
		PG_ONLY_SET=true
		shift
		;;
	--mysql-only)
		RUN_PG=false
		RUN_MYSQL=true
		MYSQL_ONLY_SET=true
		shift
		;;
	--strategy=auto | --strategy=ephemeral | --strategy=nspawn | --strategy=external | --strategy=testcontainers | --strategy=vm)
		STRATEGY="${1#*=}"
		shift
		;;
	--list)
		LIST_ONLY=true
		shift
		;;
	--)
		shift
		EXTRA_ARGS=("$@")
		break
		;;
	-h | --help)
		sed -n '2,/^set -euo/p' "$0" | sed 's/^# \?//' | sed '/^set -euo/d'
		exit 0
		;;
	*)
		EXTRA_ARGS+=("$1")
		shift
		;;
	esac
done

# ─── Capability Detection ─────────────────────────────────────────────────────

has_nix_pg() {
	command -v initdb &>/dev/null && command -v pg_ctl &>/dev/null
}

has_docker() {
	docker info &>/dev/null 2>&1
}

has_nspawn() {
	[ "$(id -u)" -eq 0 ] && command -v systemd-nspawn &>/dev/null
}

is_linux() {
	[ "$(uname -s)" = "Linux" ]
}

has_kvm() {
	[ -e /dev/kvm ]
}

# ─── Strategy Selection ───────────────────────────────────────────────────────

detect_pg_strategy() {
	if [ "$STRATEGY" != "auto" ]; then
		case "$STRATEGY" in
		external | ephemeral | testcontainers | vm)
			echo "$STRATEGY"
			return
			;;
			# nspawn is MySQL-only; fall through to auto-detect for PG
		esac
	fi
	if [ -n "${POSTGRES_TEST_DSN:-}${DATABASE_URL:-}" ]; then
		echo "external"
		return
	fi
	if has_nix_pg; then
		echo "ephemeral"
		return
	fi
	if has_docker; then
		echo "testcontainers"
		return
	fi
	if is_linux; then
		echo "vm"
		return
	fi
	echo "none"
}

detect_mysql_strategy() {
	if [ "$STRATEGY" != "auto" ]; then
		case "$STRATEGY" in
		external | nspawn | testcontainers | vm)
			echo "$STRATEGY"
			return
			;;
			# ephemeral is PG-only; fall through to auto-detect for MySQL
		esac
	fi
	if [ -n "${MYSQL_TEST_DSN:-}" ]; then
		echo "external"
		return
	fi
	if has_nspawn; then
		echo "nspawn"
		return
	fi
	if has_docker; then
		echo "testcontainers"
		return
	fi
	if is_linux; then
		echo "vm"
		return
	fi
	echo "none"
}

# ─── Execution: PostgreSQL ────────────────────────────────────────────────────

# Reset a shared external test database before a run: drops leftover test_%
# databases from crashed runs and recreates the DSN's default database, so
# repeated runs (and -count>1 loops) start from a clean slate. Opt out with
# RESET_DB=0. Warns and continues when the SQL client is missing or the
# reset fails, so tooling gaps never block a test run.
reset_shared_db() {
	[ "${RESET_DB:-1}" = "1" ] || {
		echo "==> Reset skipped (RESET_DB=0)"
		return 0
	}
	if bash "$SCRIPT_DIR/reset-db.sh" "--$1"; then
		return 0
	fi
	echo "WARNING: could not reset the $1 test database (client missing or reset failed); continuing with existing state"
	return 0
}

run_pg_external() {
	echo "==> PostgreSQL: external DSN ($POSTGRES_TEST_DSN)"
	reset_shared_db pg
	run_pg_modules
}

run_pg_ephemeral() {
	echo "==> PostgreSQL: ephemeral nixpkgs process"
	if [ ${#EXTRA_ARGS[@]} -gt 0 ]; then
		bash "$SCRIPT_DIR/ephemeral-pg.sh" "${EXTRA_ARGS[@]}"
	else
		bash "$SCRIPT_DIR/ephemeral-pg.sh"
	fi
}

run_pg_testcontainers() {
	echo "==> PostgreSQL: Docker testcontainers"
	run_pg_modules
}

run_pg_vm() {
	echo "==> PostgreSQL: QEMU VM"
	if [ ! -e /dev/kvm ]; then
		echo "WARNING: /dev/kvm not found — QEMU will use software emulation (10-50x slower)"
	fi
	if [ ${#EXTRA_ARGS[@]} -gt 0 ]; then
		bash "$SCRIPT_DIR/vm-pg.sh" "${EXTRA_ARGS[@]}"
	else
		bash "$SCRIPT_DIR/vm-pg.sh"
	fi
}

run_pg_modules() {
	local failed=0
	local timeout="${TEST_TIMEOUT:-300}"
	for mod in $PG_MODULES; do
		echo ""
		echo "--- $mod (timeout ${timeout}s) ---"
		(
			cd "$mod"
			CGO_ENABLED=1 GOWORK=off \
				timeout "$timeout" \
				go test -tags "integration goexperiment.jsonv2" ./... \
				-count=1 -v "${EXTRA_ARGS[@]}" 2>&1
		) || failed=1
	done
	if [ "$failed" -ne 0 ]; then
		echo ""
		echo "FAILED: Some PostgreSQL integration tests failed"
		return 1
	fi
}

run_pg() {
	local strategy
	strategy=$(detect_pg_strategy)
	case "$strategy" in
	external) run_pg_external ;;
	ephemeral) run_pg_ephemeral ;;
	testcontainers) run_pg_testcontainers ;;
	vm) run_pg_vm ;;
	none)
		echo "ERROR: No PostgreSQL strategy available."
		echo "  Install Nix with postgresql, or start Docker, or run on Linux."
		return 1
		;;
	esac
}

# ─── Execution: MySQL ─────────────────────────────────────────────────────────

run_mysql_external() {
	echo "==> MySQL: external DSN ($MYSQL_TEST_DSN)"
	reset_shared_db mysql
	run_mysql_modules
}

run_mysql_nspawn() {
	echo "==> MySQL: systemd-nspawn container"
	# vm-mysql-nspawn.sh falls back to vm-mysql.sh automatically if nspawn
	# is not viable at runtime.
	if [ ${#EXTRA_ARGS[@]} -gt 0 ]; then
		bash "$SCRIPT_DIR/vm-mysql-nspawn.sh" "${EXTRA_ARGS[@]}"
	else
		bash "$SCRIPT_DIR/vm-mysql-nspawn.sh"
	fi
}

run_mysql_testcontainers() {
	echo "==> MySQL: Docker testcontainers"
	run_mysql_modules
}

run_mysql_vm() {
	echo "==> MySQL: QEMU VM"
	if [ ! -e /dev/kvm ]; then
		echo "WARNING: /dev/kvm not found — QEMU will use software emulation (10-50x slower)"
	fi
	if [ ${#EXTRA_ARGS[@]} -gt 0 ]; then
		bash "$SCRIPT_DIR/vm-mysql.sh" "${EXTRA_ARGS[@]}"
	else
		bash "$SCRIPT_DIR/vm-mysql.sh"
	fi
}

run_mysql_modules() {
	local failed=0
	local timeout="${TEST_TIMEOUT:-300}"
	for mod in $MYSQL_MODULES; do
		echo ""
		echo "--- $mod (timeout ${timeout}s) ---"
		(
			cd "$mod"
			CGO_ENABLED=1 GOWORK=off \
				timeout "$timeout" \
				go test -tags "goexperiment.jsonv2" ./... \
				-count=1 -v "${EXTRA_ARGS[@]}" 2>&1
		) || failed=1
	done
	if [ "$failed" -ne 0 ]; then
		echo ""
		echo "FAILED: Some MySQL integration tests failed"
		return 1
	fi
}

run_mysql() {
	local strategy
	strategy=$(detect_mysql_strategy)
	case "$strategy" in
	external) run_mysql_external ;;
	nspawn) run_mysql_nspawn ;;
	testcontainers) run_mysql_testcontainers ;;
	vm) run_mysql_vm ;;
	none)
		echo "ERROR: No MySQL strategy available."
		echo "  Run as root with systemd-nspawn, or start Docker, or run on Linux."
		return 1
		;;
	esac
}

# ─── Dry-Run Listing ──────────────────────────────────────────────────────────

show_detection() {
	echo "=== Integration Test Strategy Detection ==="
	echo ""
	echo "Environment:"
	echo "  OS:                 $(uname -s)"
	echo "  Docker available:   $(has_docker && echo yes || echo no)"
	echo "  Nix PostgreSQL:     $(has_nix_pg && echo yes || echo no)"
	echo "  systemd-nspawn:     $(has_nspawn && echo yes || echo no)"
	echo "  KVM (/dev/kvm):     $(has_kvm && echo yes || echo no)"
	echo "  POSTGRES_TEST_DSN:  ${POSTGRES_TEST_DSN:-<not set>}"
	echo "  DATABASE_URL:       ${DATABASE_URL:-<not set>}"
	echo "  MYSQL_TEST_DSN:     ${MYSQL_TEST_DSN:-<not set>}"
	echo "  Forced strategy:    $STRATEGY"
	echo ""
	if [ "$RUN_PG" = true ]; then
		local pg_strategy
		pg_strategy=$(detect_pg_strategy)
		echo "PostgreSQL strategy: $pg_strategy"
		echo "  Modules: $PG_MODULES"
	else
		echo "PostgreSQL: skipped (--mysql-only)"
	fi
	echo ""
	if [ "$RUN_MYSQL" = true ]; then
		local mysql_strategy
		mysql_strategy=$(detect_mysql_strategy)
		echo "MySQL strategy:      $mysql_strategy"
		echo "  Modules: $MYSQL_MODULES"
	else
		echo "MySQL: skipped (--pg-only)"
	fi
	echo ""
}

# ─── Main ─────────────────────────────────────────────────────────────────────

if [ "$LIST_ONLY" = true ]; then
	show_detection
	exit 0
fi

if [ "$PG_ONLY_SET" = true ] && [ "$MYSQL_ONLY_SET" = true ]; then
	echo "ERROR: --pg-only and --mysql-only are mutually exclusive."
	exit 1
fi

if [ "$RUN_PG" = false ] && [ "$RUN_MYSQL" = false ]; then
	echo "ERROR: Nothing to run (both databases disabled)."
	exit 1
fi

show_detection

OVERALL_FAILED=0

if [ "$RUN_PG" = true ]; then
	echo ""
	echo "============================================"
	echo "  PostgreSQL Integration Tests"
	echo "============================================"
	if ! run_pg; then
		OVERALL_FAILED=1
	fi
fi

if [ "$RUN_MYSQL" = true ]; then
	echo ""
	echo "============================================"
	echo "  MySQL Integration Tests"
	echo "============================================"
	if ! run_mysql; then
		OVERALL_FAILED=1
	fi
fi

echo ""
if [ "$OVERALL_FAILED" -ne 0 ]; then
	echo "FAILED: Some integration tests failed"
	exit 1
fi

echo "All integration tests passed"
