#!/usr/bin/env bash
# measure-engine-fleet.sh — re-run ADR-0157's engine-fleet size measurements.
#
# Builds two throwaway blank-import probe modules in a temp dir and reports
# binary size, unique module count, and `go mod graph` edge count:
#   1. sqlite-only  — the minimal durable deployment candidate
#   2. all-10       — core metaengine (memory) + the 9 pure-Go engine modules
#                     (duckdb excluded: CGo; bigtable excluded: heavy GCP)
#
# Reference numbers (2026-10-10, go1.27.1, linux/amd64, proxy-resolved tagged
# versions — ADR-0157 D1):
#   sqlite-only  12,747,630 B |  64 unique modules | 168 edges
#   all-10      68,185,216 B | 221 unique modules | 871 edges
# Sizes drift with dependency updates; the RATIO is the decision-relevant
# signal. Informational tool — no gate semantics.
#
# Modes:
#   default   resolve from the module proxy (tagged releases — comparable to
#             the ADR-0157 methodology; needs network)
#   --tree    `go mod edit -replace` every go-cqrs-lite module to this
#             working tree (measures untagged local state)
set -euo pipefail

REPO_ROOT=$(cd "$(dirname "$0")/.." && pwd)
# shellcheck source=scripts/go-env.sh
source "$REPO_ROOT/scripts/go-env.sh"
mkdir -p "${TMPDIR:-/tmp}"

MODE="proxy"
[ "${1:-}" = "--tree" ] && MODE="tree"

ENGINES=(
	metaengine
	metaengine/sqliteengine
	metaengine/tursoengine
	metaengine/pgengine
	metaengine/mysqlengine
	metaengine/badgerengine
	metaengine/bboltengine
	metaengine/pebbleengine
	metaengine/dgraphengine
	metaengine/irohengine
)

imports_for() { # comma-joined subset name list -> import block body
	for path in "$@"; do
		printf '\t_ "github.com/larsartmann/go-cqrs-lite/%s/v4"\n' "$path"
	done
}

run_probe() { # <slug> <display-name> <import-paths...>
	local slug="$1" name="$2"
	shift 2
	local dir
	dir=$(mktemp -d "${TMPDIR:-/tmp}/fleet-probe-XXXXXX") || {
		echo "mktemp failed" >&2
		exit 1
	}
	trap 'rm -rf "$dir"' EXIT

	{
		printf 'package main\n\nimport (\n'
		imports_for "$@"
		printf ')\n\nfunc main() {}\n'
	} >"$dir/main.go"

	(
		cd "$dir"
		go mod init "fleetprobe/$slug" >/dev/null
		if [ "$MODE" = "tree" ]; then
			while IFS= read -r moddir; do
				[ "$moddir" = "." ] && continue
				local modpath
				modpath=$(grep -m1 '^module ' "$moddir/go.mod" | sed 's/^module //')
				go mod edit -replace "$modpath=$(realpath "$REPO_ROOT/$moddir")"
			done < <(cd "$REPO_ROOT" && find . -name go.mod -not -path './vendor/*' -printf '%h\n')
		fi
		go mod tidy >/dev/null 2>&1
		go build -o probe .
		local size modules edges
		size=$(stat -c%s probe)
		modules=$(go list -m all | tail -n +2 | wc -l | tr -d ' ')
		edges=$(go mod graph | wc -l | tr -d ' ')
		printf '| %s | %s | %s | %s |\n' "$name" "$size B" "$modules" "$edges"
	)
	rm -rf "$dir"
	trap - EXIT
}

echo "mode: $MODE (ADR-0157 reference: sqlite-only 12,747,630 B / 64 / 168; all-10 68,185,216 B / 221 / 871)"
echo "| probe | binary size | unique modules | go mod graph edges |"
echo "| --- | --- | --- | --- |"
run_probe "sqlite-only" "sqlite engine only" metaengine/sqliteengine
run_probe "all-10" "all 10 pure-Go engines" "${ENGINES[@]}"
