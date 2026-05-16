# art-dupl Improvement Report

## Executive Summary

Ran art-dupl v0.3.0 with `--semantic --sort total-tokens -t 25` on a Go monorepo (~10K LOC production code, 9 modules). Found **80 clone groups** but **~60-70% were false positives** that don't represent actionable code smell.

## Issues Found

### 1. Function Signature Detection (Critical False Positive)

**Problem**: art-dupl flags identical function signatures as "clones" even though they're required by Go interfaces.

**Example**:

```go
// All 14 of these were flagged as a clone group:
func (s *SQLEventStore) Save(ctx context.Context, aggregateType event.AggregateType, aggregateID id.AggregateID, events []event.Event, expectedVersion event.Version) error
func (s *SQLiteEventStore) Save(ctx context.Context, aggregateType event.AggregateType, aggregateID id.AggregateID, events []event.Event, expectedVersion event.Version) error
func (s *MemoryStore) Save(ctx context.Context, aggregateType event.AggregateType, aggregateID id.AggregateID, events []event.Event, expectedVersion event.Version) error
```

This is Go idiomatic - interface implementations MUST have identical signatures. Flagging these as clones is noise.

**Recommendation**: Add `--ignore-interface-methods` flag that excludes methods implementing the same interface from clone detection.

---

### 2. Lock/Unlock Pattern Detection

**Problem**: `defer s.mu.Unlock()` patterns across mutex-wrapped methods are flagged as clones.

**Example** from memory/store.go:

```go
// 9 files flagged for identical "defer mu.Unlock()" patterns
s.mu.Lock()
defer s.mu.Unlock()
```

This is standard Go concurrency pattern, not code smell.

**Recommendation**: Add `--ignore-defer-unlock` flag or pattern to skip mutex unlock patterns.

---

### 3. Error Handling Pattern Detection

**Problem**: Common error handling like `if err != nil { return err }` is flagged across many files.

**Example**:

```go
if err != nil {
    return fmt.Errorf("memory store save: %w", err)
}
```

This is Go idiomatic error handling, not duplication.

**Recommendation**: Increase minimum token threshold for error handling patterns or add `--ignore-error-propagation` flag.

---

### 4. Test Code Over-Flagging

**Problem**: ~50 of 80 clone groups were in test files. Many represent intentional test fixture setup.

**Example**:

```go
// Flagged as clone but this is intentional test setup
err = repo.Execute(ctx, aggID, "Counter", func(...) { ... })
err = repo.Execute(ctx, aggID, "Counter", func(...) { ... }) // Slightly different lambda
```

**Recommendation**:

- Default to `--exclude-test-files`
- Or add `--test-tolerance=T` threshold that only flags test duplications exceeding T lines

---

### 5. Semantic Analysis Gap

**Problem**: The tool detects syntactic similarity but misses semantic intent.

**Example**: Two functions with:

```go
err := s.CheckClosed(event.ErrStoreClosed)
if err != nil {
    return fmt.Errorf("memory store save: %w", err)
}
```

Both flagged as clone group #9, but they're:

- Required interface method implementations (signature must match)
- Same error handling pattern (idiomatic Go)
- Locked behind mutex (same concurrency pattern)

These are NOT independent duplications - they're boilerplate that cannot be eliminated without breaking interfaces or concurrency safety.

**Recommendation**: Improve semantic analysis to recognize:

1. Interface implementation requirements
2. Concurrency safety patterns
3. Error propagation conventions

---

### 6. Clone Grouping Granularity

**Problem**: Clone groups contain unrelated code snippets just because they share 2-3 tokens of similarity.

**Example**: Group #9 contained 9 files but only the `CheckClosed` call was similar - not the full function body.

**Recommendation**:

- Use finer-grained clone detection that identifies the EXACT duplicated token sequence
- Report: "Clone at line X columns Y-Z: `err := s.CheckClosed(...)`" instead of "function body clone"
- Group by actual shared code, not file proximity

---

### 7. Output Format Improvements

**Current output**:

```
found 14 clones:
  core/aggregate/repository.go:97,103
  core/aggregate/repository.go:130,136
  ...
```

**Problems**:

1. Doesn't show WHAT is cloned (just line numbers)
2. Can't understand issue without viewing source
3. Line number pairs (e.g., `97,103`) suggest range but it means "tokens 97-103"

**Recommendation**:

```
Clone Group #2: "interface method signature" (14 occurrences, 45 tokens)
├── [HIGH] core/aggregate/repository.go:97-103 - Save method signature
├── [HIGH] memory/store.go:38-44 - Save method signature
├── [INFO] testhelpers/fake_store.go:16-22 - Save method signature
└── Classification: FALSE_POSITIVE - Required by event.Store interface
```

---

### 8. Missing `--help` Content

The tool lacks comprehensive help for flags like `--semantic`, `--html`, `--sort`.

**Recommendation**: Add `--help-full` that shows all flags with examples.

---

## Suggested New Flags

```bash
art-dupl [options] --ignore-interface-methods    # Skip methods implementing interfaces
art-dupl [options] --ignore-defer-unlock        # Skip mutex unlock patterns
art-dupl [options] --ignore-error-propagation   # Skip "if err != nil { return err }"
art-dupl [options] --exclude-test-files         # Don't analyze *_test.go files
art-dupl [options] --min-clone-length=N         # Minimum N tokens for clone (default: 15)
art-dupl [options] --show-true-clones-only      # Filter out known false positive patterns
art-dupl [options] --output-json               # Machine-readable output
art-dupl [options] --group-by-pattern          # Group similar clones together
```

## Suggested Algorithm Improvements

### 1. Clone Classification

```go
type CloneType int
const (
    CloneTypeSignature CloneType = iota  // Function/method signature (often false positive)
    CloneTypeBoilerplate                 // Error handling, locking (often acceptable)
    CloneTypeSubstantial                  // Real logic duplication (actionable)
    CloneTypeTestFixture                 // Test setup (lower priority)
)
```

### 2. Weighted Token Analysis

Instead of raw token count, weight by type:

- Function signature tokens: weight 0.1 (almost always false positive)
- Error handling tokens: weight 0.3 (often boilerplate)
- Business logic tokens: weight 1.0 (actionable)
- Test fixture tokens: weight 0.5 (lower priority)

### 3. Context-Aware Detection

Check if code is:

- Part of interface implementation
- Required by language semantics
- Idiomatic pattern (defer, mutex, error handling)
- Actual business logic duplication

---

## Testing Recommendations

Test the improved algorithm against:

1. Go standard library implementations (bufio, strings, etc.) - should find ~0 signature clones
2. Go interface implementations across multiple packages
3. Idiomatic Go patterns (errors.Is, context propagation, mutex patterns)
4. Repository pattern implementations (like this codebase)

---

## Summary

| Issue                              | Severity | Impact                      | Effort to Fix |
| ---------------------------------- | -------- | --------------------------- | ------------- |
| Function signature false positives | Critical | 20+ clone groups eliminated | Medium        |
| Test code over-flagging            | High     | 50+ clone groups suppressed | Low           |
| Missing semantic classification    | High     | Better prioritization       | High          |
| Output format clarity              | Medium   | Better UX                   | Medium        |
| Pattern exclusion flags            | Medium   | Better UX                   | Low           |

The tool is useful but needs tuning for Go-specific patterns. The biggest win would be detecting and excluding interface method signatures from clone detection.

---

## Reproduction Steps

```bash
cd /home/lars/projects/go-cqrs-lite
art-dupl -t 25 . --semantic --sort total-tokens
```

Expected: ~10-15 actionable production clone groups
Actual: 80 clone groups (mostly false positives)
