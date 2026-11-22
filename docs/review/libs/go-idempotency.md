# go-idempotency — Utilization Audit · go-cqrs-lite

> Audited 2026-11-22 · Library `github.com/larsartmann/go-idempotency` @ 220c218 (latest local tag v0.6.0) · Consumer go-cqrs-lite @ feec9bc74 · Group G1

**Verdict:** UNDERUSED-MODERATE
**Score:** 2.5/7 capabilities fully leveraged = 36% (low)

The `Store` contract and its sentinel semantics are properly adopted: the
consumer's idempotency middleware is built _on_ the library's interface, and
the deprecated in-repo SQL twin implements it. But the consumer sits on a
split-pin drift (one module at v0.6.0, five still at v0.3.2), and has taken
none of the three releases' worth of capability that arrived after v0.3.2:
`Forever` permanent claims, the `idemtest` consumer test kit, and the
`WithClock` deterministic-clock option — the last of which exists precisely
to fix the wall-clock TTL sleeps this repo still has. The library's own
contract suite has never been run against the repo's SQL store.

## 1. What the library is

An interface-first idempotency SDK: the three-method `Store` contract
(`Seen`, `Record`, `CheckAndRecord` with atomicity requirements),
conflict-classified sentinels (`ErrDuplicate` → HTTP 409 non-retryable,
`ErrInvalidTTL` → 400), a deprecated dev/test-only `MemoryStore`
(`go-idempotency/store.go:150`; removal at v1.0 per
`go-idempotency/ROADMAP.md:42`), permanent claims via `idempotency.Forever`
(v0.4.0, ADR-006 there), the `idemtest` consumer test kit as a nested module
(v0.5.0, ADR-007), an injectable `MemoryStore` clock `WithClock` (v0.6.0),
an 18-invariant `contract.RunTests` suite for any backend, and a stdlib-only
`middleware` subpackage (`NewCommand`, `HTTP`). It deliberately ships no
production backends and points go-cqrs-lite users at the repo's own
`metaengine.DedupStore` capability (`go-idempotency/README.md:53`).
Extracted from this repo by ADR-0065; the in-repo `idempotency/v4` parent
shim was deleted by ADR-0128 (2026-08-14).

## 2. Usage census in go-cqrs-lite (evidence)

Pins (2026-11-22):

| Module                              | Pin        | Kind                          |
| ----------------------------------- | ---------- | ----------------------------- |
| `middleware/v4/go.mod:17`           | **v0.6.0** | direct (the only current pin) |
| `idempotency/sqlstore/v4/go.mod:10` | v0.3.2     | direct                        |
| `example/taskmanager/go.mod:30`     | v0.3.2     | direct                        |
| `integration/v4/go.mod:32`          | v0.3.2     | direct                        |
| `commandlifecycle/v4/go.mod:35`     | v0.3.2     | indirect (via middleware)     |
| `system/v4/go.mod:65`               | v0.3.2     | indirect                      |

Production import sites (3 modules):

- `middleware/v4/idempotency.go:9` — the CQRS wrapper
  `NewIdempotency`/`CommandIdempotency`/`EventIdempotency`/`QueryIdempotency`
  is built on the contract: `store.CheckAndRecord(ctx, key, ttl)` at
  `middleware/v4/idempotency.go:36`, duplicate pass-through via
  `errors.Is(err, idempotency.ErrDuplicate)` at `:37`, and fail-closed
  wrapping of store failures as Transient at `:42-47` — exactly the
  doctrine the library README's "pitfalls" section prescribes.
- `idempotency/sqlstore/v4/store.go:25` — the deprecated (ADR-0158/0167,
  v5-removal) SQL adapter implements `idempotency.Store` with the
  documented conditional-upsert `CheckAndRecord`; `engine.go:9` aliases the
  import for the `NewFromEngine` facade over `metaengine.DedupStore`.
- `example/taskmanager/setup.go:15,121` — the golden-path example wires
  `middleware.CommandIdempotency(idempotency.NewMemoryStore(0),
  idempotencyTTL, nil)` with a 10-minute TTL.

