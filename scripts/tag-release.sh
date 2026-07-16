#!/usr/bin/env bash
# tag-release.sh — Create annotated git tags for per-module releases.
#
# Usage:
#   ./scripts/tag-release.sh <module-path> <version> <description>
#
# Examples:
#   ./scripts/tag-release.sh event v4.0.1 "Fix event payload marshaling"
#   ./scripts/tag-release.sh cmd/cqrs-lint v0.1.0 "First release: 60 rules"
#   ./scripts/tag-release.sh retry v4.0.0 "First release: zero-dep retry"
#
set -euo pipefail

cd "$(git rev-parse --show-toplevel)"

if [ $# -lt 3 ]; then
  echo "Usage: $0 <module-path> <version> <description>"
  echo "Examples:"
  echo "  $0 event v4.0.1 \"Fix event payload marshaling\""
  echo "  $0 cmd/cqrs-lint v0.1.0 \"First release\""
  exit 1
fi

module="$1"
version="$2"
description="$3"
tag="${module}/${version}"

# Verify module exists
if [ ! -f "${module}/go.mod" ]; then
  echo "ERROR: ${module}/go.mod not found"
  exit 1
fi

# Verify tag doesn't already exist
if git tag -l "$tag" | grep -q .; then
  echo "ERROR: tag ${tag} already exists"
  exit 1
fi

# Verify clean working tree (no uncommitted changes)
if ! git diff-index --quiet HEAD --; then
  echo "ERROR: working tree has uncommitted changes. Commit first."
  git status --short
  exit 1
fi

# Create annotated tag
echo "Creating annotated tag: ${tag}"
git tag -a "$tag" -m "${tag}: ${description}"

# Verify
tag_type=$(git cat-file -t "$tag")
if [ "$tag_type" != "tag" ]; then
  echo "ERROR: tag is ${tag_type}, not 'tag' (annotated). Delete and retry."
  git tag -d "$tag"
  exit 1
fi

echo "✓ Created ${tag_type}: ${tag}"
echo "  Commit: $(git rev-list -1 --oneline "$tag")"
echo ""
echo "To push: git push origin ${tag}"
