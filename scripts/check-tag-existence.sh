#!/usr/bin/env bash
# check-tag-existence.sh — Verifies that all go-cqrs-lite module versions
# referenced in go.mod files have corresponding git tags.
#
# Catches the "version-sequence break" class of bug where a go.mod references
# a version that was never tagged (e.g., storage/v4.2.0 referenced but only
# storage/v4.1.0 exists).
#
# Usage: bash scripts/check-tag-existence.sh
# Exit: 0 if all tags exist, 1 if any are missing.

set -euo pipefail

cd "$(git rev-parse --show-toplevel)"

# Extract all go-cqrs-lite module references with versions from go.mod files.
# Format: module-path version
refs=$(grep -rhE 'go-cqrs-lite/[^ ]+/v[0-9]+ v[0-9]+\.[0-9]+\.[0-9]+' \
    */go.mod go.mod 2>/dev/null \
    | grep -oE 'go-cqrs-lite/[^ ]+/v[0-9]+ v[0-9]+\.[0-9]+\.[0-9]+' \
    | sort -u || true)

if [ -z "$refs" ]; then
    echo "No go-cqrs-lite module references found."
    exit 0
fi

missing=0
while IFS=' ' read -r mod ver; do
    # Convert module path + version to tag.
    # Example: "go-cqrs-lite/event/v4" + "v4.4.0" → "event/v4.4.0"
    # Example: "go-cqrs-lite/storage/pebble/v4" + "v4.1.0" → "storage/pebble/v4.1.0"
    # Rule: strip "go-cqrs-lite/" prefix and trailing "/vN" suffix, then append "/<version>"
    modpath="${mod#go-cqrs-lite/}"
    modpath="${modpath%/v[0-9]*}"
    tag="${modpath}/${ver}"

    if ! git rev-parse -q --verify "refs/tags/${tag}" >/dev/null 2>&1; then
        echo "::error::Missing tag: ${tag} (referenced in go.mod files)"
        missing=$((missing + 1))
    fi
done <<< "$refs"

if [ "$missing" -gt 0 ]; then
    echo "::error::$missing tag(s) missing — modules reference versions that were never tagged"
    exit 1
fi

echo "All referenced module tags exist."
