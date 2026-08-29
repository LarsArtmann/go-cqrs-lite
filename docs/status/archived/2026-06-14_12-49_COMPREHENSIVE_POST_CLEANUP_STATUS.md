# Comprehensive Status Report — 2026-06-14

> **Date:** 2026-06-14 12:49
> **Version:** v2.3.0 (post-release cleanup)
> **Branch:** master (synced with origin)
> **Working tree:** Clean
> **Head commit:** `da8c406b` — chore(deps): bump example/snapshot/testutil internal module versions

---

## Executive Summary

The go-cqrs-lite library is in a **healthy, production-grade state** following the v2.3.0 release. This session completed a comprehensive post-release cleanup across all 28 modules: resolving every lint issue, fixing correctness bugs, writing design docs (ADRs 0016–0018), extending the encryption module's interface coverage, adding turso indexing tests, and documenting consumer patterns.

**All 41 packages pass tests. All 23 linted modules have 0 issues. Coverage averages ~89% across library modules.**

One infrastructure concern (buildflow pre-commit hook corrupting golden files) was identified and documented but requires external action.

---

## a) FULLY DONE ✅

### Build & Test

| Check                                | Status |
| ------------------------------------ | ------ |
| All 41 packages pass tests           | ✅     |
| All 23 modules lint clean (0 issues) | ✅     |
| `nix run .#build`                    | ✅     |
| `nix run .#lint`                     | ✅     |
| `nix fmt`                            | ✅     |
| Git working tree clean               | ✅     |
| Synced with origin/master            | ✅     |

### Coverage by Module

| Module               | Coverage                                 |
| -------------------- | ---------------------------------------- |
| decider              | 100.0%                                   |
| memory               | 98.2%                                    |
| id                   | 97.5%                                    |
| otel                 | 97.3%                                    |
| dispatcher           | 98.0%                                    |
| command              | 97.1%                                    |
| listing              | 94.9%                                    |
| watermill            | 94.3%                                    |
| catalog/d2           | 94.3%                                    |
| signing/multisig     | 94.2%                                    |
| middleware           | 94.2%                                    |
| signing              | 94.0%                                    |
| catalog/asyncapi     | 93.9%                                    |
| catalog/eventcatalog | 92.8%                                    |
| event                | 92.4%                                    |
| schema               | 91.4%                                    |
| projection           | 91.4%                                    |
| catalog/docserver    | 90.1%                                    |
| cmd/cqrs-gen         | 89.9%                                    |
| snapshot             | 88.9%                                    |
| codec                | 88.9%                                    |
| storage              | 88.9%                                    |
| storage/sql          | 88.7%                                    |
| catalog              | 86.4%                                    |
| encryption           | 86.5%                                    |
| pebble               | 87.0%                                    |
| query                | 83.1%                                    |
| turso/indexing       | 77.3%                                    |
| turso                | 49.1% (sync.go needs real LibSQL server) |

### Session Accomplishments (10 commits, 75 files, +1655/-285 lines)

**Phase 1 — Quick Wins:**

- Listing godoc examples (NewListBuilder, StatusMiddleware, CacheInvalidationMiddleware)
- README performance section with benchmark numbers
- CBOR signing documentation in codec/README.md

**Phase 2 — Correctness:**

- CBOR canonical fidelity fuzz test
- MemoryStore concurrent writers benchmark
- Versioned ciphertext format for encryption (WrapCiphertext/UnwrapCiphertext)

**Phase 3a — Design Docs:**

- ADR-0016: Outbox Pattern
- ADR-0017: Schema Registry
- ADR-0018: Distributed Checkpointing

**Phase 3b — Lint Cleanup:**

- Replaced all dynamic `fmt.Errorf` errors with sentinel errors (err113 compliance)
- Fixed `rows.Err()` checks in storage/sql/query_engine.go and storage/stream.go
- Removed dead `execAndTrace` function from turso/indexing/telemetry.go
- Fixed varnamelen, ineffassign, nilerr, unconvert, gocritic issues
- Added path-based lint exclusions for style linters in .golangci.yml
- Verified all 41 `nolint:errcheck` suppressions are justified

