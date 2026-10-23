# Status: Why metaengine has no go-codec base (Tier -1 substrate gap)

> **Point-in-time snapshot. Not a living document.**
>
> Session date: 2026-10-23 17:20 CEST
> Scope: root-cause investigation only. **No production code was changed, no tests were added, no gates were run beyond one package test.**
> Trigger: owner observation "metaengine totally lost all the good things about using `/home/lars/projects/go-codec` as a base. WHY!??!"

---

## TL;DR

1. **metaengine never used go-codec.** There is no commit that removed it. The module was stdlib-only from its first commit because [ADR-0062](../../adr/0062-metaengine-dependency-boundary.md) (2026-07-25) made it "zero-dependency" by decree.
2. **A second same-day decision removed the incentive to ever revisit it.** [ADR-0066](../../adr/0066-metaengine-reify-fallback.md) chose bare JSON engine values plus a JSON round-trip (`reify`) for cross-engine typed reads, and explicitly **rejected** stamping a serialization format on stored values.
3. **The premise was later declared wrong, but the correction was narrow.** ADR-0062's 2026-08-06 addendum says the zero-dependency boundary is wrong and that dependencies are justified when they improve the planner. It only re-admitted `record/`. Nobody re-opened the codec question.
4. **Owner correction during this session (authoritative):** go-codec is **not a leaf dependency and not "Tier 3 importing down"**. It is a **Tier -1 substrate** (conceptually below Tier 0, like the standard library). Therefore there is no architectural objection at all; its absence is **encoding debt**, not a boundary being respected.
5. **Consequence:** metaengine re-implemented a slice of go-codec badly and in five places, and the strongest evidence is a test that had to pin JSON because production cannot decode the library's own default encoding (CBOR).

---

## 0. What was actually done in this session

| Step | Action | Result |
| ---- | ------ | ------ |
| 1 | Loaded the mandatory `go-cqrs-lite` skill | Confirmed the documented rule: decode payloads with a codec, never raw JSON; default event codec is CBOR |
| 2 | Read `metaengine/go.mod` | No `go-codec` require, direct or indirect |
| 3 | Traced history with `git log -S` for `go-codec` and `codec` under `metaengine/` | No production import ever existed; the only historical `go-codec` lines were indirect requires added/removed by 2026-08-13 tidy commits |
| 4 | Read ADR-0062, ADR-0066, ADR-0048, ADR-0111 | Found the two 2026-07-25 decisions that froze the design |
| 5 | Read `metaengine/stream_codec.go`, `encoded.go`, `vector_binary.go`, `serializable.go`, `cursor.go`, `reify.go`, `layout_fingerprint.go`, `vector_insert.go`, `sql_keyset.go` | Mapped the bespoke encoding surface |
| 6 | Read `system/projection_wiring.go`, `system/projection_builder.go`, `system/evolutions.go` | Proved the `system` path IS codec-aware (`codec.ForEncoding(evt.Encoding())`), so the hole is the direct/metaengine path, not the composition root |
| 7 | Counted the surface | 66 production `.go` files import `encoding/json/v2`; 145 `json.Marshal`/`json.Unmarshal` call sites; 0 production go-codec imports; 1 test-only go-codec import |
| 8 | Ran `GOWORK=off go test ./keycodec/... -count=1` in `metaengine` | GREEN (0.009s). The committed `testdata/rapid/*.fail` seeds replay green |
| 9 | Checked tree state | Clean at report time; the pre-existing modified planning doc was absorbed by the auto-commit daemon (`bfdec007e`) |

**Not done:** no repro test, no code fix, no ADR, no TODO row, no GitHub issue, no benchmark, no doc-drift fix, no full gate run.

---

## 1. The exact "why" chain

### 1.1 ADR-0062 created the gap (2026-07-25)

`docs/adr/0062-metaengine-dependency-boundary.md:13-30` states the core "depends only on the standard library and test frameworks" and lists "consumers can adopt the planner without pulling in event sourcing" as the reason. The in-repo `codec/` module was therefore excluded from day one, not lost later.

### 1.2 ADR-0066 removed the pressure to fix it (2026-07-25)

