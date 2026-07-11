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
LAYER[dedup]=0
LAYER[event]=1
LAYER[command]=1
LAYER[query]=1
LAYER[scheduling]=1
LAYER[metadata]=1
LAYER[schema]=2
LAYER[snapshot]=2
LAYER[projection]=2
LAYER[idempotency]=2
LAYER[deriver]=2
LAYER[decider]=3
LAYER[graph]=3
LAYER[scenario]=3
LAYER[projectionhost]=3
LAYER[signing]=4
LAYER[encryption]=4
LAYER[otel]=4
LAYER[stack]=6
LAYER[middleware]=5
LAYER[storage]=5
LAYER[listing]=5
LAYER[watermill]=5
LAYER[transport/http]=5
LAYER[prometheus]=5
LAYER[storage/pebble]=5
LAYER[storage/turso]=5
LAYER[storage/memory]=4
LAYER[stack/memory]=6
LAYER[stack/sqlite]=6
LAYER[stack/pebble]=6
LAYER[stack/postgres]=6
LAYER[catalog]=7
LAYER[integration]=7
LAYER[stack/bench]=7

# Some modules legitimately depend on test helpers (memory) or cross-cutting concerns (otel)
# These are documented exceptions to the strict layer rules
declare -A EXCEPTIONS
EXCEPTIONS[event]="schema snapshot storage/memory"
EXCEPTIONS[schema]="storage/memory snapshot"
EXCEPTIONS[snapshot]="storage/memory"
EXCEPTIONS[decider]="storage/memory otel"
EXCEPTIONS[storage]="listing"
EXCEPTIONS[storage/turso]="storage listing"
EXCEPTIONS[query]="snapshot storage/memory"
EXCEPTIONS[command]="snapshot storage/memory"
EXCEPTIONS[projectionhost]="storage/memory otel"

# Test-only packages that don't count against production dep budgets.
# These are test infrastructure (assertions, PBT, mocking) used across all modules.
TEST_PACKAGES="github.com/onsi/gomega github.com/onsi/ginkgo/v2 pgregory.net/rapid"

# Dependency budgets: maximum direct PRODUCTION dependencies per module.
# Budgets are intentionally tight — new deps require explicit review.
# Test-only deps (gomega, rapid, ginkgo) are excluded from the count.
declare -A DEP_BUDGET
DEP_BUDGET[id]=3
DEP_BUDGET[dispatcher]=0
DEP_BUDGET[codec]=2
DEP_BUDGET[kv]=3
DEP_BUDGET[dedup]=0
DEP_BUDGET[event]=13
DEP_BUDGET[command]=8
DEP_BUDGET[query]=8
DEP_BUDGET[scheduling]=0
DEP_BUDGET[metadata]=1
DEP_BUDGET[schema]=4
DEP_BUDGET[snapshot]=5
DEP_BUDGET[projection]=2
DEP_BUDGET[idempotency]=4
DEP_BUDGET[deriver]=4
DEP_BUDGET[decider]=10
DEP_BUDGET[graph]=3
DEP_BUDGET[scenario]=3
DEP_BUDGET[projectionhost]=9
DEP_BUDGET[signing]=5
DEP_BUDGET[encryption]=5
DEP_BUDGET[otel]=7
DEP_BUDGET[middleware]=13
DEP_BUDGET[storage]=12
DEP_BUDGET[listing]=6
DEP_BUDGET[watermill]=9
DEP_BUDGET[transport/http]=5
DEP_BUDGET[prometheus]=5
DEP_BUDGET[storage/pebble]=10
DEP_BUDGET[storage/turso]=10
DEP_BUDGET[storage/memory]=8
DEP_BUDGET[stack]=14
DEP_BUDGET[stack/memory]=10
DEP_BUDGET[stack/sqlite]=10
DEP_BUDGET[stack/pebble]=10
DEP_BUDGET[stack/postgres]=10
DEP_BUDGET[stack/bench]=10
DEP_BUDGET[catalog]=4
DEP_BUDGET[integration]=21

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

    # Subtract test-only packages from the count (they don't add production weight)
    test_deps=0
    for pkg in $TEST_PACKAGES; do
        if grep -q "[[:space:]]${pkg} " "$gomod" 2>/dev/null; then
            test_deps=$((test_deps + 1))
        fi
    done

    # Also detect deps that are only imported from _test.go files.
    # These are test-only in practice even if not in TEST_PACKAGES (e.g.,
    # internal CQRS modules used solely in tests). For each direct require,
    # check if any non-test .go file imports it; if not, it's test-only.
    direct_paths=$(awk '
        /^require \(/{found=1;next}
        /^\)/{found=0}
        found && !/\/\// && !/indirect/ && /^[[:space:]]+[^[:space:]]/{print $1}
    ' "$gomod")

    for dep_path in $direct_paths; do
        # Skip packages already caught by TEST_PACKAGES
        skip=0
        for pkg in $TEST_PACKAGES; do
            case "$dep_path" in
                *"$pkg"*) skip=1; break ;;
            esac
        done
        [ "$skip" -eq 1 ] && continue

        # If no non-test .go file imports this dep, it's test-only
        if ! grep -rl "\"${dep_path}" "${mod}/" --include='*.go' 2>/dev/null \
            | grep -vq '_test\.go$'; then
            test_deps=$((test_deps + 1))
        fi
    done

    prod_deps=$((direct - test_deps))

    if [ "$prod_deps" -gt "$budget" ]; then
        echo "BUDGET: ${mod} has ${prod_deps} production deps (budget: ${budget}, total: ${direct}, test: ${test_deps})"
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

    # Parse go.mod directly for direct go-cqrs-lite dependencies.
    # Avoids GOWORK=off + go list which fails when go.sum is stale.
    deps=$(awk '
        /^require \(/{found=1;next}
        /^\)/{found=0}
        found && /go-cqrs-lite\// && !/\/\// {print $1}
    ' "$gomod" | grep "github.com/larsartmann/go-cqrs-lite/" || true)

    for req in $deps; do
        # Skip the module itself
        # Skip the module itself (match any major version suffix)
        if echo "$req" | grep -qE "go-cqrs-lite/${mod}/v[0-9]+"; then
            continue
        fi

        # Extract module name from import path
        # github.com/larsartmann/go-cqrs-lite/EVENT/v4 -> event
        dep_mod=$(echo "$req" | sed 's|github.com/larsartmann/go-cqrs-lite/||; s|/v[0-9]\+||')

        # Skip root module and examples/cmd (not in layer map)
        if [ -z "${LAYER[$dep_mod]:-}" ]; then
            continue
        fi

        # Check exceptions
        exc="${EXCEPTIONS[$mod]:-}"
        if echo "$exc" | grep -qw "$dep_mod"; then
            continue
        fi

        # Check layer ordering (same-layer deps are allowed; only strictly higher = violation)
        dep_layer=${LAYER[$dep_mod]}
        if [ "$dep_layer" -gt "$mod_layer" ]; then
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
