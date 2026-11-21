# Evidence: the `idempotency` fleet wiring survey

> **Date:** 2026-11-17 (surveyed) · 2026-11-18 (verified against each checkout's source and committed here)
> **Status:** FINAL — input to decisions **D28** (swap timing) and **D29** (dedup durability default), and
> to the plan [`2026-11-18_06-41_SUPERB-idempotency-fleet-repair-and-linter-follow-through.md`](../planning/2026-11-18_06-41_SUPERB-idempotency-fleet-repair-and-linter-follow-through.md).
> **Method:** every row below was read out of the checkout's own source and `go.mod` (never inferred from a
> report). A second pass re-checked the first pass's claims line by line; **§7 records the corrections it
> forced**, because the first pass had four wrong mechanism claims. Re-verify before acting on any row:
> these are working trees, not tags.
> **Scope:** all ten `~/projects` checkouts that touch the `idempotency` module family (plus the two
> `DiscordSync` clones, which are one logical consumer).
> **Corrections (2026-11-21):** §2.1's claim that `idempotency/sqlstore` constructors run a background
> sweep goroutine is FALSE (dated note at §2.1); §7.5's comment verdict and §7.6's collection name are
> amended accordingly.

## 1. What is actually being surveyed

Three module identities share a name and are not interchangeable:

| Module path                                                        | Source at HEAD                                                                                                                   | Fleet consumers | Pinned                              |
| ------------------------------------------------------------------ | -------------------------------------------------------------------------------------------------------------------------------- | --------------- | ----------------------------------- |
| `github.com/larsartmann/go-cqrs-lite/idempotency/v4` (parent shim) | **DELETED** 2026-08-15 (`875bb689b`, ADR-0128) — tags v4.0.0–v4.4.0 still proxy-serve a frozen alias shell over `go-idempotency` | 8 checkouts     | all `v4.4.0`                        |
| `…/idempotency/kvstore/v4`                                         | **DELETED** 2026-10-17 at zero fleet consumers                                                                                   | 0               | tags v4.0.0–v4.3.3 proxy-resolvable |
| `…/idempotency/sqlstore/v4`                                        | present, **DEPRECATED** toward v5.0.0 (ADR-0158/0167) — no v5 successor                                                          | 3 checkouts     | all `v4.4.3`                        |

The sanctioned replacements: `github.com/larsartmann/go-idempotency` (the extracted module, v0.3.2 across
the fleet except one indirect v0.3.1) and, for durable dedup, `metaengine.DedupStore` reached through
`(*system.System).Dedup()`.

This is why "the import resolves" is not evidence of anything: the proxy keeps serving a module whose
source no longer exists. That is also the failure class `cqrs-lint` F030 now detects.

## 2. The wiring matrix

Legend: **shim** = imports the deleted parent; **direct** = imports `go-idempotency`; **sqlstore** =
imports the deprecated satellite. "Restart" = does a restart forget the dedup keys.

