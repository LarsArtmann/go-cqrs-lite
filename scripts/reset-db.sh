#!/usr/bin/env bash
# reset-db.sh — reset shared test databases before an integration test loop.
#
# Integration tests provision per-test databases (test_1, test_2, ...) and
# drop them on cleanup, but crashed or interrupted runs leave them behind,
# and repeated runs against one external DSN accumulate state (a shared-DB
# -count>1 loop polluted state mid-run on 2026-08-15). Run this before each
# loop iteration to start from a clean slate:
#
#   bash scripts/reset-db.sh                  # reset every DSN that is set
#   bash scripts/reset-db.sh --pg             # PostgreSQL only
#   bash scripts/reset-db.sh --mysql          # MySQL/MariaDB only
#   bash scripts/reset-db.sh --dry-run        # show what would be dropped
#
# test-integration.sh calls this automatically for external-DSN runs
# (opt out with RESET_DB=0).
#
# Environment:
#   POSTGRES_TEST_DSN / DATABASE_URL  — PostgreSQL target (needed for --pg)
#   MYSQL_TEST_DSN                    — MySQL/MariaDB target (needed for --mysql)
#
# DANGER: drops the DSN's default database and every test_% database on the
# target server. Only point these variables at throwaway test servers.
set -euo pipefail

SCRIPT_NAME="$(basename "$0")"

DRY_RUN=false
DO_PG=false
DO_MYSQL=false
ANY_SELECT=false

log() { echo "[$SCRIPT_NAME] $*"; }
die() {
	echo "[$SCRIPT_NAME] ERROR: $*" >&2
	exit 1
}

usage() {
	sed -n '2,/^set -euo/p' "$0" | sed 's/^# \?//' | sed '/^set -euo/d'
	exit 0
}

while [[ $# -gt 0 ]]; do
	case "$1" in
	--pg)
		DO_PG=true
		ANY_SELECT=true
		shift
		;;
	--mysql)
		DO_MYSQL=true
		ANY_SELECT=true
		shift
		;;
	--dry-run)
		DRY_RUN=true
		shift
		;;
	-h | --help)
		usage
		;;
	*)
		die "unknown argument: $1 (see --help)"
		;;
	esac
done

# Without an explicit selection, reset every DSN that is set.
if [ "$ANY_SELECT" = false ]; then
	[ -n "${POSTGRES_TEST_DSN:-}${DATABASE_URL:-}" ] && DO_PG=true
	[ -n "${MYSQL_TEST_DSN:-}" ] && DO_MYSQL=true
fi

# ─── PostgreSQL ───────────────────────────────────────────────────────────────

# Maintenance DSN pointing at the "postgres" database, so the target
# database itself can be dropped. Handles URL and keyword/value formats.
pg_maintenance_dsn() {
	local dsn="$1"
	if [[ "$dsn" == *"://"* ]]; then
		local query="" path="$dsn"
		if [[ "$dsn" == *"?"* ]]; then
			path="${dsn%%\?*}"
			query="?${dsn#*\?}"
		fi
		if [[ "$path" == *"/"* ]]; then
			echo "${path%/*}/postgres$query"
		else
			echo "$path/postgres$query"
		fi
	else
		# libpq keyword/value format: the last occurrence of a repeated
		# keyword wins, so appending overrides any earlier dbname=.
		echo "$dsn dbname=postgres"
	fi
}

reset_pg() {
	local dsn="${POSTGRES_TEST_DSN:-${DATABASE_URL:-}}"
	[ -n "$dsn" ] || die "--pg requested but POSTGRES_TEST_DSN/DATABASE_URL is not set"
	command -v psql &>/dev/null || die "psql not found on PATH (run inside: nix develop)"

	local admin
	admin="$(pg_maintenance_dsn "$dsn")"

	local default_db
	default_db="$(psql "$dsn" -tAc "SELECT current_database()")" ||
		die "cannot connect to PostgreSQL DSN"

	log "PostgreSQL: dropping test_% databases and recreating '$default_db'"

	local leftovers
	leftovers="$(psql "$admin" -tAc \
		"SELECT datname FROM pg_database WHERE datname LIKE 'test\\_%' ORDER BY datname")"

	local db
	for db in $leftovers; do
		log "  drop leftover database: $db"
		[ "$DRY_RUN" = true ] ||
			psql "$admin" -qc "DROP DATABASE \"$db\" WITH (FORCE)"
	done

	log "  drop + recreate database: $default_db"
	if [ "$DRY_RUN" = false ]; then
		psql "$admin" -qc "DROP DATABASE \"$default_db\" WITH (FORCE)"
		psql "$admin" -qc "CREATE DATABASE \"$default_db\""
	fi

	log "PostgreSQL reset complete"
}

# ─── MySQL / MariaDB ──────────────────────────────────────────────────────────

# Parses user:pass@tcp(host:port)/dbname into MYSQL_* variables.
parse_mysql_dsn() {
	local dsn="$1"
	[[ "$dsn" == *"@tcp("*")/"* ]] ||
		die "MYSQL_TEST_DSN is not in user:pass@tcp(host:port)/dbname format"

	local userpass="${dsn%%@*}"
	MYSQL_USER="${userpass%%:*}"
	MYSQL_PASS="${userpass#*:}"
	[ "$MYSQL_PASS" = "$userpass" ] && MYSQL_PASS=""

	local rest="${dsn#*@tcp(}"
	local hostport="${rest%%)*}"
	MYSQL_HOST="${hostport%%:*}"
	MYSQL_PORT="${hostport##*:}"
	[ "$MYSQL_PORT" = "$hostport" ] && MYSQL_PORT="3306"

	local after_slash="${rest#*/}"
	MYSQL_DB="${after_slash%%\?*}"
}

mysql_cli() {
	MYSQL_PWD="${MYSQL_PASS:-}" mysql \
		-h "$MYSQL_HOST" -P "$MYSQL_PORT" -u "$MYSQL_USER" "$@"
}

reset_mysql() {
	command -v mysql &>/dev/null || die "mysql client not found on PATH (run inside: nix develop)"
	parse_mysql_dsn "${MYSQL_TEST_DSN:?--mysql requested but MYSQL_TEST_DSN is not set}"

	log "MySQL: dropping test_% schemas and recreating '$MYSQL_DB'"

	local leftovers
	leftovers="$(mysql_cli -N -e \
		"SELECT schema_name FROM information_schema.schemata WHERE schema_name LIKE 'test\\_%' ORDER BY schema_name")" ||
		die "cannot connect to MySQL DSN"

	local schema
	for schema in $leftovers; do
		log "  drop leftover schema: $schema"
		[ "$DRY_RUN" = true ] ||
			mysql_cli -e "DROP DATABASE \`$schema\`"
	done

	log "  drop + recreate schema: $MYSQL_DB"
	if [ "$DRY_RUN" = false ]; then
		mysql_cli -e "DROP DATABASE \`$MYSQL_DB\`"
		mysql_cli -e "CREATE DATABASE \`$MYSQL_DB\`"
	fi

	log "MySQL reset complete"
}

# ─── Main ─────────────────────────────────────────────────────────────────────

[ "$DO_PG" = true ] || [ "$DO_MYSQL" = true ] ||
	die "nothing to do: set POSTGRES_TEST_DSN/DATABASE_URL and/or MYSQL_TEST_DSN"

[ "$DO_PG" = true ] && reset_pg
[ "$DO_MYSQL" = true ] && reset_mysql

[ "$DRY_RUN" = true ] && log "dry run: nothing was dropped"
exit 0
