# Branching-Flow Anti-Pattern Review

**Date:** 2026-06-16
**Scope:** All non-test `*.go` files across the monorepo (347 files: library modules, `example/`, `cmd/`).
**Method:** Pattern-based grep heuristics + manual inspection of every candidate site. Every finding below was read in source before reporting — no false-positive template matches.
**Status:** All actionable findings **RESOLVED**. See "Resolution Log" at bottom.

---

## Summary

The codebase is **notably clean** for branching-flow anti-patterns. The library modules (`event/`, `command/`, `query/`, `decider/`, `dispatcher/`, `memory/`, `middleware/`, `projection/`, `signing/`, `encryption/`, `codec/`, `storage/`, `pebble/`) are almost entirely free of issues. The bulk of findings concentrate in three files:

- `turso/indexing/advisor.go` (a heuristic SQL-index advisor — string/pattern-heavy by nature)
- `cmd/api-stability/main.go` (AST-walking tool — inherently nested loops)
- `cmd/cqrs-gen/main.go` (code generator — branched codegen)

No flag parameters, no arrow-shaped deep nesting, and no long if-else-if chains exist in the core library API. That is a genuinely good outcome for a public SDK.

### Severity tally

| Severity                   | Count | Where                                                                                 |
| -------------------------- | ----- | ------------------------------------------------------------------------------------- |
| **HIGH**                   | 3     | `decider/load.go`, `cmd/api-stability/main.go`, `storage/sql/base.go`                 |
| **MEDIUM**                 | 8     | mostly `turso/indexing/advisor.go`, `cmd/cqrs-gen`, `catalog/internal/caseutil`       |
| **LOW**                    | 5     | cosmetic, optional                                                                    |
| **Acceptable / by-design** | 4     | documented idioms (deprecated API, `reflect.Kind` switch, test-harness `update bool`) |

### Pareto note

Fixing just the **3 HIGH** items removes the only structurally harmful branching. The `turso` advisor is the single file that, if refactored at all, should be done as one pass (it is a self-contained heuristic). Everything else is optional polish.

---

## HIGH severity

### H1. 4-way OR comparing one variable to constants → use a `switch`

**File:** `decider/load.go:62-66`

```go
family := event.Classify(inner)
if family == event.Rejection || family == event.Conflict ||
    family == event.Transient || family == event.Corruption {
    return inner
}
return event.WrapInfrastructure(inner, "decider.op_error", inner.Error())
```

**Why it's a problem:** Comparing the same variable (`family`) against four named constants with `||` is the textbook "complex boolean expression + if-else chain in OR form". It is hard to scan, easy to mis-edit (drop one term), and duplicates information that already lives in the `event` package's 5-family taxonomy. The check is really _"is this a known non-Infrastructure family?"_, but that intent is invisible.

**Recommended fix:** a `switch`, or better, a set-membership helper in `event/` so the intent is named:

```go
// Option A — switch (minimal, local)
switch family {
case event.Rejection, event.Conflict, event.Transient, event.Corruption:
    return inner
}
return event.WrapInfrastructure(inner, "decider.op_error", inner.Error())

// Option B — name the intent (preferred; reusable across modules)
// in event/classify.go:
//   func IsKnownFamily(f Family) bool { return f >= Rejection && f <= Corruption }
// then: if event.IsKnownFamily(family) { return inner }
```

Option B also prevents `decider/` from hard-coding the family enumeration (currently it will silently break if a sixth family is added — a latent maintenance trap).

---

### H2. 7-level arrow nesting + iterates `f.Decls` twice

**File:** `cmd/api-stability/main.go:134-188` (`collectExports`)

