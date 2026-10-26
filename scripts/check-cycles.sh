#!/usr/bin/env bash
# check-cycles.sh — production import-cycle gate.
#
# WHY: the 2026-10-24 architecture review's dependency-direction score (5/5)
# rested on "0 production cycles", a number that cost a hand-written parser to
# establish and was protected by nothing. An invariant with no gate is a
# memory, not a property. This gate makes it permanent.
#
# WHAT IS CHECKED: the PACKAGE-level production import graph of every workspace
# module — `go list` with no `-test`, so `_test.go` imports are excluded by
# construction. The review's "10 apparent cycles" are exactly the test-only
# edges this excludes (event <-> eventtest, snapshot <-> storage/memory,
# metaengine <-> sqliteengine, ...); they are legal because test packages are
# not part of the build graph of their subject.
#
# SCOPE: edges are counted at package granularity (e.g. `metaengine` ->
# `metaengine/sqliteengine`); module-level edges are a strict subset, so a
# module-level cycle implies a package-level one. Package granularity is the
# stricter, and therefore the honest, invariant.
#
# Usage:
#   bash scripts/check-cycles.sh              # gate: exit 1 on any cycle
#   bash scripts/check-cycles.sh --json       # machine-readable summary
#   bash scripts/check-cycles.sh --self-test  # planted-cycle mutation proof
#
# Env overrides:
#   PKGCYCLES_GRAPH  pre-captured `go list` output (fixtures/self-test/offline).
#                    Shape: "<importpath> <dep> <dep> ..." per line.
set -euo pipefail

REPO_ROOT="$(git rev-parse --show-toplevel)"
cd "$REPO_ROOT"

PREFIX="github.com/larsartmann/go-cqrs-lite/"

# One scratch directory removed on exit. Globals, not function locals: an EXIT
# trap cannot see a local that already went out of scope (the self-test caught
# exactly that as "work: unbound variable").
WORK=""
cleanup() {
	if [ -n "$WORK" ]; then rm -rf "$WORK"; fi
}
trap cleanup EXIT

graph_source() {
	if [ -n "${PKGCYCLES_GRAPH:-}" ]; then
		cat "$PKGCYCLES_GRAPH"
		return
	fi
	# shellcheck disable=SC1091  # the shared env chain lives beside this script
	source "$REPO_ROOT/scripts/go-env.sh"

	local patterns="./..."
	local dir
	local use_dirs
	use_dirs="$(sed -n '/^use (/,/^)/p' go.work | grep -oE '\./[a-zA-Z0-9/_.-]+' || true)"
	while IFS= read -r dir; do
		[ -n "$dir" ] && patterns="$patterns $dir/..."
	done <<<"$use_dirs"

	# `Imports` is the non-test import list; `-e` would hide broken modules, so
	# a resolution failure is a hard error rather than a silent empty graph.
	go list -f '{{.ImportPath}}{{range .Imports}} {{.}}{{end}}' $patterns
}

# edges: "<pkg><TAB><dep>", repo-relative, in-repo only.
edges() {
	graph_source | awk -v prefix="$PREFIX" '
		{
			if (index($1, prefix) != 1) next
			pkg = substr($1, length(prefix) + 1)
			if (pkg == "") next
			for (i = 2; i <= NF; i++) {
				if (index($i, prefix) != 1) next
				dep = substr($i, length(prefix) + 1)
				if (dep == "" || dep == pkg) continue
				seen[pkg SUBSEP dep] = 1
			}
		}
		END {
			for (k in seen) {
				split(k, parts, SUBSEP)
				printf "%s\t%s\n", parts[1], parts[2]
			}
		}
	' | sort
}

