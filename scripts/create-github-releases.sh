#!/usr/bin/env bash
# create-github-releases.sh — create/patch GitHub Releases with the CHANGELOG
# section as the release body.
#
# release.yml's generate_release_notes produces PR-based notes. This script is
# the manual, changelog-accurate path: for each tag it extracts the matching
# "## [version]" section from the ROOT CHANGELOG.md (the single changelog per
# CONTRIBUTING.md) and creates the GitHub Release with that body — or UPDATES
# the existing release body when the release already exists.
#
# Usage:
#   ./scripts/create-github-releases.sh event/v4.0.1 command/v4.0.1
#
# Tag-to-version mapping: the module prefix is stripped ("event/v4.0.1" →
# "v4.0.1") and the CHANGELOG section whose header contains that version
# ("## [4.0.1] …" or "## [event/v4.0.1] …") is used. Release-train sections
# ("## [a/v1.0.0, b/v1.2.0 — 2026-09-19 release train (+82 more module tags)]")
# never end a header with each version, so two extra match rules apply, over
# sections scanned top-down (newest first):
#   - the full tag appears as a delimiter-bounded token anywhere in the
#     section (train bodies list every module in backticks), or
#   - the tag's last two path components do (brace-shorthand groups like
#     `metaengine/{badgerengine/v4.2.0, …}` list inner tags without the
#     module prefix).
# No match = skip with a warning (never fabricate notes).
#
# Requires: gh CLI authenticated against the repo; tags already PUSHED.
set -euo pipefail

cd "$(git rev-parse --show-toplevel)"

CHANGELOG="CHANGELOG.md"

if [ $# -eq 0 ]; then
	echo "Usage: $0 [--dry-run] <tag> [<tag> ...]"
	echo "Example: $0 event/v4.0.1 metaengine/v4.2.0"
	exit 1
fi

dry_run=0
if [ "$1" = "--dry-run" ]; then
	dry_run=1
	shift
fi

command -v gh >/dev/null 2>&1 || {
	echo "ERROR: gh CLI not found (https://cli.github.com/)" >&2
	exit 1
}

extract_section() {
	tag="$1"                          # full tag, e.g. metaengine/badgerengine/v4.2.0
	tail2="${tag#*/}"                 # badgerengine/v4.2.0 (last two path components)
	[ "$tail2" = "$tag" ] && tail2="" # single-component tag: no shorthand form
	awk -v tag="$tag" -v tail2="$tail2" -v ptr="$1" '
		function section_matches(text) {
			n = split(text, lines, "\n")
			for (l = 1; l <= n; l++) {
				gsub(/[`{}()\[\]—; ]/, ",", lines[l])
				m = split(lines[l], tok, ",")
				for (i = 1; i <= m; i++)
					if (tok[i] == tag || (tail2 != "" && tok[i] == tail2))
						return 1
			}
			return 0
		}
		function emit_train_pointer() {
			print ""
			print "Full details: [CHANGELOG.md](https://github.com/LarsArtmann/go-cqrs-lite/blob/master/CHANGELOG.md) — the wave section at the top of the file covers every subsystem change in this release train."
			exit
		}
		function flush() {
			if (sec == "" || !section_matches(sec))
				return
			n = split(sec, out, "\n")
			train = (out[1] ~ /release train/)
			for (i = 1; i <= n; i++) {
				if (train && out[i] ~ /^###/)
					emit_train_pointer()
				print out[i]
			}
			if (train)
				emit_train_pointer()
			exit # top-down: the newest matching section wins
		}
		/^## / {
			flush()
			sec = ($0 ~ /^## \[/) ? $0 : ""
			next
		}
		sec != "" { sec = sec "\n" $0 }
		END { flush() }
	' "$CHANGELOG"
}

created=0
updated=0
skipped=0

for tag in "$@"; do
	body=$(extract_section "$tag")
	if [ -z "$body" ]; then
		echo "SKIP $tag: no matching section in $CHANGELOG"
		skipped=$((skipped + 1))
		continue
	fi

	if [ "$dry_run" = 1 ]; then
		echo "[DRY] $tag: extracted $(printf '%s\n' "$body" | wc -l) lines starting: $(printf '%s' "$body" | head -1 | cut -c1-80)"
		created=$((created + 1))
		continue
	fi

	notes=$(mktemp)
	trap 'rm -f "$notes"' EXIT
	printf '%s\n' "$body" >"$notes"

	if gh release view "$tag" >/dev/null 2>&1; then
		gh release edit "$tag" --notes-file "$notes"
		echo "UPDATED $tag"
		updated=$((updated + 1))
	else
		gh release create "$tag" --title "$tag" --notes-file "$notes"
		echo "CREATED $tag"
		created=$((created + 1))
	fi
done

echo ""
echo "Done: $created created, $updated updated, $skipped skipped."
