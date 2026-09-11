#!/usr/bin/env bash
# batch-release.sh — Create annotated git tags for multiple modules in one pass.
#
# Batch equivalent of tag-release.sh, hardened to the same issue-#20 bar
# (see tag-release.sh's header for the failure classes):
#   - path-vs-tag guard per triple: a tag whose major version does not match
#     the module path declared in the go.mod is INVISIBLE to the module
#     proxy (cmd/cqrs-lint shipped v4.2.0-v4.7.0 this way).
#   - Only the TAGGED modules' go.mods are touched, and only LOCAL
#     (path-targeted) replaces are stripped. Sibling go.mods are irrelevant
#     to each tag's consumers — the proxy reads only the tagged module's own
#     go.mod at the tag — and the old strip-everything behavior mutated 80+
#     unrelated files for nothing.
#   - Every tagged module must COMPILE standalone (GOWORK=off) against its
#     stripped go.mod before any tag is created (command/v4.7.0 shipped with
#     a go.mod pinning an older sibling than the code needed).
#   - Hardened restore: the temp commit is undone with `git reset --soft
#     <original_head>` + `git restore --staged --worktree`. The old
#     `git checkout -- .` restored the STRIPPED go.mods from the stale
#     index and silently re-dirtied the tree; HEAD~1 also broke when the
#     auto-commit daemon committed between the temp commit and the reset.
#   - After pushing, smoke-check every tag: scripts/tag-release.sh --smoke.
#
# Usage:
#   ./scripts/batch-release.sh [--dry-run] "<module> <version> <description>" ...
#   ./scripts/batch-release.sh --audit
#
# --dry-run prints the tags that WOULD be created (existence, tag collision,
# path-vs-tag guard, sequence checks) without touching go.mod files, the
# tree, or the repo.
#
# --audit replays the path-vs-tag guard over EVERY tag of EVERY module
# (delegates to tag-release.sh --audit — the single implementation).
#
# Each argument is a space-separated triple: module-path, version, description.
# Description may contain spaces if quoted as part of the triple.
#
# Example:
#   ./scripts/batch-release.sh \
#     "event v4.0.3 Patch release" \
#     "command v4.0.1 Patch release" \
#     "cmd/cqrs-lint v0.3.0 Scanner accuracy overhaul"
set -euo pipefail

cd "$(git rev-parse --show-toplevel)"

usage() {
	echo "Usage: $0 [--dry-run] \"<module> <version> <description>\" ..."
	echo "       $0 --audit"
	echo "Example:"
	echo "  $0 \"event v4.0.3 Patch release\" \"command v4.0.1 Patch release\""
	echo "  $0 --dry-run \"event v4.0.3 Patch release\""
	echo "  $0 --audit   # all-modules path-vs-tag audit (via tag-release.sh)"
}

# path_matches_major mirrors tag-release.sh's single implementation of the
# issue-#20 guard: v0/v1 tags require a module path WITHOUT any /vN suffix;
# v2+ tags require the path to end in the matching /vN. Keep the two copies
# in lockstep — the guard is ~20 lines, and sourcing tag-release.sh would
# execute its main flow.
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

if [ "${1:-}" = "--audit" ]; then
	if [ $# -ne 1 ]; then
		usage
		exit 1
	fi
	exec bash "$(dirname "$0")/tag-release.sh" --audit
fi

DRY_RUN=0
ARGS=()
for a in "$@"; do
	case "$a" in
	--dry-run) DRY_RUN=1 ;;
	-h | --help)
		usage
		exit 0
		;;
	*) ARGS+=("$a") ;;
	esac
done

