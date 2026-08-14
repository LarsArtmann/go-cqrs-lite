# Brutal Project Review — go-cqrs-lite

**Date:** 2026-08-14
**Scope:** Full-project audit — core modules, storage + metaengine, docs honesty, tooling, gap hunt.
**Method:** Every claim verified against code, published tags, or CI goldens. No vibes.
**Context:** 5 months old (2026-03-15 → 2026-08-14), 5,592 commits, 86 modules, ~350K lines of Go, 1,078 test files, 125 ADRs, 1,060 tags.

> HTML version of this report: `docs/reviews/2026-08-14_14-25_brutal-self-review.html`

---

## ⚡ EXCEPTIONAL

### Engineering discipline infrastructure
`flake.nix`, `.github/workflows/ci.yml`, `cmd/api-stability`, `cmd/doc-check`

CI is real and it bites: build/vet/test/race/lint, api-stability golden with `TestEveryGoModDirIsInModulesList` drift guard, dependency budgets via `check-arch`, duplication gate (art-dupl + baseline), coverage-drift checker, doc-check, per-module `GOWORK=off` isolation, hermetic NixOS VM integration tests (postgres, mysql, duckdb, turso, dgraph). Meta-tests prevent the gates themselves from rotting. Most companies with 50 engineers don't have this.

### CGo + dependency isolation architecture
86 `go.mod` files, `metaengine/duckdbengine` `//go:build cgo`, engine-per-module

DuckDB is the only CGo dependency and every one of its 22 files is correctly gated; engine core takes `*sql.DB` so it compiles with `CGO_ENABLED=0`. Consumers who never import duckdb never need a C compiler. Dependency budgets are enforced, not aspirational.

### Immutability + zero-copy discipline in the event core
`event/event.go:121`, `event/event_new.go:89`, `event/codec.go:97`

`Payload()` clones, `Metadata()` deep-copies, constructors clone input bytes, and `PayloadReadOnly` is a documented internal escape hatch used by exactly the right serialization paths. Defensive cloning is policy, not accident.

### Test culture
fuzz, property (rapid), BDD (ginkgo), race, restart-safety, soak, contract suites, testcontainers

~50% of files are tests. Fuzzing on codecs/parsers, property tests on ring-buffer invariants, restart-safety tests on pebble/bbolt, convergence tests on iroh replication, a reusable `eventtest` contract suite, race-aware allocation thresholds. The test depth matches the code depth.

### decider singleflight + cache correctness
`decider/decider.go:157-180`, `decider/load.go:277-299`

Cache updated only after successful Save (version-conflict losers never poison it), delta folds from cache version, invalidation on error, coalesced loads. The hard concurrency logic is actually right.

### dedup/
95 lines doing exactly one thing correctly: O(1) add/has, eviction invariants property-tested, wraparound, duplicate-add, nil-receiver safety, documented capacity rationale.

---

## ✅ GOOD

### Core modules (event, command, query, decider, dispatcher, kv, id, metadata)
All rated GOOD: honest APIs, real tradeoff documentation (`id/stream_id.go:17-46` is the best doc in the repo), value semantics everywhere. Defects found are surgical (see MEH) — not systemic.

### Metaengine cost-based planner is real, not vaporware
`metaengine/cost.go`, `metaengine/rules.go`, `metaengine/store_routing.go`

8-rule pipeline, calibrated per-read-pattern costs, hysteresis deadband against re-route oscillation, live RTT EWMA, plan diff/audit, Doctor diagnostics. And an honesty note in `cost.go:3-15` admitting it's first-order. Genuine original engineering.

### SQL security posture
`storage/sql/where.go`, `metaengine/dgraphengine/injection_test.go`, `metaengine/layout.go:131`

All values parameterized everywhere, Dgraph uses QueryWithVars + has a dedicated injection test, identifier quoting with escaping. Gaps exist (see MEH) but the baseline is right.

### cqrs-lint (202 rules) + cqrs-gen + integration suite
The 202 count is machine-enforced (meta-test syncs catalog and detectors bidirectionally). ~100 rules are genuine bug-catchers. cqrs-gen is tested. `integration/` covers chaos, OTel span trees, sign+encrypt composition, time-travel loads — real implementations, no mocks.

