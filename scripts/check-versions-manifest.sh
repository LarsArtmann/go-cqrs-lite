#!/usr/bin/env bash
# check-versions-manifest.sh — module → latest published tag manifest (GitHub #27).
#
# Owns two committed artifacts, always regenerated together:
#   versions.json  — machine-readable manifest: {"modules": {"<dir>": "<latest tag>"}}
#                    Key "." is the repo-root module train; every other key is the
#                    repo-relative module dir (e.g. "stack/postgres"). Value is the
#                    FULL git tag (version = ${tag##*/}).
#   README.md      — the "<!-- versions-manifest:begin/end -->" section (a collapsed
#                    <details> table, human-readable cousin of the manifest).
#
# "Latest" = highest semver per module (sort -V), not most recent by commit date.
# The manifest lists every module prefix that EVER published a tag (105+ trains,
# including retired dirs the proxy still serves) — it is tag truth, not tree truth.
#
# UNTAG MARKING (ADR-0152): trains listed in scripts/untagged-trains.txt no
# longer cut releases, but their historical tags remain proxy-served tag truth,
# so they STAY in versions.json. Their README rows carry a "*(untagged per
# ADR-0152)*" note so humans do not read a frozen tag as a live train.
# versions.json stays pure (machine tag truth); the README carries policy.
#
# PHANTOM ROOT TRAIN: the "." row is the repo-root module (doc.go placeholder
# only). Its lone v4.0.0 tag is proxy-INVISIBLE (v4 tag on a suffix-less module
# path — the issue-#20 class), resolves for nobody, and per ADR-0152 will never
# be followed by another root tag. It is recorded because the manifest is tag
# truth, not because the train is alive.
#
# Modes:
#   bash scripts/check-versions-manifest.sh               # gate: local tags vs committed artifacts
#   bash scripts/check-versions-manifest.sh --check       # same, explicit
#   bash scripts/check-versions-manifest.sh --update      # regenerate both artifacts (local tags)
#   ... --remote                                          # tag source = origin (git ls-remote) instead
#                                                         #   of local tags — the nightly freshness leg:
#                                                         #   fails when a published tag is missing from
#                                                         #   the manifest OR the manifest cites a tag
#                                                         #   that is not on origin (unpushed/deleted)
#   bash scripts/check-versions-manifest.sh --self-test   # hermetic fixture suite (incl. stale-manifest
#                                                         #   mutation legs)
#
# Gate semantics are strict equality (regenerated == committed, both artifacts):
# a manifest that lags a tag AND a manifest that cites an unpushed tag are both
# drift worth surfacing — the same spirit as check-retracts-shipped.sh.
set -euo pipefail

MANIFEST="versions.json"
README="README.md"
BEGIN_SENTINEL='<!-- versions-manifest:begin -->'
END_SENTINEL='<!-- versions-manifest:end -->'

MODE="check"
REMOTE=0

