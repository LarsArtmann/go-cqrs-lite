# ADR Review — Inconsistencies & Questionable Decisions

**Date:** 2026-07-23
**Scope:** All 57 ADR files (0001–0057) plus `README.md`
**Method:** Full read of every file, with code-level verification of key claims

---

## A. README index is out of sync with the files

The `docs/adr/README.md` table disagrees with individual files on **Status** for at least 8 ADRs:

| ADR  | README says     | File says                                                                              |
| ---- | --------------- | -------------------------------------------------------------------------------------- |
| 0004 | Accepted        | **Superseded**                                                                         |
| 0011 | Accepted        | **Proposed**                                                                           |
| 0012 | Accepted        | **Proposed**                                                                           |
| 0017 | Proposed        | **Accepted**                                                                           |
| 0018 | Proposed        | **Accepted**                                                                           |
| 0027 | Implemented     | File agrees, but ADR-0028 declares it "superseded for new code" — status never updated |
| 0031 | Accepted        | **Implemented (v3)**                                                                   |
| 0001 | Date 2026-04-29 | Date **2026-05-03**                                                                    |

The index **stops at 0054** — ADRs **0055, 0056, 0057 are missing entirely** from the README table. Many rows for 0037–0053 also show "—" for the date even though the files contain real dates.

---

## B. Direct contradictions between ADRs

### B1. ADR-0001 file vs README mini-ADR

File says: _"The OO `aggregate` package stays for existing consumers."_
README says: _"the deprecated `core/aggregate` package was removed in Session 99."_
**Verified:** neither `core/` nor any `aggregate/` directory exists — the file's claim is false.

### B2. ADR-0009 vs ADR-0019 vs reality

ADR-0009's Decision: _"Pebble implements `event.Store` only. No Journal, no SeekableJournal, no secondary index … will never implement `event.Journal`."_
ADR-0019 then documents a `cqrs_journal:` key prefix.
**Verified:** pebble **does** implement both `ReadAll` (Journal) and `ReadFrom` (SeekableJournal) at `storage/pebble/journal.go:29,57`. ADR-0009 was simply violated; it should be superseded.

### B3. ADR-0014 vs ADR-0045

ADR-0014: _"Extracting `eventtest` as a separate module … is deferred."_
ADR-0045 then does exactly that extraction. ADR-0014 was never marked updated/superseded.

### B4. ADR-0014 dependency list is stale

Claims `event` production imports include `samber/ro`.
**Verified:** no `samber/ro` import exists in `event/` production code.

### B5. ADR-0015 vs ADR-0051/0053

ADR-0015: _"JSON remains the default."_
ADR-0051 flips `event.DefaultCodec` to CBOR; ADR-0053 flips all blind stores. ADR-0015 has no superseded note.

### B6. ADR-0020 Pattern 2 vs ADR-0049 — opposite middleware patterns

- ADR-0020: _"Pre-compute Middleware Chains — Rebuild only when middleware changes."_ (hot-path optimization)
- ADR-0049: _"The middleware chain is now rebuilt on every `Dispatch()` call."_ (acknowledges "slight performance cost")

They apply to different types (`MemoryBus` vs `dispatcher.Dispatcher`), but having two ADRs enshrine contradictory patterns without acknowledging the tradeoff is a smell.

### B7. ADR-0028 vs ADR-0027

ADR-0028 states: _"ADR-0027 (Postgres LISTEN/NOTIFY bus) is superseded for new code."_
ADR-0027's status was never updated to reflect this.

### B8. ADR-0028 vs reality — v3 removal never happened

ADR-0028 says `event.Bus`, `event.Subscriber`, `event.Middleware`, `EventBus`, `memory/bus.go`, `memory/command_bus.go`, `storage/pg_bus.go` are _"removed at the v3 boundary (T25)."_
**Verified:** all of these still exist (`event/bus.go:26,40,52`; `storage/pg_bus.go`; `command/memory_bus.go`). Either v3 never landed or T25 was abandoned — the ADR reads as a done deal.

### B9. ADR-0030 vs ADR-0037 — package resurrection

ADR-0030: _"`projection/` is removed at the v3 boundary."_
ADR-0037: _"Extract the `Projection` interface to a new `projection/` module."_
Both happened — `projection/projection.go` is the new interface-only module. But neither ADR acknowledges the name resurrection. A reader hitting 0030 first reasonably concludes the package is gone forever.

### B10. ADR-0031 vs code — aliases still exist

Claims `command.Metadata` and `query.Metadata` _"become their own structs … aliases are deleted at the v3 boundary."_
**Verified:** they are **still type aliases**:

- `command/metadata.go:24: type Metadata = metadata.CustomData[MetadataKey]`
- `query/query.go:46: type Metadata = metadata.CustomData[MetadataKey]`

They were repointed away from `event.Metadata` to `metadata.CustomData[MetadataKey]`, but they did not become "own structs." AGENTS.md also still describes them as aliases — three-way inconsistency.

### B11. ADR-0056 self-contradiction on `TimeUnix`

Context claims `CanonicalEncOptions()` defaults `Time` to `TimeUnix` which _"truncates to seconds."_
**Verified:** `codec/cbor.go:29` and `codec/cbor_compact.go:40` use **`cbor.TimeUnixDynamic`**, not `TimeUnix`.
The Consequences section even contradicts the Context: _"CBOR `TimeUnixDynamic` preserves nanos."_ The motivating problem as described may not exist.

---

## C. Stale references to deleted things

- **`example/todo/`** is cited by ADR-0004, ADR-0009, ADR-0016, ADR-0028 as a working reference.
  **Verified:** it does not exist. Only `example/getting-started`, `example/readme-quickstart`, `example/taskmanager` exist.
