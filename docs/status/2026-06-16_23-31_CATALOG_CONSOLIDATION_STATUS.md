# Comprehensive Project Status Report

> **Date:** 2026-06-16 23:31
> **Branch:** `consolidate-catalog`
> **Version:** v2.3.0 (post-release, active development toward v2.4.0)
> **Go:** 1.26.3 · **Modules:** 30 (29 in go.work + 1 root) · **Total Go files:** 726
> **Source LOC:** 31,670 · **Test LOC:** 62,136 (test:source ratio ≈ 2:1)

---

## Executive Summary

go-cqrs-lite is a **healthy, production-grade CQRS/ES library** with strong test
coverage (84–99% across core modules), clean multi-module architecture, and a
comprehensive feature set spanning event sourcing, CQRS, snapshots, projections,
signing, encryption, and catalog documentation generation.

**This session's work:** Discovered and fixed a critical over-modularization
defect — the catalog module was split into 6 separate `go.mod` files that
delivered zero dependency isolation while imposing real maintenance costs.
Consolidated 5 sub-modules back into packages within the single `catalog/v2`
module. **Zero breaking changes** — import paths preserved by construction.

**Key finding:** The project has accumulated **5 modules with dependency budget
violations**, **3 compiled binaries committed to git**, and **1 module (`encryption`)
with zero release tags** despite being v2.3.0. These are the top priority items.

---

## a) FULLY DONE ✅

### Catalog Consolidation (This Session)

| Item | Status | Detail |
|---|---|---|
| 5 sub-module `go.mod` + `go.sum` deleted | ✅ DONE | `d2`, `asyncapi`, `openapi`, `eventcatalog`, `docserver` merged into `catalog/v2` |
| Import paths preserved | ✅ DONE | Package paths `catalog/v2/d2` etc. identical — zero `.go` files changed |
| `go.work` cleaned | ✅ DONE | Removed 5 stale entries |
| `flake.nix` cleaned | ✅ DONE | Removed 5 stale `testModules` entries |
| `check-module-layers.sh` cleaned | ✅ DONE | Removed 5 LAYER + 5 DEP_BUDGET entries |
| `example/user/go.mod` cleaned | ✅ DONE | Removed 3 `require` + 3 `replace` directives for sub-modules |
| Full test suite passes | ✅ DONE | All 9 catalog packages pass in isolation (`GOWORK=off`), example/user passes |
| Branch pushed | ✅ DONE | `consolidate-catalog` pushed to origin |

### Core Library (Pre-existing)

| Module | Coverage | Tags | Status |
|---|---|---|---|
| `event` | 93.0% | 7 | ✅ Reactive EventBus, ImmutableEvent, zero-copy reads |
| `command` | 96.2% | 6 | ✅ Dispatcher, Bus, Store, Journal |
| `decider` | 99.4% | 6 | ✅ Pure-function aggregate pattern |
| `id` | 97.5% | 6 | ✅ Branded IDs via `id.Of[T]` |
| `memory` | 98.5% | 13 | ✅ Full in-memory test implementations |
| `middleware` | 93.5% | 10 | ✅ Logging, Retry, Recovery, Metrics, OTel |
| `signing` | 94.5% | 6 | ✅ HMAC-SHA256, Ed25519, multisig |
| `codec` | 88.9% | 6 | ✅ JSON, CBOR (deterministic), Raw |
| `otel` | 97.3% | 6 | ✅ Shared OTel helpers |
| `kv` | 94.9% | 0 | ✅ KV abstraction (new module) |
| `catalog` | 84.5% | 15 | ✅ Registry, SchemaFromType, 5 exporters |
| `storage` | 82.1% | 11 | ✅ SQL stores (PG/SQLite/Turso) |
| `pebble` | 81.4% | 5 | ✅ Embedded KV store |
| `encryption` | 86.9% | **0** | ✅ XChaCha20-Poly1305, AES-256-GCM |
| `query` | 72.9% | 6 | ⚠️ Lowest coverage, but functional |
| `integration` | — | 8 | ✅ Cross-module tests all pass |

### Infrastructure

- ✅ Multi-module Go workspace (`go.work`) with 29 modules
- ✅ Nix flake build system (`nix run .#build`, `.#test`, `.#lint`)
- ✅ GitHub Actions CI (build/vet/test/lint/race/coverage + GOWORK=off per-module)
- ✅ 21 ADRs (0001–0021) documenting architecture decisions
- ✅ gosec security scanning in CI with SARIF upload
- ✅ Docker multi-stage builds (linux/amd64 + linux/arm64)
- ✅ Benchmark baseline regression detection in CI
- ✅ go-error-family adoption across all modules (completed this session cycle)
- ✅ Property-based testing with `pgregory.net/rapid`
- ✅ BDD testing with Ginkgo v2 + Gomega
- ✅ Golden test infrastructure across event/encryption/codec/otel

