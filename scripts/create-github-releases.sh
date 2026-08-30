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
# ("## [4.0.1] …" or "## [event/v4.0.1] …") is used. No match = skip with a
# warning (never fabricate notes).
#
# Requires: gh CLI authenticated against the repo; tags already PUSHED.
set -euo pipefail

cd "$(git rev-parse --show-toplevel)"

CHANGELOG="CHANGELOG.md"

if [ $# -eq 0 ]; then
	echo "Usage: $0 <tag> [<tag> ...]"
	echo "Example: $0 event/v4.0.1 metaengine/v4.2.0"
	exit 1
fi

command -v gh >/dev/null 2>&1 || {
	echo "ERROR: gh CLI not found (https://cli.github.com/)" >&2
	exit 1
}

extract_section() {
	version="$1"
	awk -v v="$version" '
		$0 ~ "^## \\[.*" v "\\]" { in_section = 1; print; next }
		in_section && /^## / { exit }
		in_section { print }
	' "$CHANGELOG"
}

created=0
updated=0
skipped=0

for tag in "$@"; do
	version="${tag##*/}" # event/v4.0.1 → v4.0.1

	body=$(extract_section "$version")
	if [ -z "$body" ]; then
		echo "SKIP $tag: no '## [$version]' section in $CHANGELOG"
		skipped=$((skipped + 1))
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
