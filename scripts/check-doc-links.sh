#!/usr/bin/env bash
# check-doc-links.sh — verify relative markdown link targets resolve in-repo.
#
# Checks every markdown link of the form ](path) OUTSIDE fenced code blocks
# and inline code spans (generics like [T](x, y) live there). External
# schemes (http/https/mailto/data) and pure-anchor links are skipped; anchor
# fragments are stripped before resolution. Checked files are resolved
# through symlinks (SKILL.md -> .agents/...), so relative links resolve
# against the link's real directory. Archived trees and docs/feedback are
# frozen history and skipped by default.
#
# Usage: scripts/check-doc-links.sh [file.md ...]
# Exit 0 = all relative targets resolve; exit 1 = at least one broken link.
set -uo pipefail

repo_root="$(git rev-parse --show-toplevel 2>/dev/null)" || {
	echo "check-doc-links.sh: must run inside the git repo" >&2
	exit 1
}
cd "$repo_root"

files=("$@")
if [ "${#files[@]}" -eq 0 ]; then
	while IFS= read -r f; do files+=("$f"); done < <(
		{
			ls SKILL.md AGENTS.md README.md TODO_LIST.md FEATURES.md ROADMAP.md CHANGELOG.md CONTRIBUTING.md 2>/dev/null
			find docs .agents/skills -name '*.md' \
				-not -path 'docs/status/archived/*' \
				-not -path 'docs/planning/archived/*' \
				-not -path 'docs/feedback/*' 2>/dev/null
		} | sort -u
	)
fi

broken=0
checked=0

for f in "${files[@]}"; do
	[ -f "$f" ] || continue

	# Resolve symlinks so relative links resolve against the real location.
	real="$(readlink -f "$f")"
	[ -n "$real" ] || real="$f"
	dir="$(dirname "$real")"

	# Extract link targets outside ``` fences, with inline-code spans stripped.
	while IFS= read -r target; do
		[ -n "$target" ] || continue

		case "$target" in
		http://* | https://* | mailto:* | data:* | \#*) continue ;;
		*\ *) continue ;; # prose noise: brackets with spaces are not links
		esac

		case "$target" in
		*.go:[0-9]*) continue ;; # source line references, not file links
		esac

		path="${target%%#*}"
		[ -n "$path" ] || continue

		checked=$((checked + 1))

		if [ ! -e "$dir/$path" ]; then
			echo "BROKEN: $f -> $target"
			broken=$((broken + 1))
		fi
	done < <(awk '
		/^```/ { fence = !fence; next }
		fence { next }
		{
			line = $0
			gsub(/`[^`]*`/, "", line)
			while (match(line, /\]\(([^)]+)\)/)) {
				print substr(line, RSTART + 2, RLENGTH - 3)
				line = substr(line, RSTART + RLENGTH)
			}
		}
	' "$real" | sort -u)
done

echo "checked $checked relative link targets across ${#files[@]} files; $broken broken"
[ "$broken" -eq 0 ]
