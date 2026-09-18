# Review: How the CV Phase-0/T29 verdicts reflect on go-cqrs-lite (claims verified against source)

- **Date:** 2026-09-16
- **Trigger:** CV's Phase-0 seam-spike ADR + T29 read-model tier verdict + three CV status
  reports (2026-09-15/16) all cite this library's APIs. Every externally-made claim below
  was re-verified against this repo's source before being encoded here
  (verify-external-claims discipline; primary source = current master, build tags
  `goexperiment.jsonv2` implied, no code changed).
- **Sources (CV side):** `docs/adr/2026-09-15_phase0-seam-spike-go.md` (Accepted),
  `docs/research/2026-09-15_t29-read-model-tier-verdict.md` (with the four-tier benchmark
  addendum), status reports `2026-09-16_19-22` / `18-49` / `18-43` in `~/projects/CV/docs/status/`.

## 1. The headline reflection

CV ran the most rigorous consumer evaluation of metaengine/storage to date — a
parity-gated, four-tier comparative benchmark over a real 6.2k-event corpus — and the
outcome is **net positive for the library's core thesis and net negative for none of it**:

- The **write-path/event-system seam won**: 1041/1041 application streams fold to
  identical state through the library store path (sha256 over full domain structs),
  cross-era JSON+CBOR upcasting worked with zero failures, `system.New` boots clean with
  the sqlite engine. The migration's value is the write-path consolidation — exactly what
  the library sells.
- The **read-path loss is a designed trade, not a defect**: metaengine lost the read-tier
  comparison to CV's in-memory fold (list 7.9 ms vs 0.746 ms at 1k apps; board 38.5 ms vs
  0.9 ms). But the winning "tier" was itself a decider fold — the library's own
  architectural pattern. The fold-as-read-model IS a go-cqrs-lite shape. No API owes CV
  an apology here; the verdict keeps the tracker over the library store.

## 2. Claim-by-claim verification table

| # | CV-side claim                                                                                                                                                        | Verdict                                                                                                                                                     | Evidence (this repo, current master)                                                                                                                                                                                                     |
| - | -------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1 | No lease/single-writer primitive for engines/stores; coexistence needs a CV-owned `RegisterDriver` decorator                                                         | **Confirmed** (word-level grep: only `queue/` has lease semantics — task claims, a different concept)                                                       | `metaengine/registry.go:61` `RegisterDriver` exists as the seam; no engine/store-level lock anywhere                                                                                                                                     |
| 2 | `DomainConfig.Events` universe declaration "does not exist at system/v4 v4.7.0"                                                                                      | **Confirmed AND sharpened**: `system/v4.7.0` is the LATEST released system tag; `Events` exists on master but is **unreleased**                             | `system/config_types.go:57` + `system/coeffect_gate.go`; `git tag --contains <intro-commit>` → no system tag                                                                                                                             |
| 3 | metaengine `Scan` silently defaults to a 100-row limit                                                                                                               | **Confirmed — and worse: the doc comment lies.** `Scan` says "returns all values matching…" while the default truncates at 100                              | `metaengine/typed_reader_scan.go:15` (`scanConfig{limit: 100}`), same at `typed_reader_cursor.go:25`, `explain.go:42`; zero mention in skill `readmodels.md`                                                                             |
| 4 | `ApplyBatch` wraps no transaction (per-event commits; 3.4 s → 109 ms with `synchronous=NORMAL`)                                                                      | **Confirmed at API level**; atomicity+batching already designed for v5 and spike-validated in-repo                                                          | `metaengine/store.go:429-442` (per-event `applyWithRecord` loop); `metaengine/spike_batch_atomicity_test.go` (3 approaches validated); ADR-0123 §10 (batch boundary = the event); `Store.InTransaction` exists for Transactional engines |
| 5 | No LIKE/contains pushdown (`FilterOp` = eq/ne/lt/le/gt/ge/in)                                                                                                        | **Confirmed**                                                                                                                                               | `metaengine/enum_validation.go:71-75` — exactly those 7 ops; search degrades to client-side scan                                                                                                                                         |
| 6 | storage/v4 AutoMapper cannot scan TEXT back into `time.Time` under modernc.org/sqlite                                                                                | **Structurally confirmed** (CV measured the failure): `time.Time` maps to TEXT and the raw `*time.Time` field is passed as scan dest with no format adapter | `storage/view/auto.go:34` (mapping table), `:147` (`fv.Addr().Interface()` dest), `:193-194`; package is v5-removed anyway (ADR-0123)                                                                                                    |
| 7 | SQLViewStore removed in v5 (ADR-0123)                                                                                                                                | **Confirmed** (known/intentional)                                                                                                                           | `storage/view_aliases.go` — every symbol `Deprecated: removed in v5 (ADR-0123)`                                                                                                                                                          |
| 8 | `sqlstore`/`kvstore` cannot represent "forever"; `expiryFromTTL` gates `ttl <= 0 → ErrInvalidTTL`; helper duplicated verbatim; `expires_at NOT NULL` in all dialects | **Confirmed**                                                                                                                                               | `idempotency/kvstore/store.go:46-52` and `idempotency/sqlstore/store.go:173-179` (byte-identical bodies); `sqlstore/store.go:55,69,83` (INTEGER/BIGINT NOT NULL ×3 dialects)                                                             |
| 9 | (Implied) adapters pin go-idempotency v0.3.0                                                                                                                         | **Confirmed**                                                                                                                                               | `middleware/go.mod:16`, `idempotency/sqlstore/go.mod:9`, `idempotency/kvstore/go.mod:8` — all v0.3.0                                                                                                                                     |