`docs/adr/0066-metaengine-reify-fallback.md` decided that SQL engines store values as bare JSON and `ExecuteTyped` reifies `map[string]any` back into `R` via a marshal/unmarshal round-trip. Its alternatives section **explicitly rejects** "Store type tags alongside values" because it "couples the projection to a serialization format".

That rejection is precisely the go-codec envelope design (`WrapEncode`/`UnwrapDecode`, ADR-0044/0050). The ADR's own consequences admit the cost:

- unexported fields are lost across the SQL boundary
- unknown fields are silently ignored on reification
- the stored bytes carry no type or codec identity

### 1.3 The addendum invalidated the premise but not the consequence (2026-08-06)

`docs/adr/0062-metaengine-dependency-boundary.md` addendum says: **"The zero-dependency principle for metaengine core is superseded."** It re-lists what changes: metaengine gains `Record`, SQLite moves to its own module, projectionadapter is simplified. It never says "re-admit the codec". No follow-up ADR, TODO row, or gate ever asked the question again.

### 1.4 ADR-0111 kept the codec contract out of the shared type

`docs/adr/0111-record-type-extraction.md:30` defines `Payload []byte // encoded payload (codec-stamped)`, but the stamp is `record.Encoding`, a closed `uint8` enum (`record/encoding.go`) with only `unknown/json/cbor`. The repo knows the cost: `TODO_LIST.md` (E1 row) records that `record.Encoding` was considered for the event envelope and rejected because "its closed enum would drop custom codec stamps"; the event envelope uses `codec.Encoding` instead. The type that metaengine and event share therefore still cannot describe the encoding contract.

### 1.5 The dependency budget locks the status quo

`scripts/check-module-layers.sh:328` sets `DEP_BUDGET[metaengine]=7` and the comment enumerates exactly the seven allowed dependencies: record, id, dedup, errorfamily, go-sse, claiming, go-humanize. go-codec is not on the list. Adding it requires a budget review, and no review was ever filed. The path of least resistance has been stdlib.

### 1.6 Owner correction: the tier framing was wrong

Mid-session correction: **go-codec is a Tier -1 substrate**, not a leaf dependency. It sits conceptually *below* Tier 0 (the old in-repo `codec/` was Layer 0: `core.md:512`). So the correct statement is not "Tier 3 may import an external leaf"; it is:

> metaengine is paying a bespoke-codec tax to avoid a dependency that is architecturally beneath its own lowest tier.

That makes the finding worse than the initial report stated. There is no purity win here, only debt.

---

## 2. Evidence ledger

| Claim | Evidence |
| ----- | -------- |
| No production go-codec import | `rg -l 'larsartmann/go-codec' metaengine -g '*.go'` returns exactly one file: `metaengine/projectionadapter/adapter_record_test.go` |
| The only go-codec use in metaengine forces JSON | `metaengine/projectionadapter/adapter_record_test.go:9` imports it; line `:267` passes `event.WithCodec(codec.JSONCodec{})` so the JSON-only production decoder can read the event |
| Production default decode is JSON-only | `metaengine/projectionadapter/adapter.go:145`: `json.Unmarshal(payload, &decoded)` |
| `ApplyEncoded` is JSON-only by contract | `metaengine/encoded.go:21-33` plus the explicit doc line "For non-JSON encodings (CBOR, etc.), decode manually and use Store.Apply" |
| Events default to CBOR | `event/codec_test.go:422-428` (`TestDefaultCodec_DefaultIsCBOR`) |
| The `system` path is safe | `system/projection_builder.go:177-219` uses `codec.ForEncoding(evt.Encoding())`; `system/projection_wiring.go:14-39` prioritizes `TypeDecoder`/`EventDecoder`/auto-derived decoder |
| Stream codec swallows errors | `metaengine/stream_codec.go:14` (`data, _ := json.Marshal(v)`) and `:24` (decode failure returns the raw string) |
| Cursor is non-deterministic for maps | `metaengine/cursor.go:35` uses `json.Marshal(c.Value)`; ADR-0048 documents map-order non-determinism |
| Engine values are unstamped JSON | e.g. `metaengine/pgengine/engine.go:210`, `metaengine/duckdbengine/engine.go:197`, `metaengine/duckdbengine/planned.go:89`, plus the reify path in `metaengine/reify.go` |
| Stale zero-dep rationale still in code comments | `metaengine/projectionadapter/adapter.go:70` says the split preserves "metaengine's zero-dependency boundary", contradicted by the ADR-0062 addendum |
| Dep budget excludes codec | `scripts/check-module-layers.sh:328` (`DEP_BUDGET[metaengine]=7` plus the enumerating comment) |
| Quantified surface | 66 production files import `encoding/json/v2`; 145 production `json.Marshal`/`json.Unmarshal` sites |
| keycodec current state | `GOWORK=off go test ./keycodec/... -count=1` -> `ok ... 0.009s`; three committed `testdata/rapid/*.fail` seeds replay green |