run_gate() {
	local json="${1:-}"
	WORK="$(mktemp -d)"
	local edge_file="$WORK/edges"
	edges >"$edge_file"

	if [ ! -s "$edge_file" ]; then
		echo "::error::import graph is empty — refusing to report a green cycle check on no data (go list failed or PKGCYCLES_GRAPH is empty)."
		exit 1
	fi

	local nodes edges_n
	nodes="$({ cut -f1 "$edge_file"; cut -f2 "$edge_file"; } | sort -u | wc -l)"
	edges_n="$(wc -l <"$edge_file")"

	# Iterative-free recursive DFS with three-colour marking. 177 packages /
	# ~500 edges today: recursion depth is bounded by the longest acyclic path,
	# far below any bash stack concern.
	declare -A adj=() color=() cycle_by_key=()
	local pkg dep pair
	while IFS=$'\t' read -r pkg dep; do
		adj["$pkg"]="${adj["$pkg"]:-} $dep"
	done <"$edge_file"

	local -a stack=()

	# A cycle is a SET of packages, not a rotation: discovered from any member
	# it yields a different string. Canonicalise on the sorted member set so a
	# 2-cycle reports once, not twice.
	dfs() {
		local node="$1" next key
		color["$node"]=1
		stack+=("$node")
		for next in ${adj["$node"]:-}; do
			case "${color[$next]:-0}" in
			0) dfs "$next" ;;
			1)
				local i start=-1
				for i in "${!stack[@]}"; do
					if [ "${stack[$i]}" = "$next" ]; then
						start="$i"
						break
					fi
				done
				if [ "$start" -ge 0 ]; then
					local -a members=("${stack[@]:$start}")
					key="$(printf '%s\n' "${members[@]}" | sort | paste -sd, -)"
					if [ -z "${cycle_by_key[$key]:-}" ]; then
						cycle_by_key["$key"]="$(printf '%s>' "${members[@]}")$next"
					fi
				fi
				;;
			esac
		done
		unset 'stack[${#stack[@]}-1]'
		color["$node"]=2
	}

	local node
	while IFS= read -r node; do
		if [ "${color[$node]:-0}" = 0 ]; then
			dfs "$node"
		fi
	done < <({ cut -f1 "$edge_file"; cut -f2 "$edge_file"; } | sort -u)

	local cycles="${#cycle_by_key[@]}"

	if [ "$json" = "--json" ]; then
		printf '{"packages":%d,"edges":%d,"cycles":%d,"cycle_paths":[' "$nodes" "$edges_n" "$cycles"
		local first=1 key
		for key in $(printf '%s\n' "${!cycle_by_key[@]}" | sort); do
			[ "$first" -eq 1 ] || printf ','
			first=0
			printf '"%s"' "${cycle_by_key[$key]}"
		done
		printf ']}\n'
	else
		if [ "$cycles" -gt 0 ]; then
			echo "::error::$cycles production import cycle(s) detected:"
			for key in $(printf '%s\n' "${!cycle_by_key[@]}" | sort); do
				echo "  ${cycle_by_key[$key]}"
			done
			echo "::error::Production cycles are unbuildable by construction in Go; a cycle here means the two packages were merged into one module by accident or a test-only edge leaked into production."
			exit 1
		fi
		echo "✓ import-cycle gate: 0 production cycles ($nodes packages, $edges_n package-level edges; test-only edges excluded by construction)."
	fi

	[ "$cycles" -eq 0 ]
}

self_test() {
	local fixture="scripts/testdata/cycles"
	local rc=0
	WORK="$(mktemp -d)"
	local work="$WORK"

	run_case() { # name expected_rc
		local name="$1" want="$2" got=0
		PKGCYCLES_GRAPH="$work/$name.graph" \
			bash "${BASH_SOURCE[0]}" >"$work/$name.log" 2>&1 || got=$?
		if [ "$got" -ne "$want" ]; then
			echo "FAIL: $name — expected rc=$want, got rc=$got"
			cat "$work/$name.log"
			rc=1
		else
			echo "ok: $name (rc=$got)"
		fi
	}

	cp "$fixture/acyclic.graph" "$work/acyclic.graph"
	cp "$fixture/two-node.graph" "$work/two-node.graph"
	cp "$fixture/three-node.graph" "$work/three-node.graph"
	printf '' >"$work/empty.graph"

	run_case acyclic 0
	run_case two-node 1
	run_case three-node 1
	# An empty graph must never read as green (this was a real bug class in
	# shell gates: the loop body simply never runs).
	run_case empty 1

	if ! grep -q 'production import cycle' "$work/two-node.log"; then
		echo "FAIL: two-node case did not name the cycle"
		cat "$work/two-node.log"
		rc=1
	fi
	if ! grep -q 'refusing to report a green cycle check' "$work/empty.log"; then
		echo "FAIL: empty-graph case did not refuse"
		cat "$work/empty.log"
		rc=1
	fi

	if [ "$rc" -eq 0 ]; then
		echo "✓ import-cycle self-test: 4/4 mutation cases behaved as specified"
	fi
	return "$rc"
}

case "${1:-}" in
--self-test)
	self_test
	;;
--json)
	run_gate --json
	;;
"")
	run_gate
	;;
*)
	echo "unknown argument: $1 (expected --json or --self-test)" >&2
	exit 2
	;;
esac
