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
#   - After pushing, smoke-check every tag: scripts/tag-release.sh --smoke,
#     or one-shot for a whole wave: batch-release.sh --smoke-all <file>.
#
# SAME-BATCH SIBLING LIMITATION (documented decision): within one batch,
# `go mod tidy -e` resolves a module's sibling requires to the sibling's
# latest PUBLISHED tag — the tag being cut for that sibling in THIS batch
# does not exist yet (no network round-trip can see an unpushed tag). A
# batch therefore ships dependents pinned to the sibling's PREVIOUS version.
# Cut in dependency order across SEPARATE invocations (lowest-level module
# first, push, smoke, then dependents) when a dependent must pin the new
# version; use one batch only when today's pins are already correct.
#
# A `--verify` full-pipeline dry-run (strip + tidy + build against the real
# stripped go.mods, then restore) was CONSIDERED AND DECLINED 2026-09-13:
# --dry-run already covers every non-mutating guard, and the mutating half
# duplicates the real cut with the same restore machinery — more surface,
# no new failure class caught. Revisit only if a batch ever fails mid-cut
# in a way a pre-pass would have caught.
#
# Usage:
#   ./scripts/batch-release.sh [--dry-run] "<module> <version> <description>" ...
#   ./scripts/batch-release.sh --audit
#   ./scripts/batch-release.sh --smoke-all <file>
#
# --dry-run prints the tags that WOULD be created (existence, tag collision,
# path-vs-tag guard, sequence checks) without touching go.mod files, the
# tree, or the repo.
#
# --audit replays the path-vs-tag guard over EVERY tag of EVERY module
# (delegates to tag-release.sh --audit — the single implementation).
#
# --smoke-all <file> post-push verification for a whole wave: <file> holds
# one "<module> <version>" pair per line (# comments allowed); each line
# runs tag-release.sh --smoke (proxy serves the tag + clean-dir install
# probe). Stops at the first hard failure — dependent tags must not be
# advertised while an earlier tag in the wave is not proxy-servable — and
# exits nonzero if any line failed.
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
	echo "       $0 --smoke-all <file-with-module-version-lines>"
	echo "Example:"
	echo "  $0 \"event v4.0.3 Patch release\" \"command v4.0.1 Patch release\""
	echo "  $0 --dry-run \"event v4.0.3 Patch release\""
	echo "  $0 --audit   # all-modules path-vs-tag audit (via tag-release.sh)"
}

# path_matches_major, module_has_root_main and smoke_probe_args live in
# scripts/lib/release_common.sh (single implementation shared with
# tag-release.sh — the issue-#20 guard must never fork again).
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck disable=SC1091 # sourced release lib lives beside this script
source "${SCRIPT_DIR}/lib/release_common.sh"

if [ "${1:-}" = "--audit" ]; then
	if [ $# -ne 1 ]; then
		usage
		exit 1
	fi
	exec bash "$(dirname "$0")/tag-release.sh" --audit
fi

if [ "${1:-}" = "--smoke-all" ]; then
	if [ $# -ne 2 ]; then
		usage
		exit 1
	fi

	probes_file="$2"
	if [ ! -f "$probes_file" ]; then
		echo "ERROR: probes file ${probes_file} does not exist"
		exit 1
	fi

	total=0
	failed=0
	while IFS= read -r line || [ -n "$line" ]; do
		case "$line" in
		\#* | "") continue ;;
		esac
		mod="$(printf '%s' "$line" | awk '{print $1}')"
		ver="$(printf '%s' "$line" | awk '{print $2}')"
		if [ -z "$mod" ] || [ -z "$ver" ]; then
			echo "ERROR: malformed line in ${probes_file}: \"${line}\""
			echo "Expected: \"<module> <version>\""
			exit 1
		fi
		total=$((total + 1))
		echo ""
		echo "━━━ smoke ${mod} ${ver} ━━━"
		if ! bash "${SCRIPT_DIR}/tag-release.sh" --smoke "$mod" "$ver"; then
			echo "✗ ${mod} ${ver} FAILED — stopping: dependent tags in this wave"
			echo "  must not be advertised while this tag is broken."
			failed=$((failed + 1))
			break
		fi
	done <"$probes_file"

	if [ "$total" -eq 0 ]; then
		echo 'ERROR: no "<module> <version>" lines in '"${probes_file}"
		exit 1
	fi
	if [ "$failed" -gt 0 ]; then
		exit 1
	fi
	echo ""
	echo "✓ all ${total} tag(s) smoke-checked."
	exit 0
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

	case "$version" in
	v[0-9]*) : ;;
	*)
		echo "ERROR: malformed triple \"${arg}\""
		echo "Each argument must be ONE quoted string: \"<module> <version> <description>\""
		echo "(e.g. \"event v4.0.3 Patch release\") — got version \"${version}\"."
		exit 1
		;;
	esac

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
	build_out="$(mktemp -d)"
	build_ok=1
	# -o into a throwaway dir: `go build ./...` writes main-package binaries
	# into the module directory, silently dirtying the tree after the tag.
	# -o refuses to compile library-only modules ("no main packages to
	# build"), so those fall back to the plain build, which typechecks and
	# discards — writing nothing either way.
	if (cd "$mod" && GOWORK=off go build -o "$build_out/" ./... 2>"$build_err"); then
		:
	elif grep -q "no main packages to build" "$build_err"; then
		if ! (cd "$mod" && GOWORK=off go build ./... 2>"$build_err"); then
			build_ok=0
		fi
	else
		build_ok=0
	fi
	if [ "$build_ok" -eq 0 ]; then
		echo "ERROR: ${mod} does not compile against its published requires."
		echo "The go.mod pins a sibling older than the code needs. Bump the"
		echo "require to the published tag providing the missing symbols, then"
		echo "re-run. Build output:"
		cat "$build_err"
		rm -f "$build_err"
		rm -rf "$build_out"
		exit 1
	fi
	rm -f "$build_err"
	rm -rf "$build_out"
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