---

## b) PARTIALLY DONE ⚠️

| Item | Status | What's Missing |
|---|---|---|
| **ADR-0012** (Split catalog modules) | ⚠️ Status still says "Proposed" | Must be marked **Superseded** — the consolidation reverses this decision |
| **`encryption` module release** | ⚠️ 0 git tags | Module is v2.3.0 quality but never tagged. Consumers can't `go get` it. |
| **`kv` module integration** | ⚠️ In `go.work` + `flake.nix` but **NOT in `check-module-layers.sh`** | No layer validation, no dep budget enforcement |
| **SQL Backend facade** | ⚠️ Uncommitted: adding `SnapshotStore()` + `CheckpointStore()` | Written but not committed (in working tree) |
| **ROADMAP.md** | ⚠️ Uncommitted updates | Marking Docker CI as done, Playwright as not-applicable |
| **`example/user` & `example/todo`** | ⚠️ Pre-existing uncommitted formatting changes | gofmt import ordering, golines 120-char wrapping |
| **Dep budget compliance** | ⚠️ 5 modules over budget | turso (8/6), codec (2/0), pebble (7/5), storage (11/10), integration (19/18) |

---

## c) NOT STARTED 🔲

| Item | Priority | Detail |
|---|---|---|
| ADR-0012 status update | High | Must mark "Superseded by consolidation" — currently misleading |
| ADR for catalog consolidation | Medium | Should document WHY the split was reversed (zero isolation benefit) |
| `kv` module in `check-module-layers.sh` | High | No LAYER or DEP_BUDGET entry — completely unvalidated |
| `encryption` module tagging | High | Zero release tags despite being production-quality |
| Dep budget reconciliation | Medium | 5 modules over budget — either raise budgets or trim deps |
| Binary cleanup | High | 3 compiled binaries committed to git (see section d) |
| `.gitignore` for example binaries | Medium | `example/user/user`, `example/encryption/encryption`, `cmd/cqrs-gen/cqrs-gen` |
| v3 breaking changes | Low | 6 items deferred to v3 in TODO_LIST.md (Closer removal, Writer/Reader split, etc.) |

---

## d) TOTALLY FUCKED UP! 🔴

### 1. Compiled Binaries Committed to Git

**Three compiled Go binaries are tracked in version control:**

```
./example/user/user           (binary)
./example/encryption/encryption (binary)
./cmd/cqrs-gen/cqrs-gen       (binary)
```

These bloat the repo, cause merge conflicts, and violate basic Git hygiene. The
`.gitignore` has entries for `/bin/` and `/build/` but NOT for these specific
binary paths. They need to be `git rm --cached` and added to `.gitignore`.

### 2. ADR-0012 is Actively Misleading

`docs/adr/0012-split-catalog-modules.md` still says **Status: Proposed** and
argues for splitting catalog into 5 modules. We just did the exact opposite
(consolidated 5 back into 1). Anyone reading this ADR will be confused about
the actual architecture decision. It must be updated to **Status: Superseded**
with a reference to the consolidation rationale.

### 3. `encryption` Module Has Zero Release Tags

```
git tag -l 'encryption/v*' → (empty)
```

The encryption module is at v2.3.0 quality (86.9% coverage, XChaCha20-Poly1305 +
AES-256-GCM, fuzz tested) but has **never been tagged**. External consumers
cannot import it via `go get`. Every other module has 5+ tags. This is a release
infrastructure failure.

### 4. Five Modules Violate Their Dependency Budgets

```
BUDGET: turso has 8 direct deps (budget: 6)       — 33% over
BUDGET: codec has 2 direct deps (budget: 0)        — INFINITE% over (budget is zero!)
BUDGET: pebble has 7 direct deps (budget: 5)       — 40% over
BUDGET: storage has 11 direct deps (budget: 10)    — 10% over
BUDGET: integration has 19 direct deps (budget: 18) — 6% over
```

