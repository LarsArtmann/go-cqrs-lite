#!/usr/bin/env bash
# check-readme-links.sh — link-check every module README (row 634f).
#
# The repo link checker (check-doc-links.sh) covers SKILL/AGENTS/root docs
# but skips the ~95 module READMEs; this wrapper feeds them all through the
# same engine (fence-aware, inline-code-aware, anchor-stripping) so a moved
# or deleted file breaks CI the same way a broken skill reference does.
#
# Usage:
#   scripts/check-readme-links.sh [--self-test]
#
# Env overrides (honored by --self-test so the test never touches tracked
# files — planted-fixture pattern from calibration-gate.sh):
#   CHECK_READMES_ROOT   scan root for README.md discovery (default: repo root)
#   CHECK_READMES_FILES  explicit README list, overrides discovery entirely
set -uo pipefail

repo_root="$(git rev-parse --show-toplevel 2>/dev/null)" || {
	echo "check-readme-links.sh: must run inside the git repo" >&2
	exit 1
}
cd "$repo_root" || exit 1

self_test() {
	local fixture
	fixture="$(mktemp -d /tmp/readme-links-fixture.XXXXXX)" || return 1
	trap 'rm -rf "$fixture"' RETURN

	printf '# good\n[exists](./NEIGHBOR.md)\n' >"$fixture/GOOD.md"
	printf '# neighbor\n' >"$fixture/NEIGHBOR.md"
	printf '# bad\n[missing](./NOPE.md)\n' >"$fixture/BAD.md"

	# Leg 1 (non-vacuous detection): a broken link MUST fail the gate.
	if CHECK_READMES_FILES="$fixture/BAD.md" bash "$0" >/dev/null 2>&1; then
		echo "self-test FAILED: broken link in $fixture/BAD.md was not flagged" >&2
		return 1
	fi
	# Leg 2 (no false positive): a healthy README MUST pass.
	if ! CHECK_READMES_FILES="$fixture/GOOD.md" bash "$0" >/dev/null 2>&1; then
		echo "self-test FAILED: valid link in $fixture/GOOD.md was flagged" >&2
		return 1
	fi

	echo "check-readme-links self-test: both legs green (detection + no-FP)"
	return 0
}

[ "${1:-}" = "--self-test" ] && {
	self_test
	exit $?
}

if [ -n "${CHECK_READMES_FILES:-}" ]; then
	# shellcheck disable=SC2206 # explicit list from tests is a deliberate word split
	files=($CHECK_READMES_FILES)
else
	root="${CHECK_READMES_ROOT:-$repo_root}"
	files=()
	while IFS= read -r f; do files+=("$f"); done < <(
		find "$root" -name README.md \
			-not -path '*/.git/*' \
			-not -path '*/vendor/*' 2>/dev/null | sort
	)
fi

if [ "${#files[@]}" -eq 0 ]; then
	echo "check-readme-links: no READMEs found under ${CHECK_READMES_ROOT:-repo root}" >&2
	exit 1
fi

exec bash "$repo_root/scripts/check-doc-links.sh" "${files[@]}"
