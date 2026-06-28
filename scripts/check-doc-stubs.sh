#!/usr/bin/env bash
#
# check-doc-stubs.sh — fail if any doc.go contains a placeholder stub comment.
#
# The default goimports/go-fix template generates `// Package X provides ...`
# when no package comment is written. These stubs are useless to consumers and
# erode trust. This guard catches them before they ship.
#
# Exit 0 = all doc.go files have real descriptions, exit 1 = stubs found.

set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

echo "=== Doc Stub Check ==="

stubs=$(find . -name doc.go -not -path './vendor/*' -print0 \
  | xargs -0 grep -l '// Package .* provides \.\.\.' 2>/dev/null || true)

if [ -n "$stubs" ]; then
  echo "FAIL: placeholder doc.go stubs found:"
  echo "$stubs" | sed 's/^/  /'
  echo ""
  echo "Each file above contains '// Package X provides ...' which is the"
  echo "goimports default placeholder. Replace with a real description."
  exit 1
fi

echo "All doc.go files have real package descriptions."
exit 0
