# Open Issues Remediation — Comprehensive Status

**Date:** 2026-10-23 09:28 (+0200)
**Scope:** All 10 open GitHub issues on `LarsArtmann/go-cqrs-lite`
**State:** 8 of 10 issues implemented and unit-verified; 1 verified as already fixed; 1 (baseline feature) partially implemented. **CI gates are NOT green yet**; see Blockers.
**Authored commits:** none by this session (the auto-commit daemon absorbed work into `chore: auto-commit` commits; see Process Notes).

---

## 1. Executive Summary

I reviewed and implemented fixes for the entire open-issue backlog:

| # | Title | Status | Verified |
|---|-------|--------|----------|
| 78 | E005 misses cross-package selectors in `system.RegisterCommand` | **DONE** | Yes |
| 80 | C010 flags checked-error `if _, err := decode(...)` idiom | **DONE** | Yes |
| 81 | E003 pins every finding at `go.mod:1:1` (ID collision) | **DONE** | Yes |
| 79 | E004/F002 blind to `catalog/v4/simple` builder registrations | **DONE** | Yes |
| 82 | F010 does not recognize metaengine-native graph usage | **DONE** | Yes |
| 74 | S001 flags schema NAMES as critical hardcoded secrets | **DONE** | Yes |
| 71 | `TypedReader.Scan` truncates at 100 rows with no signal | **DONE** | Yes |
| 72 | metaengine has no keep-newest-N cap primitive | **CORE DONE**, docs recipe missing | Yes |
| 70 | `projectionadapter.EventWithID.ID` carries display form | **ALREADY FIXED** before this session | Yes |
| 83 | cqrs-lint checked-in findings baseline | **IN PROGRESS** (~60%) | No |

**Overall confidence:** High for the seven cqrs-lint fixes and both metaengine primitives; they are small, unit-tested, and follow existing conventions. Medium for the cap primitive's operational shape (explicit enforcement rather than per-insert), which is a deliberate design decision that needs your sign-off. The baseline feature is structurally complete but not yet wired to the CLI, so it currently breaks the `cmd/cqrs-lint` build (expected, not a regression in shipped code).

**Three things block "done":**

1. **File-size ratchet violations introduced by this work** (3 files) — CI will fail.
2. **Issue #83 is unfinished** (flags, run wiring, tests, docs).
3. **API-stability golden and CHANGELOG/FEATURES updates** have not been regenerated/written.

---

## 2. Method

For each issue I followed: read the issue body → locate the cited source → confirm the root cause in the current tree → implement the minimal correct fix → add a regression test that fails before / passes after → run the owning package's tests. The 10 issues split naturally into:

- **7 cqrs-lint detector fixes** (78, 80, 81, 79, 82, 74) — local AST/analysis changes plus regression fixtures.
- **1 metaengine API feature** (71) — additive option + sentinel error.
- **1 metaengine capability** (72) — new query option + store maintenance API.
- **1 already-fixed issue** (70) — confirmed by source and tests.
- **1 CLI feature** (83) — new file format, matching semantics, flags, wiring.

The draft investigation docs under `docs/drafts/2026-10-23-crm-lint-fps/` (written before this session) were used as the starting evidence and cross-checked against source. They are now tracked in git (the daemon committed them together with `scanner_generic.go` in `39754076e`).

---

## 3. Issue-by-Issue Detail

### #78 — E005 misses cross-package selectors in `system.RegisterCommand` — DONE

**Root cause confirmed:** both registration-extraction paths accepted only `*ast.Ident`, so `deal.LoseCmd` (a `*ast.SelectorExpr`) yielded `""` and the registration was never recorded, leaving every command looking handler-less.

**Fix:**
- `cmd/cqrs-lint/pkg/analyzer/scanner_generic.go` — `typeNameFromGenericArg` now returns `sel.Sel.Name` for `*ast.SelectorExpr` (handles `system.RegisterCommand[deal.LoseCmd, deal.State]`).
- `cmd/cqrs-lint/pkg/analyzer/scanner_calls.go` — `handlerTypeFromClosure` now handles `*ast.SelectorExpr` and **skips `context.Context` explicitly** via new `isContextSelector` (previously the first parameter was returned unconditionally; with the new case, `ctx context.Context` would have been misread as the handler type). The `*ast.StarExpr` leg also guards against an empty `lastIdentSegment` result.

**Tests:**
- `TestE005_NoFindingWhenRegisteredViaSystemCrossPackage`
- `TestE005_NoFindingWhenHandlerParamIsQualifiedSelector`
- Existing `TestE005_NoFindingWhenRegisteredViaSystem` still passes.

**Caveat:** resolution is by trailing type name, so two same-named command structs in different packages still collide. That limitation predates this issue and matches the existing registry design; it is not addressed.

