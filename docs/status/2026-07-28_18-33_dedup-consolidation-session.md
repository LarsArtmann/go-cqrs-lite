# Status Report: Dedup Consolidation Session

**Date:** 2026-07-28 18:33
**Session scope:** `art-dupl` clone elimination across the monorepo
**Starting state:** 3 clone groups at threshold 3
**Ending state:** 0 clone groups at threshold 3, 16 accepted in baseline

---

## A) FULLY DONE

### 1. `stack/sqlopt.OpenPrimaryBackend` — extracted open/setup/cleanup boilerplate
**Files:** `stack/sqlopt/sqlopt.go`, `stack/postgres/preset.go`, `stack/sqlite/preset.go`

Both `postgres.openBackend` and `sqlite.openBackend` had the identical structure:
`OpenDBOrErr` → `defer close-on-error` → `ctx := context.Background()` → `setup` → `newBackend` → `return`.

Extracted `OpenPrimaryBackend(openDB, setup, newBackend, backendCode)` into the shared `sqlopt` package.
Each preset now passes its dialect-specific open function and setup closures, collapsing ~30 lines of
identical control flow into one call.

- Postgres: passes `storage.PostgresInitSchema` in the setup closure
- SQLite: passes WAL/Pool/FK/InitSchema/Optimize in the setup closure
- Both reuse the same cleanup-on-error defer and backend-creation error wrapping

**Verified:** build, vet, test, race, lint, api-stability, doc-check all green.

### 2. `catalog/eventcatalog.Exporter.writeBuilderFile` — extracted builder-to-file pattern
**Files:** `catalog/eventcatalog/writer.go`, `writer_schemas_txt.go`, `writer_llms.go`

Three writer methods (`writeConfig`, `writeSchemasTxt`, `writeLLMsTxt`) each had:
`var buf strings.Builder` → fill via `WriteString`/`Fprintf` → `os.WriteFile(path, []byte(buf.String()), filePerm)`.

Extracted `writeBuilderFile(filename, fn)` which handles builder creation, file writing, and path joining.
Each writer now just passes its filename and a closure that fills the builder.

Removed now-unused `os`/`path/filepath` imports from `writer_schemas_txt.go` and `writer_llms.go`.

### 3. Dedup baseline golden updated
**File:** `.art-dupl-baseline.json`

The baseline was stale — referenced eliminated clone groups (postgres/sqlite openBackend,
catalog writer builder). Re-ran `art-dupl baseline . --threshold 3 --semantic` to record the
current 16 accepted clone groups. The `nix run .#check-duplication` gate now passes cleanly.

### 4. Full verify gate passed
`nix run .#verify` — all checks green:
- Build, Vet, Test (all 90+ modules)
- Race detector (all modules)
- Lint (all modules, 0 issues)
- API stability (golden match)
- Doc check (947 references valid across 39 packages)
- Duplication check (0 new clones vs baseline)

### 5. Pushed to remote
7 commits pushed to `origin/master`.

---

## B) PARTIALLY DONE

Nothing — every consolidation attempted was either completed or reverted.

---

## C) NOT STARTED

### Turso preset — investigated, intentionally skipped
`stack/turso/backend.go` uses turso-specific APIs (`cqrsturso.Open`, `cqrsturso.NewBackend`)
instead of `sql.Open` + `storage.NewSQLBackend`. The shape is different enough that
`OpenPrimaryBackend` would not improve it. Art-dupl confirmed no clone detected there.
This was the correct call — no action needed.

---

## D) TOTALLY FUCKED UP (and fixed)

### `decider/cache.go` closure-based `locked` helper — OVER-ENGINEERED, then REVERTED

**The mistake:** The original clone was `c.mu.Lock(); defer c.mu.Unlock()` appearing in
`Get`, `Put`, and `Invalidate` — the idiomatic Go mutex guard. I replaced it with a
closure-based `locked(ref, fn)` helper that:

1. Added indirection (closure captures, function pointer)
2. Forced ugly blank-line-separated `var` declarations (linter `wsl_v5` required it)
3. Made the code harder to read, not easier
4. Was a textbook "Accept" case per the dedup skill: "An abstraction would take more
   parameters than the duplicated code has lines"

**The fix:** Reverted to the original idiomatic `c.mu.Lock(); defer c.mu.Unlock()` pattern
with a one-line rationale comment explaining why the duplication is accepted.