```go
for _, pkg := range pkgs {            // 1
    for _, f := range pkg.Files {      // 2
        for _, decl := range f.Decls { // 3
            genDecl, ok := ...
            if !ok { continue }        // 4
            for _, spec := range genDecl.Specs {        // 4 (loop)
                ts, ok := spec.(*ast.TypeSpec)          // 5
                if !ok || !ts.Name.IsExported() { ... } // 5-6
                switch ts.Type.(type) {                 // 6
                case *ast.InterfaceType: ...            // 7
                }
            }
            if genDecl.Tok.String() == "var" || ... {   // 4
                for _, spec := range genDecl.Specs {     // 5
                    vs, ok := spec.(*ast.ValueSpec)      // 6
                    if !ok { continue }
                    for _, name := range vs.Names {      // 6 (loop)
                        if name.IsExported() {           // 7
                            exports = append(...)
                        }
                    }
                }
            }
        }
        for _, decl := range f.Decls { /* funcs */ }     // ← f.Decls iterated a 2nd time
    }
}
```

**Why it's a problem:**

1. **Arrow shape** — the innermost bodies sit at depth 6–7, the classic unreadable pyramid.
2. **Double responsibility in one loop** — type exports, value exports, AND func exports are all collected inline, and `f.Decls` is walked twice (once at line 136, once at line 175). This mixes three concerns, making the function hard to test/extend.
3. A fourth export kind (e.g. type aliases, generics) would require reaching into the most-deeply-nested block.

**Recommended fix:** extract three focused helpers that each take `(f *ast.File)` and return `[]string`, then a thin orchestrator. Each helper flattens to max 3 levels:

```go
func collectExports(pkgs []*ast.Package) []string {
    var exports []string
    for _, pkg := range pkgs {
        for _, f := range pkg.Files {
            exports = append(exports, collectTypeExports(f)...)
            exports = append(exports, collectValueExports(f)...)
            exports = append(exports, collectFuncExports(f)...)
        }
    }
    sort.Strings(exports)
    return exports
}
```

Bonus: replace `genDecl.Tok.String() == "var"` with the cheaper `genDecl.Tok == token.VAR`.

---

### H3. Boolean flag parameter on a public constructor (and a runtime setter for it)

**File:** `storage/sql/base.go:37` and `:48`

```go
func NewOwnedDBHandle(db *sql.DB, d Dialect, ownDB bool) (*OwnedDBHandle, error) { ... }

func (b *OwnedDBHandle) SetOwnership(ownDB bool) { b.ownDB = ownDB }
```

**Why it's a problem:** `ownDB bool` makes `Close()` perform two completely different actions (close the underlying `*sql.DB`, or not) chosen by a caller-supplied boolean. A reader of `handle.Close()` cannot know which path runs without tracing the flag value back to its construction site. This is the canonical "flag argument" smell. The `SetOwnership` setter is worse: it lets lifecycle semantics be silently toggled _after_ construction — a latent foot-gun for double-close/connection-leak bugs.

**Recommended fix:** replace the flag with two self-documenting constructors (the public API already has `NewSQLBackend` vs `NewSQLiteBackend`, so callers are used to choosing a constructor):

```go
// Owns the *sql.DB and will close it on Close().
func NewOwnedDBHandle(db *sql.DB, d Dialect) (*OwnedDBHandle, error)

// Borrows the *sql.DB; Close() only marks the handle closed, never closes db.
func NewBorrowedDBHandle(db *sql.DB, d Dialect) (*OwnedDBHandle, error)
```

Then delete `SetOwnership`. If a mutable ownership transfer is genuinely needed (e.g. "backend created the DB, later handed to the store"), expose an explicit `TransferOwnership()` that documents _why_, rather than a generic bool setter. Note: this is a public-API change — gate behind a minor version bump.

---

## MEDIUM severity

### M1. String-branched if/else with ~25 lines of near-duplicated codegen

**File:** `cmd/cqrs-gen/main.go:197-250` (`generate`)

`if genType == genTypeCommand { … ~25 lines … } else { … ~25 near-identical lines … }`. The two branches share the same loop skeleton and `fmt.Fprintf` cadence and differ only in (a) import path (`command/v2` vs `query/v2`) and (b) the generic type signature.

**Why it's a problem:** Adding a third generation type (e.g. events, saga steps) means extending the if/else and copy-pasting the loop a third time. The branch signal is a string, not a type, so the compiler cannot help.