---

### #80 — C010 flags the checked-error idiom — DONE

**Root cause confirmed:** any `*ast.AssignStmt` with a blank LHS and a Decode-family RHS emitted a finding, with no inspection of the enclosing statement. The blank discards the payload; the error is tested by the enclosing `if`.
 
**Fix** (`cmd/cqrs-lint/pkg/rules/correctness/swallow_helpers.go`):
- New `checkedSwallowAssigns(body)` pre-pass collects assignments whose non-blank LHS is inspected by an adjacent `if`.
- Two accepted forms:
  1. `if _, err := decode(...); err != nil { ... }` — `IfStmt.Init == assign` and `IfStmt.Cond` mentions `err`.
  2. two-line form: assignment immediately followed by `if err != nil`.
- `condInspectsAssign` uses a name set of non-blank LHS idents and an AST walk of the condition.
- Applied before emission, so both the fold path and the inline-closure callback path are covered (they share `inspectBodyForSwallowedError`).

**Tests:**
- `TestC010_NoFindingForCheckedIfInitDecode` (0 findings)
- `TestC010_NoFindingForCheckedTwoLineDecode` (0)
- `TestC010_DetectsUncheckedIfInitDecode` (still 1 when the condition ignores the error)
- Existing `TestC010_DetectsSwallowedError` / `TestC010_DetectsSwallowedSQLError` still pass (regression guard for `p, _ := decode(...)`).

---

### #81 — E003 position collision collapses findings — DONE

**Root cause confirmed:** every package-level E003 finding used `go.mod:1:1`. The finding ID is `tool:rule:file:line:col`, so N packages produced one ID; the CLI's `collectFindings` kept whichever map entry iterated first (stable count, nondeterministic content), while the toolspec path reported all N.

**Fix** (`cmd/cqrs-lint/pkg/rules/architecture/e003_e007.go`) — chose the issue's preferred option (unique real positions):
- New `packageBoundaries` accumulator records, per package, the construct kinds **and the earliest contributing source position** (file, then line, then column).
- Packages are iterated in sorted order, so output order is deterministic too.
- `pos()` uses the anchor's real position; falls back to the anchor file at `1:1` only when the position is invalid, and to `go.mod:1:1` only when no anchor exists at all.

**Tests:**
- `TestE003_EmitsDistinctIDsPerPackage` — two packages with 3 constructs each → exactly 2 findings, 2 distinct IDs, neither anchored at `go.mod`.
- Existing `TestE003_DetectsMixedConcerns` still passes.

**Behavior change to announce:** consumers will now see **N E003 findings where they previously saw 1**. Counts will rise on any multi-module/multi-package consumer (the CRM was 1 vs 5). This is the intended fix, but it is a reporting-visible change and belongs in the CHANGELOG and in consumer baselines.

**Remaining question:** the toolspec path still does not run `collectFindings`. With unique IDs that is harmless today, but a future duplicate-ID rule would diverge again. Optional hardening: run the same dedup in `pkg/toolspec/toolspec.go`.

---

### #79 — E004/F002 blind to `catalog/v4/simple` — DONE

**Root cause confirmed (two of the three cited gaps; the third is already handled by current code):**
1. `simple.Event[T](b *Builder, id string, direction, opts...)` puts the builder at arg 0; the scanner read arg 0.
2. The id often arrives as `string(eventType)`, and the catalog Event case did not unwrap a string conversion.
3. The pending-ref post-pass already uses `ExprIdentName`, which **does** resolve `*ast.SelectorExpr` to its trailing name; cross-package const values therefore work once the ref is recorded. (The issue's source citation predates that helper's use at this call site.)

**Fix:**
- `cmd/cqrs-lint/pkg/analyzer/scanner_calls.go`
  - New case qualified for `go-cqrs-lite/catalog/simple`, placed **before** the generic catalog case so it wins: reads **arg 1**, unwraps `string(...)` conversions, then records literal or pending const ref.
  - Extracted `recordCatalogEventType` so the plain `catalog.Event` and `simple.Event` paths share one implementation (removes the duplicated literal/const branch).
  - Added `catalogSimpleMessageIDArg`.
- `cmd/cqrs-lint/pkg/rules/adoption/f002_f005.go` — F002 now accepts `simple.New(...)` as a catalog declaration in addition to `catalog.NewBuilder()`.
- `cmd/cqrs-lint/pkg/rules/adoption/scan_in.go` + `helpers.go` — new `hasCallOnImportedPackage` / `projectHasCallInPath` helpers resolve the qualifier by import path when type info is available (so aliased imports work) and fall back to the qualifier name in AST-only mode.

