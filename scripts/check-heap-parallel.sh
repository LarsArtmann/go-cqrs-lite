#!/usr/bin/env bash
# check-heap-parallel.sh — heap-measurement contract enforcement.
#
# runtime.ReadMemStats snapshots PROCESS-GLOBAL state: when a parallel test
# holds live allocations during another test's snapshot, they get
# misattributed as leaks (a 63MB phantom leak cost two sessions; 13 files
# fixed 2026-08-14). The contract: a _test.go file that calls
# runtime.ReadMemStats must not call t.Parallel() in the same file.
#
# Same-file enforcement is the mechanical tripwire. Cross-file callers of
# soak runners (enginetest) remain a review concern — documented in
# AGENTS.md Testing section and enginetest/soak.go.
#
# Usage: bash scripts/check-heap-parallel.sh
# Exit: 0 = contract holds, 1 = violation(s) found.

set -euo pipefail

cd "$(git rev-parse --show-toplevel)"

errors=0
checked=0

while IFS= read -r f; do
	checked=$((checked + 1))
	if grep -qE '\bt\.Parallel\(\)' "$f"; then
		echo "ERROR: $f calls runtime.ReadMemStats AND t.Parallel() —"
		echo "       heap measurements must never run in parallel (process-global MemStats)."
		errors=$((errors + 1))
	fi
done < <(grep -rl 'runtime\.ReadMemStats' --include='*_test.go' . | sort || true)

echo "Checked $checked test file(s) calling runtime.ReadMemStats."

if [ "$errors" -gt 0 ]; then
	echo ""
	echo "Found $errors heap-measurement contract violation(s)."
	exit 1
fi

echo "✓ Heap-measurement contract holds."