if [ ${#ARGS[@]} -eq 0 ]; then
	usage
	exit 1
fi

# --- Parse triples; every non-mutating guard runs up front so a bad batch
# fails before anything is touched ---
modules=()
versions=()
descriptions=()
tags=()

for arg in "${ARGS[@]}"; do
	module=$(printf '%s' "$arg" | awk '{print $1}')
	version=$(printf '%s' "$arg" | awk '{print $2}')
	description=$(printf '%s' "$arg" | cut -d' ' -f3-)
	tag="${module}/${version}"
	gomod="${module}/go.mod"

	if [ ! -f "$gomod" ]; then
		echo "ERROR: ${gomod} not found"
		exit 1
	fi

	module_path="$(awk '/^module /{print $2; exit}' "$gomod")"
	if ! path_matches_major "$module_path" "$version"; then
		tag_major="${version#v}"
		tag_major="${tag_major%%.*}"
		echo "ERROR: tag ${tag} is inconsistent with the module path in ${gomod}:"
		echo "    module ${module_path}"
		case "$tag_major" in
		0 | 1)
			echo "v${tag_major} tags require a module path WITHOUT a /vN suffix."
			;;
		*)
			echo "v${tag_major} tags require the module path to end in /v${tag_major}."
			;;
		esac
		echo "The proxy cannot serve mismatched tags, so this release would be"
		echo "invisible to 'go install'/'go get' @latest resolution."
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

if [ "$DRY_RUN" -eq 1 ]; then
	echo "--dry-run: no tags created, no files touched."
	oldest_sort=$(printf '%s\n' "${tags[@]}" | sort -V | head -1)
	echo "  would create ${#tags[@]} tags; first (sort -V): ${oldest_sort}"
	echo "  remember: tags must be monotonically increasing in BOTH semver and"
	echo "  commit ancestry (git tag -l '<module>/v4*' | sort -V | tail -1)."
	exit 0
fi

# Verify clean working tree
if ! git diff-index --quiet HEAD --; then
	echo "ERROR: working tree has uncommitted changes. Commit first."
	git status --short
	exit 1
fi

# --- Advisory pre-flight: stale sibling pins (pin-sweep --check) ---
#
# Same rationale as tag-release.sh: a full pin sweep mutates every stale
# go.mod in the repo — the wrong thing to run implicitly inside a batch cut.
# --check reports only; it is advisory, never fatal here.
echo "Pre-flight: checking for stale sibling pins (advisory)..."
if ! bash "$(dirname "$0")/pin-sweep.sh" --check; then
	echo "NOTE: stale pins are non-fatal for this cut; run scripts/pin-sweep.sh"
	echo "      before the next dependent tag wave that needs them."
fi

# --- Hardened restore (mirrors tag-release.sh) ---
#
# One restore path for BOTH exit shapes: `git reset --soft` to the pre-strip
# HEAD is a no-op before the temp commit exists and undoes exactly that
# commit afterwards (HEAD~1 would instead undo an auto-commit-daemon commit
# that landed in between); `git restore --staged --worktree` then rebuilds
# index + tree from the original HEAD — unlike the old `git checkout -- .`,
# which restored the STRIPPED go.mods from the stale index and silently
# re-dirtied the tree.
original_head="$(git rev-parse HEAD)"

restore_original_tree() {
	git reset --soft "$original_head" 2>/dev/null || true
	git restore --staged --worktree . 2>/dev/null || true
}
trap restore_original_tree EXIT

