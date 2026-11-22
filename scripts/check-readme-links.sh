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
#
# The gate has two layers: the shared link engine (existence check — the same
# one CI runs), plus an UNTRACKED-TARGET pass that exists only locally: a
# README linking a file that is on disk but NOT in git passes the existence
# check on a dev box and breaks only on CI, where the file never lands (the
# 2026-11-22 dangling-report-row incident). On CI both layers agree because
# a fresh checkout has nothing untracked.
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

	# Leg 3 (local/CI asymmetry): an ON-DISK but UNTRACKED target must fail
	# the local run — in-repo planted fixtures (cleaned up immediately), since
	# the check is inherently git-dependent.
	local inrepo unlinked
	inrepo="$repo_root/.readme-links-selftest-$$.md"
	unlinked="$repo_root/.readme-links-selftest-untracked-$$.md"
	printf '# asym\n[untracked](%s)\n' "$(basename "$unlinked")" >"$inrepo"
	printf '# ghost\n' >"$unlinked"
	if CHECK_READMES_FILES="$inrepo" bash "$0" >/dev/null 2>&1; then
		echo "self-test FAILED: on-disk untracked target in $inrepo was not flagged" >&2
		rm -f "$inrepo" "$unlinked"
		return 1
	fi
	rm -f "$inrepo" "$unlinked"

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

	echo "check-readme-links self-test: 3/3 legs green (detection + no-FP + untracked-target)"
	return 0
}

# untracked_pass <readme...> — flag relative-link targets that exist on disk
# but are not in git. Prints "README -> target" rows; empty output = clean.
untracked_pass() {
	local readme target resolved stripped
	for readme in "$@"; do
		# shellcheck disable=SC2013 # link extraction; this tree has no spaces in doc paths
		grep -ohE '\]\([^)]+\)' "$readme" 2>/dev/null |
			sed -e 's/^](//' -e 's/)$//' |
			while IFS= read -r target; do
				case "$target" in
				http* | /* | mailto:*) continue ;;
				esac
				stripped="${target%%#*}"
				[ -n "$stripped" ] || continue
				resolved="$(dirname "$readme")/$stripped"
				[ -e "$resolved" ] || continue # missing targets are the engine's failure class
				case "${resolved#"$repo_root"/}" in
				"$resolved") continue ;; # outside the repo: not git's business
				esac
				if ! git ls-files --error-unmatch -- "$resolved" >/dev/null 2>&1; then
					printf '%s -> %s\n' "$readme" "$target"
				fi
			done
	done
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

untracked="$(untracked_pass "${files[@]}")"
if [ -n "$untracked" ]; then
	echo "::error::README link targets that exist on disk but are NOT in git (they break CI, where the files never land):" >&2
	printf '%s\n' "$untracked" >&2
	echo "::error::commit the targets (or trash the rows) before pushing — the existence check alone cannot see this class." >&2
	exit 1
fi

bash "$repo_root/scripts/check-doc-links.sh" "${files[@]}"
