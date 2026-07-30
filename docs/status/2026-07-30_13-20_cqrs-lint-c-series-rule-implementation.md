# Status Report: cqrs-lint C-series Rule Implementation

> **Date:** 2026-07-30 13:20
> **Session scope:** Implementing 6 remaining correctness rules (C018, C021, C024-C027) from `IMPROVEMENT_IDEAS.md`
> **Branch:** master

---

## a) FULLY DONE

### Rules implemented (6 new detectors + 19 test cases)

| Rule | File | Detection | Tests |
|------|------|-----------|-------|
| **C018** | `pkg/rules/correctness/c018.go` | `memory.NewMemoryStore()` used as journal fallback in type assertion/switch chains | 3 tests (type assert, type switch, negative) |
| **C021** | `pkg/rules/correctness/c021.go` | `Lock()`/`RLock()` followed by `DecodePayloadAuto`/`json.Unmarshal` before `Unlock()` (deferred unlocks ignored, function literals skipped) | 4 tests (DecodePayloadAuto, json.Unmarshal, decode outside lock, no decode) |
| **C024** | `pkg/rules/correctness/c024.go` | In-memory mutation (`s.field = ...`, `m[key] = ...`) + `*ToSQL`/`*ToDB` write call without `Begin`/`BeginTx`/`RunInTx` transaction | 3 tests (dual-write, with transaction, no mutation) |
| **C025** | `pkg/rules/correctness/c025.go` | `fmt.Errorf` without `%w` in files importing go-cqrs-lite modules (escalates D006 from info to warning for CQRS code) | 3 tests (bare errorf in CQRS file, with %w, non-CQRS file) |
| **C026** | `pkg/rules/correctness/c026.go` | TTL constant defined (`*TTL*` in name) but different literal value passed to `idempotency.NewMemoryStore` or `middleware.*Idempotency` | 4 tests (NewMemoryStore mismatch, const used, middleware mismatch, no const) |
| **C027** | `pkg/rules/correctness/c027.go` | `bus.Subscribe`/`SubscribeAll` in a codebase that also calls `projectionhost.New` | 3 tests (both present, no projectionhost, projectionhost only) |

### Wiring & registration

- All 6 detectors registered in `pkg/rules/register.go` (total: 84 detectors)
- All 6 catalog entries added to `pkg/rules/catalog.go` (correctnessRules)
- Meta-test count updated from 78 to 84 in `pkg/rules/meta_test.go`
- Catalog/register bidirectional test passes (`TestCatalogCountMatchesRegister`)

### Documentation updated

- `README.md`: rule count 78 → 84, correctness 21 → 27, rule table expanded with all 6 new rows
- `VALIDATION_REPORT.md`: scope 65 → 78 → 84
- `AGENTS.md`: `65 rules across 6 categories` → `84 rules across 8 categories`
- `IMPROVEMENT_IDEAS.md`: All 12 C-series items annotated with `~done at <hash>` markers
  - Items 1, 2, 4, 5, 7, 8 (pre-existing): annotated with original commit hashes
  - Items 3, 6, 9, 10, 11, 12 (this session): annotated with `pending` (commit hash TBD)

### Verification

- `go build -tags "goexperiment.jsonv2" ./...` — PASS
- `go vet -tags "goexperiment.jsonv2" ./...` — PASS
- `go test -tags "goexperiment.jsonv2" -count=1 ./...` — ALL PASS
- `go test -tags "goexperiment.jsonv2" -count=1 -race ./...` — ALL PASS
- `gofmt -w` applied to `c026.go` (was the only unformatted file)

---

## b) PARTIALLY DONE

### IMPROVEMENT_IDEAS.md annotations
- The 6 new rules are marked `done at pending` — the commit hash needs to be filled in after committing.
- The summary statistics table at line 484 still says `16` existing rules for Correctness — should be `27` now (16 original + 11 new across sessions).

### C025 code duplication
- `collectPkgLevelVarCalls` in `c025.go` is a copy of `collectPackageLevelVarCalls` in `consistency/d006.go`. Both scan package-level var declarations for sentinel error positions. This should be extracted to `lintutil` as a shared helper.

### C024 detection heuristic
- The dual-write detection uses name-based heuristics (`*SQL` + `*sync*`/`*write*`/`*save*`/`*persist*`). This catches the documented pattern (`syncToSQL`, `writeToDB`) but may miss methods with different naming conventions. A more robust approach would track data-flow from mutation to DB call, but that requires type information beyond AST-only analysis.

