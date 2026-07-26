#!/usr/bin/env bash
# tag-release.sh — Create annotated git tags for per-module releases.
#
# Strips LOCAL replace directives from the module's go.mod, re-resolves
# requires via `go mod tidy`, and verifies no pseudo-versions remain — so
# consumers don't hit pseudo-version / local-path errors when downloading
# from the Go module proxy.
#
# Only the tagged module's go.mod is touched (single-module scoping).
# Sibling modules are irrelevant to the tag's consumers: Go fetches the tag,
# reads <module>/go.mod, and ignores every other go.mod in the tree. Touching
# all 58 go.mod files (the old behavior) was unnecessary and risky.
#
# A "local" replace is any directive whose target is a filesystem path —
# relative (../event) or absolute (/home/lars/projects/go-finding). Both the
# go-cqrs-lite/* dev replaces AND sibling-repo replaces (go-finding, go-must)
# are dev-only and must be stripped, or a published go.mod points consumers at
# a path that does not exist on their machine.
#
# Usage:
#   ./scripts/tag-release.sh <module-path> <version> <description> [--dry-run]
#
# Examples:
#   ./scripts/tag-release.sh event v4.0.1 "Fix event payload marshaling"
#   ./scripts/tag-release.sh cmd/cqrs-lint v0.1.0 "First release: 60 rules"
#   ./scripts/tag-release.sh retry v4.0.0 "First release: zero-dep retry"
#   ./scripts/tag-release.sh metaengine v4.0.0 "First release" --dry-run
#
# --dry-run: strip + tidy + verify, print what WOULD be tagged, then exit
# without creating any commit or tag. Use to preview a release safely. The
# working tree is restored to its original state on exit.
set -euo pipefail

cd "$(git rev-parse --show-toplevel)"

usage() {
  echo "Usage: $0 <module-path> <version> <description> [--dry-run]"
  echo "Examples:"
  echo "  $0 event v4.0.1 \"Fix event payload marshaling\""
  echo "  $0 cmd/cqrs-lint v0.1.0 \"First release\""
  echo "  $0 metaengine v4.0.0 \"First release\" --dry-run"
}

# --- Parse args: peel off --dry-run / -h, keep positionals ---
dry_run=false
positionals=()
for arg in "$@"; do
  case "$arg" in
    --dry-run)
      dry_run=true
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      positionals+=("$arg")
      ;;
  esac
done

if [ "${#positionals[@]}" -lt 3 ]; then
  usage
  exit 1
fi

module="${positionals[0]}"
version="${positionals[1]}"
description="${positionals[2]}"
tag="${module}/${version}"
gomod="${module}/go.mod"

# Verify module exists
if [ ! -f "$gomod" ]; then
  echo "ERROR: ${gomod} not found"
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

# --- Restore helpers ---
#
# Two disjoint exit shapes:
#   • before any temp commit is made  → HEAD still points at the original
#     (pre-strip) commit, so `git restore --staged --worktree` from HEAD
#     brings the working tree back to the untouched go.mod/go.sum.
#   • after the temp commit is made    → HEAD points at the strip commit.
#     `undo_temp_commit` first moves HEAD back with `reset --soft` (non-
#     destructive: leaves the working tree alone), THEN restores from the
#     now-original HEAD.
#
# `git restore` is used instead of the old `git checkout -- .`, which
# restored the working tree from the INDEX — and after a `reset --soft` the
# index still held the stripped go.mod, silently re-dirtying the tree. The
# old script's "originals restored" message did not match reality.

restore_working_tree() {
  git restore --staged --worktree "$gomod" 2>/dev/null || true
  if [ -f "${module}/go.sum" ]; then
    git restore --staged --worktree "${module}/go.sum" 2>/dev/null || true
  fi
}

undo_temp_commit() {
  # Move HEAD back to the pre-strip commit (original go.mod), then discard
  # the staged strip changes in favour of that original.
  git reset --soft HEAD~1 2>/dev/null || true
  restore_working_tree
}

# --- Strip LOCAL replace directives (single module) ---
#
# A replace is local iff its target starts with '.' (relative path such as
# ../event) or '/' (absolute path such as /home/lars/projects/go-finding).
# We extract the LHS import path and drop it with `go mod edit -dropreplace`.
#
# `|| true` after the grep is mandatory: grep exits 1 when the file has no
# replace directives at all, and under `set -euo pipefail` that would abort
# the whole release for a module that simply has nothing to strip.
echo "Stripping local replace directives from ${gomod}..."

