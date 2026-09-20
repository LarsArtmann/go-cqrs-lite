#!/usr/bin/env bash
# check-golangci-hash.sh — content hash-golden tripwire for .golangci.yml.
#
# The config-corruption war (10+ incidents through 2026-09-20, ~45 min each:
# gci resurrected, depguard allow-list truncated, formatters re-enabled) is
# caught here by sha256 content comparison against a pinned golden, independent
# of shape checks (check-formatters/restore-depguard catch known-bad shapes;
# this catches ANY unreviewed byte, including reorders and whitespace edits).
#
# Usage:
#   scripts/check-golangci-hash.sh              # compare; exit 1 on mismatch
#   scripts/check-golangci-hash.sh --update     # re-pin the golden (intentional
#                                               # config change — commit BOTH
#                                               # .golangci.yml and the golden)
#   scripts/check-golangci-hash.sh --self-test  # fault-injection suite over
#                                               # planted temp fixtures
#
# Fixture hooks (self-test only, do not set in normal use):
#   GOLANGCI_HASH_CONFIG  config path   (default: <repo>/.golangci.yml)
#   GOLANGCI_HASH_GOLDEN  golden path   (default: <repo>/scripts/golangci-config-hash.golden.txt)
#
# Exit codes: 0 = match, 1 = mismatch/missing, 2 = usage.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CONFIG_FILE="${GOLANGCI_HASH_CONFIG:-$ROOT/.golangci.yml}"
GOLDEN_FILE="${GOLANGCI_HASH_GOLDEN:-$ROOT/scripts/golangci-config-hash.golden.txt}"

usage() {
	sed -n '2,20p' "${BASH_SOURCE[0]}"
	exit 2
}

hash_of() {
	sha256sum "$1" | cut -d' ' -f1
}

case "${1:-}" in
"")
	current=$(hash_of "$CONFIG_FILE")
	if [[ ! -f "$GOLDEN_FILE" ]]; then
		echo "✗ hash golden MISSING: $GOLDEN_FILE"
		echo "  Pin it (after eyeballing .golangci.yml): scripts/check-golangci-hash.sh --update"
		exit 1
	fi
	expected=$(cut -d' ' -f1 "$GOLDEN_FILE")
	if [[ "$current" == "$expected" ]]; then
		echo "PASS — .golangci.yml matches hash golden (${current:0:12}…)"
		exit 0
	fi
	echo "✗ .golangci.yml content hash MISMATCH (corruption tripwire)"
	echo "    expected sha256:${expected:0:16}…"
	echo "    actual   sha256:${current:0:16}…"
	echo "  1. INTENTIONAL change → re-pin and commit BOTH files:"
	echo "       scripts/check-golangci-hash.sh --update && git add .golangci.yml scripts/golangci-config-hash.golden.txt"
	echo "  2. UNEXPECTED edit (the config-corruption class) → inspect, then restore:"
	echo "       git diff .golangci.yml && git restore .golangci.yml"
	exit 1
	;;
--update)
	[[ -f "$CONFIG_FILE" ]] || {
		echo "✗ config not found: $CONFIG_FILE" >&2
		exit 1
	}
	hash=$(hash_of "$CONFIG_FILE")
	printf '%s  .golangci.yml\n' "$hash" >"$GOLDEN_FILE"
	echo "PINNED — golden updated to sha256:${hash:0:12}… ($GOLDEN_FILE)"
	echo "Commit BOTH: .golangci.yml + scripts/golangci-config-hash.golden.txt"
	;;
--self-test)
	SELFTEST_SELF="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/$(basename "${BASH_SOURCE[0]}")"
	SELFTEST_DIR="$(mktemp -d)"
	trap 'rm -rf "$SELFTEST_DIR"' EXIT

	self_run() {
		GOLANGCI_HASH_CONFIG="$SELFTEST_DIR/.golangci.yml" \
			GOLANGCI_HASH_GOLDEN="$SELFTEST_DIR/golden.txt" \
			bash "$SELFTEST_SELF" "$@" 2>&1
	}

	self_check() {
		local name="$1" want="$2" got="$3" out="$4" pattern="$5"
		if [[ "$got" == "$want" ]] && grep -qF -- "$pattern" <<<"$out"; then
			echo "  ✓ PASS: $name"
		else
			echo "  ✗ FAIL: $name (rc=$got, want=$want; missing '$pattern') got:"
			echo "      ${out//$'\n'/$'\n'      }"
			return 1
		fi
	}

	fails=0
	printf 'linters:\n  disable:\n    - depguard\n' >"$SELFTEST_DIR/.golangci.yml"

	rc=0
	out=$(self_run) || rc=$?
	self_check "missing golden fails loud" 1 "$rc" "$out" "hash golden MISSING" || fails=$((fails + 1))

	rc=0
	out=$(self_run --update) || rc=$?
	self_check "--update pins the fixture hash" 0 "$rc" "$out" "PINNED — golden updated" || fails=$((fails + 1))
	[[ "$(cut -d' ' -f1 "$SELFTEST_DIR/golden.txt")" == "$(hash_of "$SELFTEST_DIR/.golangci.yml")" ]] \
		|| {
			echo "  ✗ FAIL: --update did not pin the fixture hash"
			fails=$((fails + 1))
		}

	rc=0
	out=$(self_run) || rc=$?
	self_check "clean tree passes" 0 "$rc" "$out" "PASS — .golangci.yml matches" || fails=$((fails + 1))

	printf 'linters:\n  disable:\n    - depguard\n  # corrupted\n' >"$SELFTEST_DIR/.golangci.yml"
	rc=0
	out=$(self_run) || rc=$?
	self_check "content drift fails" 1 "$rc" "$out" "content hash MISMATCH" || fails=$((fails + 1))

	if ((fails == 0)); then
		echo "check-golangci-hash self-test passed."
		exit 0
	fi
	echo "check-golangci-hash self-test FAILED ($fails leg(s))."
	exit 1
	;;
-h | --help)
	usage
	;;
*)
	echo "check-golangci-hash: unknown flag: $1 (see --help)" >&2
	exit 2
	;;
esac
