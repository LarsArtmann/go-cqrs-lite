# Dedup Acceptance Log

Remaining clone groups from `art-dupl --type-aware -t 3` after the dedup
session. Each entry explains why the clone is intentional and should not be
extracted.

---

## 1. Stack preset DB open with named-return cleanup

**Files:** `stack/postgres/preset.go:147`, `stack/sqlite/preset.go:122`
**Category:** assignment | 2 occurrences

```
db, err = sqlopt.OpenDBOrErr(driver, dsn, label)
if err != nil {
    return nil, nil, err
}
defer func() {
    if err != nil && db != nil {
        _ = db.Close()
    }
}()
ctx := context.Background()
```

**Reason:** The `defer func() { if err != nil ... }` cleanup is tied to the
enclosing function's named return `err`. Extracting a helper would still leave
the conditional defer in each caller because the defer must capture the named
return from its own scope. The logic diverges immediately after this preamble
(sqlite does WAL/FK/pool config; postgres does schema init). This is the
idiomatic Go resource-lifecycle pattern for "open, then close-on-error".

---

## 2. Decider cache mutex idiom

**Files:** `decider/cache.go:69`, `decider/cache.go:117` (same file, Get + Put/Invalidate)
**Category:** unknown | 2 occurrences

```
c.mu.Lock()
defer c.mu.Unlock()

key := ref.String()
```

**Reason:** Standard Go mutex idiom. The `defer c.mu.Unlock()` must live in
the caller's scope. A closure-based `withLockKey(ref, fn)` would require
captured variables for Get's three return values, making the code worse. The
three lines are the minimal Go locking preamble.

---

## 3. Catalog writer strings.Builder with different content

**Files:** `catalog/eventcatalog/writer.go:102`, `catalog/eventcatalog/writer_schemas_txt.go:15`
**Category:** expression | 2 occurrences

```
var cfg strings.Builder
cfg.WriteString(...)
```

**Reason:** Standard `strings.Builder` usage writing completely different
content (JS config export vs Markdown schemas header). The structural
similarity is just `var X strings.Builder` followed by `X.WriteString(...)`.
The values are intentionally unique; the structure is the standard library
API.
