#!/usr/bin/env bash
# check-go-version.sh — loud toolchain + directive-drift gate (W3 Q4 ruling,
# 2026-09-20; drift lock added 2026-09-22 after the THIRD downgrade wave).
#
# Three failure classes, all once-silent:
#   1. Selected toolchain older than the go.work contract (GOTOOLCHAIN trap
#      that produced "0 findings in seconds" and phantom-green runs).
#   2. go.work directive below the dependency floor (deps require >= 1.27.1;
#      a uniform downgrade of every file stays internally consistent and only
#      explodes at build time — struck 3x: waves 4a540b02c et al).
#   3. go-directive drift: any go.mod whose `go` differs from go.work's
#      (single-file downgrade/upgrade; the workspace sweeps are supposed to
#      be lockstep — this makes them mechanically so).
#
# Usage:
#   scripts/check-go-version.sh                # contract from go.work
#   CHECK_GO_VERSION_BIN=/tmp/fake-go scripts/check-go-version.sh
#   scripts/check-go-version.sh --self-test    # stub-binary + planted-fixture suite
#
# Fixture hooks (self-test only): CHECK_GO_VERSION_BIN (a `go` stub that
# echoes GOVERSION output), CHECK_GO_VERSION_CONTRACT (contract override),
# CHECK_GO_VERSION_ROOT (planted repo tree), CHECK_GO_VERSION_FLOOR (floor
# override).
#
# Exit codes: 0 = toolchain satisfies the contract, 1 = violation, 2 = usage.
set -uo pipefail

ROOT="${CHECK_GO_VERSION_ROOT:-$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)}"
GO_BIN="${CHECK_GO_VERSION_BIN:-go}"
CONTRACT_OVERRIDE="${CHECK_GO_VERSION_CONTRACT:-}"
FLOOR_VERSION="${CHECK_GO_VERSION_FLOOR:-1.27.1}"

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

# Every go.mod's `go` directive must equal go.work's (lockstep), and go.work
# must never sit below the dependency floor. Fails loud with the offender list.
check_directive_drift() {
	local work="$1" offender_count=0 offenders mod ver
	if ! version_at_least "$work" "$FLOOR_VERSION"; then
		echo "✗ check-go-version: go.work directive go$work < floor go$FLOOR_VERSION" >&2
		echo "  (published deps require >= go$FLOOR_VERSION — the uniform-downgrade class;" >&2
		echo "   restore the directive, see docs/agents/gowork-modes.md)" >&2
		return 1
	fi
	offenders=""
	shopt -s nullglob globstar
	for mod in "$ROOT"/**/go.mod; do
		[[ "$mod" == */vendor/* ]] && continue
		ver=$(awk '/^go /{print $2; exit}' "$mod" 2>/dev/null)
		if [[ -n "$ver" && "$ver" != "$work" ]]; then
			offenders+="$mod: go$ver"$'\n'
			offender_count=$((offender_count + 1))
		fi
	done
	shopt -u nullglob globstar
	if ((offender_count > 0)); then
		echo "✗ check-go-version: go-directive drift — $offender_count go.mod file(s) differ from go.work's go$work:" >&2
		printf '%s' "$offenders" | sed 's/^/  /' >&2
		echo "  remedy: align the directive (lockstep sweep, go.work last)" >&2
		return 1
	fi
	return 0
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
	if [[ -z "$CONTRACT_OVERRIDE" ]]; then
		check_directive_drift "$CONTRACT" || exit 1
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
	# Planted repo fixture (never a live tracked file — the repo's gate
	# convention): go.work + three module go.mod files.
	fixture() {
		# $1 = go.work version, $2..$4 = module versions
		rm -rf "$TMP/repo"
		mkdir -p "$TMP/repo/a" "$TMP/repo/b" "$TMP/repo/c"
		printf 'go %s\n\nuse (\n\t./a\n\t./b\n\t./c\n)\n' "$1" >"$TMP/repo/go.work"
		printf 'module example.com/a\n\ngo %s\n' "$2" >"$TMP/repo/a/go.mod"
		printf 'module example.com/b\n\ngo %s\n' "$3" >"$TMP/repo/b/go.mod"
		printf 'module example.com/c\n\ngo %s\n' "$4" >"$TMP/repo/c/go.mod"
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

	printf '#!/bin/sh\nexit 0\n' >"$TMP/go-broken"
	chmod +x "$TMP/go-broken"
	rc=0
	CHECK_GO_VERSION_BIN="$TMP/go-broken" CHECK_GO_VERSION_CONTRACT=1.27.1 bash "$SELF" >/dev/null 2>&1 || rc=$?
	check "silent go binary fails loud" 1 "$rc"

	# Drift legs (planted fixture; CONTRACT_OVERRIDE absent so the drift path runs).
	fixture 1.27.1 1.27.1 1.27.1 1.27.1
	rc=0
	CHECK_GO_VERSION_ROOT="$TMP/repo" CHECK_GO_VERSION_BIN="$TMP/go-new" bash "$SELF" >/dev/null 2>&1 || rc=$?
	check "lockstep-clean fixture passes" 0 "$rc"

	fixture 1.27.1 1.27 1.27.1 1.27.1
	out=$(CHECK_GO_VERSION_ROOT="$TMP/repo" CHECK_GO_VERSION_BIN="$TMP/go-new" bash "$SELF" 2>&1)
	check "module below go.work fails" 1 "$?"
	grep -q "go-directive drift" <<<"$out" || {
		echo "  ✗ FAIL: drift message missing"
		fails=$((fails + 1))
	}
	grep -q "a/go.mod: go1.27" <<<"$out" || {
		echo "  ✗ FAIL: offender path missing"
		fails=$((fails + 1))
	}

	fixture 1.27.1 1.27.1 1.28 1.27.1
	out=$(CHECK_GO_VERSION_ROOT="$TMP/repo" CHECK_GO_VERSION_BIN="$TMP/go-new" bash "$SELF" 2>&1)
	check "module above go.work fails" 1 "$?"
	grep -q "b/go.mod: go1.28" <<<"$out" || {
		echo "  ✗ FAIL: above-drift offender missing"
		fails=$((fails + 1))
	}

	fixture 1.27 1.27 1.27 1.27
	out=$(CHECK_GO_VERSION_ROOT="$TMP/repo" CHECK_GO_VERSION_BIN="$TMP/go-new" bash "$SELF" 2>&1)
	check "uniform downgrade below floor fails" 1 "$?"
	grep -q "< floor go1.27.1" <<<"$out" || {
		echo "  ✗ FAIL: floor message missing"
		fails=$((fails + 1))
	}

	# CI=true leg: shared-runner passthrough must NOT weaken enforcement.
	fixture 1.27.1 1.27 1.27.1 1.27.1
	rc=0
	CHECK_GO_VERSION_ROOT="$TMP/repo" CHECK_GO_VERSION_BIN="$TMP/go-new" CI=true bash "$SELF" >/dev/null 2>&1 || rc=$?
	check "CI=true still enforces drift" 1 "$rc"

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