### Documentation honesty ~95% of the time
~20 DONE claims in FEATURES.md spot-checked and real; README quickstart is backed by a compiled example (`example/readme-quickstart`). Most libraries lie far more. The 5% that lies is concentrated and load-bearing (see BAD).

---

## 👍 DECENT

### record/ module — honest code, dishonest framing
`record/record.go:22-24`

ADR-0111 pitched `record` as the metadata unifier. Nothing in event/ or command/ embeds it — it's a **third** parallel metadata model consumed only by metaengine, bridged by converters. Also `NewStreamRef` validates nothing and `Split()` breaks on stream types containing `/`.

### TODO_LIST.md / ROADMAP.md
52 open items, fresh to 2026-08-11 — unusually current, but 4 items have false premises (codec aliases "to add" already exist at `codec/alias.go:127-130`; the idempotency/flightrecorder "shim deletion" items contradict both the code and their own doc.go files).

### example/ — flagship works, one is a ghost
taskmanager is the real deal (ES + projections + metaengine + OTel + signing + tests) but has a machine-local `replace` breaking it off-machine. metaengine-quickstart has no tests, no README, and is missing from CI `examplePaths` — it never builds anywhere.

### Engine implementations — complete but unevenly tested
All 10 engines are real implementations, zero stubs in code. But mysqlengine (env-gated) and dgraphengine (env-gated) get silently skipped in default CI, and tursoengine has no tests of its own.

---

## 😐 MEH

### "Graceful degradation" = warning + crash, not warning + slow path
pg/mysql/duckdb engines declare "degraded" support for Set/Log/Multimap/Graph/Vector in their profiles but implement only Map + Counter. The planner routes, emits a DEGRADED diagnostic, then the fold **hard-fails at runtime** (`metaengine/store.go:623`). Only graph has a runtime fallback — and it needs MultimapBackend, which mysql doesn't have. The core vision ("given one engine, serve every query") is violated by exactly the engines users will actually deploy.

### Planner model nits
- Graph op cost is `branching × depth` instead of ~`branching^depth` (`metaengine/cost.go:95`)
- Volume silently defaults to 1000
- No filter selectivity modeling
- **No plan-time validation of `Supports` vs implemented interfaces** — a conformance check would have caught every stub above

### Schema DDL exists in 4+ places
The `events` DDL is duplicated across `storage/sql/dialect.go` (4 dialects) AND `storage/sql/migrations/*.sql` — with no test asserting they match. Metaengine SQL engines copy-paste pushdown/transaction/stream_log/scan across 4 modules. Divergence bugs are pending, not hypothetical.

### SQL edge gaps
- `FilterOp`/condition ops/column names interpolated without allowlist (`storage/sql/where.go:38`)
- ORDER BY columns unquoted on the view path (`storage/view/query.go:137`)
- DSN leaked verbatim into wrapped errors (`metaengine/tursoengine/register.go:69`) — credential exposure in logs

### Resource ownership split brain
pg/mysql/duckdb own and close their `*sql.DB`; sqliteengine documents "caller owns" but its own driver factory and `tursoengine.New` open a DB that nobody ever closes — leak on `Close()`.

### Core-module surgical defects
- Singleflight captures the leader's ctx — one cancelled request fails all coalesced waiters (`decider/load.go:32`)
- Command bus middleware runs once per handler, double-counting side effects (`command/memory_bus.go:115`)
- Query audit middleware mints a fresh RequestID and drops actor/correlation/duration (`query/audit.go:95`)
- `Pagination.Offset()` underflows on zero-value structs (`query/pagination.go:40`)
- `kv.Cache` hands the same `*T` to all readers (mutation hazard)
- `id` serializes ALL generation behind one global mutex (process-wide ceiling)
- TypedQueryStore decodes with hardcoded JSON fallback while ignoring its configured codec on read (`query/typed.go:97`)
- Ghost symbol: `event.ErrBinaryNotFound` referenced nowhere (`event/errors.go:69`)

---

## ❌ BAD

### ADR-0114 "tombstone → deletion events" is documented fiction
`FEATURES.md:94`, `CHANGELOG.md:893`, `docs/migration/tombstone-to-domain-events.md`, `AGENTS.md:85`

