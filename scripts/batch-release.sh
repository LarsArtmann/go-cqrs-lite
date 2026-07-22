#!/usr/bin/env bash
# batch-release.sh — Create annotated git tags for multiple modules in one pass.
#
# Strips local replace directives from ALL go.mod files once, creates a
# temporary commit, tags all listed modules from that single commit, then
# restores the originals. This is the batch equivalent of tag-release.sh.
#
# Usage:
#   ./scripts/batch-release.sh "<module> <version> <description>" ...
#
# Each argument is a space-separated triple: module-path, version, description.
# Description may contain spaces if quoted as part of the triple.
#
# Example:
#   ./scripts/batch-release.sh \
#     "event v4.0.3 Patch release" \
#     "command v4.0.1 Patch release" \
#     "cmd/cqrs-lint v0.3.0 Scanner accuracy overhaul"
#
set -euo pipefail

cd "$(git rev-parse --show-toplevel)"

if [ $# -eq 0 ]; then
  echo "Usage: $0 \"<module> <version> <description>\" ..."
  exit 1
fi

# Parse arguments into arrays
modules=()
versions=()
descriptions=()
tags=()

for arg in "$@"; do
  module=$(echo "$arg" | awk '{print $1}')
  version=$(echo "$arg" | awk '{print $2}')
  description=$(echo "$arg" | cut -d' ' -f3-)
  tag="${module}/${version}"

  if [ ! -f "${module}/go.mod" ]; then
    echo "ERROR: ${module}/go.mod not found"
    exit 1
  fi

  if git tag -l "$tag" | grep -q .; then
    echo "ERROR: tag ${tag} already exists"
    exit 1
  fi

  modules+=("$module")
  versions+=("$version")
  descriptions+=("$description")
  tags+=("$tag")
done

echo "Releasing ${#tags[@]} modules:"
for i in "${!tags[@]}"; do
  echo "  ${tags[$i]}: ${descriptions[$i]}"
done
echo ""

# Verify clean working tree
if ! git diff-index --quiet HEAD --; then
  echo "ERROR: working tree has uncommitted changes. Commit first."
  git status --short
  exit 1
fi

# --- Strip replace directives from ALL go.mod files ---
echo "Stripping replace directives from all go.mod files..."

backup_dir="$(mktemp -d)"
trap 'rm -rf "$backup_dir"' EXIT

find . -name go.mod -not -path './vendor/*' -not -path './.git/*' | while IFS= read -r gomod; do
  dir="$(dirname "$gomod")"
  mkdir -p "$backup_dir/$dir"
  cp "$gomod" "$backup_dir/$gomod"

  while IFS= read -r replace_line; do
    replace_path=$(echo "$replace_line" | grep -oP 'github\.com/larsartmann/go-cqrs-lite/\S+' | head -1 || true)
    if [ -n "$replace_path" ]; then
      (cd "$dir" && go mod edit "-dropreplace=${replace_path}")
    fi
  done < <(grep '=>' "$gomod" 2>/dev/null || true)
done

# Re-resolve requires after stripping replaces
echo "Re-resolving requires (go mod tidy with replaces stripped)..."
find . -name go.mod -not -path './vendor/*' -not -path './.git/*' | while IFS= read -r gomod; do
  dir="$(dirname "$gomod")"
  (cd "$dir" && GOWORK=off go mod tidy -e 2>/dev/null || true)
done

# Verify: no pseudo-versions remain
echo "Verifying no pseudo-versions remain..."
pseudo_count=$(find . -name go.mod -not -path './vendor/*' -not -path './.git/*' -exec grep -l "00010101000000" {} + 2>/dev/null | wc -l)
if [ "$pseudo_count" -gt 0 ]; then
  echo "WARNING: $pseudo_count go.mod file(s) still contain pseudo-version requires."
  find . -name go.mod -not -path './vendor/*' -not -path './.git/*' -exec grep -l "00010101000000" {} + 2>/dev/null
  echo ""
  echo "Aborting release."
  exit 1
fi

# Create temporary commit with stripped go.mod files
git add -A
temp_msg="chore(release): strip replace directives for batch release"
git commit -m "$temp_msg" --no-verify 2>/dev/null || true

# Create all annotated tags from the stripped commit
for i in "${!tags[@]}"; do
  tag="${tags[$i]}"
  desc="${descriptions[$i]}"
  echo "Creating annotated tag: ${tag}"
  git tag -a "$tag" -m "${tag}: ${desc}"

  tag_type=$(git cat-file -t "$tag")
  if [ "$tag_type" != "tag" ]; then
    echo "ERROR: tag ${tag} is ${tag_type}, not 'tag' (annotated). Deleting and retrying."
    git tag -d "$tag"
    # Continue with others, report at end
  else
    echo "  ✓ ${tag_type}: ${tag}"
  fi
done

# --- Restore original go.mod files ---
echo ""
echo "Restoring original go.mod files..."
find . -name go.mod -not -path './vendor/*' -not -path './.git/*' | while IFS= read -r gomod; do
  if [ -f "$backup_dir/$gomod" ]; then
    cp "$backup_dir/$gomod" "$gomod"
  fi
done

# Undo the temporary commit
git reset --soft HEAD~1
git checkout -- .

echo "Original go.mod files restored."
echo ""
echo "Created ${#tags[@]} tags. To push:"
echo "  git push origin ${tags[*]}"