Test-only usage: `middleware/v4` tests build stores via
`idempotency.NewMemoryStore` (`middleware/v4/test_helpers_test.go:18-25`,
`idempotency_test.go`, `idempotency_ttl_test.go`);
`integration/v4/idempotency_contract_test.go:13-19` cross-checks the
middleware+MemoryStore pipeline against the sqlstore.

In-repo twin status: `idempotency/` contains **only** the deprecated
`sqlstore/v4` adapter; the parent module and `kvstore` were deleted
(ADR-0128; kvstore at zero consumers 2026-10-17 per ADR-0167). The
consumer's canonical dedup runtime is the engine capability
(`metaengine.DedupStore`, ADR-0142; `system.Dedup()` facade per ADR-0167) —
a parallel path the library itself endorses. `dedup/v4` (the bounded replay
ring, `dedup/v4/ring.go:1-12`) is a different concern (stream-boundary
overlap), not duplication.

## 3. Capability inventory

| # | Capability                                                                  | Grade    | Evidence / reason                                                                                                                                                                                                                                       |
| - | --------------------------------------------------------------------------- | -------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1 | `Store` contract (`Seen`/`Record`/`CheckAndRecord`)                         | **Full** | consumed at `middleware/v4/idempotency.go:36`, implemented by `idempotency/sqlstore/v4/store.go`                                                                                                                                                        |
| 2 | Sentinel semantics (`ErrDuplicate`, `ErrInvalidTTL`) + fail-closed doctrine | **Full** | `middleware/v4/idempotency.go:37,42-47`; error table honored end to end                                                                                                                                                                                 |
| 3 | `MemoryStore` (deprecated dev/test store)                                   | Partial  | correct in tests (`middleware/v4/test_helpers_test.go:18`); but the fleet-template example uses it as its _only_ production store (`example/taskmanager/setup.go:121`) while the library marks it dev/test-only (`go-idempotency/store.go:150`)         |
| 4 | `contract.RunTests` invariant suite                                         | Missed   | never imported; sqlstore's invariants are pinned only by hand-rolled tests (`idempotency/sqlstore/v4/property_test.go`, `integration/v4/idempotency_contract_test.go:13`) — the 18-invariant suite incl. the 200-goroutine atomicity drill runs nowhere |
| 5 | `idemtest` kit (drills, spies, failure injection)                           | Missed   | 0 imports; middleware tests hand-roll helpers and counters                                                                                                                                                                                              |
| 6 | `Forever` permanent claims (v0.4.0)                                         | Missed   | 0 uses; `EventIdempotency` consumers get TTL-windowed dedupe only, though permanent claims are the library's stated answer for consumer/event dedupe (`go-idempotency/CHANGELOG.md` [0.4.0], ADR-006 there)                                             |
| 7 | `WithClock` deterministic clocks (v0.6.0)                                   | Missed   | 0 uses; `middleware/v4/idempotency_ttl_test.go:57` still `time.Sleep(wait)`s past the TTL — the exact pattern v0.6.0 was released to eliminate                                                                                                          |
| 8 | `middleware` subpackage (`NewCommand`, `HTTP`)                              | N/A      | the consumer _is_ the CQRS middleware layer; its `middleware/v4/idempotency.go` is the sanctioned CQRS-shaped twin the library defers to                                                                                                                |
| 9 | Production backend recipes (Redis/PG/DynamoDB)                              | N/A      | consumer backends route through `metaengine.DedupStore` per ADR-0142/0158 — the engine-first divergence the library's own README endorses                                                                                                               |

Score derivation: rows N=9, N/A rows 8 and 9 → **A=7**. Full=2 (rows 1, 2),
Partial=1 (row 3 → 0.5), Missed rows 4, 5, 6, 7 → 0.
Score = round(100 × 2.5 / 7) = **36% → low band**.

## 4. Gap analysis

