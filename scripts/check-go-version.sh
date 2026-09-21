#!/usr/bin/env bash
# check-go-version.sh — loud toolchain gate (W3 Q4 ruling, 2026-09-20).
#
# The contract is the go directive in go.work (currently 1.27.1). The host
# resolves it via GOTOOLCHAIN=auto (pkgs.go_1_27 lags at 1.27.0 on this host
# — the documented env chain in docs/agents/gowork-modes.md IS the contract
# until nixpkgs ships 1.27.1). This gate fails LOUD when the selected
# toolchain is older than the contract, or when a stale GOTOOLCHAIN=local
# would silently break every build — the silent class that produced
# "0 findings in seconds" and phantom-green runs (gotchas-tooling-build).
#
# Usage:
#   scripts/check-go-version.sh                # contract from go.work
#   CHECK_GO_VERSION_BIN=/tmp/fake-go scripts/check-go-version.sh
#   scripts/check-go-version.sh --self-test    # stub-binary fixture suite
#
# Fixture hooks (self-test only): CHECK_GO_VERSION_BIN (a `go` stub that
# echoes GOVERSION output), CHECK_GO_VERSION_CONTRACT (contract override).
#
# Exit codes: 0 = toolchain satisfies the contract, 1 = violation, 2 = usage.
set -uo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
GO_BIN="${CHECK_GO_VERSION_BIN:-go}"
CONTRACT_OVERRIDE="${CHECK_GO_VERSION_CONTRACT:-}"

contract_version() {
	if [[ -n "$CONTRACT_OVERRIDE" ]]; then
		echo "$CONTRACT_OVERRIDE"
		return
	fi
	awk '/^go /{print $2; exit}' "$ROOT/go.work" 2>/dev/null
}

selected_version() {
	"$GO_BIN" env GOVERSION 2>/dev/null | sed 's/^go//'
}

version_at_least() {
	# $1 = version, $2 = floor; lexicographic dot-segment compare.
	local IFS=.
	local -a a b
	read -ra a <<<"$1"
	read -ra b <<<"$2"
	local i
	for i in 0 1 2; do
		local x="${a[i]:-0}" y="${b[i]:-0}"
		x="${x%%rc*}"
		x="${x%%beta*}"
		y="${y%%rc*}"
		y="${y%%beta*}"
		if ((10#$x > 10#$y)); then return 0; fi
		if ((10#$x < 10#$y)); then return 1; fi
	done
	return 0
}

case "${1:-}" in
"")
	CONTRACT=$(contract_version)
	if [[ -z "$CONTRACT" ]]; then
		echo "✗ check-go-version: no 'go' directive found in $ROOT/go.work" >&2
		exit 1
	fi
	SELECTED=$(selected_version)
	if [[ -z "$SELECTED" ]]; then
		echo "✗ check-go-version: '$GO_BIN env GOVERSION' produced nothing — is a Go toolchain installed and reachable?" >&2
		echo "  remedy: nix develop (provides the pinned toolchain), or export GOTOOLCHAIN=auto with a go >= $CONTRACT on PATH" >&2
		exit 1
	fi
	if ! version_at_least "$SELECTED" "$CONTRACT"; then
		echo "✗ check-go-version: selected toolchain go$SELECTED < contract go$CONTRACT (go.work)" >&2
		echo "  remedy: export GOTOOLCHAIN=auto (the newer toolchain downloads into the module cache)," >&2
		echo "          or run inside nix develop. Explicit nixpkgs pin lands when nixpkgs ships go$CONTRACT." >&2
		exit 1
	fi
	if [[ "${GOTOOLCHAIN:-auto}" == local ]] && ! version_at_least "$SELECTED" "$CONTRACT"; then
		exit 1 # unreachable today; kept symmetric with the local-mode warning below
	fi
	if [[ "${GOTOOLCHAIN:-}" == local ]]; then
		echo "WARN — GOTOOLCHAIN=local with contract go$CONTRACT: builds work only while this go is >= the contract (today: go$SELECTED)" >&2
	fi
	echo "PASS — toolchain go$SELECTED satisfies contract go$CONTRACT (GOTOOLCHAIN=${GOTOOLCHAIN:-auto})"
	exit 0
	;;
--self-test)
	SELF="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/$(basename "${BASH_SOURCE[0]}")"
	TMP="$(mktemp -d)"
	trap 'rm -rf "$TMP"' EXIT
	stub() {
		printf '#!/bin/sh\necho "go%s"\n' "$1" >"$TMP/go-$2"
		chmod +x "$TMP/go-$2"
	}
	fails=0
	check() {
		if [[ "$3" == "$2" ]]; then
			echo "  ✓ PASS: $1"
		else
			echo "  ✗ FAIL: $1 (want rc=$2, got rc=$3)"
			fails=$((fails + 1))
		fi
	}

	stub "1.27.1" new
	stub "1.26.7" old
	stub "1.27.0" lag

	out=$(CHECK_GO_VERSION_BIN="$TMP/go-new" CHECK_GO_VERSION_CONTRACT=1.27.1 bash "$SELF" 2>&1)
	check "selected >= contract passes" 0 "$?"
	grep -q "PASS — toolchain go1.27.1" <<<"$out" || {
		echo "  ✗ FAIL: pass message missing"
		fails=$((fails + 1))
	}

	out=$(CHECK_GO_VERSION_BIN="$TMP/go-old" CHECK_GO_VERSION_CONTRACT=1.27.1 bash "$SELF" 2>&1)
	check "selected < contract fails loud" 1 "$?"
	grep -q "GOTOOLCHAIN=auto" <<<"$out" || {
		echo "  ✗ FAIL: remedy missing"
		fails=$((fails + 1))
	}

	out=$(CHECK_GO_VERSION_BIN="$TMP/go-lag" CHECK_GO_VERSION_CONTRACT=1.27.1 bash "$SELF" 2>&1)
	check "nixpkgs-lag version (1.27.0) fails" 1 "$?"

	printf '#!/bin/sh\necho "go1.27.1"\n' >"$TMP/go-broken"
	printf '#!/bin/sh\nexit 0\n' >"$TMP/go-broken"
	chmod +x "$TMP/go-broken"
	bash "$SELF" >/dev/null 2>&1
	rc=0
	CHECK_GO_VERSION_BIN="$TMP/go-broken" CHECK_GO_VERSION_CONTRACT=1.27.1 bash "$SELF" >/dev/null 2>&1 || rc=$?
	check "silent go binary fails loud" 1 "$rc"

	if ((fails == 0)); then
		echo "check-go-version self-test passed."
		exit 0
	fi
	echo "check-go-version self-test FAILED."
	exit 1
	;;
-h | --help)
	sed -n '2,22p' "${BASH_SOURCE[0]}"
	exit 0
	;;
*)
	echo "check-go-version: unknown flag: $1 (see --help)" >&2
	exit 2
	;;
esac