---

## 3. Capability loss table (what go-codec gives that metaengine lacks)

| go-codec capability | metaengine today | Observable cost |
| ------------------- | ---------------- | --------------- |
| `Codec` + `Encoding` + `ForEncoding` | No codec contract at the engine/metaengine boundary; `record.Encoding` is a closed enum on `Record` only | No format negotiation, no custom codecs, no self-description |
| `AutoDetect` / `AutoDetectDebug` | Hand-rolled sniffing: marker byte in `DecodeVectorAuto`, JSON-fail-then-raw in `DecodeStreamValue` | Silent wrong-type fallbacks, no detection diagnostics |
| `TranscodeToJSON` | Absent | CBOR payloads cannot enter `ApplyEncoded`; every consumer must bridge manually |
| `WrapEncode`/`UnwrapDecode` envelope (ADR-0044/0050) | `reify` JSON round-trip (ADR-0066 explicitly rejected stamps) | Unexported fields lost, unknown fields ignored silently, legacy mixed-codec rescue absent |
| `BufferEncoder` / `EncodePooled` / `GetBuffer` | None | Allocation per stored value on the projection write path |
| `DeterministicCodec` | Fingerprint hand-sorts (good); cursor does not | Map cursors are not byte-stable; unsafe as cache keys or signatures |
| `Size` / `ObservableCodec` | None | No codec-level metrics or payload-size diagnostics |
| Established CBOR performance | Read-model values are JSON only | CBOR's documented size/speed win (go-codec README claims) is unmeasured here, but structurally unavailable |

---

## 4. Severity assessment

| Finding | Severity | Why |
| ------- | -------- | --- |
| CBOR event through `ApplyEncoded` fails | **High (correctness)** | The library's default event codec cannot feed metaengine's own encoded entry point |
| `projectionadapter` default decode is JSON-only | **High (correctness)** | CBOR events become `Corruption` and can be dead-lettered unless the consumer supplies a typed/event decoder |
| `EncodeStreamValue` swallows marshal errors | **Medium-High (data loss)** | Non-JSON-able value writes `""` with no error; `DecodeStreamValue` then returns a string where a value is expected |
| Engine rows are unstamped bare JSON | **Medium (evolution)** | Cannot introduce CBOR or a new codec per collection without a flag day; `reify` carries the compatibility burden |
| Cursor map encoding non-deterministic | **Medium (correctness/ops)** | Cursor bytes vary; unsafe as cache keys; pagination state is opaque JSON |
| No pooling / size / observability at the codec layer | **Low-Medium (perf/ops)** | Avoidable GC and no visibility on the projection write path |
| Stale "zero-dependency" comments | **Low (doc drift)** | Misleads every future reader and agent; ADR-0062 addendum already contradicts it |

---

## a) FULLY DONE