# ── Self-test (repo convention: CI-gating scripts ship with hermetic suites) ─
# Fixture: throwaway git repo + planted tags, the script run against it via cwd.
# Pins: sort -V latest-pick (v4.10.0 > v4.9.0), prerelease ordering (v2.0.0 >
# v2.0.0-rc1), root-train "." key, exact manifest golden, README table render,
# and BOTH mutation legs — stale manifest must fail, unmanifested new tag must
# fail (--update then heals both) — plus the untag-marker legs: README note
# rendered, versions.json purity, and a stripped marker failing --check.
if [ "${1:-}" = "--self-test" ]; then
	SELF="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/$(basename "${BASH_SOURCE[0]}")"
	TMP="$(mktemp -d)"
	trap 'rm -rf "$TMP"' EXIT

	# Hermetic untag lookups: the default list is the REAL repo policy file,
	# and the fixture's "." root train would inherit the real root's untag
	# marker. Pin every early leg to an empty list; the marker legs below
	# override with their own fixture list.
	: >"$TMP/untagged-none.txt"
	export UNTAGGED_TRAINS_FILE="$TMP/untagged-none.txt"

	fixture() {
		rm -rf "$TMP/repo"
		mkdir -p "$TMP/repo"
		git -C "$TMP/repo" init -q
		git -C "$TMP/repo" config user.email t@t && git -C "$TMP/repo" config user.name t
		cat >"$TMP/repo/README.md" <<RMD
# fixture

${BEGIN_SENTINEL}
${END_SENTINEL}
RMD
		tick() { git -C "$TMP/repo" commit -q --allow-empty -m "t$1"; }
		tag() { git -C "$TMP/repo" tag -a "$1" -m "$1"; }
		tick 1
		tag alpha/v4.2.0
		tick 2
		tag alpha/v4.9.0
		tick 3
		tag alpha/v4.10.0
		tick 4
		tag beta/nested/v1.2.3
		tick 5
		tag v2.0.0-rc1
		tick 6
		tag v2.0.0
	}

	fails=0
	check() {
		if [ "$3" = "$2" ]; then
			echo "  ✓ PASS: $1"
		else
			echo "  ✗ FAIL: $1 (want rc=$2, got rc=$3)"
			fails=$((fails + 1))
		fi
	}

	fixture
	rc=0
	(cd "$TMP/repo" && bash "$SELF" --update) >/dev/null 2>&1 || rc=$?
	check "update runs green on fresh fixture" 0 "$rc"

	cat >"$TMP/want.json" <<WANT
{
  "modules": {
    ".": "v2.0.0",
    "alpha": "alpha/v4.10.0",
    "beta/nested": "beta/nested/v1.2.3"
  }
}
WANT
	if diff -u "$TMP/want.json" "$TMP/repo/versions.json" >/dev/null; then
		echo "  ✓ PASS: manifest golden (sort -V pick, prerelease order, . key)"
	else
		echo "  ✗ FAIL: manifest golden mismatch"
		diff -u "$TMP/want.json" "$TMP/repo/versions.json" || true
		fails=$((fails + 1))
	fi
	grep -q '| alpha | `alpha/v4.10.0` |' "$TMP/repo/README.md" &&
		grep -qF '| . | `v2.0.0` |' "$TMP/repo/README.md" &&
		echo "  ✓ PASS: README table rendered between sentinels" || {
		echo "  ✗ FAIL: README table render"
		fails=$((fails + 1))
	}

	rc=0
	(cd "$TMP/repo" && bash "$SELF" --check) >/dev/null 2>&1 || rc=$?
	check "check green right after update" 0 "$rc"

	sed -i 's|alpha/v4.10.0|alpha/v4.9.0|' "$TMP/repo/versions.json"
	rc=0
	(cd "$TMP/repo" && bash "$SELF" --check) >/dev/null 2>&1 || rc=$?
	check "MUTATION stale manifest fails" 1 "$rc"

	sed -i 's|alpha/v4.9.0|alpha/v4.10.0|' "$TMP/repo/versions.json"
	git -C "$TMP/repo" commit -q --allow-empty -m t7
	git -C "$TMP/repo" tag -a alpha/v4.11.0 -m alpha/v4.11.0
	rc=0
	(cd "$TMP/repo" && bash "$SELF" --check) >/dev/null 2>&1 || rc=$?
	check "MUTATION unmanifested new tag fails" 1 "$rc"

	rc=0
	(cd "$TMP/repo" && bash "$SELF" --update) >/dev/null 2>&1 || rc=$?
	check "update heals after new tag" 0 "$rc"
	grep -q 'alpha/v4.11.0' "$TMP/repo/versions.json" &&
		echo "  ✓ PASS: healed manifest carries new tag" || {
		echo "  ✗ FAIL: healed manifest missing new tag"
		fails=$((fails + 1))
	}

	# UNTAG MARKING (ADR-0152): untagged trains stay in versions.json (pure
	# tag truth) but their README rows carry the policy note. Mutation legs
	# pin both directions: a stripped marker must fail --check.
	printf '# fixture untag list\nalpha\n' >"$TMP/untagged.txt"
	rc=0
	(cd "$TMP/repo" && UNTAGGED_TRAINS_FILE="$TMP/untagged.txt" bash "$SELF" --update) >/dev/null 2>&1 || rc=$?
	check "update green with untag list present" 0 "$rc"
	grep -qF '| alpha | `alpha/v4.11.0` *(untagged per ADR-0152)* |' "$TMP/repo/README.md" &&
		echo "  ✓ PASS: README row carries the untag note" || {
		echo "  ✗ FAIL: README row missing the untag note"
		fails=$((fails + 1))
	}
	grep -qF '"alpha": "alpha/v4.11.0"' "$TMP/repo/versions.json" &&
		echo "  ✓ PASS: versions.json stays pure tag truth (no policy text)" || {
		echo "  ✗ FAIL: versions.json drifted from pure tag truth"
		fails=$((fails + 1))
	}
	rc=0
	(cd "$TMP/repo" && UNTAGGED_TRAINS_FILE="$TMP/untagged.txt" bash "$SELF" --check) >/dev/null 2>&1 || rc=$?
	check "check green with marker in place" 0 "$rc"

	sed -i 's/`alpha\/v4.11.0` \*(untagged per ADR-0152)\*/`alpha\/v4.11.0`/' "$TMP/repo/README.md"
	rc=0
	(cd "$TMP/repo" && UNTAGGED_TRAINS_FILE="$TMP/untagged.txt" bash "$SELF" --check) >/dev/null 2>&1 || rc=$?
	check "MUTATION stripped untag marker fails" 1 "$rc"

	(cd "$TMP/repo" && UNTAGGED_TRAINS_FILE="$TMP/untagged.txt" bash "$SELF" --update) >/dev/null 2>&1

	if [ "$fails" -eq 0 ]; then
		echo "check-versions-manifest self-test passed."
		exit 0
	fi
	echo "check-versions-manifest self-test FAILED."
	exit 1