## 3. New findings this session produced (beyond verifying CV's claims)

### 3.1 The `UnixNano()` overflow landmine is real and pins the Forever design (executed probe)

Probe (stdlib only, replicating the adapters' exact expression
`time.Now().Add(ttl).UnixNano()`; source embedded in §3.1b so this review is
self-contained):

| TTL                                    | Expiry year | `UnixNano()`           | Stored as live?          |
| -------------------------------------- | ----------- | ---------------------- | ------------------------ |
| 100 years (smart-configs' default)     | 2126        | +4.94e18               | yes — safe               |
| 236 years                              | 2262        | **-9.21e18 (wrapped)** | **no — dead on arrival** |
| `time.Duration(math.MaxInt64)` (~292y) | 2318        | -7.43e18 (wrapped)     | no                       |

Mechanism, now verified rather than asserted: `time.Time.Add` itself survives past year
2262 (internal seconds+nanos split), the wrap is in `.UnixNano()`. Any TTL pushing the
absolute expiry beyond **2262-04-11 23:47:16 UTC** (MaxInt64 ns) stores a negative
`expires_at`; `WHERE expires_at < now` then treats the key as already expired and the
sweep deletes it. Consequences:

- Huge-TTL approximations of "forever" are not merely untidy — past ~236 years they are
  **silent no-op idempotency**, the exact unrecoverable failure direction the
  go-idempotency philosophy forbids.
- The CV verdict's fix ordering — "map `Forever` → `math.MaxInt64` **before** any
  `now.Add(ttl)` arithmetic" (write `expires_at = math.MaxInt64` directly) — is therefore
  **mandatory, not stylistic**: routing Forever through `expiryFromTTL` as a duration
  would wrap negative and expire instantly. No schema migration needed either way
  (`WHERE expires_at < now` already treats MaxInt64 as never-expiring).

### 3.1b Probe source and output (re-run 2026-09-18, go vet clean)

```go
// /tmp-regenerable; embed here so the review no longer depends on a /tmp path.
package main

import (
	"fmt"
	"math"
	"time"
)

func main() {
	now := time.Now()
	cases := []struct {
		name string
		ttl  time.Duration
	}{
		{"100 years (smart-configs default)", 100 * 365 * 24 * time.Hour},
		{"236 years", 236 * 365 * 24 * time.Hour},
		{"time.Duration(math.MaxInt64) ~292y", time.Duration(math.MaxInt64)},
	}
	fmt.Printf("now (UTC) = %s\n", now.UTC().Format(time.RFC3339))
	fmt.Printf("wrap threshold = %s\n\n", time.Unix(0, math.MaxInt64).UTC().Format("2006-01-02 15:04:05 MST"))
	for _, c := range cases {
		expiry := now.Add(c.ttl).UnixNano()
		live := expiry > now.UnixNano() // mirrors WHERE expires_at > now
		fmt.Printf("%-37s expiry year %-4d  UnixNano=%-22d live=%v\n",
			c.name, now.Add(c.ttl).UTC().Year(), expiry, live)
	}
}
```

```text
now (UTC) = 2026-09-18T16:46:14Z
wrap threshold = 2262-04-11 23:47:16 UTC

100 years (smart-configs default)     expiry year 2126  UnixNano=4943349974611977407    live=true
236 years                             expiry year 2262  UnixNano=-9214498099097574209   live=false
time.Duration(math.MaxInt64) ~292y    expiry year 2318  UnixNano=-7433622062242798402   live=false
```

(The 2026-09-16 original recorded the threshold as 2262-04-12; the re-run pins the exact
value `time.Unix(0, math.MaxInt64)` produces: **2262-04-11** 23:47:16 UTC.)

### 3.2 CV's open item 32 answered: `dedup/` is NOT a forever store, no split brain

Read in full (101 lines + README). `dedup.Ring` is a fixed-capacity in-memory ring for
replay→live boundary dedup (projectionhost, watermill CatchUpSubscriber, SSEBroker); it
deliberately **evicts** oldest IDs — the semantic opposite of forever. It is neither a
second forever-dedupe implementation nor the wire-target for a `Forever` semantic; the
`dedup/`↔`idempotency/` pairing is bounded-recency vs TTL-windowed, by design. The
Forever gap remains go-idempotency's to fill (upstream `Forever` sentinel → adapters map
to MaxInt64 per 3.1).