**Recommended fix:** split into `generateCommand(pkg string, entries []Entry)` and `generateQuery(pkg string, entries []Entry)`, or drive a single loop from a small `genSpec` struct (import path + per-entry formatter closure). Eliminates duplication and the string dispatch at once.

---

### M2. Switch-of-literals that should be a package-level map

**File:** `turso/indexing/advisor.go:228-277` (`tableQueryPatterns`)

A 4-case `switch table { case "events": return []queryPattern{…4 large inline structs…} … }`. Each arm just returns a static slice of struct literals.

**Why it's a problem:** This is data, not behavior. Encoding it as a switch means there is no `default`, no easy way to enumerate all known tables, and the function body is ~50 lines of literal noise.

**Recommended fix:** a package-level `var tableQueryPatterns = map[string][]queryPattern{ "events": {…}, … }` and reduce the method to `return tableQueryPatterns[table]`.

---

### M3. If-chain of overlapping `strings.Contains` tests → table-driven

**File:** `turso/indexing/advisor.go:360-411` (`inferIndex`)

Within each `switch table` case, a cascade of `if strings.Contains(queryUpper, "X") && strings.Contains(queryUpper, "Y") { return &Index{…} }` blocks. Each arm tests a different combination of substrings and returns a different recommended index.

**Why it's a problem:** Effectively an if-else-if chain encoded as early-returns. The mapping (pattern-substrings → recommended index) is data, but it is written as control flow. New index recommendations require editing the control flow, and the substring-coupling logic is duplicated per arm.

**Recommended fix:** a `[]struct{ needs []string; idx Index; reason string; priority Priority }` table per table; iterate and return the first match whose `needs` are all present. Collapses ~50 lines to ~15 and makes the advisor data-driven.

---

### M4. Four regexes OR'd on one line

**File:** `turso/indexing/advisor.go:331-334`

```go
if searchIndexRe.MatchString(...) || searchCoverRe.MatchString(...) ||
    usingIntegerPK.MatchString(...) || autoIndexRe.MatchString(...) { ... }
```

**Why it's a problem:** Four independent regex matchers combined with `||`. Adding a fifth regex means editing the boolean expression. Readability suffers and there is no single "list of heuristics" to audit.

**Recommended fix:** `var advisoryRegexes = []*regexp.Regexp{searchIndexRe, searchCoverRe, usingIntegerPK, autoIndexRe}` then `for _, re := range advisoryRegexes { if re.MatchString(...) { … break } }`.

---

### M5. Three `strings.Contains` joined by `&&`

**File:** `turso/indexing/advisor.go:365-367`

`strings.Contains(queryUpper, "AGGREGATE_TYPE") && strings.Contains(queryUpper, "AGGREGATE_ID") && strings.Contains(queryUpper, "VERSION")`.

**Why it's a problem:** Repeated `strings.Contains(queryUpper, …)` is noisy and the "all-substrings-present" intent is hidden. If M3 is implemented, this collapses into one table entry; otherwise a `containsAll(queryUpper, "AGGREGATE_TYPE", "AGGREGATE_ID", "VERSION")` helper expresses the intent.

---

### M6. Four-level nesting in a per-rune transform

**File:** `catalog/internal/caseutil/convert.go:13-24` (`ToSeparated`)

```go
for i, r := range runes {            // 1
    switch {                         // 2
    case unicode.IsUpper(r):
        if i > 0 {                   // 3
            // builds prevIsLower / nextIsLower, then:
            if prevIsLower || (prevIsUpper && nextIsLower) { // 4
                b.WriteRune(sep)
            }
        }
    }
}
```

**Why it's a problem:** Four levels deep with three intermediate booleans for what is a single decision ("should a separator precede this rune?"). Hard to follow.

**Recommended fix:** extract `shouldPrependSeparator(runes, i) bool` so the loop body is `if unicode.IsUpper(r) && shouldPrependSeparator(runes, i) { b.WriteRune(sep) }`. The helper is independently unit-testable (this is a `catalog/internal/` shared util used across exporters).

---

### M7. Switch on `reflect.Kind` with 10 arms

**File:** `catalog/schema/reflect.go:228-257` (`goTypeToJSON`)

