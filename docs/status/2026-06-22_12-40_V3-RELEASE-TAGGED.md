# Comprehensive Status — 2026-06-22 12:40 CEST

**Base:** `4099aeb6` (release: migrate all 38 modules from /v2 to /v4)

---

## Executive Summary

**v3.0.0 is tagged.** All 33 versioned modules carry `v3.0.0` annotated tags. The full `/v2`→`/v4` migration (706 files) is committed and passes build + vet + race tests across all 38 modules. The previous "v2-with-v3-teed-up" limbo is over — consumers can now pin a major version.

This session's work: FEATURES.md freshness audit, README positioning rewrite, CHANGELOG v3.0.0 section, stack/sqlite lint cleanup (11→0), and the mechanical `/v2`→`/v4` migration across `.go` imports, `go.mod` paths/versions/replaces, and documentation.

---

## a) Fully Done

### v3 Release (THIS SESSION)

| Item                  | Detail                                                                                                                                     | Commit     |
| --------------------- | ------------------------------------------------------------------------------------------------------------------------------------------ | ---------- |
| `/v2`→`/v4` migration | 706 files: all `.go` imports, 37 `go.mod` module paths + version pins + replace directives, `go.sum` entries, 49 `.md` documentation paths | `4099aeb6` |
| v3.0.0 tags           | 33 annotated tags created (all versioned modules)                                                                                          | git tag    |
| ComponentTracer fix   | Hardcoded `/v2` in `otel/spans.go:49` format string → `/v4` (caught by test, not by sed)                                                   | `4099aeb6` |
| API surface regen     | `docs/api_surface.txt` regenerated (1605 exports, unchanged from v2)                                                                       | `4099aeb6` |

### Documentation Polish (THIS SESSION)

| Item                  | Detail                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                   |
| --------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| FEATURES.md freshness | Deleted ghost `projection/` section (25 rows), `readmodel/cache` rows, reactive bus entries (`NewEventBus`, `NewCommandBus`, `DistinctByEventID`, `HandlerToObserver`, `ro.Observer`); fixed `memory/`→`storage/memory/` path; fixed `io.Closer` claims on 3 interfaces; fixed "all types are interfaces" lie; marked streaming reads as done; added `transport/http`, `prometheus`, `deployer-first` to Module Maturity Matrix; updated lint count from "0 across 29 modules" to honest "~60 across 38" |
| README positioning    | New "Why this library?" section + comparison table expanded to 10 capabilities vs go-cqrs/Watermill/cqrs-go; migration guide link in Status section; badge URL fixed `/v2`→`/v4`                                                                                                                                                                                                                                                                                                                         |
| CHANGELOG v3.0.0      | New `[3.0.0]` section with 11-row breaking-change table + migration guide link; cleaned stale Unreleased entries (DistributedRunner, cqrs-gen event mode — both deleted with projection/)                                                                                                                                                                                                                                                                                                                |
| V3_MIGRATION.md       | AFTER sections updated to `/v4`; BEFORE sections preserved for migration context                                                                                                                                                                                                                                                                                                                                                                                                                         |

### Stack Polish (THIS SESSION)

| Item                          | Detail                                                                                                                                                                                                                                                                                                                             |
| ----------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `stack/sqlite/preset.go` lint | 11→0 issues: extracted `buildViewOptions`/`buildPrimaryViewOptions`/`buildSecondaryViewOptions` into `views.go` (resolved nestif complexity); renamed `db`→`sqlDB` (varnamelen); converted inline error checks to plain assignments (noinlineerr); fixed exhaustruct on `defaultConfig`; split file to stay under 350-line CI gate |
| `multi_db_test.go` errcheck   | 2 `defer bundle.Close()` → `defer func() { _ = bundle.Close() }()`                                                                                                                                                                                                                                                                 |
| Coverage                      | `stack/sqlite` 45.9%→71.5% (constructor paths now exercised)                                                                                                                                                                                                                                                                       |

### Pre-existing (prior sessions, verified green)

- All 11 v3 breaking changes complete and tested
- 38 modules, 40,211 lines of production Go
- Core coverage: event 91.6%, command 90.5%, query 86.1%, decider 98.3%
- 1,605 exports (API surface stable)
- Zero panics in production code
- 5-family error taxonomy
- Full `-race` CI on both workspace and per-module matrix
- 33 ADRs

---

## b) Partially Done

### stack layer coverage (still light after this session's work)

| Module           | Coverage | Issue                                                                                   |
| ---------------- | -------- | --------------------------------------------------------------------------------------- |
| `stack`          | 45.9%    | Constructor error branches thin — presets emphasise happy paths + shared contract suite |
| `stack/postgres` | 21.2%    | Skips locally without `POSTGRES_TEST_DSN`; CI runs them in `postgres-integration` job   |
| `storage/sql`    | 61.0%    | Shared SQL infra, edge paths uncovered                                                  |