### C021 lock tracking
- The lock-state tracking is linear (source order), which works for simple `Lock() ... Unlock()` sequences but doesn't handle complex control flow (e.g., `Lock()` in one branch of an if-statement). This is a known limitation of AST-based analysis without control-flow graph construction.

---

## c) NOT STARTED

### Remaining IMPROVEMENT_IDEAS.md items (158 of 170)
The 170-item improvement plan has 12 items done (all C-series). The remaining 158 items span:
- **A-series (A002-A031):** 16 ideas — API misuse improvements and new rules
- **B-series (B016-B026):** 10 ideas — boilerplate reduction
- **E-series (E008-E015):** 8 ideas — architecture rules
- **D-series (D007-D012):** 6 ideas — consistency rules
- **S-series (S004-S007):** 4 ideas — security rules
- **P-series (P002-P013):** 12 ideas — performance rules
- **V-series:** 6 ideas — version/migration
- **T-series:** 8 ideas — testing quality
- **F-series:** 17 ideas — feature adoption coaching
- **DX & Infrastructure:** 24 ideas — linter tooling improvements
- **Misc:** 27 ideas — additional detection patterns

### `nix fmt` (treefmt) not run
- `gofmt` was applied to new files, but the full `nix fmt` (which runs `gofumpt` + `goimports` + `golines` at 120 chars) was not run. Long lines in the new files may need wrapping.

### `nix run .#verify` not run
- The comprehensive verify gate (build + vet + test + race + lint + doc-check + doc-assertions) was not run. Individual `go build`, `go vet`, `go test`, and `go test -race` were run manually.

### API-stability golden not regenerated
- No exported symbols were added/removed in the library modules (only in `cmd/cqrs-lint`), so the api-stability golden should be unaffected. But this was not verified.

---

## d) TOTALLY FUCKED UP

### Nothing is irrevocably broken.
- All code compiles, all tests pass, `go vet` is clean.
- The `IMPROVEMENT_IDEAS.md` annotation format went through 3 iterations before matching the user's desired format (`~<original text>~ done at <hash>`). This wasted time but caused no damage.
- The `hasDeferAncestor` function name collision between `c021.go` and `c023.go` was caught by the compiler and fixed immediately (renamed to `hasDeferAncestorC021`).
- The `collectPackageLevelVarCalls` cross-package reference was caught by the compiler and fixed by creating a local copy (`collectPkgLevelVarCalls`).

---

## e) WHAT WE SHOULD IMPROVE

1. **Extract `collectPkgLevelVarCalls` to `lintutil`** — D006 and C025 both have identical sentinel-var collection logic. One shared helper eliminates the duplication.
2. **C021 lock tracking needs control-flow awareness** — The current linear approach misses locks acquired in conditional branches. Consider a simplified CFG or at minimum track lock state per-branch.
3. **C024 needs broader DB-write detection** — Name-based heuristics (`*SQL*` + write verbs) will miss methods like `store.Save()` or `repo.Persist()`. Consider detecting any method call on a DB/store/sql object after in-memory mutation.
4. **C025 overlaps with D006** — Both flag `fmt.Errorf` without `%w`. C025 is CQRS-scoped + warning severity; D006 is global + info severity. Consider making D006 skip CQRS files (let C025 own them) to avoid double-reporting.
5. **C027 doesn't check event type overlap** — The rule fires on ANY `bus.Subscribe` when `projectionhost.New` exists, but the idea description says "overlapping event types." Currently it can't determine which event types a projection handles vs. which a subscription handles without type resolution.
6. **Run `nix fmt` before committing** — `gofmt` alone doesn't enforce the 120-char line limit or gofumpt rules.
7. **Run `nix run .#verify`** — The full verify gate should be run to catch anything the manual checks missed.
8. **Update IMPROVEMENT_IDEAS.md summary table** — Line 484 still says "16 existing" for Correctness; should reflect current state.
9. **C018 `mentionsJournal` is name-based** — Checks for `event.Journal`, `event.SeekableJournal`, `Journal`, `SeekableJournal` by string. A consumer aliasing the import (`ev Journal`) would be missed.
10. **Test coverage could be deeper** — C018 has no test for `event.SeekableJournal` (only `Journal`). C021 has no test for `RLock`/`RUnlock` specifically. C024 has no test for `RunInTx` as the transaction guard. C026 has no test for `EventIdempotency`/`QueryIdempotency` (only `CommandIdempotency`).

---

## f) Next 50 things to get done