- [x] Confirmed the premise: metaengine has **zero** production go-codec imports, now and historically.
- [x] Located the root cause in two dated ADRs, not in a lost refactor.
- [x] Documented the full causal chain: ADR-0062 -> ADR-0066 -> narrow addendum -> ADR-0111 closed enum -> dep-budget lock-in.
- [x] Proved the `system` composition-root path is already codec-aware; isolated the hole to direct metaengine and `projectionadapter` default paths.
- [x] Mapped 8 lost go-codec capabilities onto concrete metaengine call sites.
- [x] Quantified the bespoke surface (66 files, 145 JSON call sites, 1 test-only codec import).
- [x] Found the smoking gun: a test pins `codec.JSONCodec{}` to satisfy the JSON-only production decoder.
- [x] Verified `keycodec` tests are green (including committed rapid regression seeds).
- [x] Recorded the owner's Tier -1 correction and re-based the severity conclusion on it.

## b) PARTIALLY DONE

- [~] Capability inventory across engines: `pgengine`, `duckdbengine`, `sqliteengine` sampled; `mysqlengine`, `badgerengine`, `bboltengine`, `pebbleengine`, `bigtableengine`, `dgraphengine` not individually audited.
- [~] Runtime proof of the CBOR hole: inferred from `json.Unmarshal` semantics; **no repro test executed**.
- [~] Fix design: direction is clear (encoding-aware boundaries, then envelope), but no API shape, no compatibility window, no cost estimate.
- [~] Impact estimate: structurally argued, not measured (no size/throughput benchmark in metaengine).
- [~] Historical trace: `-S` searches are conclusive for imports and go.mod, but I did not walk every commit that touched encoding helpers (auto-commit noise makes that expensive).
- [~] Documentation drift: found two stale zero-dep claims; did not sweep all READMEs/comments.

## c) NOT STARTED

- [ ] Failing repro test for CBOR through `ApplyEncoded`.
- [ ] Failing repro test for CBOR through `projectionadapter` default path.
- [ ] Error-path tests for `stream_codec.go`.
- [ ] Fuzz tests for stream values and cursors.
- [ ] Any production code change.
- [ ] Any ADR (the Tier -1 classification deserves one).
- [ ] Any TODO_LIST row or GitHub issue.
- [ ] Any dep-budget change (`7 -> 8`).
- [ ] Any benchmark (JSON vs CBOR read-model rows).
- [ ] Any gate run (`check-arch`, `check-file-size`, `api-stability`, `doc-check`).
- [ ] Consumer blast-radius test run.
- [ ] Doc fixes (stale comments, README, skill refs, WIRE-FORMAT-KEYS).
- [ ] CHANGELOG entry.

## d) TOTALLY FUCKED UP

No destructive action was taken. The failures are analytical and procedural:

1. **Wrong tier framing, corrected by the owner.** I treated go-codec as an external leaf dependency and framed the question as "may Tier 3 import it". It is a Tier -1 substrate. My original answer understated the problem: this is not a boundary to respect, it is a foundation missing under a module that already depends on `record`, `id`, `dedup`, `claiming`, `go-sse`, and `go-humanize`. I should have classified it as substrate and called the absence debt from the first sentence.
2. **Asserted runtime behavior without executing it.** "CBOR cannot feed `ApplyEncoded`" is almost certainly true, but I did not write the 15-line test that proves it. In this repo's culture (mutation-verified pins, measured claims), that is not good enough.
3. **Stopped at diagnosis when the session could have produced an artifact.** No ADR, no TODO row, no issue, no test. The next session has to re-derive everything from this report.
4. **Sampled instead of auditing all engines.** I generalized "engine rows are bare JSON" from three engines plus shared helpers.
5. **Quoted go-codec performance claims without measuring metaengine.** "19% smaller, 32% faster" is go-codec's README, not a metaengine measurement.
6. **Did not quantify the bespoke-code tax.** LOC, allocations, and scan cost of the private codecs are unknown.
7. **Did not check whether `record.Encoding` can be extended for custom codecs**, which determines whether the shared type can carry the contract at all.
8. **Did not follow the wave-close procedure** (FEATURES/CHANGELOG/TODO harvest) because this was a question, not a wave. That was defensible, but it left the finding unactionable.
9. **Ignored the status-index convention until the user asked for a report.** The live index is part of the deliverable; I should have planned it from the start.
10. **First-go-command hygiene:** the keycodec test was run only after sourcing `scripts/go-env.sh`; earlier tool calls were `cat`/`rg` only, so nothing was poisoned, but the report should state that the full gate loop was never exercised in this environment.

