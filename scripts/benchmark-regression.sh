#!/usr/bin/env bash
# benchmark-regression.sh
# Compares current benchmark results against the baseline.
# Exit 1 if any benchmark regresses by more than the threshold (default: 20%).
#
# Usage:
#   ./scripts/benchmark-regression.sh [threshold_percent]
#
# To update the baseline:
#   nix run .#bench > benchmarks/benchmark-baseline.txt

set -euo pipefail

threshold="${1:-20}"
baseline="benchmarks/benchmark-baseline.txt"
current=$(mktemp)

echo "==> Running benchmarks..."
nix run .#bench > "$current"

regressions=0
improvements=0
stable=0

while IFS= read -r line; do
  if [[ "$line" =~ ^Benchmark[a-zA-Z]+-[0-9]+[[:space:]]+([0-9]+) ]]; then
    bench_name=$(echo "$line" | awk '{print $1}')
    current_ns=$(echo "$line" | awk '{print $3}')

    baseline_line=$(grep "^${bench_name}[[:space:]]" "$baseline" 2>/dev/null || true)
    if [[ -n "$baseline_line" ]]; then
      baseline_ns=$(echo "$baseline_line" | awk '{print $3}')

      if [[ -n "$baseline_ns" && -n "$current_ns" ]]; then
        diff_pct=$(( (current_ns - baseline_ns) * 100 / baseline_ns ))

        if [[ $diff_pct -gt $threshold ]]; then
          echo "REGRESSION: $bench_name ${baseline_ns}ns → ${current_ns}ns (+${diff_pct}%)"
          ((regressions++)) || true
        elif [[ $diff_pct -lt -5 ]]; then
          ((improvements++)) || true
        else
          ((stable++)) || true
        fi
      fi
    fi
  fi
done < "$current"

rm -f "$current"

echo ""
echo "Summary: $regressions regression(s), $improvements improvement(s), $stable stable"

if [[ $regressions -gt 0 ]]; then
  echo "FAIL: $regressions benchmark(s) regressed more than ${threshold}%"
  exit 1
fi

echo "PASS: No regressions above ${threshold}% threshold"