| #   | Checkout (HEAD)                                                          | Imports                        | Store                                                                                                                                                                                                                                                                                                                         | TTL                                         | Sweeper                                                                                                                                               | Restart                                                                                                                                                    | What is actually wrong                                                                                                                                                                                                                              |
| --- | ------------------------------------------------------------------------ | ------------------------------ | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | `InboxClean` (`1a9d257`)                                                 | shim                           | `idempotency.MemoryStore`                                                                                                                                                                                                                                                                                                     | 24h keys / **5m** ticker                    | own goroutine on `context.Background()` (not tied to the bus ctx)                                                                                     | **no** — raw SQLite rows dedup on ingest only, replay after restart works                                                                                  | `Setup.Close` closes only the bus → the sweep goroutine and ticker leak; dead imports `idempotency.Store` + `idempotency.ErrDuplicate`. Journal state lives in a SQLite file, so durable dedup keys would be semantically sound here                |
| 2   | `browser-history` (`40a4166`)                                            | shim + sqlstore                | memory **by default**, sqlite when a DB path is configured                                                                                                                                                                                                                                                                    | 10m                                         | **none on either path**                                                                                                                               | sqlite yes / memory no                                                                                                                                     | two parallel stores with divergent semantics; `idempotency.Store` interface asserted against a `MemoryStore`; `sqlite.Store(maxEntries)` silently deletes the OLDEST keys when full (env-tunable limit, "set higher for long-running servers")      |
| 3   | `browser-history-precheck` (`6a5824d`)                                   | shim + sqlstore                | an unconditional `MemoryStore` reference passed into config, discarded/replaced when the DB exists                                                                                                                                                                                                                            | 10m                                         | **none**                                                                                                                                              | sqlite yes / memory no                                                                                                                                     | the discarded `MemoryStore` still owns a 10m ticker; `go-idempotency` resolves **indirectly at v0.3.1** — the only fleet checkout off v0.3.2                                                                                                        |
| 4   | `games/KeyCountdown` (`4a38371`)                                         | shim + sqlstore                | sqlite                                                                                                                                                                                                                                                                                                                        | 10m                                         | **none**                                                                                                                                              | yes                                                                                                                                                        | ordering is actually **correct** here (`CommandIdempotency` outermost, then Retry) — see §7; the real gap is the missing sweeper, so the table grows forever                                                                                        |
| 5   | `DiscordSync` (`f8e7a20a`)                                               | **direct** + sqlstore          | sqlite                                                                                                                                                                                                                                                                                                                        | 24h (command); event keys use the store TTL | **yes** — `cmd/discordsync/idempotency_sweeper.go` (`NewIdempotencySweeper`, `idempotencySweepInterval` = 1h, immediate first pass, logs, `nil`-safe) | yes                                                                                                                                                        | capture is a non-atomic `Seen` + `Record` pair: two workers can both pass the peek and both record. Suppressed by construction (not by an atomic claim) for retries — acceptable for Discord noise, wrong for anything that must dedup exactly once |
| 6   | `games/SEC` (`81b7021b`)                                                 | shim                           | `idempotency.MemoryStore`                                                                                                                                                                                                                                                                                                     | 30s commands / 1s events / 30s default      | 1m ticker, its own goroutine                                                                                                                          | **no** (keys) — but the **journal is a persistent SQLite store** (`cqrskv`, default `data/sec.db`), so durable dedup would NOT outlive the protected state | `idempotencyTTL` default 30s looks like it could starve multi-minute checkout idempotency — review, do not assume                                                                                                                                   |
| 7   | `github-local-sync` (`75d2668`)                                          | shim                           | `idempotency.MemoryStore`                                                                                                                                                                                                                                                                                                     | 30m                                         | ticker from the same ctor                                                                                                                             | no                                                                                                                                                         | **middleware order is inverted**: `Use(Recovery, Retry, Idempotency…)` makes Retry OUTSIDE Idempotency, so attempt 2 re-enters the dedup check and fails `ErrDuplicate` — the retry is defeated by the dedup it sits outside of                     |
| 8   | `Zlota44` (`7baa086`)                                                    | shim (`ErrDuplicate` only)     | a **hand-rolled 172-line SQLite-backed `IdempotencyStore`** (`database/sql`, same `*sql.DB` as the event store, implementing `go-idempotency`'s Store directly — 2026-11-21 correction: originally recorded as a 157-line bbolt store, which was wrong); uses the shim's `ErrDuplicate` sentinel for cross-repo compatibility | 24h                                         | **none** — GC only runs lazily inside `CheckAndRecord`                                                                                                | yes                                                                                                                                                        | the shim's `Store` interface and `ErrDuplicate` are what ties this repo to a deleted module; no sweeper means the bbolt bucket only shrinks when that specific repo happens to be used; the repo **vendors** dependencies                           |
| 9   | `games/SwettySwipperWeb` (`81e6d35e`)                                    | shim + `metaengine.DedupStore` | engine collection `api_command_idempotency` (originally recorded as `wanted_ids` — wrong, see §7.6) via a local adapter                                                                                                                                                                                                       | 10m                                         | **none** — `DedupSweep` has zero callers                                                                                                              | **yes** (engine-backed)                                                                                                                                    | a hand-written adapter over `DedupStore` lives beside the deprecated sqlstore path; a dangling `replace` sits in `services/api/go.mod`; the durable half is a real product choice only Swetty has already made                                      |
| 10a | `DiscordSync` (**canonical**, `f8e7a20a`, 0 ahead / 0 behind its remote) | as row 5                       | —                                                                                                                                                                                                                                                                                                                             | —                                           | —                                                                                                                                                     | —                                                                                                                                                          | see D31                                                                                                                                                                                                                                             |
| 10b | `discordsync` (twin)                                                     | —                              | —                                                                                                                                                                                                                                                                                                                             | —                                           | —                                                                                                                                                     | —                                                                                                                                                          | **diverged from its own remote** (2 ahead / 5 behind upstream, 3 ahead of local); treat as the stale copy pending D31                                                                                                                               |

> **2026-11-21 correction, row 8 + §4 + §7.3:** Zlota44's store is SQLite, not bbolt — no `bbolt`
> matches the repo's entire git history. The false comment is the store's own "row growth is bounded
> by distinct command keys (~one per KW per command type), so no background sweeper goroutine is
> needed" — falsified by `ReDiscoverPropertyCmd`
> (`domain/property/commands.go:138`), whose key embeds a unique per-command ID
> (`"rediscover-"+KW+"-"+ID()`): the key space is unbounded and expired rows are reclaimed only by a
> `Seen` of the SAME key, which unique keys never get.

### 2.1 Library-side facts that constrain every repair

- `go-idempotency` `MemoryStore` is the **only** implementation with built-in expired-key cleanup, and it
  deletes **oldest-first blindly** — it does NOT check whether the entry is actually expired. Handing it a
  sweep interval smaller than the TTL is therefore destructive: live keys are evicted. Every fleet memory
  store must set `sweepInterval >= ttl`. (Recorded so the sweeper port in `C24` cannot repeat it.)
- `go-idempotency` `MemoryStore` opens its own ticker goroutine on construction, so an unused or discarded
  store is a live goroutine + ticker. `Close()` is the only way to stop it.
- `idempotency/sqlstore` `NewSQLiteStore`/`NewPostgresStore`/`NewMySQLStore` already run a background sweep
  goroutine when the TTL is non-zero. browser-history, precheck, KeyCountdown, and Zlota44 therefore need no
  new sweeper on the **sqlite** paths — they need to stop reaching for the memory path, or to prove the
  sqlite sweep is actually enabled. This narrows §f items 10, 12–14, and 31 considerably.

> **CORRECTED 2026-11-21 — the bullet above is FALSE.** `idempotency/sqlstore` runs NO background sweep
> goroutine on any constructor, at any TTL: `Store.Sweep` (`idempotency/sqlstore/v4/store.go:312`,
> Deprecated) is an optional, CALLER-invoked method — "Call periodically to bound table growth, or rely
> on lazy deletion in `Store.Seen` and `Store.CheckAndRecord`". Expiry reclaims lazily on reads only, so
> an idle table grows without bound. Consequences: browser-history, precheck, and KeyCountdown on the
> sqlite paths DO need a real sweeper (that work rides the D28/D29-gated bundles C18–C20); §7.5's
> "native TTL comment is ACCURATE" verdict falls with it (both comments lie). Truth of record: the
> plan's C24 preamble (2026-11-20, commit `92a4dcc2f`), re-verified in source this date.

- `metaengine.DedupStore` exposes `DedupCheckAndRecord`, `DedupSeen`, `DedupSweep`; the sweep is explicit.

## 3. Q1 — does restart-forgetting matter? (feeds D29)

Split the fleet by what breaks on restart, not by store type:

| Class                                                   | Checkouts                                                                                     | Verdict                                                                                                                                                      |
| ------------------------------------------------------- | --------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Restart bypasses dedup AND the protected state persists | `browser-history`, `browser-history-precheck`, `games/SEC`                                    | **bug** — the protected journal/rows survive, the dedup keys do not, so the exact double-submit the store exists to stop becomes possible after every deploy |
| Restart resets dedup AND the protected state resets too | `InboxClean` (replay-after-restart is a feature) , `github-local-sync` (branch index rebuild) | **ambiguous** — durable keys could be correct or actively wrong; the owner's replay intent decides (open question 3 of the 2026-11-17 status report)         |
| Already durable, retention is the only gap              | `KeyCountdown`, `Zlota44`, `SwettySwipperWeb`, `DiscordSync`                                  | **no bug** in durability; broken retention (§4)                                                                                                              |

Note that `games/SEC` is the opposite of the earlier guess: its **event journal is persistent**, so a
durable dedup key would outlive nothing — the memory store is the thing that expires while the state it
protects does not.

## 4. Q2 — which tables grow without bound? (feeds C24)

| Store                             | Retention today               | Growth                                                                     |
| --------------------------------- | ----------------------------- | -------------------------------------------------------------------------- |
| `games/SEC` memory                | 1m ticker                     | bounded                                                                    |
| `InboxClean` memory               | 5m ticker                     | bounded                                                                    |
| `github-local-sync` memory        | ctor ticker                   | bounded                                                                    |
| `browser-history` memory          | 10m ticker                    | bounded                                                                    |
| `browser-history` sqlite          | **no caller of the sweep**    | **unbounded**                                                              |
| `browser-history-precheck` sqlite | **no caller of the sweep**    | **unbounded** (+ a leaked memory ticker)                                   |
| `games/KeyCountdown` sqlite       | **no caller of the sweep**    | **unbounded**                                                              |
| `Zlota44` sqlite (hand-rolled)    | lazy, only when used          | **unbounded** — `ReDiscover` keys embed unique IDs (2026-11-21 correction) |
| `SwettySwipperWeb` engine         | **no caller of `DedupSweep`** | **unbounded**                                                              |
| `DiscordSync` sqlite              | 1h ticker, 24h keys           | bounded (the reference shape)                                              |

> **2026-11-21:** the four "no caller of the sweep" sqlite rows above were written under the false §2.1
> auto-sweep premise; their verdicts STAND and strengthen — there is no sweep to call AND no background
> sweeper exists at all. Lazy read-path reclamation is the only bound, so an idle table grows forever.

## 5. Q3 — what does each swap actually touch?

"Pure swap" = replace the deleted parent with `go-idempotency` and nothing else changes. Every symbol these
four use (`Store`, `MemoryStore`, `NewMemoryStore`, `ErrDuplicate`, `CheckAndRecord`) is exported by
`go-idempotency` v0.3.2; MVS already resolves that version and `go mod tidy` already keeps the indirect
require, so the swap is import-path-only.

| Repo                       | Pure swap?                                                                                  | Files                                                                                                                |
| -------------------------- | ------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------- |
| `InboxClean`               | **yes** (plus the `Close` leak in C23)                                                      | `internal/emailcqrs/setup.go`                                                                                        |
| `games/SEC`                | **yes**                                                                                     | `internal/cqrs/app/app.go` (+ `internal/cqrs/app/idempotency_test.go`, which re-exposes the package path)            |
| `github-local-sync`        | **yes** (plus the ordering bug in C23)                                                      | `internal/branch/service.go`                                                                                         |
| `games/SwettySwipperWeb`   | **yes** (plus the dangling `replace`)                                                       | `services/api/dispatcher.go`, `services/api/server.go`, `services/api/idempotency_integration_test.go`               |
| `games/KeyCountdown`       | no — the sqlstore wiring also changes                                                       | `internal/container/packages/cqrs.go`, `internal/cqrs/integration_test.go`, `go.mod`                                 |
| `browser-history`          | no — dual store branch must be collapsed                                                    | `api/server.go`, `api/server_setup.go`, `api/idempotency_test.go`, `api/go.mod`, `cmd/browser-history-server/go.mod` |
| `browser-history-precheck` | no — same branch plus the discarded store                                                   | `api/server.go`, `api/server_setup.go`, `api/idempotency_test.go`, `api/go.mod`, `cmd/browser-history-server/go.mod` |
| `Zlota44`                  | no — retire the hand-rolled bbolt store; vendored tree needs `go mod tidy && go mod vendor` | `internal/app/wire.go`, `internal/app/dispatcher.go`, `cmd/discover/sweep.go`, `go.mod`, `vendor/`                   |
| `DiscordSync`              | no — already direct; sqlstore retirement only                                               | `cmd/discordsync/init.go`, `go.mod`, plus the sweeper's store type                                                   |

## 6. Receipts

| Claim                       | Evidence                                                                                                                                                                            |
| --------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| parent module source gone   | `git log --diff-filter=D -- idempotency/go.mod` → `875bb689b` (2026-08-15)                                                                                                          |
| tags still resolve          | `git for-each-ref refs/tags/idempotency/v4.4.0` → 2026-08-12 (predates deletion)                                                                                                    |
| kvstore gone                | `idempotency/` at HEAD contains only `sqlstore/`                                                                                                                                    |
| 7 parent-shim consumers     | `project-dependency-graph who-uses …/idempotency --direct-only=false` (2026-11-17)                                                                                                  |
| 3 sqlstore consumers        | same tool for `…/idempotency/sqlstore`; all on `v4.4.3`                                                                                                                             |
| the false MISSING rows      | `cqrs-lint scorecard` on SEC / InboxClean / github-local-sync said `Idempotency: missing` before the catalog-hint fix; Zlota44 (direct `go-idempotency`) said `used` as the control |
| F030 silence before the fix | `cqrs-lint` on SEC and KeyCountdown produced no deleted-module / sqlstore finding                                                                                                   |

## 7. Corrections this verification pass forced

The first pass (chat-only, 2026-11-17) had these wrong. They are recorded rather than quietly fixed,
because a survey that cannot show its own error rate is not evidence:

1. **Middleware direction.** The first pass claimed the library applies the FIRST middleware as the
   INNERMOST wrapper. False: `dispatcher.Apply` iterates `slices.Backward`, so the first-added middleware is
   the OUTERMOST and runs first. Proven by a probe (`[first-before last-before handler last-after
   first-after]`) and now pinned permanently by
   `command/v4/memory_bus_test.go::TestMemoryBus_MiddlewareOrder_FirstAddedIsOutermost`. The
   **consumer-side conclusion survived** (github-local-sync's Retry is outside Idempotency and dead-ends
   retries) but the two other apps' ordering claims were inverted.
2. **KeyCountdown's ordering is correct**, not inverted, as the first pass asserted. The bug is
   github-local-sync only.
3. **Zlota44's store is bbolt, not SQLite**, and it does not implement the shim's `Store` interface
   structurally — it only reuses `ErrDuplicate`. The retirement is therefore about the sentinel and the
   dependency, not about a drop-in interface swap.

   _(2026-11-21: the backend half is wrong — see the row-8 correction: SQLite via `database/sql`,
   implementing `go-idempotency`'s Store directly (a direct v0.3.2 require beside the shim). The
   "not a drop-in swap for the SHIM's Store" half stands, and the retirement remains about the
   `ErrDuplicate` sentinel + the shim require.)_
4. **SEC's journal is persistent**, not in-memory. Durable dedup keys would be correct there; the earlier
   "keep memory because the journal is in-memory" reasoning was backwards.
5. Two stale comments claimed the absent sweep: browser-history's "native TTL" comment is ACCURATE (sqlstore
   does sweep); the lie there is the `/debug/idempotency` payload. Zlota44's "cloud sync make expensive
   sweeps unnecessary" comment is the inaccurate one.

   _(2026-11-21: half of this item is superseded — sqlstore does NOT sweep (§2.1 correction), so
   browser-history's "native TTL" comment is inaccurate too; BOTH comments lie. C22 fixed the endpoint
   payload accordingly: `SweepEnabled = TTL>0 && !UseSQLiteReadModel`, real `DedupLayers = 2`.)_

6. Swetty's engine collection is named `wanted_ids`, not `meta_dedup`.

   _(2026-11-21: RECONCILED — this correction was itself wrong. `wanted_ids` matches no file and no
   commit in Swetty's entire history (`git log --all -S "wanted_ids"` is empty); the collection is
   `api_command_idempotency`, defined as `dedupCollection` in `services/api/idempotency_dedup.go:17`
   and used by every adapter call. Row 9's `wanted_ids` is the same error. The C24-ungated sweep
   shipped 2026-11-20 (`0c82adeff`) sweeps this collection.)_

The **package doc state was a real defect and is now fixed**: `command/v4/doc.go` (and its v5 mirror
`core/v5/command/doc.go`) said "last added runs first", contradicting `dispatcher.Apply` and the
dispatcher package's own doc. Both now state the true direction and the retry/idempotency consequence.

## 8. What this evidence does and does not decide

- It **does** make D29 answerable per app, and it reduces D28's "17 files / 7 repos" to a precise
  four-repo pure-swap set plus five conditional ones.
- It **does not** decide D28 or D29. Neither the swap timing nor the durability default is a call this
  document can make; both stay on the owner decision sheet.
- It **does not** cover anything outside `idempotency`: no other module was surveyed here.