## e) WHAT WE SHOULD IMPROVE

1. **Classify dependencies by tier, not by "leaf vs heavy".** A substrate is not a convenience add; refusing it is a design statement that must be justified. Record the Tier -1 concept in the ADR/glossary so this mistake is not repeated.
2. **Never accept "JSON-only by contract" for a library whose default is CBOR.** The encoded entry point must accept an encoding stamp or sniff; a doc note is not a fix.
3. **Prefer proving over reasoning.** Every "cannot" claim about an exported API should ship with a failing test in the same session.
4. **Do not let tests encode production limitations.** The `codec.JSONCodec{}` pin in a metaengine test is a smell: it should be an explicit legacy-case test, not the only happy path.
5. **Make error handling non-negotiable in codec helpers.** Silent `_ = json.Marshal` and decode-fallback-to-raw are data-loss APIs; they need error returns or a loudly documented degradation type.
6. **Revisit ADR-0066 with a dated addendum rather than silently contradicting it.** The envelope decision and its rejected alternative need an explicit reversal, with migration semantics.
7. **Add a codec-boundary gate.** `check-arch` should know that metaengine may import go-codec, and a lint rule should flag new raw `json.Marshal` at engine/metaengine boundaries without an encoding stamp.
8. **Track capability debt in TODO_LIST, not only in status reports.** A report without a harvested row is a snapshot of a problem, not a plan.
9. **Measure before quoting dependency benchmarks.** Add a metaengine row-size and apply-throughput benchmark before claiming CBOR wins.
10. **Keep the index and docs in lockstep.** Fix the stale zero-dependency comments in the same change that adopts codec; otherwise the next agent re-derives the wrong conclusion.

## f) UP TO 50 THINGS WE SHOULD GET DONE NEXT

Ordered by dependency and value. P0 = correctness, P1 = contract, P2 = migration, P3 = polish.