**Phase 3c — CBOR Config:**

- Evaluated CoreDet vs Canonical — kept Canonical (documented in codec/cbor.go)

**Phase 3d — Features & Tests:**

- Extended `encryptedStore` with Journal, SeekableJournal, BackwardsSource (5 interfaces total)
- Added turso/indexing pure function tests (`parseStat1Rows`, `inferIndex`) — coverage 72.4% → 77.3%
- Documented field-level encryption as consumer pattern in encryption/doc.go
- Turso indexing guidance document (docs/turso-indexing-guidance.md)
- Hooks API verified as already existing (WithIndexingHooks)
- Schema evolution/migration integration verified as existing
- Health check integration documented

### TODO_LIST.md Status

- **192+** items DONE/verified
- **~60** items open (most are `[FUTURE]` or `[v3]` breaking changes)
- **~45** items planned/speculative

---

## b) PARTIALLY DONE 🟡

### Turso Coverage (49.1%)

- `turso/sync.go` (OpenSync, Push, Pull, Checkpoint, Stats) only tested for rejection paths
- Requires a real LibSQL/Turso server — **cannot be unit tested**
- Indexing subpackage at 77.3% (up from 72.4%)
- Root cause: external service dependency, not code quality issue

### Module Version Alignment

- Most modules at v2.3.0, but `decider` still at v2.2.0 in some example go.mod direct deps
- `pebble` and `projection` at v2.2.0 in example direct deps
- These are example projects, not library modules — lower priority

### Golden File Corruption (Recurring)

- `codec/testdata/golden/json_encode.json` and `middleware/testdata/golden/health-check-response.json` get reformatted by buildflow pre-commit hook on every commit
- Requires manual `git restore` after each commit
- **Root cause:** buildflow's formatting step doesn't exclude `testdata/golden/**`

---

## c) NOT STARTED ⬜

### From TODO_LIST.md — Top Open Items

