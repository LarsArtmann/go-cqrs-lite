# Session 121 — Comprehensive Status Report

**Date:** 2026-05-28 16:26
**Branch:** `master` (3 commits ahead of origin)
**Scope:** go-error-family v0.2.0 maximization — mid-way audit and reflection

---

## Executive Summary

The go-error-family v0.2.0 migration is ~55% complete across production code. Core library modules (core/event, saga, core/command, core/query, memory, watermill, projection, middleware) are converted. **139 `fmt.Errorf` wraps remain** — primarily in signing (36), catalog (29), and example (60) modules.

### Critical Finding: Build is Clean, Tests Pass (with caveats)

- **All 11 main modules build and pass tests** (1217 total tests, zero failures without race)
- **Race detector: clean** on all modules except signing (build failure due to untracked test files with broken imports)
- **Catalog golden tests: stale** — 3 golden files need refresh after recent changes
- **Stream module: zero tests** — new module, no test coverage at all
- **Signing: 7 untracked test files** from a test split that wasn't completed — they have broken imports and duplicate declarations

---

## A. Fully Done ✅

### Modules with 100% errorfamily adoption (no fmt.Errorf in production code)

| Module | Coverage | fmt.Errorf remaining | Lines | Status |
|--------|----------|---------------------|-------|--------|
| saga | 93.4% | 0 | 562 | Complete |
| watermill | 94.4% | 0 | 327 | Complete |
| projection | 96.0% | 0 | 594 | Complete |
| testhelpers | 82.1% | 0 | 1002 | Complete |

### Modules with structured error patterns in place

| Module | Coverage | fmt.Errorf remaining | Lines | Status |
|--------|----------|---------------------|-------|--------|
| core/event | 88.1% | 3 (validation errors in types.go) | ~2000 | 98% done |
| core/command | 92.5% | 0 | ~400 | Complete |
| core/query | 96.8% | 0 | ~400 | Complete |
| core/decider | 91.1% | 1 (dynamic format in load.go) | ~300 | 95% done |
| core/pkg/id | 100% | 2 (id.go dynamic format) | ~200 | 90% done |
| memory | 99.6% | 0 | 909 | Complete |
| middleware | 93.7% | 4 (validation.go + recovery.go) | 1030 | 85% done |

### Infrastructure

- **errorfamily v0.2.0 adopted** as direct dependency in core, catalog; indirect in all others
- **Re-exports added** in `core/event/errors.go`: `Wrapf`, `Newf`, `WithContext`, `ExitCode`, `HandleError`, `HandleErrorDetailed`, `RegisterTemplate`, `RegisterClassification`, `RegisterClassifications` + type aliases
- **Tests for re-exports** in `core/event/errors_taxonomy_test.go`
- **DLQ race fix** in `projection/runner_dlq_test.go` (callCount → atomic.Int32)

---

## B. Partially Done 🔧

### signing/ — 36 fmt.Errorf wraps remaining

The `263563b` commit claimed conversion but the perl script silently failed. **All 8 production files still have original fmt.Errorf wraps.**

| File | Wraps | Difficulty | Notes |
|------|-------|------------|-------|
| multisig_middleware.go | 10 | Medium | Actor type casting needed |
| middleware.go | 7 | Easy | Event type wrapping |
| multisig.go | 6 | Medium | Actor concatenation |
| multisig_extract.go | 5 | Medium | Actor type casting |
| ed25519.go | 3 | Easy | Simple wraps |
| event.go | 2 | Easy | Simple wraps |
| signer.go | 2 | Easy | Simple wraps |
| hmac.go | 1 | Easy | Simple wrap |

**Gotcha:** `Actor` is a custom string type — must cast `string(actor)` for concatenation.

### middleware/ — 4 fmt.Errorf wraps remaining

- `validation.go`: 3 wraps (command/event/query validation errors)
- `recovery.go`: 1 wrap (panic recovery with stack trace — complex)

### core/ — 6 fmt.Errorf wraps remaining

- `core/event/types.go`: 3 validation errors (source empty, version negative, schema version positive)
- `core/pkg/id/id.go`: 2 dynamic format wraps (`%T`, `%q` formatting)
- `core/decider/load.go`: 1 dynamic format wrap (`prefix+msg, args...` pattern)

### catalog/ — 29 fmt.Errorf wraps remaining

| File | Wraps | Classification |
|------|-------|---------------|
| exporter.go | 13 | Infrastructure (I/O operations) |
| writer.go | 5 | Infrastructure (file operations) |
| exporter_resources.go | 2 | Infrastructure (dir creation) |
| exporter_message.go | 3 | Infrastructure (message export) |
| exporter_resources_extra.go | 3 | Infrastructure (dir creation) |
| schema.go | 2 | Corruption (marshal/unmarshal) |
| registry.go | 1 | Rejection (domain not found) |

---

## C. Not Started 📋

### example/ — 60 fmt.Errorf wraps (5 modules)