fi

for arg in "$@"; do
	case "$arg" in
	--check) MODE="check" ;;
	--update) MODE="update" ;;
	--remote) REMOTE=1 ;;
	*)
		echo "usage: $0 [--check|--update] [--remote] | --self-test" >&2
		exit 2
		;;
	esac
done

cd "$(git rev-parse --show-toplevel)"

# is_untagged lives in the shared release lib (single implementation with
# tag-release.sh/batch-release.sh; override path via UNTAGGED_TRAINS_FILE in
# the self-test fixture).
# shellcheck disable=SC1091 # sourced release lib lives beside this script
source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib/release_common.sh"

# Tag source: local refs (dev + tag-time refresh) or origin (nightly freshness).
list_tags() {
	if [ "$REMOTE" = 1 ]; then
		git ls-remote --tags --refs origin | awk '{print $2}' | sed 's|^refs/tags/||'
	else
		git tag --list
	fi | grep -E '^([A-Za-z0-9._/-]+/)?v[0-9]+(\.[0-9]+)*(-.*)?$' || true
}

# Semver-aware latest pick. Plain sort -V ranks v2.0.0-rc1 ABOVE v2.0.0 (it
# treats the suffix as "later"), violating semver — compare base versions
# first, then prefer the release (no prerelease suffix) on a base tie.
newer_than() { # newer_than <candidate> <incumbent> → rc 0 iff candidate wins
	local cand="$1" incumbent="$2" cand_base inc_base max_base
	cand_base="${cand%%-*}"
	inc_base="${incumbent%%-*}"
	if [ "$cand_base" != "$inc_base" ]; then
		max_base="$(printf '%s\n%s\n' "$cand_base" "$inc_base" | sort -V | tail -1)"
		[ "$cand_base" = "$max_base" ]
		return
	fi
	case "$cand" in
	*-*) case "$incumbent" in
		*-*) [ "$(printf '%s\n%s\n' "$cand" "$incumbent" | sort -V | tail -1)" = "$cand" ] ;; # both prereleases: -V orders rc1 < rc2
		*) return 1 ;;                                                                        # candidate is the prerelease, incumbent the release
		esac ;;
	*) case "$incumbent" in
		*-*) return 0 ;;                                                                    # candidate is the release, incumbent the prerelease
		*) [ "$(printf '%s\n%s\n' "$cand" "$incumbent" | sort -V | tail -1)" = "$cand" ] ;; # both releases
		esac ;;
	esac
}

# Latest tag per module train. "." = the root train (tags without a slash).
declare -A latest
while IFS= read -r tag; do
	module="${tag%/*}"
	[ "$module" = "$tag" ] && module="."
	if ! [[ "$module" =~ ^[A-Za-z0-9._/-]+$ ]]; then
		echo "::error::module path '$module' (from tag $tag) has unexpected characters — refusing to emit JSON" >&2
		exit 1
	fi
	ver="${tag##*/}"
	prev="${latest[$module]:-}"
	if [ -z "$prev" ] || newer_than "$ver" "${prev##*/}"; then
		latest[$module]="$tag"
	fi
