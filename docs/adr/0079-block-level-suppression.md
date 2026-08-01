# ADR-0079: Block-Level Suppression for cqrs-lint

**Date:** 2026-08-01
**Status:** Accepted
**Related:** L4.9 from the SUPERB Metaengine Planner plan

## Context

cqrs-lint previously supported only inline suppression: `//cqrs-lint:ignore(A001)` on a single line. This is verbose when suppressing multiple findings in a block of generated code, a test helper, or a compatibility shim. Users had to repeat the comment on every line.

## Decision

Add block-level suppression with `ignore-start` / `ignore-end` pairs.

### Syntax

```go
//cqrs-lint:ignore-start          // suppresses ALL rules until ignore-end
//cqrs-lint:ignore-start(A001,A005) // suppresses specific rules only

func generatedCode() {
    // ... findings here are suppressed ...
}

//cqrs-lint:ignore-end
```

### Parser Design

- `ignore-start` with no arguments suppresses ALL rules in the block
- `ignore-start(R1,R2)` suppresses only the listed rules
- Blocks can be nested (inner `ignore-start` adds to the outer suppression)
- `ignore-end` closes the most recent open block
- The suppression filter checks backward from each finding's line to find the nearest enclosing block

### Stale Detection

The stale suppression detector (`DetectStaleSuppressions`) now also scans for block suppressions. A block is "stale" if no suppressed rule actually fires within the `ignore-start`/`ignore-end` range. This prevents dead suppression pairs from accumulating.

### Pipeline Integration

Block suppression runs after inline suppression in the filter pipeline:
1. Check inline `//cqrs-lint:ignore(RULE)` (existing)
2. Check block `//cqrs-lint:ignore-start` (new)
3. Snippet-based fallback (existing)

## Consequences

### Positive

- Dramatically reduces suppression noise for generated code blocks
- Per-rule scoping (`ignore-start(A001)`) prevents over-suppression
- Stale detection keeps suppression pairs from accumulating
- Consistent with `golangci-lint`'s `nolint` block pattern

### Negative

- Nested blocks add parser complexity
- Stale detection for blocks is O(lines × findings) per file
- No `//cqrs-lint:ignore-file` (whole-file suppression) — intentionally omitted to prevent blanket suppression abuse
