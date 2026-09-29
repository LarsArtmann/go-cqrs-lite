# Queue conformance clock seam — design proposal (M4 (c) / M16.1)

> Status: PROPOSAL (design-gated — no implementation until ratified).
> Owner decision required at the end. Harvested from the Queue M4 verification
> tail (TODO_LIST row, 2026-09-21) and the 2026-09-28 post-wave Pareto plan.

## Problem

The queue conformance suite (`queue/conformance`, run per backend: sqlite,
postgres, mysql) contains 10 fixed `time.Sleep` calls (60–650 ms) whose only
job is to let a lease or requeue delay lapse in wall-clock time. Cost:

- ~1.3 s of pure sleeping per backend suite; the suite is the hot path of
  `#test-integration` legs.
- The sleeps are lower-bounds, not guarantees: under scheduler contention a
  650 ms sleep after a 500 ms lease can still race the ms-truncated deadline
  comparison (see `lifecycle.go:247` comment — leases are stamped
  ms-truncated). Every sleep is a latent flake, and CI `-count=2` legs
  (M4 (e) sweep) would multiply exposure.

## Inventory of the sleeps (evidence)

| Site | Wait | What must lapse |
| --- | --- | --- |
| `claims.go:118` | 650 ms | 500 ms lease on crashed worker → reclaim |
| `claims.go:189` | 120 ms | short lease → visibility flip |
| `tokens.go:35` | 60 ms | guard before pre-expiry probe |
| `tokens.go:55` | 60 ms | 30 ms lease → second token mint |
| `tokens.go:89` | 60 ms | 30 ms lease → reclaim |
| `lifecycle.go:229` | 60 ms | 30 ms lease → complete must fail fenced |
| `lifecycle.go:250` | 2 ms | ms-truncation boundary (same-ms heartbeat rewrite) |
| `lifecycle_cancel.go:91` | 60 ms | 30 ms lease → cancel-after-expiry |
| `journal.go:139` | 60 ms | 30 ms lease → MarkOrphaned |
| `retry.go:249` | 600 ms | requeue `retry_in` delay → claimable again |

Key observation: every sleep waits for **store-internal time comparisons**
(lease fencing, requeue `not_before`, token mint ordering) — there is no
background poller to synchronize with.

## Why the seam is clean today

All three store backends compute time **app-side**: `now := time.Now()`
captured at operation start (`queue/mysql/claim.go:135`,
`queue/mysql/lifecycle.go:77` — deliberately captured inside the tx so
backoff counts from commit — `queue/sqlite/enqueue.go:37`, etc.). No
lease-path SQL uses DB-side `NOW()`/`CURRENT_TIMESTAMP`. One API already
takes time explicitly: `MarkOrphaned(ctx, now time.Time)`
(`journal.go:141` exercises it that way).

So the wall-clock dependency is a single, uniform `time.Now()` per operation —
exactly the shape ADR-0122 (`WithClock` — injectable time, irohengine) solved
for CRDT LWW resolution.

## Options

### Option A — harness-only: shrink leases, keep sleeps (status quo minus)

Keep the seam out; tune leases to 5 ms and sleeps to 15 ms. Rejected: still
wall-clock-dependent, still racy under `-race` + load, and ms-truncation makes
sub-10 ms leases MORE fragile (same-ms collisions are the documented trap).

### Option B — internal `nowFn` seam, conformance-only injection (recommended)

Each store gains an unexported field `now func() time.Time` (default
`time.Now`), settable only from within the `queue` module family (a tiny
`conformanceClock` wiring in each backend's test-side constructor or an
internal `queue.SetTestClock` — never exported to consumers). The conformance
`env` holds a manual clock (ADR-0122 pattern: fixed epoch + `Advance(d)`).
Every sleep above becomes `clock.Advance(lease + 1ms)`; the 2 ms boundary sleep
becomes `clock.Advance(1ms)` (deterministic ms rollover).

- Pros: zero public-API change; conformance suites become deterministic;
  sleeps deleted outright; per-op time capture semantics preserved verbatim
  (each operation still stamps "now" at its own start — only the SOURCE of
  "now" changes).
- Cons: `mysql`'s capture-inside-tx comment (`lifecycle.go:77`) must keep its
  position — trivial but must be pinned by a test; the seam is test-only
  plumbing that lives in production files (small, precedented by
  `testcontainers` skip patterns).

### Option C — exported `queue.WithClock` (ADR-0122-faithful)

Mirror irohengine: export `Clock` + `WithClock(clock)` option on the three
store constructors. Additive-only, so v4.x-legal, and it gives consumers
deterministic tests for their own queue usage.

- Pros: one seam shape across the repo (irohengine precedent); consumer-facing
  benefit; no test-only fields in production types.
- Cons: grows the public API surface (api golden + CHANGELOG + skill refs);
  public `Clock` invites "why not also inject scheduling/ event stamps?"
  scope creep that ADR-0122's scope-limit clause explicitly guards against.

## Recommendation

**B now, C deferred to v5.** B deletes all 10 sleeps with zero API growth and
unblocks CI `-count=2` legs safely. Fold C into the v5 capability-interface
work (ADR-0111 (g)) where `Clock` can join a coherent capability story instead
of landing as a lone option. If the owner wants C in v4.x anyway it is a
strict superset of B (same internal seam, plus an exported constructor option).

## Implementation sketch (when ratified, ~½ day)

1. Add `now func() time.Time` to the three stores; replace per-op
   `time.Now()` with `s.now()` (mechanical; sites listed in the inventory's
   evidence line — `mysql` keeps capture inside the tx).
2. Manual clock in `queue/conformance` (`env`), `Advance` replaces sleeps.
3. Pin the ms-truncation semantics with a dedicated test (same-ms heartbeat
   rewrite) so the seam cannot silently change deadline rounding.
4. api golden regen NOT needed under option B; needed under option C.
5. Receipt: queue M4 (c) row strike + CHANGELOG (test-infra bullet, no
   symbols).

## Open question for the owner

Ratify B (internal seam now, `WithClock` at v5) — or prefer C (export
`queue.WithClock` in v4.x) given the irohengine precedent?