| Module | Wraps | Lines |
|--------|-------|-------|
| example/todo | 35 | ~1500 |
| example/user | 10 | ~600 |
| example/projection | 8 | ~500 |
| example/storage | 7 | ~400 |
| **Total** | **60** | **~2780** |

### stream/ — New module, zero tests

- 111 lines of production code
- No test files at all
- `AggregateReader`, `ListBuilder`, `StatusPage`, cursor pagination
- No error wrapping (doesn't return errors yet — just types and interfaces)

### cmd/cqrs-gen — 1 fmt.Errorf wrap

- Single `fmt.Errorf("scan %s: %w", path, err)` in main.go

---

## D. Totally Fucked Up 💥

### 1. Signing Test Split — Incomplete & Broken

**What happened:** A previous session split `signing_test.go` (1028 lines) into 4 focused files + created 7 new test files. The split was committed (tracked files: `multisig_test.go`, `signing_test.go`, `assertions_test.go`, `benchmark_test.go`, `example_test.go`) but:
- `signing_test.go` was gutted — now only has helpers, no test functions
- `multisig_test.go` was gutted — now only has helpers, no test functions  
- 7 new untracked test files exist with **broken imports** and **duplicate declarations**
- `ed25519_test.go`, `hmac_test.go`, `middleware_test.go`, `signature_test.go` are tracked but modified (git shows `M signing/signing_test.go` and untracked `ed25519_test.go` etc.)

**The race detector build fails on signing because these files have unused imports and undefined symbols.**

**Fix needed:** Either complete the split (fix imports, track all new files) or revert to the pre-split state.

### 2. Catalog Golden Tests — Stale

3 golden files are out of date:
- `catalog/testdata/golden/asyncapi.yaml`
- `catalog/testdata/golden/eventcatalog-config.js`
- `catalog/testdata/golden/package.json`

**Fix:** `go test ./catalog/... -update` (already verified this works)

### 3. Tombstone ErrNilEvent — Fixed but Disconnected

`core/event/tombstone.go` referenced `ErrNilEvent` which didn't exist. Fixed in commit `b95e80a` but this means the latest commits from previous sessions were already pushed without this fix being in place.

---

## E. What We Should Improve

### 1. Type Model Improvements

**Actor type safety:** `signing.Actor` is `string` but needs explicit casting everywhere. Consider:
```go
type Actor string
func (a Actor) String() string { return string(a) }
```
This already exists but the fmt.Errorf wraps still do `string(actor)` manually.

**Error code naming convention:** Currently inconsistent:
- `event.` prefix: `event.nil_aggregate_id`
- `middleware.` prefix: `middleware.cb_invalid_failure_threshold`
- `catalog.` prefix: `catalog.nil_schema`
- `signing.` prefix: `signing.no_verifier`
- `memory.` prefix: `memory.snapshot_load_at_version_failed`
- `saga.` prefix: `saga.nil_definition`

Should be formalized: `<module>.<package>.<error_description>`

### 2. Library Improvements to Consider

| Library | Purpose | Current State |
|---------|---------|---------------|
| `go-error-family` v0.2.0 | Structured error taxonomy | Already adopted, 55% migrated |
| `err113` linter | Prevent new fmt.Errorf | Could add to golangci-lint |
| `goerr` | Error builder pattern | Not needed — errorfamily covers this |

### 3. Architecture Debt

- **stream module has zero tests** — new code with no verification
- **signing test files are in a broken state** — blocks race detection
- **`replace` directives in every go.mod** — required until v1.0.0 tags pushed
- **Catalog error handling is I/O-heavy** — all wraps are file operation failures, correctly classified as Infrastructure

### 4. Consistency Issues

- **middleware/recovery.go** uses `fmt.Errorf` with stack traces — this is a legitimate case where `Wrapf` or keeping `fmt.Errorf` makes sense since the stack trace is formatted dynamically
- **core/decider/load.go** uses `fmt.Errorf(prefix+msg, args...)` — this is a variadic pattern that `Wrapf` could handle
- **core/pkg/id/id.go** uses `%T` and `%q` formatting — these need `Wrapf` or `Newf`

---

## F. Top 25 Things We Should Get Done Next

### Priority 1: Fix Broken Things (7 tasks)

| # | Task | Impact | Effort | Est. |
|---|------|--------|--------|------|
| 1 | Fix signing test split — complete or revert | Race detection blocked | 30min | 30min |
| 2 | Update catalog golden test files | CI green | 5min | 5min |
| 3 | Commit and push all uncommitted changes | Safety | 5min | 5min |
| 4 | Add stream module tests (basic coverage) | 0% → 80%+ | 60min | 60min |
| 5 | Add `go.work sync` to verify module graph consistency | Build hygiene | 10min | 10min |
| 6 | Add stream module to integration tests | Cross-module validation | 30min | 30min |
| 7 | Verify signing test coverage after fix | Regression check | 15min | 15min |

### Priority 2: Complete errorfamily Migration (9 tasks)

| # | Task | Impact | Effort | Est. |
|---|------|--------|--------|------|
| 8 | Convert signing/ 36 fmt.Errorf → errorfamily | Highest-value remaining | 45min | 45min |
| 9 | Convert catalog/ 29 fmt.Errorf → errorfamily | Catalog is consumer-facing | 30min | 30min |
| 10 | Convert middleware/ 4 fmt.Errorf → errorfamily | Consistency | 15min | 15min |
| 11 | Convert core/event/types.go 3 validation errors | Core module completeness | 10min | 10min |
| 12 | Convert core/pkg/id/id.go 2 dynamic wraps | Core module completeness | 10min | 10min |
| 13 | Convert core/decider/load.go 1 dynamic wrap | Core module completeness | 5min | 5min |
| 14 | Convert cmd/cqrs-gen 1 wrap | Completeness | 5min | 5min |
| 15 | Convert example/ 60 fmt.Errorf → errorfamily | Example quality | 60min | 60min |
| 16 | Remove `fmt` imports from files that no longer need it | Code cleanliness | 15min | 15min |

### Priority 3: Quality & Consistency (5 tasks)

| # | Task | Impact | Effort | Est. |
|---|------|--------|--------|------|
| 17 | Add `err113` linter rule to prevent new fmt.Errorf | Prevention | 10min | 10min |
| 18 | Formalize error code naming convention (`module.package.error`) | Consistency | 15min | 15min |
| 19 | Add error classification tests for signing | Coverage | 20min | 20min |
| 20 | Add error classification tests for catalog | Coverage | 20min | 20min |
| 21 | Run full race + coverage report | Verification | 10min | 10min |

### Priority 4: Future-Proofing (4 tasks)

| # | Task | Impact | Effort | Est. |
|---|------|--------|--------|------|
| 22 | Document error taxonomy in AGENTS.md or docs/ | Onboarding | 20min | 20min |
| 23 | Plan v1.0.0 tag strategy to remove replace directives | Publishability | 30min | 30min |
| 24 | Add `stream` module error wrapping patterns | Architecture | 15min | 15min |
| 25 | Review `example/` as integration test surface | Test strategy | 30min | 30min |

---

## Test Coverage Summary

| Module | Coverage | Tests | Status |
|--------|----------|-------|--------|
| core/command | 92.5% | ✅ | Green |
| core/decider | 91.1% | ✅ | Green |
| core/event | 88.1% | ✅ | Green |
| core/pkg/dispatcher | 100.0% | ✅ | Green |
| core/pkg/id | 100.0% | ✅ | Green |
| core/query | 96.8% | ✅ | Green |
| memory | 99.6% | ✅ | Green |
| signing | 93.9% | ⚠️ | Race build fails (broken test files) |
| saga | 93.4% | ✅ | Green |
| watermill | 94.4% | ✅ | Green |
| projection | 96.0% | ✅ | Green |
| middleware | 93.7% | ✅ | Green |
| storage | 89.9% | ✅ | Green |
| catalog | 96.3% | ⚠️ | Golden tests stale |
| testhelpers | 82.1% | ✅ | Green |
| integration | N/A | ✅ | Green |
| stream | 0.0% | ❌ | No tests |
| example/todo | — | ✅ | Green |
| example/user | — | ✅ | Green |

**Total: 1217 tests passing across 11 main modules**

---

## Error Taxonomy Adoption

| Family | Production Uses |
|--------|----------------|
| Rejection | 49 (validation, nil checks, not-found) |
| Infrastructure | 124 (I/O, network, store operations) |
| Corruption | 41 (data integrity, codec failures) |
| Conflict | 9 (duplicate registration, optimistic locking) |
| Transient | 4 (temporary failures, retries) |

**Total structured errors: 227** (vs 139 remaining `fmt.Errorf`)

---

## G. Top #1 Question I Cannot Figure Out Myself

**Should the `example/` modules be converted to errorfamily at all?**

Arguments for: consistency, demonstrates the library pattern to consumers, example code should model best practices.
Arguments against: example code is not imported by anyone, adds 60 changes with zero consumer impact, example error handling is typically simplified for readability.

**Recommendation:** Convert example/ last, if at all. The 80/20 says focus on signing (36) + catalog (29) + middleware (4) + core (6) = **75 wraps** for maximum consumer impact.

---

## Git Status

```
On branch master, 3 commits ahead of origin/master:
  b95e80a fix(core/event): replace undefined ErrNilEvent in tombstone.go with NewRejection
  7b38e1a refactor(signing): split signing_test.go into focused files
  14b4692 test(signing): add multisig tests for signing module

Untracked:
  signing/ed25519_test.go (modified, untracked)
  signing/hmac_test.go (modified, untracked)  
  signing/middleware_test.go (modified, untracked)
  signing/signature_test.go (modified, untracked)
  signing/multisig_*_test.go (7 new files, untracked)
  cqrs-gen/ (build artifact)

Modified:
  signing/signing_test.go (gutted — helpers only)
  signing/multisig_test.go (gutted — helpers only)
  catalog/testdata/golden/* (3 stale golden files)
```