CHANGELOG documents a **BREAKING rename** (TombstonePolicy → DeletePolicy, `applyTombstonePolicy` → `applyDeletePolicy`) that never landed. Verified: `listing/types.go:42` still exports `TombstonePolicy`; the CI's own api-surface golden confirms it; `WithDeleteTypes`/`DeletePolicy`/`StatusDeleted` exist nowhere. The published migration guide contains code that will not compile. Meanwhile README's headline feature table still sells "tombstone soft-delete" — the thing ADR-0114 declares deprecated. Four documents tell three contradictory stories about the same feature.

### Deprecated-module story told three ways
`codec/`, `retry/`, `idempotency/`, `flightrecorder/`

codec + retry: consistently deprecated. idempotency's doc.go says kvstore/sqlstore are *permanent residents that "will never move"* while TODO plans to delete the module. flightrecorder is described in TODO as a "pure re-export alias" — it is the full implementation with 4 internal consumers. 50 go.mod files still carry the deprecated codec path as indirect dep. Nobody can tell what the deprecation policy actually is.

### Benchmark sprawl: 5 systems, 2 baselines, 0 truth
`benchkit`, `cmd/cqrs-bench`, `stack/bench`, `metaengine/bench`, `integration/`, `scripts/bench-*.sh`

Five overlapping harnesses. The regression baseline used by `benchmark-regression.sh` is a v2-era artifact (references `event/v2`, June ANSI escapes) while CI compares against a different curated baseline — and only ever emits a warning, never fails. `benchmarks/` dir is a stale dump. `metaengine/bench` is a whole Go module containing only test files, tracked by api-stability for zero exports. `reports/jscpd-report.json` is a JavaScript copy-paste detector report in a Go repo, duplicating the art-dupl gate.

### SESSION_MILESTONES.md dead, status split-brained
`docs/sessions/SESSION_MILESTONES.md`, `docs/status/`

Died at Session 100; nothing from August; "reality" lives in docs/status/ which nothing links as the canonical source. Module counts differ across README (68), ROADMAP (86), actual (88 go.mod files).

### Repo junk
`t/tasks.buf`, `result/bin` (16MB, root-owned), `reports/coverage.out` (empty), `codec/testdata`, `codec/reports`

`t/tasks.buf` is tracked-in-git temp junk referenced nowhere. `result/` is a 16MB root-owned Nix build leak inside the repo. Empty coverage artifact. Dead directories the TODO already knows about but hasn't trashed.

---

## 🔥 REALLY BAD

### 1. The published release chain does not build
`id/v4.4.0` (latest tag) missing `actor_id.go`; `command/metadata.go` uses `id.ActorID`; `command/go.mod` requires `id v4.4.0`

Verified: `id.ActorID` exists in the working tree, is used by `command` and required at `v4.4.0` — but the symbol exists in **no published id tag** (v4.4.0 is the latest and lacks it). Any consumer building command/metaengine/record with `GOWORK=off` fails to compile against published versions. While this is open, every "production-ready" claim is false. **This is the single highest-priority fix in the repo.**

### 2. Engine capability fraud: profiles declare what code doesn't implement
`metaengine/pgengine/engine.go:172-192`, `metaengine/mysqlengine/engine.go:139-153`, `metaengine/duckdbengine/engine.go:158-178`

Six engines declare `Supports` for backends that do not exist in their code. The planner trusts the profile, the test harness auto-skips missing backends (`metaengine/adttest/harness.go:106`), so nothing catches it. For a library whose stated north star is "developers never need to think about the storage layer," routing a query to an engine that then hard-fails is the worst possible failure mode. bbolt and badger engines are the honest ones — declare exactly what they implement.

### 3. Known broken code shipped and left broken
`metaengine/dgraphengine/counter.go:158`

Dgraph `CounterIncrement` is completely broken — DQL requires `$key0: string`, the code emits `$key0 string`. Verified present today. It's a single-character fix sitting in TODO_LIST for days. Plus the Dgraph journal off-by-one that corrupts checkpoint resumption. Multiple 2026-08-11 sessions shipped without running the verify gate ("stale-GREEN backlog" — the repo's own AGENTS calls this out as worse than no claim).

### 4. v4/v5 chimera — the cut was never made
ADR-0123, TODO_LIST "v5 Unification"

Phases 1-7 shipped; Phase 8 (the actual deletion: `stack/`, all 8 presets, `Materialize`, `RelationalProjection`, `graph.GraphProjection`, migration guide, v5.0.0 tag) is 0% done — all 7 boxes unchecked. Users live on a half-migrated chimera with two composition roots alive simultaneously and no migration guide for either direction.

