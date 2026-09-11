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
	echo "       $0 --smoke <module-path> <version>"
	echo "       $0 --audit"
	echo "Examples:"
	echo "  $0 event v4.0.1 \"Fix event payload marshaling\""
	echo "  $0 cmd/cqrs-lint v0.1.0 \"First release\""
	echo "  $0 metaengine v4.0.0 \"First release\" --dry-run"
	echo "  $0 --smoke cmd/cqrs-lint v4.10.0   # AFTER pushing the tag"
	echo "  $0 --audit                         # all-modules path-vs-tag audit"
}

# path_matches_major reports whether a module path is consistent with a tag
# version's major number: v0/v1 tags require a module path WITHOUT any /vN
# suffix; v2+ tags require the path to end in the matching /vN. Mismatched
# tags are INVISIBLE to the module proxy (the issue-#20 class), so the
# per-release guard below and `--audit` route through this one implementation.
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

# --- Post-cut proxy smoke-check (--smoke): proves proxy.golang.org serves
# the freshly pushed tag AND that the tag actually builds. The proxy fetches
# a tag on first request after the push (can lag seconds to ~a minute); the
# retry loop keeps `go list -m module@tag` going until the version resolves,
# so dependent modules never tidy against a tag the proxy has not absorbed
# (the tag-interleaving mechanic, AGENTS §Module Management). For modules
# whose root package is `main`, the check continues with a clean-dir
# `go install <module>@<tag>` + run — `go list` proves the proxy SERVES the
# version, not that it COMPILES, and the install probe is what catches the
# poisoned cqrs-lint v4.8.0 class (issue #20).
proxy_smoke_check() {
	local mod="$1"
	local ver="$2"
	local full_tag="${mod}/${ver}"

	local module_path
	module_path="$(grep '^module ' "${mod}/go.mod" | awk '{print $2}')"
	if [ -z "$module_path" ]; then
		echo "ERROR: cannot read module path from ${mod}/go.mod"
		exit 1
	fi

	if ! git ls-remote --exit-code origin "refs/tags/${full_tag}" >/dev/null 2>&1; then
		echo "ERROR: tag ${full_tag} is not pushed to origin — push first:"
		echo "  git push origin ${full_tag}"
		exit 1
	fi

	echo "Waiting for proxy.golang.org to serve ${module_path}@${ver} ..."
	local attempt
	for attempt in $(seq 1 12); do
		if env GOFLAGS='' GOPRIVATE='' GONOSUMDB='*' GONOSUMCHECK='*' \
			go list -m "${module_path}@${ver}" >/dev/null 2>&1; then
			echo "✓ proxy serves ${module_path}@${ver} (attempt ${attempt})"
			return 0
		fi
		sleep 10
	done

	echo "ERROR: proxy.golang.org still does not serve ${module_path}@${ver}"
	echo "after 12 attempts (~2 min). Check https://proxy.golang.org/ and the"
	echo "tag's ancestry; do NOT build dependent tags until this resolves."
	exit 1
}

# smoke_install_and_run installs module@ver from the PROXY into a clean
# GOBIN and runs it with --help. The install is a hard gate: a tag that
# fails standalone compilation ships broken to every consumer (the poisoned
# cqrs-lint v4.8.0 shipped `const version = 4.8.0`, unquoted — invisible to
# the tagger's pre-bump build, caught only by exactly this probe). A
# non-zero --help exit is a WARNING, not a failure: probe exit semantics
# are tool-specific (subcommand CLIs may reject the bare flag).
smoke_install_and_run() {
	local mod="$1"
	local ver="$2"

	if ! grep -qs '^package main$' "${mod}"/*.go; then
		echo "ℹ ${mod} has no root main package; install+run probe skipped"
		return 0
	fi

	local module_path
	module_path="$(awk '/^module /{print $2; exit}' "${mod}/go.mod")"

	local bin_name="${module_path%/v[0-9]*}"
	bin_name="${bin_name##*/}"
	local tmpbin
	tmpbin="$(mktemp -d)"

	echo "Clean-dir install probe: go install ${module_path}@${ver} ..."
	if ! env GOFLAGS='' GOPRIVATE='' GOBIN="$tmpbin" go install "${module_path}@${ver}"; then
		echo "ERROR: clean-dir install of ${module_path}@${ver} failed — the tag"
		echo "does not compile standalone. Do NOT advertise this release; fix and"
		echo "re-tag (this is the poisoned v4.8.0 failure class)."
		rm -rf "$tmpbin"
		return 1
	fi

	local rc=0
	"$tmpbin/${bin_name}" --help >/dev/null 2>&1 || rc=$?
	if [ "$rc" -eq 0 ]; then
		echo "✓ installed ${bin_name} runs (--help exited 0)"
	else
		echo "WARNING: installed ${bin_name} --help exited ${rc}. Install succeeded;"
		echo "eyeball a manual run before advertising the release."
	fi

	rm -rf "$tmpbin"
	return 0
}

