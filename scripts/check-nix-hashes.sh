#!/usr/bin/env bash
# check-nix-hashes.sh — placeholder-hash tripwire for flake.nix.
#
# Problem (2026-10-23 incident): an auto-configure pass rewrote every pinned
# source/vendor hash in flake.nix to the all-zero placeholder
# (sha256-AAAA…=) and the auto-commit daemon absorbed it. Eval still worked,
# so the breakage surfaced only when a derivation was finally realised —
# `nix run .#check-duplication` died on a fixed-output hash mismatch minutes
# later, and EVERY nix gate (build/test/lint/verify) would have failed the
# same way. This is the sibling of the .golangci.yml corruption class: a
# machine-authored rewrite that no test covers because it is valid syntax.
#
# The check is deliberately dumb and total: a hashes file that ships to CI
# must contain no placeholder, ever. Restoring a real hash is a one-line fix;
# discovering it mid-#verify costs a full cycle.
#
# Usage: bash scripts/check-nix-hashes.sh [files...]   (default: flake.nix)
#        bash scripts/check-nix-hashes.sh --self-test
# Exit:  0 clean, 1 placeholder found or file missing, 2 usage.

set -euo pipefail

# NH_ROOT: internal hook for --self-test fixtures (same pattern as
# CALIB_GATE_LOADAVG_FILE / RD_ROOT — point the script at a planted tree).
if [ -n "${NH_ROOT:-}" ]; then
	cd "$NH_ROOT"
else
	cd "$(git rev-parse --show-toplevel)"
fi

# Placeholder forms: the nixpkgs lib.fakeHash constant, its literal
# all-zero sha256 renderings (base64 and hex), and an empty hash.
PLACEHOLDER_RE='lib\.fakeHash|lib\.fakeSha256|lib\.fakeSha512|sha256-AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=|sha256-0000000000000000000000000000000000000000000000000000|hash = ""|vendorHash = ""'

scan() {
	local failed=0 file

	for file in "$@"; do
		if [ ! -f "$file" ]; then
			echo "ERROR: $file not found" >&2
			failed=1
			continue
		fi

		local hits
		hits="$(grep -nE "$PLACEHOLDER_RE" "$file" || true)"

		if [ -n "$hits" ]; then
			echo "✗ $file carries placeholder hash(es):" >&2
			printf '%s\n' "$hits" | sed 's/^/    /' >&2
			failed=1
		fi
	done

	return "$failed"
}

if [ "${1:-}" = "--self-test" ]; then
	NH_TMP="$(mktemp -d)"
	trap 'rm -rf "${NH_TMP:-}"' EXIT

	failures=0

	# Leg 1: a clean file passes.
	printf 'src = fetchFromGitHub { hash = "sha256-KOjiTspiVFweUiSYPCPb+Lnk+sqymnfIyujxduk9tOY="; };\n' \
		>"$NH_TMP/flake.nix"
	if ! NH_ROOT="$NH_TMP" "$0" flake.nix >/dev/null 2>&1; then
		echo "  ✗ FAIL: clean fixture must pass" >&2
		failures=$((failures + 1))
	fi

	# Leg 2: the all-zero placeholder is caught.
	printf 'src = fetchFromGitHub { hash = "sha256-AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="; };\n' \
		>"$NH_TMP/flake.nix"
	if NH_ROOT="$NH_TMP" "$0" flake.nix >/dev/null 2>&1; then
		echo "  ✗ FAIL: zero-hash placeholder must fail" >&2
		failures=$((failures + 1))
	fi

	# Leg 3: the nixpkgs constant is caught.
	printf 'vendorHash = lib.fakeHash;\n' >"$NH_TMP/flake.nix"
	if NH_ROOT="$NH_TMP" "$0" flake.nix >/dev/null 2>&1; then
		echo "  ✗ FAIL: lib.fakeHash must fail" >&2
		failures=$((failures + 1))
	fi

	# Leg 4: a missing file is loud, never a silent pass.
	if NH_ROOT="$NH_TMP" "$0" absent.nix >/dev/null 2>&1; then
		echo "  ✗ FAIL: missing file must fail" >&2
		failures=$((failures + 1))
	fi

	if [ "$failures" -gt 0 ]; then
		echo "self-test: ${failures} failure(s)"
		exit 1
	fi

	echo "self-test: all green"
	exit 0
fi

FILES=("$@")
if [ "${#FILES[@]}" -eq 0 ]; then
	FILES=(flake.nix)
fi

if ! scan "${FILES[@]}"; then
	echo "" >&2
	echo "FAIL: placeholder hashes in ${FILES[*]}." >&2
	echo "Fix: restore the real hash (nix reports the expected value as 'got:' on the first build attempt)." >&2
	exit 1
fi

echo "OK: no placeholder hashes in ${FILES[*]}."