### 3.3 CV's open item 33 answered: the go-idempotency consumer census

Direct `.go` importers across `~/projects` (excluding the library itself): **6 projects**
— bank-sync (middleware + own sqlite idempotency store), cqrs-htmx, DiscordSync,
file-and-image-renamer, Standup-Killer, Zlota44 (test-only); plus go-appkit/cqrs via
go.mod (indirect). **CV is not among them** — CV uses its own `MemoryIdempotencyStore`
and is the _prospective_ consumer blocked by the TTL wall. So: the library has real
reach (the "multiple repos" intuition holds), but the verified _forever-need_ count is
still **one (CV)** — YAGNI-defensible until a second forever-need appears, per CV's own
Q2 framing.

## 4. Reflection → action items for this repo (ranked)

1. **Fix the `Scan` doc lie + document the default (S, do now).** `metaengine/typed_reader_scan.go:8`
   says "returns all values" while defaulting to 100. Either document the default loudly
   ("Scan returns up to 100 rows unless `WithLimit` is passed — `WithLimit(0)` for
   unbounded") or flip the default to unbounded for v5 (breaking-change window). Add the
   limit note to skill `readmodels.md`. This is the one finding where the current API
   actively misleads.
2. **Consider a first-class single-writer/lease story (M, design).** A real consumer
   (CV) had to plan a driver-factory decorator just to coexist with a foreign
   `<dsn>.lease` holder. A minimal `system`/engine-level option (advisory file lock,
   `sqlite` `BEGIN EXCLUSIVE`, or an engine open-mode) would remove a whole decorator
   class. The `queue/` module already proves lease semantics in-tree. Decide before v5
   freezes engine construction surfaces.
3. **Release the coeffect gate (already-written work).** `DomainConfig.Events` + the
   dangling-subscription gate sit unreleased on master while the newest consumer
   (CV, pinned at system v4.7.0 = latest tag) enforces its universe CV-side. A system
   tag would let CV delete its `AllEventTypes()` gate. (Routine release mechanics, not
   new work.)
4. **ApplyBatch batching/atomicity (M, already planned).** ADR-0123 §10 owns this; the
   spike owns feasibility. CV's measurement (3.4 s vs 109 ms for 6.2k events; fsync-bound
   per-event commits) adds the missing performance argument and a real replay consumer.
   The v5 FoldOp-closure design from the spike is the vehicle; `synchronous=NORMAL` as a
   documented engine pragma is the stopgap consumers can use today.
5. **LIKE/contains pushdown (M, v5 FilterOp window).** `FilterContains`/`FilterPrefix`
   would serve the search pattern CV measured at 2.3–28 ms (client-side scan). Fits the
   existing `FilterOp` enum; engines without native LIKE can still evaluate in the
   closure fallback.
6. **AutoMapper `time.Time` (won't fix, by design).** The whole `storage/view` package is
   v5-removed; CV's int64-epoch-millis contract is the right call for any SQL tier and
   matches what metaengine SQL engines already do. No action beyond noting the caveat
   lived its life.
7. **go-idempotency Forever (upstream-first, M).** When green-lit: `Forever` sentinel in
   go-idempotency (v0.4.0), then adapters write MaxInt64 directly (per 3.1), dedup the
   verbatim-copied `expiryFromTTL` (both adapters, 9 lines each) while touching them,
   and add an overflow test at the `expiryFromTTL` boundary. Pin-bump coordination via
   the go-ecosystem-upgrade skill across the three v0.3.0 consumers.

## 5. What CV's process reflects back (meta)

- The parity-gated four-tier benchmark is the template for evaluating THIS library's
  claims: cross-tier parity before any number is trusted. Worth borrowing for future
  engine-vs-engine comparisons here (`benchkit` profiles exist; the parity gate does
  not).
- CV's measured asymmetry caveat (library tiers on constructor defaults vs hand-indexed
  RawSQL) is fair and this repo should hold itself to it: any future metaengine latency
  claim should be tuned-vs-tuned (`BuildLayoutPlanFromType` composite layouts exist at
  `metaengine/layout.go:199` and were NOT what CV measured).
- One CV status report briefly recorded a wrong verdict ("metaengine selected") before
  reconciliation — a reminder that decision docs must reconcile against existing
  evidence before landing. No action for this repo; the corrected verdict is what this
  document consumed.

_Point-in-time verification: 2026-09-16, master @ 240368b57. No code was changed in this
review. Probe source and output embedded in §3.1b (re-verified 2026-09-18; the /tmp copy
was regenerable and is no longer load-bearing)._