# --- One-shot all-modules path-vs-tag audit (--audit) ---
#
# The issue-#20 class, repo-wide: a tag whose module path (as declared in
# the go.mod AT that tag) does not match the tag's major version can never
# be served by the module proxy — `@latest` silently resolves to an older,
# pre-suffix version instead (cmd/cqrs-lint shipped v4.2.0-v4.7.0 this way).
# This replays the same guard the per-release flow enforces over EVERY tag
# of EVERY module, so a whole history of invisible tags surfaces at once.
audit_all_tags() {
	local violations=0
	local checked=0
	local skipped=0
	local gomod dir tag_glob tag version path_at_tag

	while IFS= read -r gomod; do
		dir="${gomod#./}"
		dir="${dir%/go.mod}"

		# The root module (go.mod at ./go.mod) has BARE tags ("v4.0.0"); every
		# nested module's tags carry the "<dir>/" prefix.
		if [ -z "$dir" ]; then
			tag_glob='v*'
		else
			tag_glob="${dir}/*"
		fi

		while IFS= read -r tag; do
			[ -z "$tag" ] && continue
			# The "<dir>/*" glob also matches DEEPER nested tags (storage/*
			# matches storage/memory/v4.5.0); a module owns only tags of the
			# exact form "<dir>/<version>" — deeper ones belong to nested
			# modules and are audited under their own go.mod.
			case "${tag#"$dir"/}" in
			*/*) continue ;;
			esac
			version="${tag##*/}"
			# `|| true` INSIDE the substitution: with pipefail, a missing
			# file makes git show exit 128 and would otherwise abort via
			# set -e; putting the guard before awk (a || true | awk) would
			# instead short-circuit awk on SUCCESS and leak the raw go.mod.
			path_at_tag="$(git show "${tag}:${gomod}" 2>/dev/null | awk '/^module /{print $2; exit}' || true)"
			if [ -z "$path_at_tag" ]; then
				echo "SKIP  ${tag} (no go.mod at ${gomod} in the tagged tree)"
				skipped=$((skipped + 1))
				continue
			fi

			checked=$((checked + 1))
			if path_matches_major "$path_at_tag" "$version"; then
				continue
			fi
			echo "FAIL  ${tag} → module path ${path_at_tag} cannot serve ${version}"
			violations=$((violations + 1))
		done < <(git tag -l "$tag_glob")
	done < <(find . -name go.mod -not -path './vendor/*' -not -path './.git/*')

	echo ""
	echo "Audit: ${checked} tag(s) checked, ${violations} violation(s), ${skipped} skipped."
	if [ "$violations" -gt 0 ]; then
		echo "FAIL tags were never servable by the module proxy; @latest on the"
		echo "affected paths resolves to an older version instead. If the path is"
		echo "dead (superseded by a /vN path), ship a deprecation stub tag (see"
		echo "cmd/cqrs-lint/v0.2.1 and cmd/cqrs-bench/v0.1.1)."
		return 1
	fi
	return 0
}

if [ "${1:-}" = "--smoke" ]; then
	if [ $# -ne 3 ]; then
		usage
		exit 1
	fi
	proxy_smoke_check "$2" "$3"
	smoke_install_and_run "$2" "$3"
	exit 0
fi

if [ "${1:-}" = "--audit" ]; then
	if [ $# -ne 1 ]; then
		usage
		exit 1
	fi
	audit_all_tags
	exit $?
fi

# --- Parse args: peel off --dry-run / -h, keep positionals ---
dry_run=false
positionals=()
for arg in "$@"; do
	case "$arg" in
	--dry-run)
		dry_run=true
		;;
	-h | --help)
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

# --- Verify the module path's major version matches the tag ---
#
# The proxy refuses any tag whose major version does not match the module
# path declared in the go.mod at that tag. A v4 tag over a suffix-less
# module path is INVISIBLE to the proxy: @latest silently resolves to the
# newest pre-suffix version instead (cmd/cqrs-lint shipped v4.2.0-v4.7.0
# this way; `go install ...@latest` served v0.2.0 for years). v0/v1 tags
# require the opposite: no /vN suffix at all. The decision logic lives in
# path_matches_major (shared with --audit); only the messaging is local.
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
	echo "invisible to 'go install'/'go get' @latest resolution. Fix the"
	echo "module path (or pick the matching major version), then re-run."
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

