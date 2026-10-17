# SUPERB — Disk-Truth Pareto Plan v2 (2026-10-17 20:13, revised 20:4x)

> **Scope:** the disk-truth wedge (StreamMarker fix debt + on-disk schema review) — now
> **sequenced onto the delivery chain and the v5 cut**, not planned as an isolated track.
> Predecessors: [2026-10-11 trust-green-tags](2026-10-11_20-16_SUPERB-trust-green-tags-to-v5-cut-pareto-plan.md)
> (owns the 126-row backlog; its 1% = changed-set trust green + system/systemscenario/schema tag
> waves — **we JOIN that wave, we do not duplicate it**), [2026-10-09 v5 schema evolution]
> (2026-10-09_v5-declarative-schema-evolution.md) (owns upcasting/fingerprint machinery).
> `TODO_LIST.md` stays the living source.
>
> **v1 → v2 deltas (what the challenge surfaced):**
> 1. **The 1% was wrong.** Verification without delivery = polish in a drawer. The real 1% is the
>    DELIVERY CHAIN (changed-set green → extend the schema-v4.6.0 wave with the six fixed modules
>    → tag → cqrs-htmx/go-appkit bumps) — TODO_LIST itself rules "all shipped value is dead code
>    until consumers import it".
> 2. **v1 duplicated the trust-green plan's W1.** Changed-set verification already has an owner;
>    v2 contributes modules to that changed set instead of re-running a parallel track.
> 3. **No wave acceptance criteria.** v2 defines DONE per wave.
> 4. **No decision gates / risk register.** v2 adds both (incl. the "integration finds a real
>    regression" gate — answer: fix-forward is safe pre-tag; no released tag carries bare writes).
> 5. **Fine table violated its own ≤12min rule** (compound `12×3`/`12×5` rows). v2 splits them.
> 6. **No quantification.** v2 quantifies blast radius; estimates calibrated against prior
>    executed plans (27 medium/150 fine ≈ multi-day waves).

## Errata (Step 0)

| # | Correction |
|---|---|
| E1 | PC-5 partially stale: `LayoutPlan.Fingerprint` + stamp ledger shipped 2026-10-10 (T3+T5). Review corrected; gap re-scoped to default-on + conformance (D→A18). |
| E2 | Tag wave owned by TODO_LIST §V5-schema-evolution open row (schema v4.6.0 + system minor). We EXTEND it with event/command/storage/commandlifecycle/projectionhost. |
| E3 | Upcaster machinery fleet-proven (T2 + 3 consumers) — recipes.md doc only (A21). |
| E4 | go.sum untidies belong to the go-codec v0.3.2 train — tidy-or-confirm-owner only (A24d). |

## Quantified blast radius

- Typed WAL migration touches **4 SQL engines** (sqlite+turso delegate, pg, mysql, duckdb), 1 table each + 1 conformance suite.
- Key escaping touches **3 KV engines** + **2 legacy KV stores** (storage/pebble, storage/bbolt) with a one-release dual-read window.
- Tag wave extends an already-open wave by **6 modules**; consumer bumps: **2 repos** (cqrs-htmx ~30 trains, go-appkit 8 files).
- Pins/laws add ~**10 tests** across 5 modules. inferColumnType deletion touches **1 module** (metaengine core).

---

## Pareto Breakdown (v2)

### The 1% that delivers 51% — THE DELIVERY CHAIN

Everything this session produced sits in `[Unreleased]`; pinned v4 tags still write prefixed
bytes. One delivery chain converts 100% of it into fleet value:
**changed-set suites green (join trust-green W1) → extended schema-v4.6.0 wave (6 modules) →
tag → cqrs-htmx + go-appkit bumps.** ~½ day, mostly machine time. Skip it and every other task
in this plan polishes code nobody imports.

### The 4% that deliver 64% — the "never again" cluster

1. Delete `inferColumnType` (the only finding that corrupts FUTURE data silently).
2. Raw-bytes persistence pins (the class round-trip tests cannot see — proven twice).
3. keycodec sep/esc escaping + round-trip law (+ legacy KV dual-read).
4. cqrs-lint branded-`.String()` rule, warn-first.

### The 20% that deliver 80% — the disk truth

Typed WAL columns on all SQL engines (constraint-backed journal of record: UNIQUE event id,
UNIQUE(stream,version), real metadata) + layout-stamp default-on + docs truth (gotchas,
CHANGELOG four-layer compat, taskmanager example repair).

### The other 20% → 100%

Envelope unification (timers→gcdc), time discipline (UnixNano), wire tags, cursor/sweep audits,
legacy decode fixture, goal.db, recipes entry, benchkit key, harvest.

---

## Table A — Comprehensive plan (30–100 min, 24 tasks, delivery-first order)

| # | Task | Tier | Impact | Effort | Gate/Acceptance |
|---|------|------|--------|--------|-----------------|
| A1 | Join trust-green W1: run changed-set `GOWORK=off` suites for the 6 StreamMarker modules + schema/system; triage reds (own vs parallel) | 1% | unblocks the wave | M | all 8 modules green |
| A2 | Extend the open schema-v4.6.0 release row: add event/command/storage/commandlifecycle/projectionhost; co-release via batch-release smoke | 1% | fleet binaries stop writing prefixed | L | tags resolve; `go get` probe ok |
| A3 | Consumer bumps: cqrs-htmx then go-appkit (ADR-0152 order); run their suites; confirm prefixed writes gone in a systemscenario | 1% | value delivered | M | both consumer suites green |
| A4 | Run integration battery over the tagged set (`#test-integration`, pg/mysql-vm/dgraph/redis legs) — post-A2, pre-consumer (A3) | 1% | real-engine proof | L | matrix recorded in CHANGELOG |
| A5 | Delete `inferColumnType`; unknown → TEXT + loud warning; repin layout goldens; api golden if surface moves | 4% | kills silent mistyping | M | `contentType`-like fields → TEXT w/ warn |
| A6 | Raw-bytes pins: WAL event+command envelopes (no `StreamMarker:`, no base64 payload, metadata JSON) | 4% | leak visible to CI forever | M | pin red on pre-fix code (demonstrated) |
| A7 | goal-shaped-app end-to-end pin `TaskView.ID == doneID.Get()`; inspect/regenerate goal.db if prefixed | 4% | mirrors #70 pattern | S | story asserts bare id |
| A8 | keycodec sep/esc escaping (v2 family prefix) + rapid round-trip law over adversarial corpus (`:`,`\x00`,255+) on badger/bbolt-meta/pebble-meta | 4% | kills separator injection | L | law green; v1 key still readable |
| A9 | storage/pebble + storage/bbolt escaped keys, dual-read window one release (incl. `cqrs_journal_idx` rekey) | 4% | same fix, legacy KV | L | both suites green; README migration note |
| A10 | cqrs-lint branded-`.String()` rule (warn-first, slog/attribute whitelist, waiver path) | 4% | mechanical prevention | M | fixtures 3 good/3 bad; ratchet doc |
| A11 | sqliteengine typed WAL: additive columns + json_extract backfill + UNIQUE(event_id) + UNIQUE(collection,stream_id,version); journal COPIED never rewritten (ADR-0143) | 20% | journal gets constraints | L | duplicate-append conflicts |
| A12 | pgengine + mysqlengine typed WAL (JSONB metadata / online-DDL-safe indexes; NO table recreate on MySQL) | 20% | engine parity | L | same conformance green |
| A13 | duckdbengine typed WAL (ADD COLUMN path; UNIQUE via engine check + dedup table where DDL lacks it — documented) | 20% | parity without recreate risk | M | conformance adapted + green |
| A14 | WAL conformance into adttest: duplicate-append + (stream,version) conflict, every SQL engine | 20% | invariants proven per engine | M | 4 engines × 2 cases |
| A15 | Layout-stamp default-ON for planned tables + conformance: column add/rename trips `LayoutStampDiffs` on every engine | 20% | closes re-scoped PC-5 | M | diff fires on all engines |
| A16 | Docs truth pack: gotchas (sweep-complete, cursor-pairing, four-layer compat, raw-bytes-pin convention) + CHANGELOG four-layer wording | 20% | docs stop overclaiming | S | doc-check + md-go green |
| A17 | Fix example/taskmanager tests (`idStreamID`/`servertiming`, dep prune) → deriver.go covered | 20% | example = consumer docs | M | suite green |
| A18 | Timers payload → ADR-0044 gcdc envelope (decode-old-wrap-new) | tail | one envelope system | M | migration test green |
| A19 | Time discipline: typed-WAL writers store UnixNano (rides A11–A13); UTC rule documented | tail | kills PC-6 ordering trap | S | order-by test across offsets |
| A20 | Wire tags for `Causation`/`TombstoneMark` + golden-bytes pin | tail | PC-12 wire lock-in | S | golden committed |
| A21 | recipes.md §schema: legacy payload-ID strip via `Chain.Upcasters()` (doc only — E3) | tail | consumer playbook | S | doc-check green |
| A22 | Cursor/sweep audit: pebble/bbolt/watermill keyset readers vs stored form; tests where suspect; `StreamRef.Split()` consumer grep | tail | closes cursor bug class | M | findings rowed or tests added |
| A23 | Legacy prefixed-row decode fixture (golden v4-era bytes → current code) + benchkit `countKey` alignment | tail | compat proof + hygiene | S | fixture green |
| A24 | Closeout: docs-health HARVEST (incl. `ParseStreamID` strictness as owner question, go.sum tidy-or-confirm-owner per E4) | tail | backlog hygiene | S | TODO_LIST rows land |

## Table B — Fine breakdown (≤12 min each, no compound rows)

| # | Fine task | Parent | Est |
|---|-----------|--------|-----|
| B01 | Enumerate changed set (6 modules + schema/system); kick changed-set suites | A1 | 12 |
| B02 | Triage reds: own vs parallel-session; fix own (each fix its own row if >12min) | A1 | 12 |
| B03 | Record changed-set green matrix in TODO_LIST release row | A1 | 8 |
| B04 | Add 6 modules to schema-v4.6.0 wave row; note co-release intent | A2 | 8 |
| B05 | Dry-run batch-release --smoke over the wave | A2 | 12 |
| B06 | Tag wave; verify proxy resolution per module | A2 | 12 |
| B07 | Post-tag `go get` probe from a scratch consumer | A2 | 10 |
| B08 | cqrs-htmx: bump pins, run suite | A3 | 12 |
| B09 | go-appkit: bump pins, run suite | A3 | 12 |
| B10 | systemscenario in consumer: assert no prefixed write reaches store | A3 | 12 |
| B11 | `#test-integration` local-SQL leg (sqlite/pebble/bbolt/duckdb) | A4 | 12 |
| B12 | `#integration-pg` ephemeral leg | A4 | 12 |
| B13 | `#integration-mysql-vm` leg | A4 | 12 |
| B14 | `#integration-dgraph` + `#integration-redis` legs | A4 | 12 |
| B15 | Results matrix → CHANGELOG appendix | A4 | 10 |
| B16 | Delete `inferColumnType` + word lists | A5 | 10 |
| B17 | `sqlTypeOf` → (string, error); unknown→TEXT+warn | A5 | 12 |
| B18 | Repin layout goldens (price/count/age cases → TEXT) | A5 | 12 |
| B19 | CHANGELOG row + api golden regen | A5 | 10 |
| B20 | WAL pin: write event via sqlite backend, read raw row | A6 | 12 |
| B21 | Assert: bare id, raw payload, JSON metadata, UTC/nano time | A6 | 10 |
| B22 | WAL pin: command envelope twin | A6 | 10 |
| B23 | Demonstrate pin red on pre-fix commit (sanity of the pin itself) | A6 | 10 |
| B24 | goal-story assert bare ID | A7 | 10 |
| B25 | goal.db grep prefixed ids; regenerate if local | A7 | 10 |
| B26 | keycodec: escape/unescape + unit table | A8 | 12 |
| B27 | v2 family prefixes wired into all key builders | A8 | 12 |
| B28 | rapid round-trip law, adversarial corpus | A8 | 12 |
| B29 | 3 KV engine suites green | A8 | 12 |
| B30 | storage/pebble: escaped builder + legacy dual-read | A9 | 12 |
| B31 | storage/bbolt: same + journal_idx rekey | A9 | 12 |
| B32 | Migration notes in both READMEs | A9 | 10 |
| B33 | cqrs-lint: match `.String()` on branded receivers | A10 | 12 |
| B34 | Whitelist slog/attribute/log positions | A10 | 12 |
| B35 | Fixtures 3 good/3 bad; warn severity | A10 | 12 |
| B36 | Register rule + ratchet + waiver example | A10 | 12 |
| B37 | sqliteengine: guarded ALTER ADD typed columns | A11 | 12 |
| B38 | Backfill UPDATE via json_extract per column | A11 | 12 |
| B39 | Add UNIQUEs (index path; no recreate) | A11 | 12 |
| B40 | Writer fills typed columns (BLOB payload, JSON metadata, nanos) | A11 | 12 |
| B41 | Reader prefers typed, falls back to value JSON | A11 | 12 |
| B42 | pgengine: typed columns + JSONB metadata variant | A12 | 12 |
| B43 | pgengine: backfill + unique indexes + reader | A12 | 12 |
| B44 | mysqlengine: typed columns (VARCHAR(768) ids, JSON) | A12 | 12 |
| B45 | mysqlengine: online-DDL indexes + backfill + reader | A12 | 12 |
| B46 | duckdbengine: ADD COLUMN path + engine-level unique check | A13 | 12 |
| B47 | duckdbengine: backfill + reader + doc'd constraint story | A13 | 12 |
| B48 | adttest: duplicate-append case ×4 engines | A14 | 12 |
| B49 | adttest: (stream,version) conflict case ×4 engines | A14 | 12 |
| B50 | Flip stamp default ON (config inversion + migration note) | A15 | 12 |
| B51 | Conformance: column-add trips diff per engine | A15 | 12 |
| B52 | Conformance: column-rename trips diff per engine | A15 | 12 |
| B53 | gotchas: sweep-complete block (8 sites) | A16 | 10 |
| B54 | gotchas: cursor-pairing invariant | A16 | 8 |
| B55 | gotchas: four-layer compat checklist | A16 | 10 |
| B56 | gotchas: raw-bytes-pin convention | A16 | 8 |
| B57 | CHANGELOG: four-layer compat wording swap | A16 | 10 |
| B58 | taskmanager: servertiming import/usage fix | A17 | 12 |
| B59 | taskmanager: idStreamID/parseIDString helpers | A17 | 12 |
| B60 | taskmanager: dep prune + suite | A17 | 12 |
| B61 | timers: WrapEncode replace + legacy decode | A18 | 12 |
| B62 | timers: migration test old→new envelope | A18 | 12 |
| B63 | typed-WAL nanos writer + cross-offset order test | A19 | 12 |
| B64 | UTC/unixnano rule doc | A19 | 8 |
| B65 | Causation/TombstoneMark json tags + golden bytes test | A20 | 12 |
| B66 | Cursor audit grep + suspect tests (2) | A22 | 12 |
| B67 | StreamRef.Split consumer grep + note | A22 | 10 |
| B68 | Legacy prefixed-row golden fixture | A23 | 12 |
| B69 | benchkit countKey → Get() | A23 | 8 |
| B70 | HARVEST rows + ParseStreamID ruling prep + go.sum owner check | A24 | 12 |

---

## Execution graph

```mermaid
flowchart TD
    subgraph D["1% — DELIVERY CHAIN"]
        A1[A1 changed-set green<br/>joins trust-green W1] --> A2[A2 extended tag wave]
        A2 --> G1{G1: tags resolve?}
        G1 -- no --> A2
        G1 -- yes --> A4[A4 integration battery]
        A4 --> G2{G2: real regression?}
        G2 -- yes --> FIX[fix-forward: pre-tag, no released bare writes] --> A4
        G2 -- no --> A3[A3 consumer bumps]
    end
    subgraph P["4% — never-again cluster"]
        A5[A5 kill inferColumnType]
        A6[A6 raw-bytes pins] -. guards .-> A11
        A7[A7 story pin]
        A8[A8 key law] --> A9[A9 legacy KV dual-read]
        A10[A10 lint rule]
    end
    subgraph T["20% — disk truth"]
        A11[A11 sqlite typed WAL] --> A12[A12 pg+mysql] --> A13[A13 duckdb] --> A14[A14 conformance]
        A15[A15 stamps default-on]
        A16[A16 docs truth] & A17[A17 taskmanager]
    end
    subgraph X["tail → 100%"]
        A18 & A19 & A20 & A21 & A22 & A23 --> A24[A24 harvest]
    end
    D --> P --> T --> X
    A5 -.no name typing.-> A15
    A2 -.v4.6.0 wave.-> A18
```

## Decision gates

- **G1 (post-tag):** module proxy resolves + `go get` probe ok — else re-tag per go-release skill (never debug downstream first).
- **G2 (post-integration):** regression found → **fix-forward** (no released tag carries bare writes; journal bytes are additive) — never revert the wave.
- **G3 (pre-A9 merge):** dual-read window declared in release notes; one release later, drop legacy range.
- **G4 (A15):** stamp default-on ships WARN mode; hard-fail only at v5 (established rollout pattern).

## Risk register

| Risk | Mitigation |
|------|------------|
| MySQL table recreate locks `meta_stream_log` | Additive ALTER + online ADD INDEX only; no recreate (A12) |
| DuckDB UNIQUE limitations | Engine-level check + dedup table fallback, documented (A13) |
| Parallel session owns tag train | A2 extends THEIR row (E2); no independent release |
| cqrs-lint rule false positives | slog/attribute whitelist + waiver path + warn-first (A10) |
| Key migration dual-read miss | v2 family prefix keeps v1 range intact; G3 drop gate |
| Pins too loose (assert decode, not bytes) | B23 proves pin red on pre-fix commit |

## Wave acceptance criteria

- **W-DONE (1%):** consumers run retagged modules; a systemscenario in cqrs-htmx proves no prefixed write reaches any store; CHANGELOG rows moved under the version heading.
- **P-DONE (4%):** a scratch checkout of the pre-fix commit fails the new pins/laws (StreamMarker, separator injection, name-typing) and current HEAD passes all.
- **T-DONE (20%):** duplicate-append conflicts on all 4 SQL engines; `json_extract` no longer required to find an event's identity; `LayoutStampDiffs` fires on column drift per engine; docs/gotchas updated and gates green.
- **X-DONE (tail):** TODO_LIST harvested; zero orphan findings from either session report.

## What this plan deliberately does NOT do

- No v5 API breaks (String()/Display() flip, aggregate_*→stream_* column renames) — v5-cut work, owned by the 2026-10-11/10-08 plans.
- No journal rewriting ever (ADR-0143).
- No new envelopes, formats, or abstractions beyond unifying onto the existing gcdc envelope.