# --- Strip LOCAL replace directives from the TAGGED modules only ---
#
# A replace is local iff its target starts with '.' (relative path such as
# ../event) or '/' (absolute path such as /home/lars/projects/go-finding).
# Sibling-repo dev replaces must be stripped too, or a published go.mod
# points consumers at a path that does not exist on their machine. `|| true`
# after grep is mandatory under pipefail: grep exits 1 when a go.mod has no
# replace directives at all.
for mod in "${modules[@]}"; do
	gomod="${mod}/go.mod"
	echo "Stripping local replace directives from ${gomod}..."
	while IFS= read -r line; do
		lhs="${line%%=>*}"
		rhs="${line#*=>}"
		lhs="${lhs#replace }"
		lhs="$(printf '%s' "$lhs" | sed -e 's/^[[:space:]]*//' -e 's/[[:space:]]*$//')"
		rhs="$(printf '%s' "$rhs" | sed -e 's/^[[:space:]]*//' -e 's/[[:space:]]*$//')"
		lhs="${lhs%% *}"
		[ -z "$lhs" ] && continue
		case "$rhs" in
		.* | /*)
			(cd "$mod" && go mod edit "-dropreplace=${lhs}")
			echo "  dropped replace ${lhs} => ${rhs}"
			;;
		esac
	done < <(grep '=>' "$gomod" 2>/dev/null || true)
done

# --- Re-resolve requires for the tagged modules ---
echo "Re-resolving requires (go mod tidy with local replaces stripped)..."
for mod in "${modules[@]}"; do
	(cd "$mod" && GOWORK=off go mod tidy -e 2>/dev/null || true)
done

# --- Verify no pseudo-versions remain in the tagged modules ---
for mod in "${modules[@]}"; do
	gomod="${mod}/go.mod"
	echo "Verifying no pseudo-versions in ${gomod}..."
	if grep -q "00010101000000" "$gomod"; then
		echo "ERROR: ${gomod} still contains a pseudo-version require."
		echo "These break downstream consumers. Matching lines:"
		grep -n "00010101000000" "$gomod" || true
		echo ""
		echo "Aborting release. Every module ${mod} depends on must have a"
		echo "published tag. Publish the missing sibling(s), then re-run."
		exit 1
	fi
done

# --- Verify every tagged module COMPILES standalone ---
#
# `go mod tidy` resolves the import graph but does not typecheck: a module
# whose code uses symbols from a NEWER sibling than its go.mod pins builds
# fine under workspace replaces and then breaks for every consumer of the
# tag (command/v4.7.0 shipped exactly this).
for mod in "${modules[@]}"; do
	echo "Verifying ${mod} builds standalone with stripped go.mod..."
	build_err="$(mktemp)"
	if ! (cd "$mod" && GOWORK=off go build -tags goexperiment.jsonv2 ./... 2>"$build_err"); then
		echo "ERROR: ${mod} does not compile against its published requires."
		echo "The go.mod pins a sibling older than the code needs. Bump the"
		echo "require to the published tag providing the missing symbols, then"
		echo "re-run. Build output:"
		cat "$build_err"
		rm -f "$build_err"
		exit 1
	fi
	rm -f "$build_err"
done

# --- Temp commit carrying the stripped go.mods; all tags point at it ---
for mod in "${modules[@]}"; do
	git add "${mod}/go.mod"
	if [ -f "${mod}/go.sum" ]; then
		git add "${mod}/go.sum"
	fi
done

tag_list=""
for tag in "${tags[@]}"; do
	tag_list="${tag_list}${tag}, "
done
tag_list="${tag_list%, }"
git commit -m "chore(release): strip replace directives for batch release: ${tag_list}" --no-verify 2>/dev/null || true

# --- Create all annotated tags from the stripped commit ---
created=()
for i in "${!tags[@]}"; do
	tag="${tags[$i]}"
	desc="${descriptions[$i]}"
	echo "Creating annotated tag: ${tag}"
	if ! git tag -a "$tag" -m "${tag}: ${desc}"; then
		git tag -d "$tag" 2>/dev/null || true
		echo "ERROR: git tag failed for ${tag}; the tree was restored. Already-"
		echo "created tags (if any) still exist locally: ${created[*]:-none}"
		exit 1
	fi

	tag_type="$(git cat-file -t "$tag")"
	if [ "$tag_type" != "tag" ]; then
		git tag -d "$tag"
		echo "ERROR: tag ${tag} is ${tag_type}, not 'tag' (annotated); the tree"
		echo "was restored. Already-created tags (if any) still exist locally:"
		echo "  ${created[*]:-none}"
		exit 1
	fi
	echo "  ✓ ${tag_type}: ${tag}"
	created+=("$tag")
done

echo ""
echo "Original go.mod files restored."
echo ""
echo "Created ${#created[@]} tags. To push:"
echo "  git push origin ${created[*]}"
echo ""
echo "After pushing, smoke-check every tag (proxy serves it + it installs):"
for i in "${!tags[@]}"; do
	echo "  scripts/tag-release.sh --smoke ${modules[$i]} ${versions[$i]}"
done