**Lesson:** `mu.Lock(); defer mu.Unlock()` is NOT duplication — it is the Go idiom for
mutex guards. Extracting it into a closure helper is always wrong. The dedup skill
explicitly calls this out: "An abstraction would take more parameters than the
duplicated code has lines."

### Forgotten git commits between changes
The auto-commit daemon handled most commits, but I should have committed each
self-contained change explicitly. The session produced 7 commits but they were
batched by the daemon rather than logically separated.

### Forgotten baseline update
After the catalog + stack consolidations, I initially forgot to update
`.art-dupl-baseline.json`. This would have caused `nix run .#check-duplication`
to fail on the next session. Caught and fixed in the reflection round.

---

## E) WHAT WE SHOULD IMPROVE

### Process improvements
1. **Commit after each self-contained change** — don't rely on the auto-commit daemon.
   Each consolidation should be its own commit with a clear message.
2. **Update the dedup baseline in the same change** as the consolidation — not as an
   afterthought.
3. **Run `nix run .#check-duplication` immediately** after a consolidation, not just
   at the end via `nix run .#verify`.
4. **Recognize "Accept" cases faster** — `mu.Lock` patterns, standard error wrapping,
   and idiomatic boilerplate should be accepted on sight, not refactored.
5. **The AGENTS.md dedup helper section** should document the `writeBuilderFile` and
   `OpenPrimaryBackend` patterns as examples of good extraction.

### Code improvements (broader, noticed during this session)
6. **`writeBuilderFile` could be promoted** — the pattern (fill a `strings.Builder`,
   write to file) is common across the codebase beyond catalog. Consider a shared
   `fileutil.WriteBuilder(dir, filename, fn)` if more call sites exist.
7. **`OpenPrimaryBackend` could serve future SQL presets** — if a MySQL or CockroachDB
   preset is ever added, it should use this helper.
8. **Turso's `applySchemaAndPragmas`** is a good extraction — the same pattern exists
   in sqlite's setup closure now. Could these share code? Probably not worth it given
   the turso-specific `InitSchemaWithIndexesAndOptimizations` branch.

---

## F) Up to 50 things we should get done next

### Dedup / code quality (immediate)
1. **Audit the 16 accepted baseline clones** — review each for whether it can be eliminated now
2. **`storage/memory` wrap helpers** (`withWriteLock`, `withReadLock`) — are they still needed
   after this session's changes?
3. **`signing/hmac.go` + `signing/cose.go`** clone (hash computation) — investigate
4. **`storage/pebble/command_read.go` + `query_read.go`** clone — investigate
5. **`projectionhost/dlq.go` + `middleware/deadletter.go`** clone — investigate
6. **`metaengine/fold.go` + `storage/view/auto.go`** clone — investigate
7. **`kv/mem.go` + `dispatcher/lifecycle.go`** clone — investigate
8. **`storage/pg_bus_dispatch.go` + `command/memory_bus.go`** clone (appears twice in baseline) — investigate
9. **Run `art-dupl` at threshold 5** — catch larger structural clones
10. **Run `art-dupl --html`** for visual review of the 16 baseline groups

### Catalog writers (follow-up)
11. **`catalog/eventcatalog` has 3 more writers** — check if `writeServices`, `writeMessages`,
    etc. can use `writeBuilderFile`
12. **Audit all `os.WriteFile` + `strings.Builder` patterns** repo-wide for `writeBuilderFile` adoption
13. **Consider a `writeJSONFile` companion** — the JSON writers (`writePackageJSON`, etc.) have a
    similar marshal-then-write pattern

### Stack presets (follow-up)
14. **Add `OpenPrimaryBackend` to `stack/sqlopt` README** — document the helper
15. **Consider extracting `ConfigureSQLitePool` + `SQLiteEnableWAL` into a single
    `SQLiteSetup` function** — they're always called together
16. **Multi-DB presets** — `openSecondaryBackend` in sqlite/postgres/turso still duplicates;
    check if `sqlopt.NewSecondaryBackend` already covers it (it does — verify all presets use it)

### Type model improvements
17. **`config` struct in each preset** — `stack/sqlite/config`, `stack/postgres/config`,
    `stack/turso/config` all embed `sqlopt.DSNConfig` + `sqlopt.PragmaConfig` but add
    preset-specific fields. Consider a shared `sqlopt.PresetConfig` base.