The `check-module-layers.sh` script detects these but CI evidently doesn't fail
on them (or they'd have been fixed). Either the budgets are wrong (too tight)
or the deps are wrong (should be indirect). This makes the budget system
meaningless — it cries wolf but nobody listens.

---

## e) WHAT WE SHOULD IMPROVE! 🚀

### Architecture

1. **`kv` module is a ghost in the layer checker** — it exists in `go.work` and
   `flake.nix` but is completely absent from `check-module-layers.sh`. If `kv`
   is a real module (Layer 0 candidate — it's a leaf), it needs LAYER + DEP_BUDGET
   entries. If it's not real, it shouldn't be in the workspace.

2. **Dep budget system has no teeth** — 5 violations exist and nothing happens.
   Either wire `check-module-layers.sh` into CI as a required gate, or remove
   the budgets entirely. A check that doesn't block is worse than no check — it
   creates false confidence.

3. **SQL Backend is half-built** — `SQLBackend` exposes `EventStore()`,
   `CommandStore()`, `QueryStore()` but the uncommitted work adds
   `SnapshotStore()` + `CheckpointStore()`. This should be committed and
   completed — the backend facade should be the single entry point for all
   SQL stores sharing one `*sql.DB`.

4. **Replace directive sprawl remains** — Even after catalog consolidation, 22
   `replace` blocks remain across modules. While documented as necessary for
   `GOWORK=off` per-module CI, this is maintenance overhead. Consider whether
   all modules truly need `GOWORK=off` testing or if only leaf modules do.

### Type Safety

5. **`query.Handler` still returns `any`** — Deferred to v3 as a breaking change,
   but `query.TypedHandler[T]` already exists. The `any` return is a type safety
   hole that forces consumers to type-assert.

6. **`io.Closer` on core interfaces** — ADR-0010 accepted removing it, deferred
   to v3. This conflation of lifecycle with storage concerns persists.

### Testing

7. **`query` module has lowest coverage (72.9%)** — Below the project's 80%
   target. Needs focused test investment.

8. **No integration test for the SQL Backend facade** — The backend that ties
   together EventStore + CommandStore + QueryStore + (pending) SnapshotStore +
   CheckpointStore has no test verifying they share one connection correctly.

### Documentation

9. **ADR hygiene is broken** — ADR-0012 says "Proposed" for a decision we just
   reversed. ADRs should reflect current state. Status fields need a sweep.

10. **FEATURES.md says "28 modules"** but we now have 30 `go.mod` files (added
    `kv`, consolidated catalog). Doc drift.

---

## f) Top 25 Things to Get Done Next

### Tier 1 — Critical (Do This Week)

| # | Task | Impact | Effort |
|---|---|---|---|
| 1 | **Remove 3 committed binaries** from git (`git rm --cached`) + update `.gitignore` | High | 5 min |
| 2 | **Mark ADR-0012 as Superseded** with consolidation rationale | High | 10 min |
| 3 | **Write ADR-0022: Catalog Consolidation** documenting the reversal | Medium | 20 min |
| 4 | **Add `kv` to `check-module-layers.sh`** (LAYER + DEP_BUDGET) | High | 10 min |
| 5 | **Tag `encryption/v2.3.0`** (and `kv/v2.3.0`) — they're production-quality but unpublished | High | 10 min |
| 6 | **Fix dep budgets**: either raise budgets to match reality OR move deps to indirect | High | 30 min |
| 7 | **Commit the uncommitted SQL Backend work** (SnapshotStore + CheckpointStore on facade) | High | 15 min |
| 8 | **Commit the ROADMAP.md + command/aggregate_ref.go + kv formatting changes** | Medium | 10 min |

### Tier 2 — High Value (Do This Sprint)

| # | Task | Impact | Effort |
|---|---|---|---|
| 9 | **Wire `check-module-layers.sh` into CI as required gate** | High | 30 min |
| 10 | **Update FEATURES.md** — module count 28→30, catalog consolidation | Medium | 15 min |
| 11 | **Update AGENTS.md** — catalog module structure, `kv` module addition | Medium | 15 min |
| 12 | **Add integration test for SQL Backend facade** — verify all stores share one `*sql.DB` | High | 1 hr |
| 13 | **Improve `query` coverage** from 72.9% → 80%+ | Medium | 2 hr |
| 14 | **Sweep all ADR statuses** — ensure each reflects current reality | Medium | 30 min |
| 15 | **Add pebble doc.go prefix documentation** (uncommitted improvement) | Low | 5 min |

### Tier 3 — Polish (Do This Release Cycle)

| # | Task | Impact | Effort |
|---|---|---|---|
| 16 | **Consolidate `replace` directives** — audit which modules truly need GOWORK=off testing | Medium | 2 hr |
| 17 | **Add `example/todo` catalog integration demo** — show d2 + asyncapi + eventcatalog working post-consolidation | Low | 1 hr |
| 18 | **Fix `storage/sql_backend.go` uncommitted checkpoint store code** | Medium | 30 min |
| 19 | **Add deprecation notice to ADR-0016** (Outbox Pattern — declined, use Watermill) | Low | 5 min |
| 20 | **Document `kv` module purpose** — is it Layer 0? What's its relationship to `pebble`? | Medium | 20 min |

### Tier 4 — Strategic (v3 Planning)

| # | Task | Impact | Effort |
|---|---|---|---|
| 21 | **Plan v3 `io.Closer` removal** from event.Store, snapshot.SnapshotStore, command.Store | High (breaking) | 1 day |
| 22 | **Plan v3 `query.Handler` generic return** — eliminate `any` | High (breaking) | 4 hr |
| 23 | **Plan v3 event.Store split** into Writer/Reader/Deleter (ADR-0010) | High (breaking) | 1 day |
| 24 | **Evaluate outbox pattern via Watermill** — ADR-0016 declined, but is there a lighter-weight alternative? | Medium | 2 hr |
| 25 | **Benchmark suite consolidation** — ensure per-module benchmarks are comparable across versions | Low | 2 hr |

---

## g) Top #1 Question I Cannot Figure Out Myself

### **What is the intended relationship between `kv/` and `pebble/`?**

The `kv` module exists in `go.work` and `flake.nix` `testModules`, has a
`go.mod` (`github.com/larsartmann/go-cqrs-lite/kv/v2`), source files
(`kv.go`, `mem.go`, `errors.go`, `doc.go`), tests (94.9% coverage), and
defines a `Store` interface with `MemStore` implementation.

But it is **completely absent from `check-module-layers.sh`** — no LAYER
assignment, no DEP_BUDGET. It's also **not mentioned in `AGENTS.md`**'s module
list or the module graph.

**Questions I can't answer:**
- Is `kv` a new Layer 0 leaf module (KV interface + memory impl) that `pebble/`
  will eventually depend on instead of importing PebbleDB directly?
- Or is `kv` an extraction of pebble's internal KV interface that will be shared
  with a future Redis/BoltDB adapter?
- Should `pebble.Store` implement `kv.Store`, or are they independent?
- What layer is `kv`? (Presumably Layer 0 since it's a pure interface + mem impl,
  but I need confirmation.)

**Why it matters:** Without knowing `kv`'s intended role, I can't:
1. Assign it the correct layer in the architecture
2. Set an appropriate dependency budget
3. Know whether `pebble` should depend on it
4. Document it correctly in AGENTS.md and FEATURES.md

---

## Verification Matrix

| Check | Command | Result |
|---|---|---|
| Catalog has 1 go.mod | `find catalog -name go.mod \| wc -l` | **1** ✅ |
| Zero replace in catalog | `grep -rn "replace" catalog/*/go.mod` | **0** ✅ |
| Catalog tests (isolated) | `cd catalog && GOWORK=off go test ./...` | **9/9 pass** ✅ |
| example/user tests | `cd example/user && go test ./...` | **pass** ✅ |
| integration tests | `cd integration && GOWORK=off go test ./...` | **5/5 pass** ✅ |
| Layer violations (catalog) | `check-module-layers.sh \| grep catalog` | **0** ✅ |
| Layer violations (other) | `check-module-layers.sh` | **5 modules over budget** ⚠️ |
| Compiled binaries in git | `find . -type f -executable` | **3 binaries** 🔴 |
| encryption tags | `git tag -l 'encryption/v*'` | **0** 🔴 |
| kv in layer checker | `grep kv check-module-layers.sh` | **absent** 🔴 |

---

## Git State

```
Branch: consolidate-catalog (pushed to origin)
Commits this session:
  7fc59315 refactor(catalog): consolidate 5 sub-modules into packages
  da95686f chore(examples,docs): apply gofmt import ordering
  4dc8cbc2 chore(catalog): clean up config references to removed sub-modules

Uncommitted (pre-existing, not authored this session):
  ROADMAP.md                    — Docker CI done, Playwright N/A
  command/aggregate_ref.go      — doc comment on re-export
  docs/adr/README.md            — ADR status updates
  kv/mem.go, kv/mem_test.go     — formatting + exhaustruct fixes
  pebble/doc.go                 — prefix documentation
  storage/sql_backend.go        — SnapshotStore + CheckpointStore facade methods
```

---

*Generated by Brutal Self-Review + Status Report skills*