A 10-case `switch k` mapping `reflect.Kind → JSON type`.

**Why it's borderline:** This is the largest switch in the codebase. However, mapping `reflect.Kind` to a JSON schema type is _inherently_ a switch-shaped problem and Go idiom strongly favors a switch here. Converting the simple cases to a `map[reflect.Kind]Type` is possible but gains little (most arms already group kinds, e.g. `reflect.Int, reflect.Int8, …`).

**Verdict:** Leave as a switch (idiomatic). Listed only so it is on record as _considered and accepted_.

---

### M8. Three-way OR mixing exact-match constants and a suffix test

**File:** `catalog/openapi/exporter.go:185`

```go
if lower == "id" || lower == "aggregate_id" || strings.HasSuffix(lower, "_id") {
```

**Why it's borderline:** Readable, but the two exact-string checks (`"id"`, `"aggregate_id"`) are subsumed by the suffix test only partially (`"id"` has no `_id` suffix). A `var idFieldNames = map[string]struct{}{"id": {}, "aggregate_id": {}}` plus the suffix check makes the special-cased set explicit and extensible. Minor; fix only if touching this exporter.

---

## LOW severity (optional polish)

| File:Line                                                                 | Pattern                                                                                   | Note                                                                                                                |
| ------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------- |
| `catalog/d2/connections.go:39`                                            | `len(svc.Commands)==0 && len(svc.Events)==0 && len(svc.Queries)==0`                       | Extract `svc.hasMessages()` / `svc.IsEmpty()` — improves readability of a repeated guard.                           |
| `decider/decider.go:74`                                                   | `r.snapshotStrategy != nil && (r.snapshotStore == nil \|\| r.codec == nil)`               | A clear invariant-validation guard. Acceptable as-is; could extract `snapshotConfigComplete() bool` only if reused. |
| `catalog/internal/cattest/catalog.go:60` & `event/eventtest/golden.go:14` | `AssertGolden(..., update bool)`                                                          | Common test-harness idiom (the `update` golden flag). Acceptable; not a production API.                             |
| `cmd/cqrs-gen/main.go:123`                                                | `d.IsDir() \|\| !strings.HasSuffix(path, ".go") \|\| strings.HasSuffix(path, "_test.go")` | Three conditions but each is a distinct, readable filter clause. Fine.                                              |
| `catalog/internal/caseutil/convert.go:20`                                 | `i+1 < len(runes) && runes[i+1] >= 'a' && runes[i+1] <= 'z'`                              | Dense rune-range idiom; collapses if M6's `shouldPrependSeparator` helper is added.                                 |

---

## Acceptable / by-design (no action)

| Site                                                | Pattern                                              | Why it's fine                                                                                                     |
| --------------------------------------------------- | ---------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------- |
| `event/event_new.go:76`                             | `switch v := payload.(type)` (3 cases incl. default) | Idiomatic Go type switch; only distinguishes `[]byte` / `json.RawMessage` / marshal-default.                      |
| `codec/raw.go:18`                                   | `switch b := v.(type)`                               | Same — tiny type switch, fine.                                                                                    |
| `event/tombstone.go:23`, `listing/types.go:48`      | `switch` over a small enum (3 cases)                 | Correct use of switch for enum-to-string / enum semantics.                                                        |
| `middleware/circuit_breaker.go:79,106`              | `switch circuitState(...)`                           | State machine; switch is the right tool.                                                                          |
| `event/replay.go:38` `WithReplay(ctx, replay bool)` | Bool param                                           | **Already deprecated** in favor of `WithProcessingMode(ctx, ModeReplay)`. Intentionally retained for back-compat. |

---

## Recommended fix order (Pareto)

1. **H1** (`decider/load.go`) — 2-minute fix, removes a latent maintenance trap. Do first.
2. **H3** (`storage/sql/base.go`) — public API; pair with the next minor release.
3. **H2** (`cmd/api-stability/main.go`) — internal tool, low risk, big readability win.
4. **M1** (`cmd/cqrs-gen/main.go`) — dedup the codegen branches.
5. **turso advisor M2–M5** — refactor `advisor.go` as one pass (it is self-contained and all four findings are symptoms of the same "data-as-control-flow" smell). Convert to table-driven + a regex slice.
6. **M6** (`catalog/internal/caseutil`) — extract the separator helper.

