# makezero Linter Evaluation

## Current State

`makezero.always: false` in `.golangci.yml`. This means `makezero` only flags `make([]T, 0)` without a subsequent append, NOT the valid `make([]T, len(x))` index-fill pattern.

## The Problem with `always: true`

`makezero.always: true` flags ALL `make([]T, N)` where N > 0, including the idiomatic Go pattern:

```go
// Index-fill pattern (valid, zero-allocation):
result := make([]string, len(items))
for i, item := range items {
    result[i] = item.Name
}
```

This pattern appears ~115 times in the codebase. `makezero` treats each as a potential "forgot to fill" bug. In practice, the index-fill pattern is a standard Go idiom for pre-allocated slices where every element is assigned by index.

## Recommendation

**Keep `always: false`.** The `always: true` mode generates too many false positives for the index-fill pattern to be useful. The `always: false` mode still catches the real bug class (zero-length makes that should have been followed by append) without the noise.

If the linter adds a `--allow-index-fill` flag in the future, `always: true` could be re-evaluated.
