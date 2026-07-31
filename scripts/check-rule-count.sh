#!/usr/bin/env bash
# check-rule-count.sh — Verifies that the rule count documented in FEATURES.md,
# ROADMAP.md, and AGENTS.md matches the actual count from rules.AllRules().
#
# This prevents doc drift: when rules are added/removed, the docs must be
# updated in the same change.
#
# Usage: bash scripts/check-rule-count.sh
# Exit: 0 if all docs match, 1 if any drifted.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT

# Write a tiny Go program that prints the rule count.
cat > "$TMPDIR/count.go" << 'GOEOF'
package main

import (
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules"
)

func main() {
	ctx := &analyzer.AnalysisContext{}
	detectors := rules.RegisterAll(ctx)
	fmt.Print(len(detectors))
}
GOEOF

# Run it from the cqrs-lint module directory.
ACTUAL=$(cd "$REPO_ROOT/cmd/cqrs-lint" && \
  GOWORK=off go run -tags "goexperiment.jsonv2" "$TMPDIR/count.go" 2>/dev/null) || {
  echo "ERROR: could not get rule count from AllRules()"
  exit 1
}

echo "==> Actual rule count: $ACTUAL"

# Check each doc file for the documented count.
FAILED=0

check_doc() {
  local file="$1"
  local pattern="$2"
  local label="$3"

  if [ ! -f "$REPO_ROOT/$file" ]; then
    echo "SKIP: $file not found"
    return
  fi

  DOC_COUNT=$(grep -oE "$pattern" "$REPO_ROOT/$file" | grep -oE '[0-9]+' | head -1 || true)

  if [ -z "$DOC_COUNT" ]; then
    echo "WARN: could not find rule count in $file (pattern: $pattern)"
    return
  fi

  if [ "$DOC_COUNT" != "$ACTUAL" ]; then
    echo "FAIL: $label says $DOC_COUNT rules, actual is $ACTUAL"
    FAILED=1
  else
    echo "OK:   $label says $DOC_COUNT rules (matches)"
  fi
}

check_doc "FEATURES.md" '[0-9]+-rule' "FEATURES.md"
check_doc "ROADMAP.md" '[0-9]+ rules shipped' "ROADMAP.md"
check_doc "AGENTS.md" '[0-9]+ rules across' "AGENTS.md"

if [ "$FAILED" -eq 1 ]; then
  echo ""
  echo "Doc drift detected. Update the rule count in the files marked FAIL above."
  echo "Actual count: $ACTUAL"
  exit 1
fi

echo ""
echo "All doc rule counts match actual ($ACTUAL)."