### 5. Flagship example doesn't build off-machine; flagship linter flags the flagship example
`example/taskmanager/go.mod:88`, `cmd/cqrs-lint` E005, `cmd/cqrs-lint/testdata/taskmanager_golden.txt:20-29`

taskmanager carries `replace github.com/larsartmann/go-must => /home/lars/projects/go-must` — a machine-local path. And the cqrs-lint golden enshrines 10 E005 false positives because E005 doesn't recognize `system.RegisterCommand` — the linter is blind to the SDK's own composition layer. The "canonical clean reference" is neither clean nor a valid reference.

---

## 🕳️ Totally Missed (never done)

| Gap | Status |
| --- | --- |
| **Transactional outbox** | Designed (ADR-0016), zero code, then a REMOVE-OUTBOX-PLAN.md. Most conspicuous hole in an ES library — every serious competitor has one. |
| **Broker transports (NATS, Redis)** | `docs/design/transport-{nats,redis}.md` both say "Accepted, implementation pending". transport/ is http+grpc only. Ephemeral broker scripts exist with zero Go tests. |
| **Per-module CHANGELOGs** | 6 of 86 modules have one. Users importing one module cannot see what changed in it. |
| **Distributed projection running** | Leader election / sharding for projectionhost — design doc only. |
| **Event archival / stream compaction** | Design docs only; long-lived streams grow unbounded. |
| **Security hardening beyond markdown** | SECURITY.md supports "v3.x" (two majors stale); govulncheck failures swallowed in release.yml (`|| echo "WARN: skipped"`). |
| **Docs website / published versioned docs** | None; dashboard is a "Raw Idea". |
| **Module compatibility matrix** | 86 independently tagged modules, no published matrix. |
| **Multi-tenancy** | One paragraph in remaining-ideas.md. |
| **Schema-evolution codegen** | Runtime upcasters exist, the tooling doesn't. |
| **Kafka** | Never even planned. |

---

## Glossary

- **RTT** = Round-Trip Time — network latency to the database server.
- **EWMA** = Exponentially Weighted Moving Average — a smoothed average that weights recent samples more than old ones; recovers from stale data without keeping all history.
- **"Live RTT EWMA"** in this report means: the metaengine continuously pings remote engines (PG, MySQL, Dgraph, Turso), measures actual round-trip latency, and maintains a smoothed rolling average (`LatencyTracker`: ring buffer + incremental EWMA + P50/P95/P99, fed by `ProbeEngine` background loops) that the cost-based planner uses for routing decisions instead of hardcoded latency guesses.

---

## What To Do (Pareto order)

1. **Fix the release chain TODAY** — re-tag `id` (and dependents), verify `GOWORK=off go build` against published versions only. Nothing else matters until `go get` works.
2. **Stop the capability fraud** — one afternoon: make every engine profile declare exactly what it implements, add a plan-time `Supports`-vs-interface conformance test. Turns runtime crashes into honest DEGRADED diagnostics (or removes the lie).
3. **Fix the two Dgraph bugs** (colon = 1 char, off-by-one) — they're in TODO with XS/S effort, shipped broken.
4. **Reconcile the ADR-0114 story** — either land DeletePolicy or rewrite FEATURES/CHANGELOG/AGENTS/migration-guide to tell the truth. Pick one reality.
5. **Remove taskmanager's local `replace`** and teach E005 about `system.RegisterCommand`; regenerate the lint golden.
6. **Cut v5 or abandon it** — the chimera is worse than either endpoint. The WAL + store-middleware dedup plans (both 0% done) fold into this decision.
7. **One bench system** — pick benchkit+cqrs-bench, delete the rest, one baseline, make regression CI fail on breach.
8. **Tell one deprecation story** — pick the policy for codec/retry/idempotency/flightrecorder, apply it everywhere in the same edit.
9. **Trash the junk** — `t/`, `result/`, `reports/`, `codec/testdata`, `benchmarks/` dump, `metaengine/bench` module.
10. **Then, and only then**: outbox (biggest user-facing gap), NATS/Redis transports (or delete the design docs), per-module CHANGELOGs.

---

**Final verdict:** Top-1% engineering velocity and discipline in code and tests. Bottom-half discipline in releases, docs honesty, and system consolidation. The gap between how good the code is and how broken the release chain is — that's the whole story of this repo.
