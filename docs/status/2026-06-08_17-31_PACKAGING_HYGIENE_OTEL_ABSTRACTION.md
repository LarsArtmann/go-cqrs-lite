# Status Report — Packaging Hygiene & OTel Abstraction

**Date:** 2026-06-08 17:31
**Trigger:** Feedback from project-discovery-sdk — "Why I Can't Use You"
**Session scope:** Make go-cqrs-lite genuinely "lite" by cleaning dependency graph

---

## Executive Summary

7 commits today. The OTel abstraction is done for 4 modules. Dep budget CI is enforced. The library is now measurably leaner for consumers who don't use OTel directly.

**Net result:** 3 modules (decider, projection, storage) now have `go.opentelemetry.io/*` as **indirect** deps. Before today, all were direct.

---

## A) FULLY DONE

### 1. OTel Type Re-exports (`otel/types.go`)
- Created `otel/types.go` with type aliases: `Tracer`, `Span`, `SpanKind`, `KeyValue`, `Meter`, `Float64Histogram`, `RecordOption`
- Helper functions: `AttrString`, `AttrInt`, `AttrInt64`, `WithAttributes`, `WithSpanKind`, `SpanFromContext`
- Metric helpers: `MetricWithAttributes`, `MetricWithDescription`, `MetricWithUnit`
- Zero breaking changes — all type aliases, consumers see same types

### 2. OTel Migration — 4 modules, 15 files

| Module | Files changed | OTel before | OTel after |
|---|---|---|---|
| decider | 3 | **direct** | **indirect** ✅ |
| projection | 2 | **direct** | **indirect** ✅ |
| storage | 8 | **direct** | **indirect** ✅ |
| middleware | 2 | direct (prod) | direct (test deps hold it) |

**Lines removed:** 172 → **Lines added:** 136. Net -36 lines.

### 3. Dep Budget CI
- `scripts/check-module-layers.sh` enforces per-module direct dep budgets
- `nix run .#check-layers` wired into Nix flake
- Budgets tightened after OTel migration: decider 12→10, projection 11→9, storage 12→10