### ~60 lint issues across 12 modules

All style, not correctness. CI reports but doesn't gate (per ADR). Breakdown: wrapcheck (12), errcheck (9), varnamelen (8), exhaustruct (6), noinlineerr (5), gochecknoglobals (4), staticcheck (3), ineffassign (3), err113 (3), nestif (2), gosec G115 (2). Worst offender was `stack/sqlite` (11 issues — **now fixed** this session). Remaining are spread across transport/http, codec, catalog, watermill.

### Stale go.sum entries

~25 `go.sum` files still contain `v2` checksum lines for sibling modules. These are harmless (workspace replace directives resolve siblings locally) but cosmetic noise. They will auto-clean once tags are pushed to remote and per-module `go mod tidy` runs.

### Transport adapters: 1 of 4 shipped

`transport/http/` (SSE) done. `transport/grpc/`, `transport/nats/`, `transport/redis/` are accepted ADRs with zero code. Correctly waiting for consumer signal.

---

## c) Not Started

| Item                                       | Status                                                | Blocked on                                                    |
| ------------------------------------------ | ----------------------------------------------------- | ------------------------------------------------------------- |
| Push v3.0.0 tags to remote                 | Not pushed                                            | Manual `git push --tags` (user decision on timing)            |
| API surface reduction                      | Brainstorm exists (`docs/brainstorming/`), unactioned | Deferred to v4 (per user instruction this session)            |
| Hot-State cache (decider)                  | TODO_LIST entry, no code                              | Needs a real hot-aggregate workload to justify                |
| Read-pressure snapshot strategy            | TODO_LIST entry, no code                              | Depends on Hot-State cache landing first                      |
| Secondary indexes on `kv.Store`            | ROADMAP entry                                         | `Materialize.List` does full table scan; works at small scale |
| gRPC/NATS/Redis transport adapters         | ADR-0025 accepted                                     | Consumer signal                                               |
| Documentation site (Docusaurus/MkDocs)     | ROADMAP raw idea                                      | Consumer signal                                               |
| PostgreSQL testcontainers                  | ROADMAP entry                                         | Docker CI setup                                               |
| Per-module `go mod tidy` with `GOWORK=off` | Stale go.sum entries remain                           | Tags must be pushed first, then tidy resolves                 |

---

## d) Totally Fucked Up

### Status report pollution — 373 files and counting

`docs/status/` contains **373 files**. 363 of them are from before this session — most reference deleted modules (`projection/`, `readmodel/`), old import paths (`/v2`), and breaking changes that are now done. The `archive/` subdirectory exists but has not been used since the last report mentioned it. The doc-to-code ratio is **5.5×** (222,724 doc lines vs 40,211 code lines). This report is #374.

### go.sum stale entries are technically debt

25 modules have `go.sum` files with `/v2` checksum lines for sibling deps. While functionally harmless (replace directives resolve locally), they're visual noise that a careful consumer running `go mod verify` would notice. The fix is: push tags → `cd each-module && GOWORK=off go mod tidy`. Not done yet because tags aren't pushed.

### The `doc-files-age-check` BuildFlow hook is lying

The pre-commit hook has a `doc-files-age-check` that passed on this commit, yet 373 status reports (many months old) exist. The check is either too narrow or not checking `docs/status/`. This means the doc bloat problem has no automated guardrail.

---

## e) What We Should Improve

### 1. Push the tags and clean go.sum entries

The v3.0.0 release is not real until tags are on the remote. `git push --tags` + per-module `go mod tidy` closes the loop. Until then, `go get .../event/v4` fails for external consumers.

### 2. Archive old status reports — enforce it

Move all `docs/status/*.md` older than 7 days into `docs/status/archive/`. This is a 10-second script that reduces 373 files to ~5 active. The `doc-files-age-check` hook needs to actually enforce this or be removed as theater.

### 3. Fix remaining ~49 lint issues

`stack/sqlite` went from 11→0 this session. The same mechanical work applies to the remaining offenders: `transport/http` (wrapcheck), `codec` (gochecknoglobals), `watermill` (gosec G115). Each module is a 10-minute focused pass.

### 4. Add stack/ constructor-error tests

`stack` at 45.9% and `stack/postgres` at 21.2% are the consumer entry points. The presets emphasise happy paths. Constructor-error branches (bad DSN, nil DB, schema migration failure) are the paths that fail in production and are untested.

### 5. Document the deployer-first pattern prominently

The deployer-first stack (`example/deployer-first/`) is the canonical production pattern but it's buried. It should be the first thing a new consumer sees — in the README, not just in an example subdirectory.