### Immediate (this session's loose ends)
1. Commit the 6 new rules and fill in the `pending` hash annotations in `IMPROVEMENT_IDEAS.md`
2. Run `nix fmt` on the new files
3. Run `nix run .#verify` for full gate verification
4. Update IMPROVEMENT_IDEAS.md summary table (line 484: 16 → 27 existing correctness rules)
5. Extract `collectPkgLevelVarCalls` to `lintutil` as shared helper
6. Make D006 skip CQRS-importing files (let C025 own them, avoid double-reporting)
7. Add deeper test cases: C018 with `event.SeekableJournal`, C021 with `RLock`/`RUnlock`, C024 with `RunInTx`, C026 with `EventIdempotency`/`QueryIdempotency`

### A-series rules (next priority from IMPROVEMENT_IDEAS.md)
8. A002: Detect `marshalPayload` helper two-step pattern
9. A014: Verify ALL `event.NewEvent` calls are flagged
10. A016: Context-aware idempotency (custom store detection)
11. A017: Check for snapshot strategy, not just store (`WithSnapshotStore` without `WithSnapshotStrategy`)
12. A020: Custom event.Bus reimplementation detection
13. A021: Custom event.Store reimplementation detection
14. A022: Raw `otel.Tracer()` instead of `cqrsotel`
15. A023: Custom in-memory snapshot store reimplementation
16. A024: Decorative event sourcing (decider shape, no wiring)
17. A025: Command/query only, no events
18. A026: Event bus only, no CQRS pipeline
19. A028: cqrs-htmx used only for HTTP middleware
20. A029: `bus.UsePublish` stub returning nil
21. A030: In-memory checkpoint store with persistent event store (similar to C017)
22. A031: In-memory DLQ with persistent event store (similar to C017)

### B-series rules
23. B016: Manual checkpoint replay table
24. B017: Manual read model rebuild from scratch on startup
25. B018: Repeated bus.Subscribe boilerplate (3+ with identical error handling)
26. B019: O(n²) read model (repo.Load per event in SubscribeAll) — partially covered by P001
27. B020: Manual legacy field upcasting instead of schema.Upcaster
28. B022: Manual correlation enricher instead of CommandCausalityEnricher
29. B025: Missing state cache on repository
30. B026: Manual event type registration instead of catalog

### E/D/S/P/V/T/F-series (selection)
31. E008: cqrs-htmx primary path bypasses stack presets
32. E009: No HTTP integration for CQRS
33. E013: Signing configured but disabled by default
34. E014: No read-your-writes consistency
35. D007: Inconsistent event creation API (event.New vs event.NewEvent)
36. D008: Inconsistent codec usage (DecodePayload vs DecodePayloadAuto)
37. D010: Generic error code "internal"
38. D012: Schema version not stamped on events
39. S004: PII data without encryption
40. S005: Event signing available but disabled
41. P002: Full read model rebuild on every startup
42. P003: Mutex held during payload decode (overlaps with C021 — consolidate)
43. P004: Multiple repository instances (overlaps with C019 — consolidate)
44. V-series: Version mixing detection rules
45. T-series: Test quality rules (scenario tests, coverage)
46. F-series: Feature adoption coaching (scorecard, profile command)

### Linter DX & Infrastructure
47. Incremental analysis — cache AST results for faster re-runs
48. Feature adoption scorecard — show what each project uses vs misses
49. Profile command — detailed usage analysis per project
50. cqrs-htmx-aware rules — different defaults for framework consumers

---

## g) Questions I cannot answer myself

1. **Should C025 and D006 be consolidated into one rule, or kept separate?** C025 escalates D006's detection from info to warning for CQRS files. If they coexist, the same `fmt.Errorf` call gets reported twice (once as C025 warning, once as D006 info). Should I make D006 skip files that import go-cqrs-lite modules, or is double-reporting acceptable (different audiences)?

2. **Should C021 (mutex held during decode) and P003 (same pattern, performance framing) be consolidated?** The IMPROVEMENT_IDEAS.md lists both C021 and P003 as the same anti-pattern with different severity framings (correctness vs performance). C021 is implemented; P003 is not. Should P003 be dropped in favor of C021, or should P003 detect the pattern with a performance-focused message?

3. **Should the IMPROVEMENT_IDEAS.md `pending` placeholders be filled with the actual commit hash after committing, or is there a different workflow for tracking this?** The pre-existing items have real hashes (`b31eb572`, `ec402374`, etc.). The 6 new items say `pending` because no commit exists yet. Should I commit and then amend the file with the hash, or leave `pending` for a future session to fill?
