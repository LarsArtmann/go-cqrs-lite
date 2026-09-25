#!/usr/bin/env bash
# check-example-standalone.sh — every example must be buildable STANDALONE
# (GOWORK=off, published pins only, no sibling replace leakage).
#
# The B6f class: an example's go.mod carried a replace pinning a sibling at
# an UNPUBLISHED version; the wave's forward-pin abort surfaced only at tag
# time. This audit fails when any example/go.mod has LOCAL (path) replaces
# it should not have, or requires a sibling version no published tag serves.
#
# Allowed (documented) exceptions:
#   - pre-release replace stripping is a WAVE decision (mesh-demo's
#     `replace ../../catalog` is deliberate until the catalog tag lands);
#     such cases are baselined below with their reason and MUST carry the
#     same comment in their go.mod.
#
# Usage:
#   bash scripts/check-example-standalone.sh           # audit
#   bash scripts/check-example-standalone.sh --build   # + GOWORK=off build each
#   bash scripts/check-example-standalone.sh --self-test
set -uo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

# Deliberate local replaces (module => reason). Keep SHORT: each entry dies
# at the tag wave that obsoletes it.
ALLOWED_REPLACES=''

DO_BUILD=0
if [ "${1:-}" = "--build" ]; then
	DO_BUILD=1
elif [ "${1:-}" = "--self-test" ]; then
	tmp="$(mktemp -d)"
	trap 'rm -rf "$tmp"' EXIT
	echo "━━━ check-example-standalone self-test ━━━"
	# Fixture: an example with a sneaky path replace.
	mkdir -p "$tmp/example/bad" "$tmp/example/good"
	printf 'module example.com/bad\n\ngo 1.27.1\n\nrequire x/y v1.0.0\n\nreplace x/y => ../../y\n' >"$tmp/example/bad/go.mod"
	printf 'module example.com/good\n\ngo 1.27.1\n' >"$tmp/example/good/go.mod"

	out="$(cd "$tmp" && bash "$ROOT/scripts/check-example-standalone.sh" 2>&1)"
	rc=$?
	echo "$out" | grep -q "example/bad" && echo "  ✓ PASS: path-replace example flagged" ||
		echo "  ✗ FAIL: path-replace example not flagged"
	[ "$rc" -eq 1 ] && echo "  ✓ PASS: audit exits nonzero" || echo "  ✗ FAIL: audit should exit 1"
	rm -rf "$tmp/example/bad"
	out="$(cd "$tmp" && bash "$ROOT/scripts/check-example-standalone.sh" 2>&1)"
	[ $? -eq 0 ] && echo "  ✓ PASS: clean example set passes" || echo "  ✗ FAIL: clean set should pass"
	exit 0
fi

failures=0
for gomod in "$ROOT"/example/*/go.mod; do
	example_dir="$(dirname "$gomod")"
	name="$(basename "$example_dir")"

	# Leg 1: no local path replaces (the standalone killer).
	while IFS= read -r line; do
		[ -z "$line" ] && continue
		if echo "$ALLOWED_REPLACES" | grep -qF "[$name]"; then
			continue
		fi
		echo "✗ example/$name carries a local replace — standalone builds cannot resolve it:" >&2
		echo "    $line" >&2
		echo "  (strip at the wave that publishes the sibling, or baseline it above WITH reason)" >&2
		failures=$((failures + 1))
	done < <(grep -E '^replace .* =>' "$gomod" | grep -E '=> \.\.?/' || true)

	# Leg 2 (--build): GOWORK=off compile against published pins only.
	if [ "$DO_BUILD" = 1 ]; then
		if ! (cd "$example_dir" && GOWORK=off go build ./... >/dev/null 2>&1); then
			echo "✗ example/$name does not build standalone (GOWORK=off)" >&2
			failures=$((failures + 1))
		else
			echo "  ✓ example/$name standalone build OK"
		fi
	fi
done

echo "example-standalone audit: $failures finding(s)"
[ "$failures" -eq 0 ] || exit 1
exit 0