**Tests:**
- `TestE004_NoFindingForCatalogSimpleBuilderLiteral` (0)
- `TestE004_NoFindingForCatalogSimpleBuilderConst` (0, covers `string(Const)` plus multi-file const resolution)
- `TestE004_DetectsUnregisteredSimpleBuilderEvent` (1, guards the arg-1 read)
- `TestF002_NoFindingWithCatalogSimpleBuilder` (0)

**Caveat:** only `Event` registrations feed `EventTypesInCatalog`. `simple.Command` / `simple.Query` are not recorded, matching the existing catalog scanner's event-only behavior; no current rule consumes command/query catalog declarations.

---

### #82 — F010 does not recognize metaengine-native graph usage — DONE

**Root cause confirmed:** the only skip was `importsPathIn(files, "go-cqrs-lite/graph")`; a consumer projecting `metaengine.Edge`/`EdgeRemoval` folds failed it and got coached to migrate onto the module it deliberately left.

**Fix** (`cmd/cqrs-lint/pkg/rules/adoption/f010_f011.go`, `scan_in.go`):
- New `usesMetaengineGraph(files)`: requires an import containing `go-cqrs-lite/metaengine` **and** an AST reference to a selector named `Edge` or `EdgeRemoval`.
- Skip condition is now `graph import OR metaengine Graph ADT`.
- Suggestion text now leads with the metaengine Graph ADT and offers `graph.NewGraphProjection` as the alternative, instead of presenting the standalone module as the only path.

**Tests:**
- `TestF010_NoFindingWithMetaengineGraphADT` (0)
- `TestF010_FindingWhenMetaengineImportedWithoutGraphADT` (1 — import alone is not evidence)
- Existing F010 tests still pass.

**Caveat:** the reference check is the same cheap string-match budget the rule already spent. A project importing metaengine and using an unrelated `.Edge` selector (e.g. a mesh library) would suppress the coaching. F010 is `Info`/low-confidence adoption coaching, so the tradeoff favors fewer false accusations; tightening to `IsQualifierFor(gf, sel, "go-cqrs-lite/metaengine")` is a possible follow-up.

---

### #74 — S001 flags schema NAMES as critical secrets — DONE

**Root cause confirmed:** the identifier substring list includes `token`/`key`/`secret`, and the only value escapes were length ≥ 8 and `IsURLOrPlaceholder`, so `const enrollTokensCollection = "enroll_tokens"` was a `critical` build-gating finding with unactionable "use env vars" advice.

**Fix:**
- `cmd/cqrs-lint/pkg/rules/lintutil/url_placeholder.go` — new `IsNameShapedValue` (pure lowercase snake_case: `^[a-z][a-z0-9_]*$`, no leading digit/underscore).
- `cmd/cqrs-lint/pkg/rules/security/rules.go`
  - The `check` closure now takes `isConst` and applies the name-shape allowlist **only when the binding itself says "identifier"**: a `const` declaration, or an identifier carrying a storage/schema suffix (`Collection`, `Table`, `Queue`, `Bucket`, `Topic`, `Index`, `Schema`, `ViewName`, including `*Name` forms).
  - Const-bound true positives are **downgraded from `critical` to `warning`** and get an actionable suggestion: schema vocabulary needs no action; real key material must move to a `var` read from the environment/secret manager.