done < <(list_tags)

if [ "${#latest[@]}" -eq 0 ]; then
	echo "::error::no semver tags found ($([ "$REMOTE" = 1 ] && echo 'on origin' || echo 'locally')) — refusing to write an empty manifest" >&2
	exit 1
fi

render_manifest() {
	echo '{'
	echo '  "modules": {'
	first=1
	for module in $(printf '%s\n' "${!latest[@]}" | LC_ALL=C sort); do
		if [ "$first" = 1 ]; then first=0; else printf ',\n'; fi
		printf '    "%s": "%s"' "$module" "${latest[$module]}"
	done
	printf '\n'
	echo '  }'
	echo '}'
}

render_readme_table() {
	echo '| Module | Latest published tag |'
	echo '| --- | --- |'
	for module in $(printf '%s\n' "${!latest[@]}" | LC_ALL=C sort); do
		note=""
		if is_untagged "$module"; then
			note=' *(untagged per ADR-0152)*'
		fi
		printf '| %s | `%s`%s |\n' "$module" "${latest[$module]}" "$note"
	done
}

if [ "$MODE" = "update" ]; then
	render_manifest >"$MANIFEST"
	if ! grep -qF "$BEGIN_SENTINEL" "$README" || ! grep -qF "$END_SENTINEL" "$README"; then
		echo "::error::$README lacks the versions-manifest sentinels — add the section scaffold (see scripts/check-versions-manifest.sh header) instead of appending blindly." >&2
		exit 1
	fi
	tmp_readme=$(mktemp)
	trap 'rm -f "$tmp_readme"' EXIT
	# Splice: everything up to and including the begin sentinel, then the fresh
	# table, then everything from the end sentinel on.
	awk -v begin="$BEGIN_SENTINEL" '
		index($0, begin) > 0 { print; exit }
		{ print }' "$README" >"$tmp_readme"
	render_readme_table >>"$tmp_readme"
	awk -v end="$END_SENTINEL" '
		start { print; next }
		index($0, end) > 0 { start = 1; print }' "$README" >>"$tmp_readme"
	mv "$tmp_readme" "$README"
	echo "✓ wrote $MANIFEST (${#latest[@]} module trains) + refreshed README section ($([ "$REMOTE" = 1 ] && echo 'from origin' || echo 'from local tags'))"
	exit 0
fi

# --check: strict equality, both artifacts, diff shown for actionable failures.
rc=0
gen=$(mktemp)
trap 'rm -f "$gen"' EXIT
render_manifest >"$gen"
if ! diff -u "$MANIFEST" "$gen" >"$gen.diff"; then
	echo "::error::$MANIFEST is stale vs $([ "$REMOTE" = 1 ] && echo 'origin' || echo 'local') tags."
	echo "::error::If a tag row is MISSING below (-), publish flow skipped the refresh — run: bash scripts/check-versions-manifest.sh --update"
	echo "::error::If a tag row is EXTRA below (+), it is not on the tag source (unpushed or deleted tag) — push it or regen after removal."
	sed 's/^/  /' "$gen.diff"
	rm -f "$gen.diff"
	rc=1
else
	rm -f "$gen.diff"
fi
gen_table=$(mktemp)
render_readme_table >"$gen_table"
if ! awk -v begin="$BEGIN_SENTINEL" -v end="$END_SENTINEL" '
	index($0, begin) > 0 { in_sec = 1; next }
	index($0, end) > 0 { in_sec = 0; next }
	in_sec { print }' "$README" >"$gen.sec"; then
	echo "::error::failed to extract README section"
	rc=1
elif ! diff -u "$gen.sec" "$gen_table" >"$gen.tdiff"; then
	echo "::error::README versions section is stale vs $MANIFEST — run: bash scripts/check-versions-manifest.sh --update"
	sed 's/^/  /' "$gen.tdiff"
	rm -f "$gen.tdiff"
	rc=1
else
	rm -f "$gen.tdiff"
fi
rm -f "$gen_table" "$gen.sec"
if [ "$rc" = 0 ]; then
	echo "✅ versions manifest fresh (${#latest[@]} module trains, $([ "$REMOTE" = 1 ] && echo 'origin' || echo 'local') tags)."
fi
exit "$rc"
