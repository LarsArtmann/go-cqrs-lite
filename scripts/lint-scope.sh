#!/usr/bin/env bash
# lint-scope.sh — lint ONLY the modules a task window touched.
#
# The full #lint gate walks every testModule (~98 dirs); a session that
# changed three modules needs those three linted NOW, not after the sweep.
# This is the scoped twin: the same per-module golangci-lint invocation and
# the same ✅/❌ summary shape as the #lint app, over the module list in
# GT_LINT_MODULES (space-separated repo-relative module dirs).
#
# Usage:
#   GT_LINT_MODULES="metaengine metaengine/sqliteengine" nix run .#lint-scope
#   scripts/lint-scope.sh --self-test
#
# Env:
#   GT_LINT_MODULES  space-separated module dirs (required unless --self-test;
#                    each must contain a go.mod — a typo'd path fails loud,
#                    never silently skipped)
#   GOLANGCI_BIN     golangci-lint binary (default: golangci-lint from PATH)
#   GOLANGCI_CONFIG  config path (default: <root>/.golangci.yml)
#   LINT_SCOPE_ROOT  repo root (default: $PWD)
#
# Exit codes: 0 all clean; 1 findings, usage error, or unknown module.

set -euo pipefail

info() { printf 'lint-scope: %s\n' "$*"; }

GOLANGCI_BIN="${GOLANGCI_BIN:-golangci-lint}"
GOLANGCI_CONFIG="${GOLANGCI_CONFIG:-}"
# Default the repo root to the git toplevel so the script works from any
# directory inside the repo (the flake app always runs it from the root).
LINT_SCOPE_ROOT="${LINT_SCOPE_ROOT:-$(git rev-parse --show-toplevel 2>/dev/null || printf '%s' "$PWD")}"

run_scope() {
	local root="$1" config="$2"
	shift 2
	local mods=("$@") mod failed=0 failedMods="" total=0

	for mod in "${mods[@]}"; do
		if [ ! -f "$root/$mod/go.mod" ]; then
			echo "lint-scope: unknown module dir '$mod' (no go.mod under $root)" >&2
			echo "usage: GT_LINT_MODULES=\"mod1 mod2\" scripts/lint-scope.sh" >&2
			return 1
		fi
	done

	for mod in "${mods[@]}"; do
		total=$((total + 1))
		echo "==> Linting $mod"
		if (cd "$root/$mod" && "$GOLANGCI_BIN" run --config "$config" ./...); then
			:
		else
			failed=1
			failedMods="$failedMods $mod"
		fi
	done

	if [ "$failed" -eq 0 ]; then
		echo "✅ Lint: $total/$total modules clean"
	else
		echo "❌ Lint: findings in:$failedMods"
	fi

	return "$failed"
}

self_test() {
	local tmp
	tmp=$(mktemp -d)
	trap 'rm -rf "$tmp"' EXIT

	local root="$tmp/repo"
	mkdir -p "$root/clean/mod" "$root/dirty/mod"

	printf 'module example.test/clean\n\ngo 1.27\n' >"$root/clean/go.mod"
	printf 'package mod\n\nfunc Used() int { return 1 }\n' >"$root/clean/mod/mod.go"

	printf 'module example.test/dirty\n\ngo 1.27\n' >"$root/dirty/go.mod"
	printf 'package mod\n\nvar unusedVar = 2\n' >"$root/dirty/mod/mod.go"

	printf 'version: "2"\nlinters:\n  default: none\n  enable:\n    - unused\n' >"$tmp/config.yml"

	export GOLANGCI_LINT_CACHE="$tmp/cache"
	export LINT_SCOPE_ROOT="$root"
	local rc=0 out=""

	info "self-test: clean module passes with the ✅ summary"
	out=$(run_scope "$root" "$tmp/config.yml" clean 2>&1) || rc=$?
	if [ "$rc" -ne 0 ] || ! grep -q '✅ Lint: 1/1 modules clean' <<<"$out"; then
		echo "  FAIL(self-test): clean module must pass with the summary shape, got rc=$rc: $out" >&2
		return 1
	fi
	echo "  PASS: clean module passes"

	info "self-test: dirty module fails with the ❌ summary naming it"
	rc=0
	out=$(run_scope "$root" "$tmp/config.yml" dirty 2>&1) || rc=$?
	if [ "$rc" -eq 0 ] || ! grep -q '❌ Lint: findings in: dirty' <<<"$out"; then
		echo "  FAIL(self-test): dirty module must fail naming itself, got rc=$rc: $out" >&2
		return 1
	fi
	echo "  PASS: dirty module fails naming itself"

	info "self-test: unknown module dir is a loud usage error"
	rc=0
	out=$(run_scope "$root" "$tmp/config.yml" no-such-module 2>&1) || rc=$?
	if [ "$rc" -eq 0 ] || ! grep -q "unknown module dir 'no-such-module'" <<<"$out"; then
		echo "  FAIL(self-test): unknown module must fail loud, got rc=$rc: $out" >&2
		return 1
	fi
	echo "  PASS: unknown module fails loud"

	rm -rf "$tmp"
	trap - EXIT
	info "self-test: all mutations detected"
}

case "${1:-}" in
--self-test)
	if ! command -v "$GOLANGCI_BIN" >/dev/null 2>&1; then
		echo "lint-scope: self-test needs golangci-lint on PATH (or GOLANGCI_BIN)" >&2
		exit 1
	fi
	self_test
	;;
"")
	if [ -z "${GT_LINT_MODULES:-}" ]; then
		echo "lint-scope: GT_LINT_MODULES is empty" >&2
		echo "usage: GT_LINT_MODULES=\"mod1 mod2\" scripts/lint-scope.sh" >&2
		exit 1
	fi
	config="${GOLANGCI_CONFIG:-$LINT_SCOPE_ROOT/.golangci.yml}"
	# shellcheck disable=SC2086 # GT_LINT_MODULES is a deliberate space-separated list
	run_scope "$LINT_SCOPE_ROOT" "$config" ${GT_LINT_MODULES}
	;;
*)
	echo "usage: scripts/lint-scope.sh [--self-test] (modules come from GT_LINT_MODULES)" >&2
	exit 1
	;;
esac
