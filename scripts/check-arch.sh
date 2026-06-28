#!/usr/bin/env bash
# check-arch.sh — Two-layer architecture enforcement
#
# Layer 1: Cross-module rules via check-module-layers.sh (go.mod parsing)
# Layer 2: Intra-module package rules via go-arch-lint (per-module)
#
# This script is the CI-enforceable architecture gate. It replaces the
# standalone go-arch-lint check (which cannot handle multi-module workspaces)
# with a per-module approach that works correctly.

set -euo pipefail

PROJECT_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
FAILED=0

echo "━━━ Layer 1: Cross-module dependency rules (check-module-layers.sh) ━━━"
if bash "$PROJECT_ROOT/scripts/check-module-layers.sh"; then
    echo "  ✓ Module layer check passed"
else
    echo "  ✗ Module layer check FAILED"
    FAILED=1
fi

echo ""
echo "━━━ Layer 2: Intra-module package rules (go-arch-lint per-module) ━━━"

# Find all modules with a local .go-arch-lint.yml
for archfile in $(find "$PROJECT_ROOT" -name ".go-arch-lint.yml" -not -path "*/vendor/*" | sort); do
    moddir=$(dirname "$archfile")

    # Skip the workspace-level config (handled by Layer 1 documentation)
    if [ "$moddir" = "$PROJECT_ROOT" ]; then
        continue
    fi

    modname=$(basename "$moddir")
    echo "  Checking $modname..."

    if (cd "$moddir" && go-arch-lint check --project-path "$moddir" 2>&1) | grep -q "shouldn't depend\|not attached"; then
        echo "    ✗ $modname has architecture violations"
        (cd "$moddir" && go-arch-lint check --project-path "$moddir" 2>&1) | grep "shouldn't depend\|not attached" | head -10
        FAILED=1
    else
        echo "    ✓ $modname passed"
    fi
done

echo ""
if [ "$FAILED" -eq 0 ]; then
    echo "━━━ All architecture checks passed ━━━"
    exit 0
else
    echo "━━━ Architecture checks FAILED ━━━"
    exit 1
fi