Everything in **LOW** is optional; **Acceptable** items need no change.

---

## Conclusion

No branching-flow anti-patterns reach the **library's public API surface** with one exception (H3, the `ownDB` flag — already borderline-common). The genuinely harmful branching lives entirely in `cmd/` tools and the `turso` advisor heuristic, which is the right place for "ugly but contained" logic. Fixing the three HIGH items and the single `turso` pass would leave the codebase effectively free of branching-flow smells.

---

## Resolution Log

All actionable findings resolved. Tests + lint pass for all changed modules.

| Finding | Resolution | Files Changed |
| --- | --- | --- |
| **H1** (4-way OR → switch) | **Already resolved** by commit `adb8e5e1` (error-family migration rewrote `opError` to use `event.Compose`/`event.Wrapf` — no more OR-chain). Fixed 2 pre-existing lint issues in the same file to unblock pipeline. | `decider/load.go` |
| **H2** (arrow nesting) | **Already resolved** — `collectExports` already split into `collectFileExports` + `collectGenDeclExports` + `typeExportName` helpers (max 3 levels). | — |
| **H3** (ownDB bool flag) | Added `NewBorrowedDBHandle(db, d)` and `NewOwningDBHandle(db, d)` as preferred constructors. Deprecated `NewOwnedDBHandle(db, d, ownDB bool)` and `SetOwnership(bool)`. Updated all 3 production callers + tests to use clean API. | `storage/sql/base.go`, `storage/sql/coverage_test.go`, `storage/event_store.go`, `storage/command_store.go`, `storage/query_store.go` |
| **M1** (string-branched codegen) | Extracted `genSpec` map with `writeEntry` closures. Two named writer functions (`writeCommandHandler`, `writeQueryHandler`). Adding a new gen type is now a single map entry. | `cmd/cqrs-gen/main.go` |
| **M2** (switch → map) | `tableQueryPatterns` now a package-level `queryPatternsByTable` map; method is a one-liner lookup. | `turso/indexing/advisor_data.go` (new), `turso/indexing/advisor.go` |
| **M3** (if-chain → table-driven) | `inferIndex` now iterates `indexInferenceRules[table]` — a data table of `{needs []string, index, reason, priority}`. First match wins, preserving exact semantics. | `turso/indexing/advisor_data.go`, `turso/indexing/advisor.go` |
| **M4** (regex OR → slice) | `advisoryRegexes` is now a `[]*regexp.Regexp` slice; `recommendationFromDetail` loops over it. | `turso/indexing/advisor_data.go`, `turso/indexing/advisor.go` |
| **M5** (3× Contains → helper) | `containsAll(s, substrs...)` helper used by M3's table-driven matching. | `turso/indexing/advisor_data.go` |
| **M6** (4-level nesting) | Extracted `shouldPrependSepBeforeUpper` and `shouldPrependSepBeforeDigit` helpers. `ToSeparated` body now max 2 levels. | `catalog/internal/caseutil/convert.go` |
| **M7** (reflect.Kind switch) | **Accepted** — idiomatic switch for reflect.Kind→type mapping. No action needed. | — |
| **M8** (id field OR-chain) | Extracted `isIDField(lower string) bool` helper. Also removed redundant `"aggregate_id"` check (subsumed by `_id` suffix test). | `catalog/openapi/exporter.go` |
| **LOW** (d2 hasMessages) | Extracted `hasMessages(svc)` helper. | `catalog/d2/connections.go` |
| **LOW** (decider snapshot guard) | Extracted `snapshotConfigIncomplete()` method. | `decider/decider.go` |
| **Pre-existing lint fixes** | Fixed 3 pre-existing lint issues to unblock pipeline: ST1023 + wsl_v5 in `decider/load.go`, gci in `catalog/schema/yaml.go`, varnamelen in `encryption/codec.go`. | 3 files |
