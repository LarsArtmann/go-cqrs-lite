#!/usr/bin/env bash
# check-module-layers.sh
# Validates that go-cqrs-lite module dependencies follow the layered architecture.
# go-arch-lint is unsuitable for multi-module Go workspaces because it treats
# inter-module imports as external vendor dependencies. This script parses go.mod
# files directly to enforce layer rules.

set -euo pipefail

# Layer definitions (higher layers may depend on lower layers only)
declare -A LAYER
LAYER[id]=0
LAYER[dispatcher]=0
LAYER[codec]=0
LAYER[kv]=0
LAYER[event]=1
LAYER[command]=1
LAYER[query]=1
LAYER[schema]=2
LAYER[snapshot]=2
LAYER[decider]=3
LAYER[memory]=4
LAYER[signing]=4
LAYER[otel]=4
LAYER[middleware]=5
LAYER[storage]=5
LAYER[projection]=5
LAYER[listing]=5
LAYER[watermill]=5
LAYER[pebble]=5
LAYER[turso]=5
LAYER[catalog]=6
LAYER[integration]=6

# Some modules legitimately depend on test helpers (memory) or cross-cutting concerns (otel)
# These are documented exceptions to the strict layer rules
declare -A EXCEPTIONS
EXCEPTIONS[event]="memory"
EXCEPTIONS[schema]="memory snapshot"
EXCEPTIONS[snapshot]="memory"
EXCEPTIONS[decider]="memory otel"
EXCEPTIONS[storage]="listing"
EXCEPTIONS[turso]="storage listing"
EXCEPTIONS[command]="snapshot"
EXCEPTIONS[query]="snapshot"

# Dependency budgets: maximum direct (non-indirect) dependencies per module.
# Budgets are intentionally tight — new deps require explicit review.
declare -A DEP_BUDGET
DEP_BUDGET[id]=3
DEP_BUDGET[dispatcher]=0
DEP_BUDGET[codec]=0
DEP_BUDGET[kv]=1
DEP_BUDGET[event]=13
DEP_BUDGET[command]=8
DEP_BUDGET[query]=7
DEP_BUDGET[schema]=4
DEP_BUDGET[snapshot]=5
DEP_BUDGET[decider]=10
DEP_BUDGET[memory]=8
DEP_BUDGET[signing]=5
DEP_BUDGET[otel]=7
DEP_BUDGET[middleware]=13
DEP_BUDGET[storage]=10
DEP_BUDGET[projection]=9
DEP_BUDGET[listing]=6
DEP_BUDGET[watermill]=5
DEP_BUDGET[pebble]=5
DEP_BUDGET[turso]=6
DEP_BUDGET[catalog]=4
DEP_BUDGET[integration]=18

failed=0

# ── Dependency count check ──

for mod in "${!DEP_BUDGET[@]}"; do
    gomod="${mod}/go.mod"
    if [ ! -f "$gomod" ]; then
        continue
    fi

    budget=${DEP_BUDGET[$mod]}
    direct=$(awk '
        /^require \(/{found=1;next}
        /^\)/{found=0}
        found && !/\/\// && !/indirect/ && /^[[:space:]]+[^[:space:]]/{count++}
        END{print count+0}
    ' "$gomod")

    if [ "$direct" -gt "$budget" ]; then
        echo "BUDGET: ${mod} has ${direct} direct deps (budget: ${budget})"
        failed=1
    fi
done

# ── Layer ordering check ──

for mod in "${!LAYER[@]}"; do
    gomod="${mod}/go.mod"
    if [ ! -f "$gomod" ]; then
        continue
    fi

    mod_layer=${LAYER[$mod]}

    # Use GOWORK=off to get only this module's direct dependencies
    deps=$(cd "$mod" && GOWORK=off go list -m -json all 2>/dev/null | awk '
        /"Path":/ { path=$2; gsub(/[",]/, "", path) }
        /"Main":/ { if ($2 == "true") print path }
    ' | grep "github.com/larsartmann/go-cqrs-lite/" || true)

    for req in $deps; do
        # Skip the module itself
        if echo "$req" | grep -q "go-cqrs-lite/${mod}/v2"; then
            continue
        fi

        # Extract module name from import path
        # github.com/larsartmann/go-cqrs-lite/EVENT/v2 -> event
        dep_mod=$(echo "$req" | sed 's|github.com/larsartmann/go-cqrs-lite/||; s|/v2||')

        # Skip root module and examples/cmd (not in layer map)
        if [ -z "${LAYER[$dep_mod]:-}" ]; then
            continue
        fi

        # Check exceptions
        exc="${EXCEPTIONS[$mod]:-}"
        if echo "$exc" | grep -qw "$dep_mod"; then
            continue
        fi

        # Check layer ordering
        dep_layer=${LAYER[$dep_mod]}
        if [ "$dep_layer" -ge "$mod_layer" ]; then
            echo "VIOLATION: ${mod} (layer ${mod_layer}) depends on ${dep_mod} (layer ${dep_layer}) via ${req}"
            failed=1
        fi
    done
done

if [ "$failed" -eq 1 ]; then
    echo "::error::Module layer violations detected"
    exit 1
fi

echo "Module layer check passed"