1. **Playwright E2E tests for example/user/** — Browser-based end-to-end tests
2. **Arena allocation experiment** — Go arena API for zero-copy event processing
3. **Zero-allocation event encoding** — Pool-based buffer reuse
4. **SIMD serialization** — Hardware-accelerated JSON/CBOR
5. **jsonv2 codec** — Wait for Go's new encoding/json/v2 package
6. **cqrs-gen v2** — Improved code generator with plugin system
7. **WASM compilation target** — Verify modules compile to WebAssembly
8. **gRPC adapter** — Protocol buffer transport
9. **NATS adapter** — Message queue integration
10. **Redis adapter** — Cache/message broker
11. **pprof endpoints** — Built-in profiling middleware
12. **Prometheus metrics exporter** — /metrics endpoint
13. **Structured logging middleware** — slog integration
14. **Distributed tracing examples** — OTel collector setup guide
15. **Log compaction** — Event store compaction strategy
16. **Multi-tenant store** — Namespace isolation
17. **S3/GCS archival** — Cold storage for old events
18. **Dashboard** — Visual event store browser
19. **Migration generator** — Auto-generate SQL migrations from schema
20. **Property testing expansion** — More rapid-based property tests
21. **Chaos engineering** — Fault injection framework
22. **Performance dashboard** — Continuous benchmark visualization
23. **Docker CI** — Already has Dockerfile, needs CI pipeline
24. **govulncheck in CI** — Vulnerability scanning
25. **PostgreSQL adapter** — Native PG driver support

---

## d) TOTALLY FUCKED UP 💥

### Buildflow Pre-Commit Hook — Golden File Corruption

**Severity: HIGH (recurring data corruption)**

The `buildflow` pre-commit hook (`~/.git/hooks/pre-commit`) silently corrupts JSON golden files on **every commit**:

- `codec/testdata/golden/json_encode.json`: Compacts `{ "email": ... }` → `{"email":...}` (actually it's the reverse — it ADDS spaces, turning compact into spaced)
- `middleware/testdata/golden/health-check-response.json`: Collapses multi-element arrays to single-line

**Impact:**

- Every commit requires manual `git restore` of golden files afterward
- If missed, tests fail on next run
- Has already caused 2+ corrupted commits (c8e5f32a, 084e8b1c) that needed fix-up commits
- This is the #1 developer experience problem in the repo right now

**Root Cause:** buildflow's formatting step processes JSON files in `testdata/golden/` and applies a JSON formatter that doesn't match what Go's `encoding/json` produces. The `flake.nix` treefmt config already excludes `**/testdata/golden/**`, but buildflow has its own independent formatter that doesn't respect treefmt excludes.

**Workaround:** Run `git restore codec/testdata/golden/ middleware/testdata/golden/` after every commit. This is fragile and error-prone.

**Fix needed:** Configure buildflow to exclude `testdata/golden/**` from its formatting steps, or uninstall the pre-commit hook entirely and rely on `nix fmt` + `nix run .#lint` instead.

### Go Build Cache Fragility

**Severity: MEDIUM**

The Go build cache at `/home/lars/.cache/go-build/` has repeatedly gotten corrupted during this session:

1. First corruption: Cache entries for stdlib packages disappeared, causing "could not import fmt" errors
2. After `rm -rf`, some directories couldn't be removed ("directory not empty") — required multiple attempts
3. `go clean -cache` also fails with "unlinkat directory not empty"

**Impact:** Lint silently returns "0 issues" when the cache is corrupted (typecheck fails silently). This masked real lint issues across encryption (56), storage (50), and turso/indexing (105) for multiple sessions.

**Root Cause:** Likely disk space pressure (90% full, 55GB free) causing incomplete cache writes. The `/tmp` directory has 47GB of data.

### Disk Space Pressure

**Severity: MEDIUM**

- Root filesystem at **89% full** (60GB free of 512GB)
- `/tmp` consuming 47GB
- Go build cache consumed 20GB before cleanup
- Tests fail with "No space left on device" during link phase when cache is too full

---

## e) WHAT WE SHOULD IMPROVE 🔧

### Architecture & Code Quality

1. **Break up turso/sync.go** — It's the only function set we can't test. Extract sync logic into an interface that can be mocked, even if the real implementation needs a server.
2. **Consolidate lint exclusions** — `.golangci.yml` now has 15+ per-path exclusions. Consider whether some linters (nlreturn, wsl_v5, noinlineerr) should be globally disabled instead.
3. **Golden file regeneration workflow** — Add a `nix run .#update-golden` command that regenerates all golden files deterministically, then verify with `git diff`.
4. **Version bump consistency** — Some example projects still reference v2.2.0 modules. A `nix run .#align-versions` command would help.
5. **Coverage gate for turso** — Currently excluded from the 80% gate. Consider whether the sync.go exclusion is documented well enough.

### Developer Experience

6. **Fix or remove buildflow pre-commit hook** — It's causing more harm than good. The golden file corruption is a serious reliability issue.
7. **Add `nix run .#check`** — A single command that runs build + vet + test + lint + format-check, for quick pre-push validation.
8. **Document the cache corruption workaround** — Add to AGENTS.md: "If lint shows 0 issues but seems wrong, run `rm -rf ~/.cache/go-build && mkdir -p ~/.cache/go-build`"
9. **Add disk space monitoring** — A simple check at the start of `nix run .#test` that warns if disk is >85% full.

### Testing

10. **Integration test for the full encryption pipeline** — Sign + encrypt + store + load + decrypt + verify roundtrip.
11. **Property-based tests for storage** — SQL store invariants (ordering, versioning, pagination).
12. **Race detector in CI for turso** — The indexing module has concurrent operations.

### Documentation

13. **Consumer migration guide** — v2.2.0 → v2.3.0 breaking changes (if any).
14. **Module dependency graph visualization** — Auto-generate from go.mod files.
15. **API stability report** — Run `cmd/api-stability` and publish results.

---

## f) Top 25 Things to Get Done Next

### 🔴 Critical (Do First)

1. **Fix buildflow golden file corruption** — Either configure exclusions or uninstall the hook. This is causing data loss.
2. **Clean up /tmp** — 47GB is excessive. Identify and remove old temp files.
3. **Free disk space** — Root at 89%. Clean Docker images, old nix generations, etc.

### 🟡 High Impact

4. **govulncheck in CI** — Add vulnerability scanning to the CI pipeline
5. **Docker build in CI** — The Dockerfile exists, needs CI integration
6. **Prometheus /metrics endpoint** — Built-in metrics exporter middleware
7. **pprof middleware** — Built-in profiling endpoints for production debugging
8. **Structured logging (slog) middleware** — Standardize logging across modules
9. **Distributed tracing examples** — OTel collector setup guide with code
10. **Performance dashboard** — Continuous benchmark visualization (GitHub Pages)

### 🟢 Medium Impact

11. **cqrs-gen v2** — Plugin system, template customization
12. **PostgreSQL native adapter** — Beyond the current database/sql abstraction
13. **gRPC transport adapter** — Protocol buffer message serialization
14. **NATS message queue adapter** — For event distribution
15. **Redis adapter** — Cache + pub/sub
16. **Multi-tenant store** — Namespace isolation for SaaS use cases
17. **S3/GCS archival** — Cold storage for old events
18. **Log compaction strategy** — Prune/snapshot old events
19. **Migration generator** — Auto-generate SQL from schema module
20. **Property testing expansion** — More rapid-based tests for edge cases

### 🔵 Polish

21. **Playwright E2E for example/user/** — Browser-based integration tests
22. **WASM compilation verification** — Ensure modules compile to WebAssembly
23. **Arena allocation experiment** — Go arena API exploration
24. **jsonv2 codec** — When Go's new encoding/json/v2 stabilizes
25. **Chaos engineering framework** — Fault injection for resilience testing

---

## g) Top Question I Cannot Figure Out Myself 🤔

**Why does the `buildflow` pre-commit hook reformat JSON golden files when `flake.nix` treefmt explicitly excludes `testdata/golden/**`?\*\*

The `flake.nix` treefmt configuration at lines 92–96 explicitly excludes `**/testdata/golden/**`:

```
excludes = [
  "**/testdata/golden/**"
];
```

Yet buildflow still reformats `codec/testdata/golden/json_encode.json` and `middleware/testdata/golden/health-check-response.json` on every commit. Buildflow appears to have its own JSON formatter that operates independently of treefmt.

I cannot determine:

1. Whether buildflow reads treefmt's excludes or has its own config
2. Where buildflow's formatting rules are configured (no `.buildflow.yml` or similar in repo)
3. Whether this is a bug in buildflow or a configuration issue on this machine

**This needs user input** — either configure buildflow to respect golden file excludes, or uninstall the hook and rely on `nix fmt` (which correctly excludes golden files).

---

## Session Metrics

| Metric               | Value                                                |
| -------------------- | ---------------------------------------------------- |
| Commits this session | 10                                                   |
| Files changed        | 75                                                   |
| Lines added          | +1,655                                               |
| Lines removed        | -285                                                 |
| Packages tested      | 41 (all pass)                                        |
| Modules linted       | 23 (all clean)                                       |
| Avg library coverage | ~89%                                                 |
| Total Go files       | 690                                                  |
| Total modules        | 28 (22 library + 1 integration + 3 examples + 2 cmd) |
| ADRs                 | 18                                                   |
| Go version           | 1.26.3                                               |

---

## Commit History This Session

```
da8c406b chore(deps): bump example/snapshot/testutil internal module versions
824483cc fix(lint): resolve all lint issues, fix rows.Err checks, remove dead code
f8d36004 chore: bump internal module dependencies to v2.3.0
0593bd7d chore(projection): require event/v2 v2.3.1 for ProcessingMode API
a37fad87 fix(modules): add missing memory replace directive and align dep versions
084e8b1c feat(encryption): implement Journal/SeekableJournal/BackwardsSource on EncryptedStore
70c6397d fix(codec): restore compact JSON golden file corrupted by c8e5f32a
c8e5f32a docs: apply consistent markdown table formatting across ADR-0016/0017/0018
bf4111eb docs: post-v2.3.0 comprehensive cleanup — ADR-0016/0017/0018, TODO cleanup
3d5ec978 feat(phase1-2): quick wins + correctness improvements
```

---

_Generated by Crush — 2026-06-14_
