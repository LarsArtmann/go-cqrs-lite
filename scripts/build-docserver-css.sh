#!/usr/bin/env bash
# Build the docserver UI stylesheet (catalog/docserver/static/docs-ui.css)
# from docs-ui.src.css + templ-components sources.
#
# Usage:
#   scripts/build-docserver-css.sh          # rebuild and write docs-ui.css
#   scripts/build-docserver-css.sh --check  # rebuild to a temp file and fail
#                                           # on drift vs the committed output
#
# Requires: go, tailwindcss (nix: pkgs.tailwindcss_4). Run via
# `nix run .#build-docserver-css` to get the pinned tailwind binary.
set -euo pipefail

REPO_ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
CATALOG_DIR="$REPO_ROOT/catalog"
STATIC_DIR="$CATALOG_DIR/docserver/static"
SRC_CSS="$STATIC_DIR/docs-ui.src.css"
OUT_CSS="$STATIC_DIR/docs-ui.css"

if ! command -v go >/dev/null 2>&1; then
  echo "ERROR: go is required to resolve the templ-components module dir" >&2
  exit 1
fi

if ! command -v tailwindcss >/dev/null 2>&1; then
  echo "ERROR: tailwindcss not found. Run via 'nix run .#build-docserver-css'." >&2
  exit 1
fi

DOCSERVER_DIR="$CATALOG_DIR/docserver"
GOMODCACHE="${GOMODCACHE:-$(go env GOMODCACHE)}"
TC_VERSION=$(cd "$CATALOG_DIR" && GOWORK=off go list -m -f '{{.Version}}' github.com/larsartmann/templ-components)
TC_DIR="$GOMODCACHE/github.com/larsartmann/templ-components@$TC_VERSION"
if [ ! -d "$TC_DIR" ]; then
  (cd "$CATALOG_DIR" && GOWORK=off go mod download github.com/larsartmann/templ-components)
fi

TMP_DIR=$(mktemp -d)
trap 'rm -rf "$TMP_DIR"' EXIT

sed "s|__DOCSERVER_DIR__|$DOCSERVER_DIR|g" "$SRC_CSS" > "$TMP_DIR/docs-ui.entry.css"
cat >> "$TMP_DIR/docs-ui.entry.css" <<EOF

/* Generated: templ-components module sources ($TC_VERSION). */
@source "$TC_DIR/display/**/*.{templ,go}";
@source "$TC_DIR/layout/**/*.{templ,go}";
@source "$TC_DIR/navigation/**/*.{templ,go}";
@source "$TC_DIR/utils/**/*.{templ,go}";
@import "$TC_DIR/templates/templ-components-theme.css";
@import "$TC_DIR/templates/custom.css";
EOF

tailwindcss -i "$TMP_DIR/docs-ui.entry.css" -o "$TMP_DIR/docs-ui.css" --minify

if [ "${1:-}" = "--check" ]; then
  if ! diff -u "$OUT_CSS" "$TMP_DIR/docs-ui.css"; then
    echo "ERROR: catalog/docserver/static/docs-ui.css is stale." >&2
    echo "Fix: nix run .#build-docserver-css (or edit docs-ui.src.css) and commit." >&2
    exit 1
  fi
  echo "docs-ui.css up to date"
else
  cp "$TMP_DIR/docs-ui.css" "$OUT_CSS"
  echo "wrote $OUT_CSS ($(wc -c < "$OUT_CSS") bytes)"
fi
