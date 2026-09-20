# Queue arc M4 DONE — claim tokens, dep validation, FactTx, MySQL engine

**Date:** 2026-09-19 (midday session)

> **RESOLVED-BY-ROUTING (2026-09-19 docs-health 8th pass):** struck items above = verified shipped (CHANGELOG 2026-09-19 ADR-0142/ADR-0143 entries; TODO_LIST `[x]` rows). Open remainder tracked in TODO_LIST "Metaengine Universal Storage Substrate": T18b load-sweep + benchmark re-baseline (quiet-window gated), T19–T21 (v5-gated), tag waves, claim-metrics parity owner decision. ARCHIVED.
> **Scope:** the queue TODO item's M4 remainder (T14–T17) — executed from
> the TODO_LIST under the "execute everything, keep going" directive.
> **Repos:** go-cqrs-lite only. Concurrent arc (ADR-0142 substrate) was
> mid-flight in parallel; overlap points are called out in (d).

---

## a) FULLY DONE (all verified this session)

| Task                                    | What landed                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                       | Verification                                                                                                                                                                                                                   |
| --------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| **T15 — ADR-0134 claim tokens**         | `queue.Claim.Token` + `queue.NewClaimToken` (crypto/rand); `lease_token` column (NULL when unclaimed) with idempotent migrations (SQLite pragma-probe ALTER; PG `ADD COLUMN IF NOT EXISTS`); finalize signatures take the TOKEN (owner string superseded — remains attribution via `ClaimDue` arg / `lease_owner` / fact `Owner` read in-tx from the row); token predicates on Complete/Fail/FailPermanent/Requeue/Heartbeat/CancelOwned; reclaim mints a fresh token; ADR-0134 status → Accepted-for-queue with adoption addendum                | sqlite + PG suites green; new conformance Tokens suite (mint-per-claim/no-reuse, forged-token refusal on every finalize, full theft story with journal attribution to the reclaiming worker)                                   |
| **T14 — dep validation + cycle policy** | `queue.ErrDanglingDep` (Rejection, `queue.dangling_dep`): every `task.New.Deps` entry must exist at enqueue, uniformly on 3 engines (SQLite json_each anti-join; PG jsonb_array_elements_text anti-join; MySQL per-dep EXISTS); cycles are UNREPRESENTABLE by construction (deps fixed at enqueue + store-minted IDs) — documented as the policy in queue/doc.go; unblock-bump evaluated and REJECTED (claim-time gating unblocks transactionally; bounded aging covers starvation); cancelled/dead deps block forever (rescue re-opens) — pinned | Deps conformance suite (4 pins incl. total-rejection, dead-dep rescue re-open, chain drain, would-be-cycle rejection); error-taxonomy gate extended to queue (bidirectional)                                                   |
| **T16 — FactTx + Watermarks**           | `queue.FactTx`/`queue.FactSink`: `WithFacts(ctx, fn)` runs fn in ONE tx, sink appends commit/roll back together (ADR-0001 lineage upstreamed); implemented by sqlite+postgres+mysql over the task-tables' connection domain; `queue.Store.Watermarks(ctx)` operator list surface                                                                                                                                                                                                                                                                  | conformance pin: rollback leaves HeadSeq unchanged, commit lands both ordered + time-stamped; Watermarks list pinned in pinWatermark                                                                                           |
| **T17 — `queue/mysql/v4`**              | Third engine: MySQL 8+/MariaDB 10.6+; two-statement claims (SELECT … FOR UPDATE SKIP LOCKED → token-fenced UPDATE + RowsAffected fence); BIGINT unix-ms everywhere (deliberate deviation from the plan's DATETIME(3) note — one encoding across all engines, no tz traps); per-statement DDL (no multiStatements DSN requirement); nullable-`dedup_key` UNIQUE emulation of the partial index; InnoDB deadlock (1213/1205) in-engine retry in ClaimDue; per-dep EXISTS validation; DSN-gated conformance (fresh throwaway database per subtest)   | Green vs live ephemeral MariaDB 11.4 (`MYSQL_TEST_DSN`), incl. `-race -count=2`. Three dialect realities found+fixed live: multi-statement DDL split, `last_error` NOT NULL needs explicit '', deadlock retry                  |
| **Ceremony**                            | go.work + flake testModules + api-stability slice + LAYER[queue/mysql]=5 + DEP_BUDGET 2 + cqrs-lint exclusion + queue/.go-arch-lint mysql exclusion; module-map + modules.md + README rows (queue row refreshed for tokens/FactTx/dep-validation); CHANGELOG entry (5 bullets); api golden regen (7400 exports at my regen; 7413 after merging their Engine surface — see (d))                                                                                                                                                                    | api-stability `TestEvery*` green (-count=1); changelog-symbols 138 citations green; error-taxonomy 525 codes green; check-module-layers green; doc-check 1154 refs valid exit 0; file-size: ZERO queue violations; gofmt clean |

## b) Verification evidence (the runs that count)

- `queue/sqlite`: `-race -count=2` ok (128s)
- `queue/postgres`: `-tags integration -race` ok (102s, pgtestcontainer)
- `queue/mysql`: `MYSQL_TEST_DSN=… -race -count=2` ok (5.6s, live MariaDB)
- `queue` (contract): ok
- Gates: api-stability meta-tests, check-changelog-symbols,
  check-error-taxonomy, check-module-layers (Layer 1), cqrs-lint catalog
  coverage, doc-check, check-file-size (queue slice), gofmt — all green.

## c) Flake kills (suite determinism, the 14-46 §e2 discipline)

1. `pinExpiryReclaim` had a 40ms lease racing its pre-expiry probe —
   passed by timing luck until this session's `-race` run. Now 500ms
   lease + 650ms expiry wait (forced clock gap both sides).
2. `pinRequeue` had a 50ms delay racing the follow-up `Get`'s
   `NotBefore.Before(now)` assertion — now 500ms + 600ms.
   Same latent class as the heartbeat ms-race fixed 2026-09-14; the
   suite is deterministic now, not lucky.

## d) Interactions with the concurrent ADR-0142 arc

- **Their claimkit MySQL landed MID-SESSION**: while this arc built the
  queue/mysql Store, the substrate arc fixed the claimkit MySQL dialect
  AND wired `queue/mysql`'s Engine surface (engine.go/register.go/
  engine_test.go, claimkit capabilities over the same database) via
  daemon commits. Reconciled: their wiring is included in the final
  verification below (full queue/mysql suite incl. their engine_test
  green with `-race` vs live MariaDB); the merged api golden is
  7413 exports, `TestEvery` green; their budget/doc fixes to my
  ceremony entries were kept as-written.
