#!/usr/bin/env bash
# check-libreports.sh — gate for the fleet-libs audit artifacts:
#   leg 1: every path:line citation in docs/review/libs/*.md resolves against
#          the consumer repo and the sibling library checkouts (citation-sweep.py)
#   leg 2: docs/review/libs/index.html links every report file and the render
#          asset, and no report is orphaned
#
# Local gate by construction: leg 1 resolves against ~/projects checkouts that
# CI runners do not have. Self-test: LIBREPORTS_ROOT re-points both legs at a
# fixture tree with planted faults (a stale citation, a dangling index link).
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="${LIBREPORTS_ROOT:-$(cd "$SCRIPT_DIR/.." && pwd)}"

fail() {
	echo "✗ libreports check FAILED: $*" >&2
	exit 1
}

if [[ "${1:-}" == "--self-test" ]]; then
	tmp="$(mktemp -d)"
	trap 'rm -rf "$tmp"' EXIT
	mkdir -p "$tmp/docs/review/libs" "$tmp/consumer"

	cat >"$tmp/consumer/file.go" <<'EOF'
package x

// line 4 exists so citations at :1 and :4 resolve; :99 does not.
// line 5 pads the file so the :4 repair stays in range.
// line 6 pads too.
EOF

	cat >"$tmp/docs/review/libs/good.md" <<'EOF'
# good
- consumer/file.go:1 works
EOF

	cat >"$tmp/docs/review/libs/bad.md" <<'EOF'
# bad
- consumer/file.go:99 stale
EOF

	cat >"$tmp/docs/review/libs/index.html" <<'EOF'
<a href="good.md">good</a>
<a href="missing.md">dangling</a>
<img src="../assets/2026-11-22_fleet-libs-audit-index.png">
EOF
	mkdir -p "$tmp/docs/assets"
	touch "$tmp/docs/assets/2026-11-22_fleet-libs-audit-index.png"

	# Expect failure: stale citation (leg 1) and dangling link (leg 2).
	if SELF_TEST=0 LIBREPORTS_ROOT="$tmp" bash "$0" >/dev/null 2>&1; then
		fail "self-test: planted faults did not trip the gate"
	fi

	# Green control: repair the faults, expect success.
	sed -i 's/file.go:99/file.go:4/' "$tmp/docs/review/libs/bad.md"
	sed -i 's/missing.md/bad.md/' "$tmp/docs/review/libs/index.html"

	if ! SELF_TEST=0 LIBREPORTS_ROOT="$tmp" bash "$0" >/dev/null 2>&1; then
		fail "self-test: repaired fixture still failed"
	fi

	echo "✓ check-libreports self-test green"
	exit 0
fi

# ── leg 1: citations ────────────────────────────────────────────────────────
if ! python3 "$SCRIPT_DIR/citation-sweep.py" --root "$ROOT"; then
	fail "stale citations (re-run scripts/citation-sweep.py after report edits)"
fi

# ── leg 2: index ↔ report links ────────────────────────────────────────────
libs_dir="$ROOT/docs/review/libs"
index="$libs_dir/index.html"

[[ -f "$index" ]] || fail "missing $index"

for report in "$libs_dir"/*.md; do
	name="$(basename "$report")"
	grep -q "$name" "$index" || fail "index.html does not link report $name"
done

# Every report link in the index resolves.
while IFS= read -r link; do
	[[ -e "$libs_dir/$link" ]] || fail "index.html links missing file $link"
done < <(grep -o 'href="[^"]*\.md"' "$index" | sed 's/href="//;s/"//')

# The render asset is linked and present.
grep -q '2026-11-22_fleet-libs-audit-index.png' "$index" ||
	fail "index.html does not reference the render asset"
[[ -f "$ROOT/docs/assets/2026-11-22_fleet-libs-audit-index.png" ]] ||
	fail "render asset missing from docs/assets/"

echo "✓ libreports check: citations + index/report links green"
