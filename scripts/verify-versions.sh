#!/usr/bin/env bash
# verify-versions.sh — Check that all consumer projects use the same go-cqrs-lite version.
#
# Usage: ./scripts/verify-versions.sh [projects-dir]
#
# Exit code 0 = all consumers on the same version.
# Exit code 1 = version mismatch detected.
set -euo pipefail

PROJECTS_DIR="${1:-/home/lars/projects}"
GO_CQRS_LITE_MODULE="github.com/larsartmann/go-cqrs-lite"

declare -A version_map
mismatch_found=0

for dir in "$PROJECTS_DIR"/*/; do
    proj=$(basename "$dir")
    gomod="$dir/go.mod"

    [ -f "$gomod" ] || continue
    grep -q "$GO_CQRS_LITE_MODULE" "$gomod" 2>/dev/null || continue

    # Extract all go-cqrs-lite versions (direct + indirect)
    versions=$(grep "$GO_CQRS_LITE_MODULE" "$gomod" | \
        grep -oP 'v\d+\.\d+\.\d+' | sort -u)

    if [ -z "$versions" ]; then
        continue
    fi

    # Check if multiple versions exist
    version_count=$(echo "$versions" | wc -l)
    if [ "$version_count" -gt 1 ]; then
        echo "⚠️  $proj has MULTIPLE versions:"
        echo "$versions" | sed 's/^/    /'
        mismatch_found=1
    else
        version=$(echo "$versions" | head -1)
        version_map["$version"]+="$proj "
    fi
done

echo ""
echo "=== Version Summary ==="
for version in "${!version_map[@]}"; do
    count=$(echo "${version_map[$version]}" | wc -w)
    echo "v$version: $count projects"
done

if [ "$mismatch_found" -eq 1 ]; then
    echo ""
    echo "❌ Version mismatch detected. Run 'go mod tidy' in affected projects."
    exit 1
fi

echo ""
echo "✅ All consumers on consistent versions."