1. [P0] Write `TestApplyEncoded_CBORPayload` that fails today.
2. [P0] Write `TestProjectionAdapter_DefaultDecoder_CBOR` that fails today.
3. [P0] Write `TestEncodeStreamValue_UnmarshalableValue` proving the silent `""` loss.
4. [P0] Write `TestDecodeStreamValue_MalformedYieldsRawString` pinning the type confusion.
5. [P0] Write `FuzzStreamCodecRoundTrip`.
6. [P0] Write `FuzzCursorRoundTrip` including maps, `[]byte`, and scalars.
7. [P0] Audit all production `json.Marshal` sites for discarded errors (`pgengine/aggregations.go`, `duckdbengine/aggregations.go`, `pushdown.go`, etc.).
8. [P0] Decide whether `DecodeStreamValue`'s fallback is a contract or a bug; either way, test it.
9. [P1] Write ADR: "metaengine encoding boundaries and Tier -1 substrate" with the Tier -1 definition.
10. [P1] Add a dated addendum to ADR-0066 reversing the "no type tags" choice, or keep it and explain the alternative.
11. [P1] Raise `DEP_BUDGET[metaengine]` to 8 and document go-codec in the budget comment.
12. [P1] Choose the `ApplyEncoded` API shape: `ApplyEncodedWithEncoding(type, payload, enc)` vs `AutoDetect` by default.
13. [P1] Choose whether `AutoDetect` is default-on or opt-in for direct callers.
14. [P1] Decide the engine-value envelope format (stamped JSON/CBOR) and the dual-read window.
15. [P1] Decide cursor wire contract: keep opaque JSON, or version + stamp it.
16. [P1] Decide whether custom/encrypted codecs are in scope for engine values.
17. [P1] Define the compatibility promise for rows written before any envelope change.
18. [P0] Route `ApplyEncoded`/`ApplyEncodedRecord` through `codec.ForEncoding`/`AutoDetect`.
19. [P0] Replace `projectionadapter`'s default `json.Unmarshal` with an encoding-aware decode.
20. [P0] Keep `Corruption` classification for genuinely undecodable payloads after the codec attempt.
21. [P1] Add an encoding-aware `EncodeStreamValue` variant; deprecate the silent one.
22. [P1] Change `DecodeStreamValue` to return an error or a typed fallback.
23. [P1] Make `Cursor.Encode` binary-safe and deterministic (or reject map values explicitly).
24. [P1] Add a cursor format version byte/prefix.
25. [P1] Keep a legacy cursor decode test for every prior shape.
26. [P2] Prototype the envelope in `sqliteengine` map rows with dual-read.
27. [P2] Extend to `tursoengine`.
28. [P2] Extend to `pgengine`.
29. [P2] Extend to `mysqlengine`.
30. [P2] Extend to `duckdbengine`.
31. [P2] Extend to `bigtableengine`.
32. [P2] Extend to the LSM family (`pebbleengine`, `bboltengine`, `badgerengine`) via `keycodec`.
33. [P2] Replace the `reify` JSON round-trip fast path with envelope-aware decode.
34. [P2] Keep `reify` as the legacy fallback and mark it so.
35. [P2] Add a cross-engine mixed-codec conformance test.
36. [P2] Add a legacy-row fixture per engine (pre-envelope bytes read post-upgrade).
37. [P1] Fix stale zero-dependency comments (`projectionadapter/adapter.go:70` and any twins).
38. [P1] Update `metaengine/README.md` "ApplyEncoded" and toolchain sections.
39. [P1] Update skill `core.md` §3.3 to state the metaengine boundary explicitly.
40. [P1] Add a FAQ entry: "why did my CBOR projection become Corruption?"
41. [P1] Document the engine row wire format in `docs/WIRE-FORMAT-KEYS.md`.
42. [P1] Add a CHANGELOG `[Unreleased]` entry when the first slice lands.
43. [P0] Run `nix run .#check-file-size` after adding code/tests.
44. [P0] Run `nix run .#check-arch` after the dep-budget change.
45. [P0] Regenerate the API golden (`cmd/api-stability ... --update`).
46. [P0] Run `cmd/doc-check` on every touched doc.
47. [P1] Harvest this report into TODO_LIST with the chosen P0 slice.
48. [P1] File a GitHub issue for the CBOR/`ApplyEncoded` correctness hole.
49. [P2] Add a JSON-vs-CBOR read-model benchmark and a regression guard.
50. [P3] Remove the test-only `codec.JSONCodec{}` pin once the production default is codec-aware, and convert it into an explicit legacy test.

## g) QUESTIONS ONLY THE OWNER CAN ANSWER

1. **Direction ruling:** should metaengine adopt go-codec at all encoding boundaries (your Tier -1 framing), or should it keep a tiny internal codec contract to avoid the module dependency? This determines whether we write an adoption ADR or an isolation ADR.
2. **Compatibility scope:** do any v4 metaengine deployments hold materialized rows that must remain readable in their current bytes, and how long a dual-read window do you want before legacy JSON-only rows may be dropped?
3. **First-slice appetite:** fix the small, no-schema-change correctness hole first (`ApplyEncoded` + `projectionadapter` default decode), or go straight for the stamped engine-value envelope that touches every engine? I can recommend the small slice first, but the risk appetite and release timing are yours.

---

## Appendix A: Self-review prompts

**What did I forget?**
- The repro tests. The error-path tests. The full engine audit. The `record.Encoding` extensibility question. The artifact (ADR/TODO/issue). The benchmark. The doc sweep.

**What could I have done better?**
- Classified go-codec as Tier -1 from the start. Proved the CBOR failure with a test instead of reasoning from `json.Unmarshal`. Checked all engines, not a sample. Measured instead of quoting README numbers. Produced at least one durable artifact.

**What can I still improve?**
- Everything in §f, starting with items 1-4 and 9-10. The report is only useful if the next session converts it into a failing test, a one-page ADR, and a TODO row.

## Appendix B: One-sentence verdict

> metaengine did not lose go-codec; it was designed without the substrate that sits beneath its own lowest tier, then grew five private codecs to compensate, and the bill is now visible as a CBOR-shaped hole in its encoded entry point plus a test that pins JSON to keep production happy.
