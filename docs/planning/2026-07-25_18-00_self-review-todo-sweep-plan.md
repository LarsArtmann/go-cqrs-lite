# Comprehensive Execution Plan — Self-Review TODO Sweep

**Created:** 2026-07-25
**Source:** [`docs/status/2026-07-25_17-32_brutal-self-review-and-comprehensive-status.md`](../status/2026-07-25_17-32_brutal-self-review-and-comprehensive-status.md) (section f: 50 next-steps)
**Mandate:** Execute the WHOLE TODO list. Every item either DONE, fixed-in-place, or explicitly deferred-with-tracking.

## Prioritization Key

`Impact × CustomerValue ÷ Effort`, sorted descending. Each task ≤12 min.

## Workstream A — Gates & Correctness (Tier 1, do first)

| ID | Task | Impact | Effort | Origin |
|----|------|--------|--------|--------|
| A1 | Root-cause gopls phantom DuplicateMethod (clear cache, document) | High | 10m | f5 |
| A2 | Pebble `safeInt64` → real `math.MaxInt64` clamp (drop nolint) | High | 5m | f3 |
| A3 | CHANGELOG `[Unreleased]` Fixed: idempotency Record TTL-no-extend | High | 5m | f2 / Q2 |
| A4 | Run `#check-api-stability`; regen golden if stale; add step to `#verify` | High | 10m | f1,f6,f48 |

## Workstream B — Idempotency Contract Depth (Tier 2)

| ID | Task | Impact | Effort | Origin |
|----|------|--------|--------|--------|
| B1 | 3-way Record contract test (memory+kvstore+sqlstore) in integration/ | High | 12m | f4,f8 |
| B2 | kvstore concurrent Record (SetIfAbsent contention) test | Med | 8m | f12 |
| B3 | kvstore Seen lazy-delete + Record reclaims-expired tests | Med | 8m | f39,f40 |
| B4 | Idempotency TTL-under-load soak test | Low | 10m | f22 |
| B5 | Document idempotency Store contract in DOMAIN_LANGUAGE.md | Med | 8m | f41 |
| B6 | Track RefreshTTL optional capability in TODO_LIST.md (defer) | Low | 3m | f16 |

## Workstream C — Metaengine Test Depth (Tier 2/3)

| ID | Task | Impact | Effort | Origin |
|----|------|--------|--------|--------|
| C1 | ADTSortedMap complexity honesty test (ONLogN not OLogN) | Med | 8m | f7 |
| C2 | CounterBackend concurrent-increment test | Med | 8m | f8 |
| C3 | ExecuteTyped typed-mismatch error test | Med | 8m | f20 |
| C4 | Close() no-op contract test (idempotent, caller-owns-DB) | Low | 6m | f15 |
| C5 | LogBackend append+tail restart test | Low | 10m | f27 |
| C6 | SetBackend concurrent SetAdd idempotency test | Low | 8m | f28 |
| C7 | GraphBackend BFS restart test | Low | 10m | f29 |
| C8 | Plan picks memory over SQLite for point lookups (cost) test | Med | 8m | f37 |
| C9 | Cursor round-trip across all value types test | Low | 8m | f38 |
| C10 | README SQLite example + jsonv2 portability note | Med | 10m | f26,f36 |
| C11 | 3 ADRs: reify fallback, tx-MapUpdate, multimap seq-seed | Med | 12m | f42,f43,f44 |
| C12 | Reconcile metaengine tier placement (AGENTS.md vs ADR-0046) | Low | 6m | f25 |

## Workstream D — Benchkit Test Depth (Tier 2/3)

| ID | Task | Impact | Effort | Origin |
|----|------|--------|--------|--------|
| D1 | GOMAXPROCSSweep restore-after-panic test | Low | 8m | f30 |
| D2 | WorkerSweep scaling monotonicity test | Low | 8m | f31 |
| D3 | Repeat median-selection test | Low | 8m | f32 |
| D4 | PrintSweep handles FAILED+success mix test | Low | 6m | f46 |
| D5 | ScalingSweep preserves result ordering test | Low | 6m | f47 |

## Workstream E — Infrastructure / Shared (Tier 2)

| ID | Task | Impact | Effort | Origin |
|----|------|--------|--------|--------|
| E1 | Extract raceEnabled → testutil/race.go (build-tag helper) | Med | 10m | f10 |
| E2 | Document raceEnabled build-tag pattern in AGENTS.md | Med | 5m | f5(proc),f11 |
| E3 | Lint sweep: uint64→int64 casts (benchkit/cpu_unix.go) | Med | 8m | f13 |
| E4 | Lint sweep: otel varnamelen + godoc/nlreturn/wrapcheck confirm-clean | Low | 8m | f14,f17,f18,f19 |
| E5 | projectionhost host_reset typo-fix regression test | Low | 8m | f23 |
| E6 | Coverage delta measurement + report (metaengine, kvstore, benchkit) | Med | 8m | f9 |

## Workstream F — Docs / Tracking (Tier 3, defer-with-tracking)

| ID | Task | Impact | Effort | Origin |
|----|------|--------|--------|--------|
| F1 | TODO_LIST.md: RefreshTTL, cqrs-lint Record rule, cqrs-bench metaengine profile, daemon msg triage, CI badge, recurring lint sweep | Low | 10m | f16,f24,f35,f45,f34,f50 |

Explicitly **deferred** (feature/process work, not fixes — tracked, not half-built):
- `RefreshTTL(ctx,key,ttl)` optional capability (new API → TODO_LIST)
- cqrs-lint rule for Record contract (new rule → TODO_LIST)
- cqrs-bench metaengine SQLite profile (new bench profile → TODO_LIST)
- `-race` on #check-api-stability (N/A: it's `go run`, not tests → documented)
- Auto-commit daemon message triage (prior decision: leave as-is → TODO_LIST)
- Recurring lint-sweep scheduling (process → TODO_LIST)
- `#verify-full` superseded by adding api-stability to `#verify` (A4)

## Workstream G — Final Verification & Report

| ID | Task | Impact | Effort |
|----|------|--------|--------|
| G1 | `nix fmt` | High | 2m |
| G2 | `nix run .#verify` green | High | 5m |
| G3 | `nix run .#check-api-stability` green | High | 3m |
| G4 | Final completion table report | High | 5m |

## Execution Rules

1. Verify (`go build`/`go test`) after each task that touches code.
2. Run `nix run .#lint` after lint-sweep + code workstreams.
3. Keep `#verify` green at every workstream boundary.
4. Never edit files outside a task's stated scope without flagging.
5. Real fixes over `//nolint`; behavioral tests over cosmetic ones.