- **ADR-0018** references `projection.Runner` and `projection/runner.go` — dissolved by ADR-0030.
- **ADR-0026** lists "Arena allocation" as an experimental feature under exploration. AGENTS.md confirms it was removed (zero consumers). ADR-0026 also says _"All 7 core modules compile to WASM"_ then lists 6 (`id`, `codec`, `dispatcher`, `event`, `command`, `query`) — off-by-one.
- **ADR-0017** references `catalog/schema/` and `schema.FromReflect()`. The catalog split (ADR-0012, still Proposed) would move this.
- **ADR-0040** section heading _"Implementation plan (deferred)"_ with bullets like _"Define Deriver type + Then/Filter combinators (~100 LOC)"_ — but the status line says _"implemented: `deriver/` module."_ Section is stale.

---

## D. Broken intra-ADR links

- **ADR-0052 "Related"** links to `0044-self-describing-blind-stores.md`. Actual file is `0044-blind-store-encoding-stamps.md`.
- **ADR-0044** references `docs/v4-WISHLIST.md` item #8 and `docs/migration/JSON_TO_CBOR.md` — unverified, worth checking.

---

## E. Questionable decisions worth re-examining

### E1. ADR-0027 (Postgres LISTEN/NOTIFY bus) — likely wasted effort

ADR-0027 ships a full implementation (re-fetch, lifecycle, channel allow-listing, three CI integration tests, dedicated `pgxpool`) and then **one day later** ADR-0028 declares it superseded for new code in favor of Watermill. The whole PgxListener/re-fetch/visibility-retry machinery was built knowing (or immediately before deciding) it would be deprecated. The ADR-0027 → 0028 sequence reads as build-then-deprecate.

### E2. ADR-0046 — "Four-Tier Model" is misnamed

The model has **seven** numbered tiers (0–6). The name "four-tier" doesn't match the content. The doc even lists them as "Tier 0 … Tier 6." Either compress to four tiers or rename to "Seven-Tier Dependency Model."

### E3. ADR-0009 — over-confident "never"

_"Pebble will never implement `event.Journal` or `event.SeekableJournal`"_ was overturned by ADR-0019 within ~3 weeks. Absolute "never" claims in ADRs age badly; soften to "deferred unless a consumer need emerges."

### E4. ADR-0010 documents a decision that wasn't taken

The Decision section proposes a new `Lifecycle` interface. The Status line then says _"No new `Lifecycle` type was introduced — `io.Closer` already fills that role."_ The ADR body sells one design; the implementation chose another. Should be rewritten so the Decision matches what was actually decided.

### E5. ADR-0040 leans too hard on TypeDB for legitimacy

The Deriver design references TypeDB's reasoning engine at length, then rejects nearly every TypeDB feature (recursion, query-time eval, rule-as-schema, n-ary relations). What's left is a 100-LOC `Deriver` function type with `Then`/`Filter` — bog-standard Go functional composition. The TypeDB framing inflates a simple combinator API into an academic comparison.

### E6. ADR-0026 WASM target oversells

Only 6–7 core modules compile to WASM; the storage/middleware tiers (Pebble, SQL, OTel SDK) don't. Calling WASM a supported target when the practical surface is a handful of leaf modules is generous.

### E7. ADR-0042/0043 — DLQ split is a real wart

Keeping `middleware.DeadLetterEntry` and `projectionhost.DeadLetterEntry` separate is justified on its merits, but the ADRs don't note that consumers using both retry middleware _and_ projectionhost must learn two parallel DLQ APIs with divergent field shapes (`Error error` vs `Error string`, `AggregateID id.AggregateID` vs `string`). The "this is not a split brain" defense is protest-too-much.

---

## F. Minor / cosmetic

- ADR-0003 file says "9 modules"; README mini-ADR says "16 modules"; AGENTS.md says "52"; actual count is **55**. The ADR-0003 file should be marked historical.
- ADR-0046 says _"38 of 48 modules"_ depend on codec; AGENTS.md says _"38 of 52."_ Now 55 modules total — both stale.
- ADR-0002 README says "38 sentinel errors"; file says "~20." Both unverified, but they disagree.

---

## Recommended cleanup priorities

All items resolved on 2026-07-23. Details below.

| Priority | Action | Effort | Status |
| -------- | ------ | ------ | ------ |
| 1 | **Fix the README index** — add 0055–0059, fix the 8 status mismatches, fill in dates | 10 min | **Done** |
| 2 | **Mark violated/superseded ADRs**: 0009, 0014, 0015 (top notes + status), 0027 (loud deprecation), 0028 (note v3 not executed) | 20 min | **Done** |
| 3 | **Rewrite ADR-0010's Decision** to match what was actually implemented (io.Closer, not Lifecycle) | 10 min | **Done** |
| 4 | **Reconcile ADR-0031** with the actual alias-based metadata shape | 30 min | **Done** (status note added) |
| 5 | **Fix the ADR-0056 `TimeUnix` misstatement** | 5 min | **Done** |
| 6 | **Sweep dead `example/todo/` references** from ADRs 0004, 0009, 0016 | 10 min | **Done** |
| 7 | **Rename ADR-0046** to "Seven-Tier Dependency Model" | 5 min | **Done** |
| 8 | **Trim TypeDB framing** from ADR-0040 | 10 min | **Done** |
| 9 | **Cross-reference ADR-0020 ↔ ADR-0049** middleware pattern contradiction | 5 min | **Done** |
| 10 | **Add consumer burden section to ADR-0043** + draft ADR-0059 | 20 min | **Done** |
| 11 | **ADR-0058** added to README index | 2 min | **Done** |