# --- Advisory pre-flight: stale sibling pins (pin-sweep --check) ---
#
# A full pin sweep (even with --no-build) MUTATES every stale go.mod in the
# repo — the wrong thing to run implicitly inside a single-module cut, which
# this script deliberately scopes to one go.mod. The non-mutating inverse
# runs instead: --check reports pins older than the latest local tag. It is
# advisory, never fatal here (unrelated stale pins must not block a cut);
# the standalone build gate below is the hard stop for the failure mode
# that actually breaks consumers of THIS tag.
echo "Pre-flight: checking for stale sibling pins (advisory)..."
if ! bash "$(dirname "$0")/pin-sweep.sh" --check; then
	echo "NOTE: stale pins are non-fatal for this cut; run scripts/pin-sweep.sh"
	echo "      before the next dependent tag wave that needs them."
fi

# --- Restore helpers ---
#
# Two disjoint exit shapes:
#   • before any temp commit is made  → HEAD still points at the original
#     (pre-strip) commit, so `git restore --staged --worktree` from HEAD
#     brings the entire working tree back to the untouched state.
#   • after the temp commit is made    → HEAD points at the strip commit.
#     `undo_temp_commit` first moves HEAD back to original_head with
#     `reset --soft` (non-destructive: leaves the working tree alone), THEN
#     restores all tracked files.
#
# `git restore` is used instead of the old `git checkout -- .`, which
# restored the working tree from the INDEX — and after a `reset --soft` the
# index still held the stripped go.mod, silently re-dirtying the tree. The
# old script's "originals restored" message did not match reality.
#
# We restore ALL tracked files (not just go.mod/go.sum) because the auto-
# commit daemon may have committed other files between the temp commit and
# the reset, leaving staged deletions or modifications in the index.
# Restoring only go.mod/go.sum left those behind.

# Save the exact original HEAD before any modifications. Using HEAD~1 to
# undo the temp commit is fragile: if the auto-commit daemon commits between
# the temp commit and the reset, HEAD~1 points at the daemon's commit, not
# the pre-strip commit.
original_head="$(git rev-parse HEAD)"

restore_working_tree() {
	git restore --staged --worktree . 2>/dev/null || true
}

undo_temp_commit() {
	# Move HEAD back to the pre-strip commit (original go.mod), then discard
	# all staged/working-tree changes in favour of that original.
	git reset --soft "$original_head" 2>/dev/null || true
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
	.* | /*)
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

# (The old cmd/cqrs-lint version-constant bump block is gone: resolvedVersion()
# now reports the toolchain-embedded version (debug.ReadBuildInfo), so a cut
# no longer needs to mutate any source file — and the sed that poisoned
# v4.8.0 with an unquoted const cannot happen again.)

# --- Verify the stripped module actually COMPILES standalone ---
#
# `go mod tidy` resolves the import graph but does not typecheck: a module
# whose code uses symbols from a NEWER sibling than its go.mod pins builds
# fine under workspace replaces and then breaks for every consumer of the
# tag (command/v4.7.0 shipped exactly this — metadata.Metadata existed only
# at metadata/v4.5.0). One GOWORK=off build against the stripped go.mod
# proves the tagged go.mod is self-sufficient.
echo "Verifying ${module} builds standalone with stripped go.mod..."
build_err="$(mktemp)"
build_out="$(mktemp -d)"
build_ok=1
# -o into a throwaway dir: `go build ./...` writes main-package binaries
# into the module directory, silently dirtying the tree after the tag.
# -o refuses to compile library-only modules ("no main packages to
# build"), so those fall back to the plain build, which typechecks and
# discards — writing nothing either way.
if (cd "$module" && GOWORK=off go build -o "$build_out/" -tags goexperiment.jsonv2 ./... 2>"$build_err"); then
	:
elif grep -q "no main packages to build" "$build_err"; then
	if ! (cd "$module" && GOWORK=off go build -tags goexperiment.jsonv2 ./... 2>"$build_err"); then
		build_ok=0
	fi
else
	build_ok=0
fi
if [ "$build_ok" -eq 0 ]; then
	echo "ERROR: ${module} does not compile against its published requires."
	echo "The go.mod pins a sibling older than the code needs. Bump the"
	echo "require to the published tag providing the missing symbols, then"
	echo "re-run. Build output:"
	cat "$build_err"
	rm -f "$build_err"
	rm -rf "$build_out"
	restore_working_tree
	exit 1
fi
rm -f "$build_err"
rm -rf "$build_out"

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
echo "After pushing, verify the proxy serves it:"
echo "  $0 --smoke ${module} ${version}"
echo "Note: the tag points to a temporary commit that strips local replace"
echo "      directives. The temporary commit was undone locally; the original"
echo "      go.mod is restored."