18. **`storage.SQLBackend` dialect** — the backend knows its dialect but callers must still
    pass `storage.NewSQLiteBackend` vs `storage.NewSQLBackend`. Could a dialect-tagged
    factory simplify this?
19. **`errorfamily` codes** — each preset uses unique codes (`sqlite_preset.open_primary`,
    `postgres_preset.open_primary`). Consider a structured `ErrorCode{Module, Operation}`
    type instead of string conventions.

### Library adoption / ecosystem
20. **`hashicorp/go-multierror` or `errors.Join`** — the preset cleanup paths
    (`_ = db.Close()` in defers) silently discard errors. Consider collecting them.
21. **`cleanerr` or similar** — the wrap-then-return pattern could benefit from a
    dedicated helper library. The project already has per-module `wrapXOrOK` helpers;
    consider standardizing.

### Testing
22. **Add tests for `OpenPrimaryBackend`** — it's currently untested in isolation
    (only exercised via preset integration tests)
23. **Add tests for `writeBuilderFile`** — currently only tested via catalog export tests
24. **Property-test the LRU cache** — `decider/cache.go` has table tests but no rapid-based
    property tests for concurrent access
25. **Test the dedup gate in CI** — verify `nix run .#check-duplication` runs in GitHub Actions

### Documentation
26. **Update `AGENTS.md` dedup helper section** with `OpenPrimaryBackend` + `writeBuilderFile`
    as examples of good extraction
27. **Add a "when to accept duplication" section** to the dedup skill — document the
    `mu.Lock` lesson
28. **Document the 16 accepted baseline clones** in a `dedup-acceptance.md` file per the skill
29. **Update `stack/sqlopt/doc.go`** to mention `OpenPrimaryBackend`

### Broader architecture
30. **Audit all `openBackend` variants** — memory, pebble, turso all have their own.
    Is there a shared `BackendOpener` interface worth defining?
31. **Consider a `stack.Preset` interface** — each preset (`sqlite.New`, `postgres.New`,
    `turso.New`) has the same shape: options → config → open backend → wire stores → return bundle.
    A preset interface could enable preset-level testing helpers.
32. **`metaengine` integration** — how does the metaengine storage planner relate to the
    preset backends? Could `OpenPrimaryBackend` feed into a metaengine-aware preset?
33. **Projection host + preset integration** — presets could optionally include a
    projectionhost pre-wired to the event store.

### CI / tooling
34. **Add `art-dupl --html` output to CI artifacts** — for visual review on each PR
35. **Add a "clone budget" check** — fail if clone count increases beyond baseline + N
36. **Run `art-dupl` at multiple thresholds** (3, 5, 10) in CI for trend tracking
37. **Add `gocyclo` or `cyclop`** to the lint gate — catch complexity before it becomes clones

### Cleanup
38. **Remove `//nolint:wrapcheck` where `OpenPrimaryBackend` now handles wrapping** —
    verify no stale nolint directives remain
39. **Audit unused `//nolint` directives** repo-wide — `nolintlint` catches some but not all
40. **Check if `storage.ConfigureSQLitePool` is still called outside presets** — if not,
    consider making it unexported
41. **Verify `filePerm` constant** is consistent across catalog writers (it is — they share it)

---

## G) Questions I CANNOT figure out myself

### 1. Should `writeBuilderFile` live in `catalog/eventcatalog` or a shared `fileutil` package?
The pattern (fill a `strings.Builder`, write to a file) is not catalog-specific. But extracting
it to a shared package requires deciding: which module? `testutil`? A new `fileutil`?
Or is it too trivial to justify a package? I cannot decide this without knowing your preference
on utility package granularity.

### 2. Should the 16 accepted baseline clones be documented in `dedup-acceptance.md`?
The dedup skill says: "When many groups are accepted, record the rationale in a
`dedup-acceptance.md` file." There are 16 groups. Should I create this file with one-line
rationales for each, or is the baseline JSON sufficient?

### 3. Should `OpenPrimaryBackend` replace the turso preset's `openLocalBackend` too?
Turso uses `cqrsturso.Open` (not `sql.Open`) and `cqrsturso.NewBackend` (not
`storage.NewSQLBackend`). The control flow is similar enough that `OpenPrimaryBackend`
*could* work if we generalize the open and newBackend function signatures further.
But this would make the helper more abstract for one call site. Is the consistency
worth the abstraction cost?
