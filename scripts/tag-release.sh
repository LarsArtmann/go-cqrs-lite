#!/usr/bin/env bash
# tag-release.sh — Create annotated git tags for per-module releases.
#
# Strips local replace directives from the module's go.mod before tagging,
# so consumers don't hit pseudo-version errors when downloading from the
# Go module proxy. Replace directives are restored after tagging.
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

# --- Strip replace directives from ALL go.mod files ---
# Local replace directives (../foo) are needed for development but break
# consumer builds when published to the Go module proxy. We create a
# temporary commit with stripped go.mod files, tag from that commit,
# then restore the originals.

echo "Stripping replace directives from all go.mod files..."

# Save original go.mod files
backup_dir="$(mktemp -d)"
trap 'rm -rf "$backup_dir"' EXIT

find . -name go.mod -not -path './vendor/*' -not -path './.git/*' | while IFS= read -r gomod; do
  dir="$(dirname "$gomod")"
  mkdir -p "$backup_dir/$dir"
  cp "$gomod" "$backup_dir/$gomod"

  # Drop all replace directives
  while IFS= read -r replace_line; do
    # Extract the LHS module path from "replace foo/bar => ../baz" or block form
    replace_path=$(echo "$replace_line" | grep -oP 'github\.com/larsartmann/go-cqrs-lite/\S+' | head -1)
    if [ -n "$replace_path" ]; then
      (cd "$dir" && go mod edit "-dropreplace=${replace_path}")
    fi
  done < <(grep '=>' "$gomod" 2>/dev/null || true)
done

# Create temporary commit with stripped go.mod files
git add -A
temp_msg="chore(release): strip replace directives for ${tag}"
git commit -m "$temp_msg" --no-verify 2>/dev/null || true
strip_commit=$(git rev-parse HEAD)

# Create annotated tag from the stripped commit
echo "Creating annotated tag: ${tag}"
git tag -a "$tag" -m "${tag}: ${description}"

# Verify
tag_type=$(git cat-file -t "$tag")
if [ "$tag_type" != "tag" ]; then
  echo "ERROR: tag is ${tag_type}, not 'tag' (annotated). Delete and retry."
  git tag -d "$tag"
  git reset --soft HEAD~1
  exit 1
fi

echo "✓ Created ${tag_type}: ${tag}"
echo "  Commit: $(git rev-list -1 --oneline "$tag")"
echo ""

# --- Restore original go.mod files ---
echo "Restoring original go.mod files..."
find . -name go.mod -not -path './vendor/*' -not -path './.git/*' | while IFS= read -r gomod; do
  if [ -f "$backup_dir/$gomod" ]; then
    cp "$backup_dir/$gomod" "$gomod"
  fi
done

# Undo the temporary commit (keep changes as unstaged)
git reset --soft HEAD~1
git checkout -- .

echo "Original go.mod files restored."
echo ""
echo "To push: git push origin ${tag}"
echo "Note: The tag points to a temporary commit that strips replace directives."
echo "      The temporary commit was undone locally; originals are restored."
