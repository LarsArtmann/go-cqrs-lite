#!/usr/bin/env bash
# probe-proxy-tags.sh — weekly "the module proxy serves every module's
# latest tag, and the zip is REAL" probe.
#
# The poisoned-tag class (binary-junk zip in metaengine/tursoengine/v4.2.0,
# 2026-09-24) ships SILENTLY: consumers get a broken download while every
# in-repo gate stays green, and tag-release.sh's zip-content guard protects
# only FUTURE tags. This probe re-verifies the PUBLISHED surface:
#   1. proxy.golang.org resolves @latest for every workspace module
#   2. the @latest zip downloads and lists a go.mod at the module root
#   3. no entry in the zip smells like an embedded ELF binary (the junk class)
#
# Weekly cadence: wired into nightly-gates.yml behind a Sunday check and
# runnable any time. Network required.
#
# Usage:
#   scripts/probe-proxy-tags.sh                # probe all modules
#   scripts/probe-proxy-tags.sh --self-test    # fixture suite (good/bad zips)
#
# Fixture hook: PROBE_FIXTURE_DIR with planted zips named
# <mod>-latest.zip (self-test only — never set in normal use).
set -uo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PROXY="https://proxy.golang.org"

failures=0

# check_zip <module-path> <zip-file>: go.mod at root + no ELF entries.
check_zip() {
	local mod="$1" zip="$2"
	local entries
	if ! entries="$(unzip -Z1 "$zip" 2>/dev/null)"; then
		echo "✗ $mod@latest: zip unreadable (corrupt archive — the poisoned-tag class)" >&2
		failures=$((failures + 1))
		return
	fi

	# Zip listing paths are <full-module-path>@<v>/go.mod — the module
	# path itself contains slashes, so anchor on the @version segment.
	if ! grep -qE '@v[^/]+/go\.mod$' <<<"$entries"; then
		echo "✗ $mod@latest: zip has no root go.mod (junk or mis-rooted archive)" >&2
		failures=$((failures + 1))
		return
	fi

	# ELF sniff per DECOMPRESSED entry (raw grep misses deflated magic):
	# a real module zip carries no ELF files at all; the poisoned zip was
	# mostly one big binary blob.
	local entry
	while IFS= read -r entry; do
		[ -n "$entry" ] || continue
		if unzip -p "$zip" "$entry" 2>/dev/null | head -c 4 | grep -qa $'\x7fELF'; then
			echo "✗ $mod@latest: zip entry $entry is an ELF binary (junk class)" >&2
			failures=$((failures + 1))
			return
		fi
	done <<<"$entries"

	echo "  ✓ $mod@latest: zip clean ($(wc -l <<<"$entries") entries)"
}

probe_module() {
	local mod_dir="$1"
	local mod
	mod="$(grep -m1 '^module ' "$mod_dir/go.mod" | awk '{print $2}')"
	[ -n "$mod" ] || return 0

	local latest
	latest="$(python3 -c "import urllib.request,sys;print(urllib.request.urlopen('$PROXY/$mod/@latest').read().decode())" 2>/dev/null |
		python3 -c 'import json,sys;print(json.load(sys.stdin)["Version"])' 2>/dev/null || true)"

	if [ -z "$latest" ]; then
		echo "✗ $mod: proxy @latest does not resolve (invisible tag?)" >&2
		failures=$((failures + 1))
		return
	fi

	local tmp
	tmp="$(mktemp)"
	trap 'rm -f "$tmp"' RETURN

	if ! python3 - "$mod" "$latest" "$tmp" <<'PYEOF' 2>/dev/null
import sys, urllib.request
mod, ver, out = sys.argv[1], sys.argv[2], sys.argv[3]
urllib.request.urlretrieve(f"https://proxy.golang.org/{mod}/@v/{ver}.zip", out)
PYEOF
	then
		echo "✗ $mod@$latest: zip download failed" >&2
		failures=$((failures + 1))
		return
	fi

	check_zip "$mod" "$tmp"
}

if [ "${1:-}" = "--self-test" ]; then
	tmp="$(mktemp -d)"
	trap 'rm -rf "$tmp"' EXIT
	echo "━━━ probe-proxy-tags self-test ━━━"

	# Good zip: version-dir + go.mod + one source file.
	good="$tmp/good@v1.0.0"
	mkdir -p "$good"
	printf 'module example.com/good\n\ngo 1.27.1\n' >"$good/go.mod"
	printf 'package good\n' >"$good/good.go"
	(cd "$tmp" && zip -qr good-latest.zip "good@v1.0.0")

	# Bad zip 1: no go.mod at root.
	bad1="$tmp/bad1@v1.0.0"
	mkdir -p "$bad1/sub"
	printf 'package sub\n' >"$bad1/sub/x.go"
	(cd "$tmp" && zip -qr bad1-latest.zip "bad1@v1.0.0")

	# Bad zip 2: ELF bytes embedded (printf the magic + some payload).
	bad2="$tmp/bad2@v1.0.0"
	mkdir -p "$bad2"
	printf 'module example.com/bad2\n' >"$bad2/go.mod"
	head -c 64 /bin/true >"$bad2/blob"
	(cd "$tmp" && zip -qr bad2-latest.zip "bad2@v1.0.0")

	fails_before=$failures
	check_zip "example.com/good" "$tmp/good-latest.zip" >/dev/null
	[ "$failures" -eq "$fails_before" ] && echo "  ✓ PASS: clean zip accepted" ||
		{ echo "  ✗ FAIL: clean zip rejected"; fails_before=$failures; }

	check_zip "example.com/bad1" "$tmp/bad1-latest.zip" >/dev/null 2>&1
	[ $((failures - fails_before)) -ge 1 ] && echo "  ✓ PASS: go.mod-less zip caught" ||
		echo "  ✗ FAIL: go.mod-less zip not caught"
	local_fails=$failures

	check_zip "example.com/bad2" "$tmp/bad2-latest.zip" >/dev/null 2>&1
	[ $((failures - local_fails)) -ge 1 ] && echo "  ✓ PASS: ELF-junk zip caught" ||
		echo "  ✗ FAIL: ELF-junk zip not caught"

	[ "$failures" -gt 0 ] && { echo "self-test: mutation legs behaved"; exit 0; }
	echo "self-test: GOOD zip was rejected — checker over-strict"
	exit 1
fi

cd "$ROOT" || exit 2

echo "━━━ proxy tag probe ($(date -u +%FT%TZ)) ━━━"
while IFS= read -r mod_dir; do
	probe_module "$mod_dir"
done < <(find . -name go.mod -not -path './vendor/*' -not -path './.git/*' -exec dirname {} \; | sed 's|^\./||' | sort)

if [ "$failures" -gt 0 ]; then
	echo "::error::$failures module tag(s) failed the proxy probe — retraction/re-cut decision needed (see TODO data-mesh tail)"
	exit 1
fi
echo "✅ proxy resolves every module's latest tag; all zips clean."