while IFS= read -r line; do
  # Split on the first '=>'.
  lhs="${line%%=>*}"
  rhs="${line#*=>}"
  # Drop the `replace ` keyword (single-line form); block-form lines have none.
  lhs="${lhs#replace }"
  # Trim surrounding whitespace.
  lhs="$(printf '%s' "$lhs" | sed -e 's/^[[:space:]]*//' -e 's/[[:space:]]*$//')"
  rhs="$(printf '%s' "$rhs" | sed -e 's/^[[:space:]]*//' -e 's/[[:space:]]*$//')"
  # LHS may carry a version ("foo v1.0.0"); keep only the path component.
  lhs="${lhs%% *}"
  [ -z "$lhs" ] && continue
  # Only strip LOCAL targets. Versioned replaces (foo v1 => bar v2) and
  # module-path redirects (foo => other.tld/foo) are left untouched.
  case "$rhs" in
    .*|/*)
      (cd "$module" && go mod edit "-dropreplace=${lhs}")
      echo "  dropped replace ${lhs} => ${rhs}"
      ;;
  esac
done < <(grep '=>' "$gomod" 2>/dev/null || true)

# --- Re-resolve requires ---
#
# With the local replaces gone, `go mod tidy` resolves each require to a real
# published tag instead of the pseudo-version the replace was masking.
# GOWORK=off is mandatory — the workspace's use directives would otherwise
# re-inject local paths and skip proxy resolution. `-e` + `|| true` so a
# module whose sibling dep is not yet tagged does not crash the script; the
# pseudo-version check below catches and reports that case explicitly.
echo "Re-resolving requires (go mod tidy with local replaces stripped)..."
(cd "$module" && GOWORK=off go mod tidy -e 2>/dev/null || true)

# --- Verify no pseudo-versions in the tagged module ---
#
# `grep -q` returns 1 on no-match, which is exactly the "clean" case here and
# is safe under errexit because it sits in an `if` condition (no pipeline,
# no pipefail surface). The old version piped `find -exec grep | wc -l` and
# relied on the command-substitution exemption from errexit — fragile and
# confusing. This form is obviously correct.
echo "Verifying no pseudo-versions remain in ${gomod}..."
if grep -q "00010101000000" "$gomod"; then
  echo "WARNING: ${gomod} still contains a pseudo-version require."
  echo "These break downstream consumers. Matching lines:"
  grep -n "00010101000000" "$gomod" || true
  echo ""
  echo "Aborting release. Every module ${module} depends on must have a"
  echo "published tag. Publish the missing sibling(s), then re-run."
  restore_working_tree
  exit 1
fi

# --- Dry-run preview ---
if $dry_run; then
  echo ""
  echo "=== DRY RUN — no commit or tag will be created ==="
  echo "Would create annotated tag: ${tag}"
  echo "  module:      ${module}"
  echo "  version:     ${version}"
  echo "  description: ${description}"
  echo ""
  echo "Stripped ${gomod} (diff against original):"
  git diff --no-color -- "$gomod" 2>/dev/null || true
  if [ -f "${module}/go.sum" ]; then
    git diff --no-color -- "${module}/go.sum" 2>/dev/null || true
  fi
  restore_working_tree
  echo ""
  echo "Dry run complete. Working tree restored; no commit or tag created."
  exit 0
fi

# --- Create temp commit + annotated tag ---
git add "$gomod"
if [ -f "${module}/go.sum" ]; then
  git add "${module}/go.sum"
fi

temp_msg="chore(release): strip replace directives for ${tag}"
git commit -m "$temp_msg" --no-verify 2>/dev/null || true

echo "Creating annotated tag: ${tag}"
git tag -a "$tag" -m "${tag}: ${description}"

# Verify the tag is annotated (an object, not a lightweight ref).
tag_type="$(git cat-file -t "$tag")"
if [ "$tag_type" != "tag" ]; then
  echo "ERROR: tag is ${tag_type}, not 'tag' (annotated). Cleaning up..."
  git tag -d "$tag"
  undo_temp_commit
  exit 1
fi

echo "✓ Created ${tag_type}: ${tag}"
echo "  Commit: $(git rev-list -1 --oneline "$tag")"
echo ""

# --- Undo the temp commit and restore the original go.mod/go.sum ---
undo_temp_commit

echo "Original ${gomod} restored (working tree clean)."
echo ""
echo "To push: git push origin ${tag}"
echo "Note: the tag points to a temporary commit that strips local replace"
echo "      directives. The temporary commit was undone locally; the original"
echo "      go.mod is restored."