---

## f) Top 25 Things To Do Next

| #   | Task                                                                                           | Impact   | Effort  | Category     |
| --- | ---------------------------------------------------------------------------------------------- | -------- | ------- | ------------ |
| 1   | **Push v3.0.0 tags to remote** (`git push --tags`)                                             | Critical | Trivial | Release      |
| 2   | **Per-module `go mod tidy`** to clean stale go.sum v2 entries                                  | High     | Small   | Hygiene      |
| 3   | **Archive `docs/status/*.md`** older than 7 days → `archive/`                                  | High     | Trivial | Hygiene      |
| 4   | **Fix `doc-files-age-check` hook** to actually check `docs/status/`                            | High     | Small   | Tooling      |
| 5   | **Fix remaining ~49 lint issues** (transport/http, codec, watermill)                           | Medium   | Medium  | Quality      |
| 6   | **Add constructor-error tests to `stack/`** (45.9%→>70%)                                       | High     | Medium  | Testing      |
| 7   | **Add constructor-error tests to `stack/postgres`** (21.2%→>50%)                               | High     | Medium  | Testing      |
| 8   | **Document deployer-first pattern in README**                                                  | Medium   | Small   | Docs         |
| 9   | **Write "first 15 minutes" quickstart** that runs without setup                                | High     | Small   | Adoption     |
| 10  | **Clean stale `samber/ro` indirect deps** from go.mod files (9 modules)                        | Low      | Trivial | Hygiene      |
| 11  | **Add `cmd/api-stability` tests** — tool guards breaking changes but is untested (0% coverage) | Medium   | Small   | Testing      |
| 12  | **Add turso `OpenSync` error-path tests** (39%→50%)                                            | Medium   | Small   | Testing      |
| 13  | **Add `storage/sql` `LoadWithSpan` error branch tests** (61%→70%)                              | Medium   | Small   | Testing      |
| 14  | **Golden tests for `signing`** (HMAC/Ed25519 signature format stability)                       | Medium   | Small   | Testing      |
| 15  | **Golden tests for `storage`** (DDL schemas, metadata roundtrip)                               | Medium   | Small   | Testing      |
| 16  | **Document `POSTGRES_TEST_DSN`** prominently in CONTRIBUTING                                   | Low      | Trivial | Docs         |
| 17  | **Write a short blog post / Reddit post** announcing v3.0.0                                    | High     | Small   | Adoption     |
| 18  | **Link benchmarks from README** with a one-line perf claim                                     | Medium   | Trivial | Adoption     |
| 19  | **Decide and ADR: is multi-master permanently out of scope?**                                  | Medium   | Small   | Architecture |
| 20  | **Hot-State cache prototype** (only if a real hot-aggregate workload emerges)                  | Low      | Medium  | Feature      |
| 21  | **Read-pressure snapshot strategy** (only after Hot-State cache)                               | Low      | Medium  | Feature      |
| 22  | **Secondary indexes on `kv.Store`** (only if a workload needs them)                            | Low      | Large   | Feature      |
| 23  | **gRPC transport adapter** (ADR-0025, only if consumer asks)                                   | Low      | Large   | Feature      |
| 24  | **NATS/Redis transport adapters** (ADR-0025, only if consumer asks)                            | Low      | Large   | Feature      |
| 25  | **Property-based integration testing** with state machine verification                         | Low      | Medium  | Testing      |

---

## g) Top #1 Question I Cannot Answer

### Should the v2 tags and `/v2` module paths be kept alive for backward compatibility, or is v2 dead on arrival of v3?

The v3 migration deleted `/v2` paths from all 706 files. The `v2.6.0` tags still exist in git, but the code at those tags uses APIs that no longer exist in the current codebase (e.g., `projection/`, `readmodel/`, `Decider.Fold`, `event.Event` as interface). There is no v2→v3 compatibility shim layer.

**What I can't determine:**

- Whether any existing consumer is pinned to `/v2` and would break if v2 is abandoned
- Whether a v2.7.0 backport release (with bug fixes but not the breaking changes) is needed
- Whether the v2.6.0 tags should be documented as the "final v2" or just left to rot

**Why it matters:** If consumers exist on `/v2`, the migration guide is their only lifeline. If no consumers exist on `/v2`, the old tags are dead weight. The answer determines whether we need to maintain a v2 backport branch or can treat v3 as the single canonical version.

Every release-process decision (branch strategy, backport policy, `go mod` proxy behavior) hinges on this. I cannot answer it from the codebase alone — it requires knowledge of who is importing this library and at what version.

---

_Report generated: 2026-06-22 12:40 CEST · Base commit: `4099aeb6` · 38 modules · 1,605 exports · 33 v3.0.0 tags_