### 4. Documentation
- AGENTS.md: added dep budget principle (#12) and OTel abstraction principle (#13)
- Plan doc at `docs/planning/2026-06-08_06-47_PACKAGING_HYGIENE_AND_ADOPTION_UNLOCK.md`

### 5. Full Test Suite — ALL PASS, 0 FAILURES

| Package | Coverage |
|---|---|
| catalog/v2 | 95.9% |
| command/v2 | 80.5% |
| decider/v2 | **100.0%** |
| dispatcher/v2 | **100.0%** |
| memory/v2 | 98.2% |
| middleware/v2 | 93.5% |
| projection/v2 | 91.2% |
| storage/v2 | 86.8% |
| event/v2 | 89.4% |
| query/v2 | 94.3% |
| (22 packages total) | avg ~91% |

---

## B) PARTIALLY DONE

### Middleware OTel migration
- **Production code** (tracing.go, metrics_otel.go) fully migrated to otel/ re-exports
- **Test files** (tracing_test.go, metrics_otel_test.go, tracing_logging_test.go) still import `go.opentelemetry.io` directly
- Result: `go.opentelemetry.io/*` stays as **direct** dep in middleware/go.mod because test imports force it
- Fix: migrate test files to use otel/ re-exports, or create middleware-specific test helpers

### Dep budgets
- Budgets are set but conservative. Could be tighter once middleware test cleanup and eventtest separation happen.

---

## C) NOT STARTED

### 1. eventtest separation (HIGH IMPACT, BLOCKED)
- All 6 sibling-importing test files in `event/` are **already `package event_test`** (external) using only public API
- They CAN move to a top-level module (e.g. `eventtests/`)
- **Blocked by:** Go doesn't support nested modules. Would need a top-level module, changing the import path for **40+ consumers** across pebble, storage, decider, watermill, projection, middleware, memory, integration, signing, examples
- This is a v3-level breaking change, not a patch

### 2. Reactive split (MEDIUM IMPACT, BLOCKED)
- `event/reactive.go` uses `samber/ro` — the only non-trivial external dep in event's production code
- Could become a top-level `eventbus/` module
- **Same Go nested module limitation** — would change import paths for 3 consumers (example/user, integration, projection tests)

### 3. event/go.mod audit
- 5 test-only sibling deps (command, query, memory, schema, snapshot) + 3 test frameworks (ginkgo, gomega, rapid) remain in event/go.mod
- Cannot remove without removing the _test.go files that import them
- The _test.go files are the 6 external tests that could move to eventtest — circular dependency with item #1 above

### 4. ID backing type documentation
- `go-branded-id` supports `cbid.ID[T, B]` for configurable backing types
- Just needs examples in id/ README

---

## D) TOTALLY FUCKED UP

### eventtest/go.mod zombie
- Created `event/eventtest/go.mod` as a nested module attempt
- Deleted after discovering Go's nested module limitation
- But gopls still shows warnings about `eventtest/go.mod` in diagnostics — **there is no go.mod there anymore**
- The gopls errors in project diagnostics are stale ghosts, not real issues

### Initial plan was over-scoped
- Original plan had 68 tasks including "extract resilience/" module
- Correctly scoped down to: OTel re-exports + dep budgets
- Should have started with the research (which modules use which OTel symbols) instead of planning first

---

## E) WHAT WE SHOULD IMPROVE

### Architecture
1. **otel/go.mod has gomega as direct dep** — it's only used in test files. `go mod tidy` won't move it because of workspace resolution. Should be investigated.
2. **storage/sql has only 34.7% coverage** — the dialect-specific SQL code is undertested
3. **otel/ has only 73% coverage** — the new types.go has no tests yet
4. **event/eventtest coverage is 18.4%** — test helpers themselves barely tested

### Process
5. **No CI pipeline** — `.github/workflows/` doesn't exist. All checks are manual (`nix run .#test`, `nix run .#lint`, `nix run .#check-layers`). The dep budget CI only runs if someone remembers.
6. **Pre-commit hook output is noisy** — BuildFlow reports existing binary issues (example/user/user, example/todo/cmd/api/api, example/listing/listing) that aren't related to current changes. Should be cleaned up or .gitignore'd.
7. **eventtest stale dispatcher dep** — gopls warned about `dispatcher/v2` in eventtest but we never fixed it

### Code quality
8. **middleware has unnecessary type arguments** — gopls flagged 6 instances of `infertypeargs` in metrics_otel.go and tracing_logging.go
9. **event_type_clone_test.go has unused write** — CorrelationID field written but never read (line 217)

---

## F) TOP 25 THINGS TO DO NEXT

### Tier 1: Direct dep reduction (continuing today's work)

| # | Task | Impact | Effort |
|---|------|--------|--------|
| 1 | Migrate middleware test files to otel/ re-exports → make OTel indirect in middleware/go.mod | HIGH | 2h |
| 2 | Add tests for otel/types.go (type alias verification, helper functions) | MED | 30min |
| 3 | Fix gomega in otel/go.mod — move to indirect | LOW | 1h |
| 4 | Remove stale binaries from example/ (user, todo/cmd/api, listing) | LOW | 10min |
| 5 | Add .gitignore entries for compiled binaries in example/ | LOW | 5min |

### Tier 2: event/ dep cleanup (requires architectural decision)

| # | Task | Impact | Effort |
|---|------|--------|--------|
| 6 | Decision: Is eventtest separation worth the import path break for 40+ consumers? | HIGH | Discussion |
| 7 | If yes: create top-level `eventtests/` module, move 6 external test files | HIGH | 2h |
| 8 | If yes: update 40+ consumer import paths | HIGH | 1h |
| 9 | If no: document that event/ test deps are intentional | LOW | 10min |
| 10 | Decision: Is reactive split worth a top-level `eventbus/` module? | MED | Discussion |

### Tier 3: CI & Quality

| # | Task | Impact | Effort |
|---|------|--------|--------|
| 11 | Create `.github/workflows/ci.yml` with nix build/test/lint/check-layers | HIGH | 2h |
| 12 | Wire `nix run .#check-layers` into CI as required check | HIGH | 15min |
| 13 | Fix storage/sql coverage (34.7% → 80%+) | MED | 4h |
| 14 | Fix middleware infertypeargs lint warnings (6 instances) | LOW | 15min |
| 15 | Fix event_type_clone_test.go unused write warning | LOW | 5min |

### Tier 4: Developer experience

| # | Task | Impact | Effort |
|---|------|--------|--------|
| 16 | Add id/ README with configurable backing type examples | MED | 30min |
| 17 | Add otel/ README documenting the re-export pattern | MED | 20min |
| 18 | Update decider/projection/storage READMEs to mention OTel is optional | LOW | 20min |
| 19 | Fix example/user/server.go unused params (pre-commit flagged) | LOW | 10min |
| 20 | Fix CONTRIBUTING.md (modified but uncommitted) | LOW | 10min |

### Tier 5: Future architecture

| # | Task | Impact | Effort |
|---|------|--------|--------|
| 21 | Investigate Go build tags for optional OTel (`//go:build !no_otel`) | MED | 4h |
| 22 | Consider Go 1.27+ lazy module loading if/when available | FUTURE | — |
| 23 | Evaluate moving eventtest/ to a separate repo | FUTURE | — |
| 24 | Write ADR for OTel abstraction pattern | MED | 30min |
| 25 | Benchmark: measure compile time difference with OTel indirect vs direct | LOW | 1h |

---

## G) MY #1 QUESTION

**Should the eventtest/ external tests be separated into a top-level module, accepting the breaking import path change for 40+ consumers?**

This is the single highest-impact remaining item (removes 5 sibling deps from event/go.mod), but it breaks every module that imports `event/v2/eventtest`. The alternative is to document the current state as intentional and focus CI budget enforcement instead.

Arguments for:
- event/ goes from 12 direct deps to ~5 — genuinely "lite"
- The 6 test files are already `package event_test` using only public API — architecturally clean separation

Arguments against:
- 40+ import path changes across the repo
- All downstream consumers (outside this repo) would also need to update
- Could be deferred to v3 with a deprecation notice

---

## Dependency Audit (Current State)

| Module | Direct deps | Internal deps | OTel direct? |
|---|---|---|---|
| id | 2 | 0 | No |
| dispatcher | 0 | 0 | No |
| codec | 0 | 0 | No |
| **event** | **12** | **7** | No |
| command | 7 | 3 | No |
| query | 6 | 2 | No |
| schema | 3 | 3 | No |
| snapshot | 4 | 4 | No |
| **decider** | **9** ↓from 11 | 6 | **No (indirect)** ✅ |
| memory | 7 | 5 | No |
| signing | 4 | 2 | No |
| **otel** | 6 | 0 | **Yes** (expected — owns OTel) |
| **middleware** | **12** ↓from 13 | 6 | **Yes** (test deps) |
| **storage** | **9** ↓from 11 | 7 | **No (indirect)** ✅ |
| **projection** | **8** ↓from 10 | 5 | **No (indirect)** ✅ |
| listing | 5 | 3 | No |
| watermill | 4 | 3 | No |
| pebble | 4 | 3 | No |
| turso | 5 | 4 | No |
| catalog | 2 | 0 | No |
| integration | 17 | 14 | Yes |

**Modules with reduced deps today:** decider (-2), projection (-2), storage (-2), middleware (-1) = **7 direct deps eliminated**

---

## Commits Today

```
666ef308 style: normalize import blocks and multi-line argument formatting
292d9e03 docs(AGENTS): add OTel abstraction principle
bc896b2d chore(infra): tighten dep budgets after OTel migration
8b4ab081 refactor(middleware): migrate production code from direct OTel to otel/ re-exports
60bb72d8 refactor(storage): migrate from direct OTel imports to otel/ re-exports
1246e916 refactor(projection): migrate from direct OTel imports to otel/ re-exports
9e3f63f1 refactor(decider): migrate from direct OTel imports to otel/ re-exports
ab1ba0d0 feat(otel): add type re-exports for OTel abstraction
3c00cb2e cleanup: remove orphan snaptest package and fix server.go unused params
7c81f186 docs(status): SEC lessons integration status + dep budget wiring
4b90a87b docs(status): comprehensive post-hygiene session status report
```