1. **Split-pin drift is real but cosmetic — and hides missed capability.**
   Why the split: `middleware/v4` reached v0.6.0 only as an MVS pull during
   an unrelated go-branded-id tidy sweep (commit f8e1c6933, 2026-11-20: "lets
   MVS pull system/v4 + systemtest/v4 up to the same pin. No source
   changes."). No v0.4.0-v0.6.0 feature is used anywhere —
   `rg 'idempotency\.Forever|WithClock|idemtest' --type go` returns nothing
   for this library's symbols. MVS masks the split in workspace builds; each
   stale module's standalone build still resolves v0.3.2 semantics as its
   minimum. The sweep to v0.6.0 is mechanical and unlocks gaps 2-4 below.

2. **The contract suite has never certified the repo's own store.**
   `idempotency/sqlstore/v4` implements `idempotency.Store` but is verified
   by bespoke tests only. `contract.RunTests` (plus its `Forever` invariants
   once pinned ≥ v0.4.0) is one call:

   ```go
   // skip-validate
   func TestSQLStoreSatisfiesContract(t *testing.T) {
       contract.RunTests(t, func(tb testing.TB) idempotency.Store {
           return newSQLiteStoreForTest(tb)
       })
   }
   ```

   Until v5 deletes the module (ADR-0158), this is the cheapest
   exactly-once insurance in the repo.

3. **TTL tests sleep; the pinned release exists to stop that.**
   `middleware/v4/idempotency_ttl_test.go:22-28` sizes a `(ttl, wait)` pair
   to survive `-race` scheduling and `:57` sleeps it out. With v0.6.0:

   ```go
   // skip-validate
   clock := &fakeClock{}
   store := idempotency.NewMemoryStore(0, idempotency.WithClock(clock.Now))
   // first dispatch passes, duplicate rejected...
   clock.advance(2 * ttl) // ...then the key is reclaimable, instantly
   ```

4. **Event dedupe is TTL-windowed everywhere.** `EventIdempotency` takes a
   TTL and no caller passes `idempotency.Forever`, so a redelivery after the
   window re-processes — the corruption class the library says permanent
   claims exist for. One line at a webhook/sink call site flips it:

   ```go
   // skip-validate
   mw := middleware.EventIdempotency(store, idempotency.Forever, nil)
   ```

5. **The example teaches a deprecated production posture.** The
   golden-path app's only dedup store is `MemoryStore`
   (`example/taskmanager/setup.go:121`), which the library documents as
   dev/testing-only and will remove at v1.0 (`go-idempotency/ROADMAP.md:42`).
   Fleet apps copy the example. Either comment it as demo-scoped or wire the
   engine-backed path the ADR-0167 facade will make one call.

6. **Stale references to the deleted twin.**
   `storage/pebble/v4/adapter_test.go:560` still credits "idempotency.KVStore
   support" (module deleted 2026-10-17), and `versions.json` still lists
   `idempotency` (v4.4.0) and `idempotency/kvstore` (v4.3.3) entries for
   modules that no longer exist on disk.

## 5. Version currency

- Latest local tag **v0.6.0** (2026-11-17). Pinned: v0.6.0 in 1 go.mod
  (middleware), **v0.3.2 in 5** → SPLIT-PIN DRIFT, mostly behind.
- What v0.3.2 → v0.6.0 changed (all additive, `go-idempotency/CHANGELOG.md`):
  v0.4.0 `Forever` permanent claims + corrected backend-adapter recipes;
  v0.5.0 the `idemtest` kit as nested module; v0.6.0 `WithClock`.
  [Unreleased] adds `idemtest.RequireAllowsAfterExpiryPinned` + `PinnedClock`
  — release intent: deterministic TTL drills are the active direction.
- None of the consumer's 6 pins is _broken_ by the gap (the API the consumer
  touches is stable across v0.3.2-v0.6.0), but the drift means two fleet
  behaviors coexist: middleware-compiled binaries already run v0.6.0
  semantics; standalone sqlstore/taskmanager builds declare v0.3.2 minimums.