**Why the allowlist is scoped (important deviation from the issue's literal proposal):** a bare `^[a-z][a-z0-9_]*$` value allowlist suppresses real lowercase API keys (`supersecretvalue123`, `sk_live_...`) and immediately broke the existing positive test `TestS001_DetectsHardcodedKey`. The two-signal form (`name-shaped` + `const`/structural name) kills the reported class without weakening var-assignment detection.

**Tests:**
- `TestS001_NoFindingForSchemaVocabularyConst` (0)
- `TestS001_NoFindingForSchemaVocabularyVariable` (0)
- `TestS001_ConstSecretIsDowngradedToWarning` (1 finding, severity asserted `warning`)
- Entire pre-existing S001 suite passes, including `TestS001_AllowlistKeepsRealCredentials` and `TestS001_StillDetectsHashesAndSecrets`.

**Decision for you:** confirm the suffix list (`Schema`, `ViewName` are included; `View` is not, to avoid matching `Review`/`Preview`) and whether `warning` is the right downgrade level (vs `info`).

---

### #71 — `TypedReader.Scan` truncation signal — DONE

**Root cause confirmed:** `Scan` returns `([]V, error)`; the internal `scanItems` computes `pageMeta.hasMore` and discards it, so a 100-row default result is indistinguishable from a complete one.

**Fix (additive; no signature change, so the api-stability contract holds):**
- `metaengine/scan_options.go` — `scanConfig` gains `exact bool`; new `WithExact()` option.
- `metaengine/errors.go` — internal `errScanTruncated` + exported `ErrScanTruncated`.
- `metaengine/typed_reader_scan.go`
  - `scanItems` now applies options and delegates scanning to a new `scanItemsConfigured(ctx, cfg)`; after scanning, if `cfg.exact && meta.hasMore`, returns `ErrScanTruncated` with collection, limit, and remediation guidance.
  - The prefetch-cache fast path now reports `hasMore` (`len(cached) > cfg.limit`) instead of always `false`, so `WithExact` is not silently defeated when a PrefetchCache is attached.
  - `Scan`'s godoc now documents the 100-row default, `WithLimit(0)`, `WithDefaultLimit`, and points at `WithExact` / `ScanPage` as the two escape paths.

**Tests:** `TestWithExact_MakesTruncationLoud` covers truncated (error + nil rows), exact fit (5/5, no error), and unbounded (`WithLimit(0)`, no error). Full `metaengine` package suite passes.

**Semantics note:** `WithExact` answers "was this the whole result?", not "return more rows". Combined with `WithLimit(0)` it can never fail. `ScanPage` also honors the option (it shares `scanItems`), which is correct but worth documenting.

**Follow-up (not done):** a metadata-returning variant, e.g. `ScanWithMeta() ([]V, ScanMeta, error)`, would let callers branch without turning truncation into an error. The issue accepted `WithExact` as sufficient; flag if you want the richer API.

---

### #72 — metaengine keep-newest-N cap — CORE DONE

**Design chosen:** query-declared policy + explicit store enforcement (issue option 1's declaration plus option 2's helper), not automatic per-insert trimming.

**Why not per-insert:** memory/KV engines implement scans by materializing and sorting the whole collection, so an automatic scan on every insert would make every capped write O(N) or worse. Explicit enforcement keeps the write path clean; callers trim after a batch or on a timer, which is the real cadence of bounded read models.

**Implementation:**
- `metaengine/query_config.go`
  - `QueryConfig` gains `capRows` / `capOrderBy`, public getters `CapRows()` / `CapOrderBy()`.
  - New `WithCap(maxRows int, orderBy string) QueryOption`. `maxRows <= 0` disables; docs state the Map/key requirements and the enforcement entry points.
- `metaengine/cap.go` (new)
  - `validateCapConfig(q)`: rejects empty order-by, non-Map ADT, and result types that do not carry the query's key type. Wired into `Plan()` and `RegisterQuery()` **before** the store is built.
  - `capKeyExtractor(q)`: derives a value→key extractor from the query's `insertFold` (key type + result value type), reusing `buildKeyExtractor` and its ambiguity guard.
  - `Store.EnforceCap(ctx, queryName) (int, error)`:
    1. under the query's fold lock (so it serializes against fold writes and replays of the same query),
    2. reads the **newest `capRows`** rows (descending order-by) as the keep page,
    3. if more rows exist, pages through the older remainder in batches of 500 and deletes every row,
    4. returns the row count removed. No converging loop: descending order makes the keep set a prefix, so every page after it is entirely overflow.
  - `Store.EnforceCaps(ctx)`: sorted query-name order, totals removed rows, joins per-query errors (`errors.Join`), so one bad collection does not hide the others.
  - Deletion honors temporal engines (`MapDeleteAt(…, time.Now())` when `VersionVersionsCells`) and notifies live subscribers (`notifyLive`), matching `applyFoldRemove`.
  - Exported `ErrCapRequiresMap`, `ErrCapUnsupported`; internal sentinels for cursor-missing and config errors.
- `metaengine/planner.go`, `metaengine/register_query.go` — validation wiring.

**Tests (`metaengine/cap_test.go`):**
- `TestWithCap_EnforceCapKeepsNewestRows` — 6 inserts, cap 3 → 3 removed, surviving `Seq` are 4/5/6 (proves "newest", not "first").
- `TestWithCap_EnforceCapIsIdempotent` — under-cap no-op; second pass removes 0.
- `TestWithCap_EnforceCapsOnlyTrimsCappedCollections` — batch trimming + uncapped query returns 0/nil.
- `TestWithCap_RequiresOrderBy` — Plan error names `orderBy`.
- `TestWithCap_RejectsNonMapADT` — `ErrCapRequiresMap` for a Counter query.
- Full `metaengine` suite passes (25.9s).

**Architectural caveat that must be documented loudly:** trimming removes projection rows **outside the event log**. Reset + replay restores the full, uncapped row set, so a replay must be followed by another `EnforceCap`. This is compaction-style derived-state maintenance (ADR-0136 invertibility ladder), not a fact. If replay-deterministic capping is required, the cap has to move into the fold application path and accept the per-write scan cost; that is a design decision, not a bug fix.

**Not done:** the docs recipe (option 3). Planned locations: `metaengine/COOKBOOK.md` and, if consumer-facing, a fence in `.agents/skills/go-cqrs-lite/references/recipes.md` (which requires a `recipes_catalog*.go` classification entry because `TestRecipesCatalogCoversFile` is strict).

---

### #70 — `EventWithID.ID` display form — ALREADY FIXED (verified)

The current source already stamps the **bare** stream value and documents the distinction:

- `metaengine/projectionadapter/typed_decoder.go` — both `Register` and `RegisterString` use `evt.StreamID().Get()`; the `EventWithID` godoc explicitly contrasts it with `StreamID.String()`.
- `metaengine/projectionadapter/typed_decoder_test.go` pins the bare form and performs a raw-ID lookup through a real store.
- `GOWORK=off go test ./... -count=1` in `metaengine/projectionadapter` → **ok** (0.323s).

The fix predates this session (absorbed by an earlier auto-commit; current tip `b30156686` touched the file). **Recommendation:** close #70 with a verification comment rather than re-implementing; no code change is warranted.

---

### #83 — cqrs-lint findings baseline — IN PROGRESS

**What exists:** `cmd/cqrs-lint/baseline.go` (untracked, 357 lines) implements the full matching/persistence core:

- `baselineDocument` / `baselineEntry`: `version`, `tool`, and entries of `rule`, `file`, `line`, `column`, optional `reason`.
- **Portable keying:** entries store the file **relative to the baseline file's directory**, slash-normalized, so a checked-in baseline works across machines and OSes. The finding ID embeds absolute paths and is therefore *not* stored; matching uses `(rule, file, line, column)`.
- `splitBaseline(doc, findings, root)` → `New` / `Known` / `Stale`. Stale entries are sorted deterministically.
- `entriesFromFindings` preserves reasons from the previous document on regeneration (the operator's adjudication notes are never silently dropped).
- `writeBaseline`: deterministic entry sort (rule, file, line, column), 2-space JSONC-compatible JSON via `encoding/json/v2`, trailing newline, atomic temp-file + rename.
- `loadBaseline` rejects unknown versions with guidance; `loadBaselineIfPresent` treats a missing file as empty for the first `--update-baseline`.
- `applyBaseline(cfg, active)` prints (non-quiet) "N findings matched the baseline", "Baseline shrunk: N … run --update-baseline", and per-stale-entry lines; returns `Reported` (new findings only), `Known`, `Stale`, `Updated`.

**What is missing (this is why the build currently fails):**

1. `AppConfig` fields and cmdguard tags:
   ```go
   // skip-validate
   Baseline       string `default:"" flag:"baseline" help:"Path to a findings baseline file (JSON)" local:"true"`
   UpdateBaseline bool   `default:"false" flag:"update-baseline" help:"Rewrite the baseline from the current findings" local:"true"`
   ```
2. `run()` wiring after `filterFindings`:
   - call `applyBaseline`;
   - `--update-baseline` → write, print notice, exit 0 (maintenance run);
   - otherwise output **only** `Reported`;
   - `shouldExitWithError(cfg, Reported)` so baselined findings cannot fail the build;
   - keep the health score on the full `unsuppressed` set so baselining does not inflate the score (deliberate; document it).
3. Tests:
   - split: known/new/stale, relative-path portability, reason preservation;
   - deterministic rewrite (two runs byte-identical, sorted);
   - missing/invalid/version-mismatch errors;
   - `--update-baseline` without `--baseline` errors;
   - end-to-end: 3 findings → baseline → exit 0/quiet; add one → only the new one reported + exit non-zero; fix one → "baseline shrunk".
4. Docs: README/`RULES.md` section for the file, flags, exit semantics, and the stale-entry policy.

**Open decisions for #83:**
- Should stale baseline entries be **fatal** under an opt-in flag (`--fail-on-stale-baseline`), matching `--fail-on-stale-suppressions`? Current draft: visible notice, non-fatal.
- Should baseline matching fall back to `(rule, file, message)` when line numbers move, to reduce churn? Current draft follows the issue and keys on line/column.
- Should `--baseline` be allowed in `.cqrs-lint.json` (it is, via the flag-tag loader) and resolved relative to the config file rather than the CWD? Current draft resolves relative to CWD.

---

## 4. Verification Evidence

Run from the repo root unless noted. All commands used `GOWORK=off` per the multi-module rule.

| Check | Command | Result |
|-------|---------|--------|
| metaengine package | `cd metaengine && GOWORK=off go test . -count=1` | **ok** (25.870s) |
| projectionadapter (#70) | `cd metaengine/projectionadapter && GOWORK=off go test ./... -count=1` | **ok** (0.323s) |
| E003/E004/E005 rules | `cd cmd/cqrs-lint && GOWORK=off go test ./pkg/rules/architecture/ -count=1` | **ok** |
| F-series adoption rules | `... go test ./pkg/rules/adoption/ -count=1` | **ok** (1.087s) |
| C010 correctness rules | `... go test ./pkg/rules/correctness/ -count=1` | **ok** |
| S001 security rules | `... go test ./pkg/rules/security/ -count=1` | **ok** |
| analyzer (scanner changes) | `... go test ./pkg/analyzer/ -count=1` | **ok** |
| toolspec | `... go test ./pkg/toolspec/ -count=1` | **ok** |
| gofmt on all touched files | `gofmt -l <files>` | **clean** (no output) |
| cqrs-lint full suite | `... GOWORK=off go test ./... -count=1` | **FAIL** only in `pkg/rules/version` (2 tests) — see below |
| File-size ratchet | `bash scripts/check-file-size.sh` | **FAIL** — includes 3 violations caused by this work |

**Pre-existing failure (NOT caused by this work):**

```
--- FAIL: TestV007_F090b_AttributesDotImportedRemovedSymbol
--- FAIL: TestV007_F090b_SilentWhenTypedInfoOff
pkg/rules/version/v007_typed_test.go:67/105:
  BuildContext(typedfixture): no packages loaded … the module graph is likely broken
```

Reproduced directly in the fixture:

```
$ cd cmd/cqrs-lint/testdata/typedfixture && GOWORK=off go build ./...
go: updates to go.mod needed; to update it:
        go mod tidy
```

The fixture's `go.mod` (`go 1.27.2`, pinned `schema/v4 v4.6.0` + transitive set) is stale relative to the current module graph. No file in this session touched that fixture or the `version` rules. **Recommended:** `go mod tidy` in the fixture as a separate, small fix (or confirm the owning session already has it in flight).

**File-size ratchet failures caused by this work:**

```
./cmd/cqrs-lint/pkg/analyzer/scanner_calls.go grew 360 → 401 lines (baselined files may only shrink)
./cmd/cqrs-lint/pkg/rules/adoption/scan_in.go has 386 lines (max 350) — NEW offender
./cmd/cqrs-lint/baseline.go has 357 lines (max 350) — NEW offender
```

**Pre-existing, unrelated ratchet failures observed in the same run** (not mine; flagging for ownership):

```
./cmd/doc-check/recipes_catalog_meta3.go        408 lines (NEW)
./catalog/eventcatalog/exporter.go              grew 382 → 392
./catalog/types_resources.go                    359 lines (NEW)
./catalog/architecture/index.go                 415 lines (NEW)
./catalog/architecture/selection.go             636 lines (NEW)
```

Note: the baseline (56 entries) is enforced by `scripts/check-file-size.sh`; the gate's message says "max 350" for new offenders and "may only shrink" for baselined files. `metaengine/planner.go` is baselined at 562 and currently 474, so my +4 lines there are still under the ratchet and safe.

**Not yet run (must run before claiming done):**

- `nix run .#verify` (build + vet + test + race + lint + doc-check + doc-assertions)
- `nix run .#check-arch` (dependency budgets)
- `nix run .#check-lint-config` (golangci config + depguard)
- `nix run .#check-md-go`, `check-adr-numbering` (if docs/ADRs are added)
- `nix run .#check-coverage`, `check-duplication`
- `cd cmd/api-stability && GOWORK=off go run . --update` (new exported surface)

---

## 5. Blockers (must fix before merge/CI green)

### B1. File-size ratchet (three files)

Recommended splits (no behavior change):

1. **`cmd/cqrs-lint/pkg/analyzer/scanner_calls.go`** — move the `catalog/simple` case body plus `catalogSimpleMessageIDArg` and `recordCatalogEventType` into a new `scanner_catalog_simple.go`; the target is ≤ 360 lines to satisfy the ratchet. Alternatively move `handlerTypeFromClosure` / `isContextSelector` into `scanner_calls_helpers.go` (already a helpers home).
2. **`cmd/cqrs-lint/pkg/rules/adoption/scan_in.go`** — move `usesMetaengineGraph` into `scan_in_metaengine.go` and `hasCallOnImportedPackage` into `scan_in_calls.go`, or one new `scan_in_paths.go` file.
3. **`cmd/cqrs-lint/baseline.go`** — split the document/IO layer (`loadBaseline`, `loadBaselineIfPresent`, `writeBaseline`) into `baseline_file.go`; keep matching + `applyBaseline` in `baseline.go`.

Do **not** run `--update-baseline` to paper over these; the policy reserves that for structural shifts.

### B2. #83 is unfinished

Without the `AppConfig` fields, `cmd/cqrs-lint` does not compile. This is the highest-priority remaining coding task.

### B3. Public API surface changed without golden regeneration

New exported symbols requiring `cmd/api-stability` golden updates:

- `metaengine`: `WithExact`, `ErrScanTruncated`, `WithCap`, `EnforceCap`, `EnforceCaps`, `ErrCapRequiresMap`, `ErrCapUnsupported`, `QueryConfig.CapRows`, `QueryConfig.CapOrderBy`.
- No cqrs-lint API surface changed (it is a binary; behavior changed).

Also required by repo procedure: `CHANGELOG.md` `[Unreleased]` rows and `FEATURES.md` inventory rows for #71/#72 (and the cqrs-lint behavior changes), plus doc-check after any skill-reference edits.

### B4. Documentation debt for #71/#72 and #83

- `metaengine/COOKBOOK.md`: keep-newest-N recipe (`WithCap` + `EnforceCap(s)`, cadence, replay caveat) and `WithExact` usage.
- Consumer-facing skill recipe if desired (`references/recipes.md` §2.x) with a `recipes_catalog*.go` classification entry.
- cqrs-lint README/`RULES.md`: baseline file format, flags, exit semantics.
- CHANGELOG: E003 count change is a reporting-visible behavior change for consumers.

---

## 6. Risks and Review Notes

1. **#81 changes reported counts for every consumer.** The CRM goes from 1 E003 to 5. Any consumer CI that gates on counts or snapshots output will move. This is the intended fix, but it needs a release note and belongs in consumer baselines once #83 lands.
2. **#74 deviates from the issue's literal allowlist.** The issue proposed skipping all pure snake_case values independently; that would suppress lowercase API keys and broke an existing positive test. The shipped form is a conjunction (name shape **and** const/structural binding). If you want the literal form, expect a real detection regression; I recommend keeping the conjunction.
3. **#72 is not replay-deterministic by itself.** Trimming is outside the event log. Reset + replay restores all rows. Documented in code; must also be in the recipe. A fold-level cap would be replay-safe but pay a scan per insert.
4. **#72 enforcement is explicit, not automatic.** A consumer that declares `WithCap` but never calls `EnforceCap` gets no trimming. This is a deliberate performance tradeoff; if you prefer "declared means enforced", the enforcement must move into `applyFoldInsert` and the cost model must be revisited.
5. **#82's string-match skip is permissive.** Any `.Edge` selector in a metaengine-importing project suppresses the coaching. Tightening to qualifier resolution is cheap if you prefer precision.
6. **#71's `WithExact` on `ScanPage`** makes a paginated call error when more pages exist (if the caller passes `WithExact`). That is consistent ("exact" means complete), but it is a footgun worth a godoc sentence on `ScanPage`.
7. **`go.work.sum` grew by 116 lines** and was committed by the daemon inside `39754076e`, alongside this session's `scanner_generic.go` edit and the user's draft docs. Not code-affecting, but it is unrelated churn in a "chore" commit and should be reviewed before release.
8. **The auto-commit daemon destroyed authored history.** All work landed as `chore: auto-commit N changed file(s) (heuristic)` commits. For a release wave, the per-issue narrative exists only in this report and the issue threads. No `git commit` was run by me (per policy, the user must ask).

---

## 7. What Remains — Ordered Plan

**Phase 1 — make the tree green (half day-ish):**

1. Split the three over-size files (B1).
2. Finish #83: `AppConfig` flags, `run()` wiring, exit-code semantics, tests, README/RULES docs (B2).
3. `gofmt` + package tests again after the splits.

**Phase 2 — docs and gates:**

4. `metaengine/COOKBOOK.md` recipes for #72/#71; optional skill recipe + catalog entry.
5. Regenerate `cmd/api-stability` golden; run `doc-check`.
6. `CHANGELOG.md` `[Unreleased]` + `FEATURES.md` rows (including the E003 count-change note).
7. Run, in this order: `nix run .#check-file-size`, `.#check-arch`, `.#check-lint-config`, `.#check-error-taxonomy`, `.#check-md-go`, `.#vulncheck`, then `nix run .#verify` (exclusively — it cannot run concurrently with integration suites).

**Phase 3 — consumer proof and issue hygiene:**

8. Reproduce the real-consumer deltas on `LarsArtmann/crm @ 34f26a6` (CLI + BuildFlow toolspec): E005 25→0, E004 21→0, F002 1→0, C010 11→0, F010 1→0, E003 1→5 stable and identical on both surfaces, plus #74 on the webphone site. I have not run these yet (consumer checkout path/access not confirmed in this session).
9. Add a minimal fixture repo test for the baseline CLI semantics (the "To verify" list in #83).
10. Close #70 (verified fixed) with evidence; close #74, #78, #79, #80, #81, #82 after gates pass; keep #83 open until its checklist is complete; comment on #71/#72 with the shipped API and the caveats above.

**Phase 4 — follow-ups (separate issues, not blockers):**

11. Toolspec dedup parity for future duplicate-ID rules.
12. `ScanWithMeta()` non-error truncation API.
13. Opt-in `--fail-on-stale-baseline`.
14. Tidy `cmd/cqrs-lint/testdata/typedfixture` (`go mod tidy`) and re-run `pkg/rules/version`.
15. Decide ownership for the pre-existing `catalog/*` and `cmd/doc-check/recipes_catalog_meta3.go` ratchet violations.

---

## 8. Process Notes / Workspace State

- **Working tree at report time:**
  - Untracked (new, not yet committed): `cmd/cqrs-lint/baseline.go`, `metaengine/cap_test.go`.
  - Everything else is committed by the daemon across `39754076e`, `0ba621775`, `41a19694c`, `52e256914`, `72fae9bae`, `7b8a5aca1`, `ee20c8f77`.
- **Draft docs:** `docs/drafts/2026-10-23-crm-lint-fps/01..06` are now tracked; `05` and `06` were untracked at session start and were absorbed by the daemon with an unrelated code edit.
- **No GitHub writes yet:** no issues closed, no comments posted, no PR created. I deliberately waited for gates before touching issue state.
- **Commands used were `GOWORK=off` per module**, as required by the repo's multi-module rules. `nix run .#verify` has not been run yet and must run exclusively.
- **Environment caveat:** the ambient shell's Go toolchain is sufficient for the packages tested, but `nix`-based gates are the authoritative check and are pending.

---

## Appendix A — Files Changed by Issue

| Issue | Files |
|-------|-------|
| 78 | `cmd/cqrs-lint/pkg/analyzer/scanner_generic.go`, `scanner_calls.go`, `pkg/rules/architecture/new_rules_test.go` |
| 80 | `cmd/cqrs-lint/pkg/rules/correctness/swallow_helpers.go`, `new_rules_test.go` |
| 81 | `cmd/cqrs-lint/pkg/rules/architecture/e003_e007.go`, `new_rules_test.go` |
| 79 | `cmd/cqrs-lint/pkg/analyzer/scanner_calls.go`, `pkg/rules/adoption/f002_f005.go`, `helpers.go`, `scan_in.go`, `f001_f009_test.go`, `pkg/rules/architecture/new_rules_test.go` |
| 82 | `cmd/cqrs-lint/pkg/rules/adoption/f010_f011.go`, `scan_in.go`, `f010_f017_test.go` |
| 74 | `cmd/cqrs-lint/pkg/rules/security/rules.go`, `new_rules_test.go`, `pkg/rules/lintutil/url_placeholder.go` |
| 71 | `metaengine/errors.go`, `scan_options.go`, `typed_reader_scan.go`, `with_exact_test.go` |
| 72 | `metaengine/cap.go` (new), `cap_test.go` (new), `query_config.go`, `planner.go`, `register_query.go` |
| 70 | none (already fixed; verified only) |
| 83 | `cmd/cqrs-lint/baseline.go` (new, incomplete wiring) |

## Appendix B — Test Commands (copy-paste)

```bash
# Metaengine (includes WithExact + cap tests)
cd metaengine && GOWORK=off go test . -count=1

# Projection adapter (issue #70 verification)
cd metaengine/projectionadapter && GOWORK=off go test ./... -count=1

# cqrs-lint rule packages
cd cmd/cqrs-lint
GOWORK=off go test ./pkg/rules/architecture/ ./pkg/rules/adoption/ \
  ./pkg/rules/correctness/ ./pkg/rules/security/ -count=1
GOWORK=off go test ./pkg/analyzer/ ./pkg/toolspec/ -count=1

# Formatting check
gofmt -l <touched files>

# Gates (not yet run / expected failures listed in §4)
bash scripts/check-file-size.sh
nix run .#verify
```

## Appendix C — Open Questions for You

1. **#72 enforcement model:** keep explicit `EnforceCap(s)`, or auto-trim per insert (and accept the scan cost)? I recommend explicit + docs.
2. **#83 stale semantics:** notice-only (current draft) or opt-in fatal flag?
3. **#74 severity:** `warning` for const-bound true positives — or `info`?
4. **#82 precision:** keep the cheap `.Edge` string match, or resolve the metaengine qualifier via type info?
5. **#81 release note:** are you comfortable with consumer counts rising (E003 1 → N) in the next release?
6. **Ratchet:** do you want me to fix the pre-existing `catalog/*` / `recipes_catalog_meta3.go` violations too, or leave them to their owning session?
7. **History:** do you want an authored, per-issue commit series (rewriting/restructuring the daemon's chore commits), or is the current history acceptable for the release?

---

*Report generated at 2026-10-23 09:28 +0200. No further commands were run after the request to stop; all pending work is listed above.*