- **Transient gate failures observed then resolved**: check-module-layers
  and the cqrs-lint catalog were red MID-FLIGHT from their un-swept
  ceremony (`scheduling/engine` uncatalogued — fixed on sight as a
  1-line exclusion; layer/budget entries landed via daemon commits
  between runs). Final state green with both arcs' work in tree.
- **Still red, NOT mine**: `check-file-size` flags metaengine/*engine
  growth (duckdb 429→444, pebble 725→744, pg 416→431, engine 700→704) —
  the substrate arc's pending ratchet tail. Zero queue files flagged.

## e) Not done (deliberately / owner-gated)

- **Tag wave** (`claiming` v4.0.0 + queue family v4.0.0): owner-gated
  per the standing rule; T15 changed finalize signatures pre-release, so
  the whole family should tag in ONE wave (TODO_LIST updated to say so).
- **MySQL integration leg in nix** (`#integration-mysql-*`): the
  MYSQL_TEST_DSN gate is the portability surface; folding it into the
  nix legs belongs with the toolchain-split resolution (their g-1).
- **T18–T23** (metaengine read-adapter, taskmanager consumer, tq
  re-open ADR, PapDashboard eval): unchanged, tracked in TODO_LIST.

---

_Point-in-time snapshot; (e) is TODO_LIST fuel._