## 6. The fleet argument

"I have MANY projects and limited time; if every project re-implements or
reintegrates everything I will only fix bugs all day long." Exactly-once
logic is the highest-bug-cost, lowest-visibility kind of code — the library
exists so the fleet never re-proves atomicity, TTL reclaim, or fail-closed
behavior. This repo already absorbed that argument at the contract level;
it has not absorbed it at the verification level: hand-rolled property
tests and wall-clock sleeps re-derive what `contract.RunTests` and
`idemtest` prove continuously upstream, and any future invariant fix (the
[Unreleased] drill tightening is a live example) will not reach tests that
don't run the suite.

## 7. Ranked opportunities

1. **Sweep the five v0.3.2 pins to v0.6.0** (high, mechanical): per-module
   `go mod tidy`; one commit; closes the drift and unlocks 2-4.
2. **Run `contract.RunTests` against `idempotency/sqlstore/v4`** (high,
   small): the store stays until v5; certify it with the library's
   18-invariant suite while it does.
3. **Adopt `idemtest` drills + `WithClock` in `middleware/v4` tests**
   (medium): `RequireExactlyOnce`/`RequireOneWinner`/`RequireFailClosed`
   replace hand-rolled counters; `WithClock` deletes the TTL sleeps
   (`middleware/v4/idempotency_ttl_test.go:57`).
4. **Offer `Forever` where events are deduped** (medium): document or wire
   permanent claims for webhook/sink-style `EventIdempotency` consumers.
5. **Fix the example's store posture** (medium): comment or replace
   `example/taskmanager/setup.go:121`'s MemoryStore with the engine-backed
   path per ADR-0167.
6. **Clean stale twin references** (trivial):
   `storage/pebble/v4/adapter_test.go:560` comment; `versions.json` rows for
   the deleted `idempotency`/`idempotency/kvstore` modules.

## 8. Reconsider when

- The v5 cut deletes `idempotency/sqlstore/v4` (ADR-0158): the contract
  suite's local run retires with it; `idemtest` drills over `sys.Dedup()`
  become the successor verification.
- `system.Dedup()` (ADR-0167) ships: the example can demonstrate
  engine-backed dedup in one line — opportunity 5 flips from comment to
  rewrite.
- A fleet app adds an HTTP ingress with `Idempotency-Key` headers: the
  library's `middleware.HTTP` (409/400/503 contract, `idemtest` HTTP drills)
  flips from N/A to the reference implementation for that app.
- `MemoryStore` removal lands at go-idempotency v1.0: the example and all
  test helpers must move to a real backend that day.

## 9. How to verify

```
cd /home/lars/projects/go-cqrs-lite
rg -n 'go-idempotency' -g go.mod                      # 6 pins: 1× v0.6.0, 5× v0.3.2
rg -l 'larsartmann/go-idempotency' --type go -g '!*_test.go'   # middleware, sqlstore, example
sed -n '36,47p' middleware/v4/idempotency.go          # CheckAndRecord + ErrDuplicate + fail-closed wrap
sed -n '121p' example/taskmanager/setup.go            # NewMemoryStore(0) as the example's prod store
sed -n '57p' middleware/v4/idempotency_ttl_test.go    # time.Sleep(wait)
rg -n 'idempotency\.Forever|idemtest|idempotency\.WithClock' --type go   # 0 hits for this lib's symbols
rg -n 'contract' idempotency/sqlstore/v4/*_test.go | head -3   # no contract.RunTests import
sed -n '560p' storage/pebble/v4/adapter_test.go       # stale KVStore comment
python3 -c "import json;d=json.load(open('versions.json'));print([k for k in d['modules'] if k.startswith('idempotency')])"
git -C /home/lars/projects/go-idempotency log -1 --format='%H'
git -C /home/lars/projects/go-idempotency tag --list | sort -V | tail -1   # v0.6.0
```

Sibling reports in this directory: `go-error-family.md`, `go-codec.md`,
`go-retry.md`.
