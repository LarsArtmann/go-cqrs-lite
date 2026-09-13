#!/usr/bin/env bash
# release_common.sh — functions shared by the release scripts.
#
# Sourced, never executed. tag-release.sh and batch-release.sh both source
# this file so the issue-#20 guard and the smoke-probe helpers have ONE
# implementation: the second copy of path_matches_major was a two-copy
# lockstep risk (a fix in one copy silently missing the other).

# path_matches_major <module-path> <version>: v0/v1 tags require a module
# path WITHOUT any /vN suffix; v2+ tags require the path to end in the
# matching /vN. Mismatched tags are INVISIBLE to the module proxy (the
# issue-#20 class), so both release scripts route every guard here.
path_matches_major() {
	local module_path="$1"
	local version="$2"
	local tag_major="${version#v}"
	tag_major="${tag_major%%.*}"

	local path_major=""
	case "$module_path" in
	*/v[0-9]*)
		path_major="${module_path##*/v}"
		;;
	esac

	case "$tag_major" in
	0 | 1)
		[ -z "$path_major" ]
		;;
	*)
		[ "$path_major" = "$tag_major" ]
		;;
	esac
}

# module_has_root_main <module-dir>: exits 0 iff the module's root package
# is `main`. The install+run smoke probe only applies to CLI modules; a
# library module takes the "probe skipped" path in the caller.
module_has_root_main() {
	grep -qs '^package main$' "$1"/*.go
}

# smoke_probe_args <module-dir> [probes-file]: prints the explicit probe
# invocation declared for the module in scripts/smoke-probes.txt (one
# "<module-path> <args...>" line per module, # comments allowed), or prints
# nothing when the module has no entry — the caller then falls back to the
# bare --help default. An explicit probe replaces the default entirely.
smoke_probe_args() {
	local module_dir="$1"
	local probes_file="${2:-}"
	if [ -z "$probes_file" ]; then
		probes_file="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/smoke-probes.txt"
	fi
	[ -f "$probes_file" ] || return 0
	awk -v m="$module_dir" '!/^#/ && $1 == m { $1 = ""; sub(/^ /, ""); print; exit }' "$probes_file"
}
