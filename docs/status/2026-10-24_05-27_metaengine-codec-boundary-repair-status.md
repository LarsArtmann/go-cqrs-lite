# Status: metaengine codec boundary repair (first slice)

> **Session date:** 2026-10-24 05:27 CEST
> **Scope:** the metaengine encoded-ingress / projection-adapter / stream-codec / cursor slice only.
> **Plan:** [`docs/planning/2026-10-23_17-31_SUPERB-metaengine-codec-boundary-repair.md`](../planning/2026-10-23_17-31_SUPERB-metaengine-codec-boundary-repair.md)
> **Parent diagnosis:** [`docs/status/2026-10-23_17-20_metaengine-go-codec-tier-minus-one-gap-status.md`](2026-10-23_17-20_metaengine-go-codec-tier-minus-one-gap-status.md)
> **Branch state:** local `master` is 4 commits ahead of `origin/master`; HEAD `20a95af2c`. The same HEAD is pushed to `origin/agent/metaengine-codec-boundary-repair`. Master was deliberately NOT pushed (reason in §5.1).

---

## 0. TL;DR

The correctness hole is fixed and verified: the library's own default event payload (CBOR) now flows through `Store.ApplyEncoded` / `ApplyEncodedRecord` and through a zero-config `projectionadapter`, and decode failures classify as Corruption so a dead-letter store can quarantine them. Two of the plan's design assumptions were disproved by the RED tests and replaced with a safer design, documented in the plan's §9.2.

Four things did not go well:

1. The auto-commit daemon raced my commit; the authored commit and its detailed message never landed. HEAD is a `chore: auto-commit 55 changed file(s)` commit that also carries a **parallel session's** in-flight typed-WAL feature.
2. That foreign feature does not compile (`undefined: runSuffix`), which blocks the requested `git push` to master.
3. No ADR was written for the boundary decision, and replay of the new CBOR marker is reasoned about rather than pinned by a test.
4. Two of four commits contain dependency/`go.sum` churn authored by another session, so the history does not read as a coherent change set.

Everything else (tests, goldens, gates, docs, budget, taxonomy) is green; the remaining reds belong to other sessions' in-flight work.

---

## 1. What this session actually changed

Authored change set (21 files, ~646 insertions / 179 deletions across the codec work, plus the plan/log/docs):

| Area | Files |
| ---- | ----- |
| Core fix | `metaengine/encoded.go` (rewritten ingress + fold decode), `metaengine/stream_codec.go`, `metaengine/cursor.go`, `metaengine/consistency.go` (doc) |
| Adapter | `metaengine/projectionadapter/adapter.go` |
| Engine write paths | `metaengine/pgengine/stream_copy.go`, `metaengine/duckdbengine/stream_log.go`, `metaengine/mysqlengine/stream_log.go`, `metaengine/dgraphengine/stream_log.go` |
| Tests | `metaengine/encoded_codec_test.go` (Ginkgo, committed earlier), `metaengine/projectionadapter/adapter_default_codec_test.go` (committed earlier), `metaengine/stream_codec_test.go`, `metaengine/cursor_deterministic_test.go` |
| Wiring / goldens | `metaengine/go.mod` + `go.sum` (go-codec), `scripts/check-module-layers.sh`, `docs/error-taxonomy.md`, `docs/api_surface.txt` |
| Docs | `metaengine/README.md`, `.agents/skills/go-cqrs-lite/references/{recipes,modules,faq}.md`, `CHANGELOG.md`, the plan doc |

---

## 2. a) FULLY DONE

Each item cites the code and the proof.

1. **RED-first reproduction before touching production code.** Four test files, three of them failing for the right reason (output archived in the plan §9.1):
   - `metaengine/encoded_codec_test.go` — CBOR through `ApplyEncoded` (sniffed) and through `ApplyEncodedRecord` (stamped), plus JSON regression and non-JSON/CBOR rejection.
   - `metaengine/projectionadapter/adapter_default_codec_test.go` — JSON **and** CBOR through a nil decoder. The JSON case was the decisive one: the documented `map[string]any` default panicked every typed fold (`reflect: Call using map[string]interface {} as type ...CounterIncremented`), independent of CBOR.
   - `metaengine/stream_codec_test.go` — encode error on unrepresentable values, decode error on corrupt stored values, legacy fallback contract.
   - `metaengine/cursor_deterministic_test.go` — 64-key map cursor encoded 9 times must be byte-identical and round-trip.
2. **`ApplyEncoded` and `ApplyEncodedRecord` are codec-aware.** `metaengine/encoded.go:47` and `:70` build a pipeline marker via `encodePayload` (`:102`): JSON keeps the historical `jsontext.Value` marker (`rawJSON`, `:145`), CBOR becomes `encodedPayload{raw, encoding}` (`:93`). Auto-detection via `codec.AutoDetect`; a known `Record.Encoding` is honored (`codecEncodingOf`, `:131`). Empty payload and unknown codec fail as Corruption (`metaengine.encoded.empty_payload`, `metaengine.encoded.unknown_encoding`).
3. **Per-fold codec decode, not a schema-free transcode.** `decodeEncodedFoldPayload` (`metaengine/encoded.go:180`) resolves `codec.ForEncoding` and decodes into `reflect.New(fold sample)`; `[]byte`-sample folds get the raw bytes (`isByteSample`). This is the design correction described in §4.
4. **Corruption classification on the encoded path.** `corruptionDecodeError` (`metaengine/encoded.go:205`) wraps deterministic decode failures, so projection hosts dead-letter instead of crash-looping. This preserved the adapter's pre-existing DLQ contract.
5. **Zero-config adapter delegates instead of building a map.** `metaengine/projectionadapter/adapter.go:148-158` calls `Store.ApplyEncodedRecord` when neither `PayloadDecoder` nor `EventDecoder` is set. The two stale doc claims ("map[string]any", "zero-dependency boundary") were corrected at `:21` and `:65-70`.
6. **Stream-value silent loss closed on all write paths.**
   - New API: `EncodeStreamValueErr` (`metaengine/stream_codec.go:28`), `DecodeStreamValueErr` (`:62`); `EncodeStreamValue` (`:19`) is now a documented `// Deprecated:` wrapper with byte-identical behavior.
   - Migrations: `metaengine/pgengine/stream_copy.go` (COPY rows and multi-VALUES insert), `duckdbengine/stream_log.go` (`StreamAppend`, `StreamAppendExpected`), `mysqlengine/stream_log.go` (both), `dgraphengine/stream_log.go` (both).
   - Bonus defect found and fixed in the same functions: dgraph's `StreamAppendExpected` discarded its own `json.Marshal` error (`data, _ :=`); it now returns it.
   - Zero production callers of the deprecated `EncodeStreamValue` remain (verified with ripgrep: 0 hits outside `stream_codec.go`).
7. **Deterministic cursor bytes.** `metaengine/cursor.go:38` uses `json.Marshal(c.Value, json.Deterministic(true))`. The existing wire golden (`metaengine/cursor_wire_golden_test.go`, struct-shaped cursors) is unaffected and still green.
8. **go-codec is an explicit dependency at the right layer.** `metaengine/go.mod:8` requires `go-codec v0.3.2` as a direct dep; `DEP_BUDGET[metaengine]` raised 7 to 8 with a written rationale (`scripts/check-module-layers.sh:323-328`).
9. **Error taxonomy is honest.** Four new codes documented (`docs/error-taxonomy.md:768-773`); `scripts/check-error-taxonomy.sh` green with `pool=erraudit` after switching the two f-form wraps to the plain `WrapCorruption` form so the extractor can see them.
10. **API surface golden updated and justified.** `docs/api_surface.txt` gains exactly `metaengine/func DecodeStreamValueErr` and `metaengine/func EncodeStreamValueErr`; `cmd/api-stability` meta-tests green (`TestEvery`).
11. **Docs corrected where they were false, not just appended.**
    - `metaengine/README.md`: "JSON Event Payloads" became "Encoded Event Payloads (JSON or CBOR)" with the time.Time rationale; the toolchain section now names the codec substrate; the `ApplyEncoded` section no longer claims a JSON-only zero-copy path.
    - `.agents/skills/.../recipes.md`: the "Encoded Applies" section previously instructed readers to hand-decode CBOR; both the intro and the caveat bullets are now correct.
    - `.agents/skills/.../modules.md`: the `ApplyEncodedRecord` one-liner no longer says "takes JSON bytes".
    - `.agents/skills/.../faq.md`: new entry for the CBOR/DLQ symptom (cause, fix, and the explicit "schema-free transcode is not the fix" warning), placed next to the pre-existing `time.Time` drift entry.
    - `CHANGELOG.md`: one Fixed entry covering the ingress fix and the same-boundary sweep.
12. **Verification matrix actually executed** (details in §9): metaengine suite, projectionadapter, system, six engine modules, builds for all eight changed modules, api golden + meta-tests, layers/budgets, error taxonomy, doc-check (1349 refs), recipes compile gate, `nix fmt`, module-scoped `golangci-lint` on metaengine (3 findings, all in untouched `keycodec/` files).
13. **The plan was reconciled, not rewritten.** §9 of the plan records the RED output verbatim, the two design corrections, what shipped per task, the gate results, and the pre-existing reds. The original design text is intact.

---

## 3. b) PARTIALLY DONE

1. **Push.** Work is durable on `origin/agent/metaengine-codec-boundary-repair`, but `master` is NOT pushed. See §5.1.
2. **Commit message quality.** The requested "very detailed commit message" did not land. The daemon committed my staged tree (plus foreign files) as `chore: auto-commit 55 changed file(s) (heuristic)` before my `git commit` acquired the ref. Amending it now would require force-pushing a branch that is already public, which needs your approval.
3. **Dependency alignment review.** `go-codec v0.3.2` declares `go-error-family v0.12.0`, so adding go-codec to `metaengine` forced `metaengine` from `v0.11.0` to `v0.12.0`. I verified it compiles and the suites are green, but I did not review what changed between those error-family versions, and I briefly churned through `v0.12.0 → v0.11.0 → v0.12.0` while trying to avoid the bump. The final state is correct; the process was sloppy.
4. **"No persisted bytes change" is verified by reading, not by a pin.** I know the JSON path still produces `jsontext.Value` and that the CBOR marker never reaches an engine (it is decoded per fold before the engine call), but there is no runtime test asserting that engine rows / EventLog payloads are byte-identical to pre-change behavior. The repo has exactly this discipline for `system` (`system/wal_raw_bytes_pin_test.go`); metaengine does not.
5. **Read-side `DecodeStreamValue`.** Deliberately left lossy (11 production files still use it). It is documented and its fallback is pinned by one test, but the corruption-hiding behavior remains in every read path.
6. **Engine-level error-propagation proof.** I migrated the write sites and built/tested the modules, but no test drives each engine's `StreamAppend` with an unrepresentable value to prove the error now surfaces. The new unit tests cover the helpers, not the call sites.
7. **`docs/WIRE-FORMAT-KEYS.md` (plan T7.5).** Correctly skipped: that document owns the aggregate→stream key renames. It has no payload-codec section, and the change does not touch those keys. Recorded here so the omission is a decision, not an oversight.

---

## 4. c) NOT STARTED

1. **ADR for the boundary decision.** The repo's planning discipline wants an ADR when a dated design premise is revisited. ADR-0062 ("zero-dependency core") explicitly rejected external codecs for metaengine, and ADR-0066 rejected stamped engine values; the encoded-ingress decision (per-fold codec decode, schema-free transcode rejected) is exactly the kind of reversal that should be recorded. Not written.
2. **Replay test for the new CBOR marker.** `encodedPayload` is unexported and travels through `EventInput.Payload` (documented at `metaengine/consistency.go:19-25`). `Verify`, `Backfill`, `DemoteEngine` catch-up and `failover.go` all pass it through `Apply`/`dispatchFolds`, which the type switch handles. I reasoned this through; nothing pins it. A test that runs `WithEventLog` + `Verify` over a CBOR-encoded apply is the missing proof.
3. **Tests for the remaining contract edges:**
   - `ApplyEncodedRecord` with `record.EncodingUnknown` + CBOR bytes (the sniff-through-record path; my test used the explicit CBOR stamp).
   - A lying stamp (JSON stamp, CBOR bytes) - documented as "surfaces as a decode error", not pinned.
   - Empty payload (`metaengine.encoded.empty_payload`) and unknown codec (`metaengine.encoded.unknown_encoding`).
   - `[]byte`-sample folds receiving raw CBOR bytes.
   - `codecEncodingOf` value table.
   - A payload-shape matrix for `AutoDetect` (map, array, text string, int, float, bool, null) - my CBOR tests only exercise structs/maps.
4. **Patch-level pin that stored bytes are unchanged** (see §3.4).
5. **Performance evidence.** The encoded ingress now does a type switch plus a `reflect.New` + codec decode per fold. For JSON the decode is the same `encoding/json/v2` call as before; for CBOR it is new work on a path that previously failed outright. No benchmark, no allocation measurement, no comparison against `Apply(struct)`.
6. **Reader-side hardening** (see §3.5).
7. **Repo-wide gates:** no `nix run .#verify` / `#verify-fast` / `#verify-ci`, no `nix run .#lint`, no `#check-coverage`, no `#check-api-stability` with `-race`, no `#check-modsums`. I ran the targeted equivalents only.
8. **`system`-level proof of the adapter default path.** `system` always supplies a decoder (`system/projection_wiring.go` priority chain), so the nil-decoder delegation is only proven by the projectionadapter unit test, not through a `system.New` boot.
9. **Cross-module search for JSON-only assumptions in consumer decoders** (`projectionadapter.EventWithID`, examples, docs snippets). Only the two repositories' reference docs were checked.
10. **Engine-value envelope (ADR-0066 reversal), custom/encrypted codec support for read-model values, `reify` replacement.** Deferred by design, still deferred.

---

## 5. d) TOTALLY FUCKED UP

### 5.1 The push is blocked by a foreign, non-compiling file inside my HEAD

`git status` was clean at the end, but HEAD contains another session's in-flight typed-WAL feature:

```
metaengine/pgengine/typed_wal_test.go:209:27: undefined: runSuffix
```

Reproduced twice, minutes apart, and the file's mtime is frozen at 18:55 while the session ran past 05:27. `go build ./...` passes (tests are excluded from build), which is why the pre-commit hook's "build all modules" gate did not catch it.

Consequences:

- `git push origin master` would publish a commit whose `pgengine` test package does not compile. I refused.
- The safety branch `origin/agent/metaengine-codec-boundary-repair` exists, but it publishes the same foreign WIP.
- I cannot fix the file (I did not author it, I do not know what `runSuffix` is supposed to be, and the owning session may be mid-edit).

### 5.2 The auto-commit daemon beat my commit, twice

- I staged exactly 21 files; the daemon committed 55 (mine + foreign) before my `git commit` could take the ref: `fatal: cannot lock ref 'HEAD': is at 20a95af2c...`.
- Earlier in the session the daemon also auto-committed my intermediate `metaengine/encoded_normalize.go` (with exported `NormalizeEncoded`/`NormalizeEncodedWithEncoding`), which I then deleted. Those symbols existed in local history for a while and were never in the API golden. Churn, not a correctness problem, but it means an exported API surface was briefly "shipped" that never should have existed.

### 5.3 Git hygiene mistake

`git push -u origin HEAD:refs/heads/agent/...` re-pointed local `master`'s upstream to the side branch. I fixed it with `git branch --set-upstream-to=origin/master master`, but the initial invocation was careless (the `-u` was unnecessary).

### 5.4 Churned the dependency graph while trying to avoid a bump

`go get go-codec@v0.3.2` upgraded error-family to v0.12.0; my attempt to pin it back silently downgraded go-codec to v0.3.1; I then set both explicitly. Net effect is right, but I should have checked go-codec's own `go.mod` (which requires v0.12.0) before attempting the pin.

### 5.5 Claimed a design in the plan that the tests disproved

The plan's T2/T3 prescribed CBOR→JSON transcoding via `codec.TranscodeToJSON`. I implemented it, ran the tests, saw `time.Time` fail, and had to redesign mid-flight. Writing the plan one step earlier (a five-line probe, which I had already done in a scratch module for `AutoDetect`) would have caught it before the plan was committed. The RED evidence ultimately made this a strength, but the plan was wrong on paper for an hour.

---

## 6. e) WHAT WE SHOULD IMPROVE

Direct answers to the three questions in the prompt:

### What did I forget?

1. **An ADR.** The boundary reversal is undocumented as a decision (§4.1).
2. **A replay test.** The marker's journey through `Verify`/`Backfill`/`DemoteEngine` is reasoned, not executed (§4.2).
3. **Record-stamp sniff test.** I tested the explicit `EncodingCBOR` stamp and the no-stamp `ApplyEncoded` path, but not "unknown stamp + sniff" through `ApplyEncodedRecord`.
4. **A byte-level "stored bytes unchanged" pin**, which is this repo's standard for wire claims (§3.4).
5. **A `[]byte`-sample CBOR test** - the `isByteSample` branch of the new decoder is untested.
6. **The performance question.** I added a decode path and never measured it.
7. **Phase-boundary commits.** AGENTS.md says the daemon will absorb mid-phase edits; I relied on one big staged commit and lost it.
8. **That `metaengine/sqliteengine/encoding.go`'s `fmt.Sprintf("%v", v)` fallback** is the same silent-loss class I just fixed elsewhere. Spotted during research, left untouched (out of slice), not even recorded in the plan.

### What could I have done better?

1. **Probe the transcode behavior for `time.Time` before writing the plan**, not after implementing it.
2. **Commit the RED tests as their own commit** (they passed the gate and prove the bug) instead of leaving them in a 13-file auto-commit.
3. **Write the replay test alongside the marker**, because the marker is the riskiest part of the design (an unexported type crossing public `EventInput.Payload`).
4. **Fix the error-code extraction mismatch first** (the `WrapCorruptionf` blind spot) instead of writing the code with the f-form and then rewriting it for the gate.
5. **Ask the daemon question up front** (see §8) rather than discovering it mid-commit.
6. **Surgical baseline work**: the two "new" `DEP_BUDGET`/taxonomy edits required discovering that the taxonomy doc was already stale for an unrelated reason (`ErrNoDatabase` rows). I chose a surgical doc edit over regenerating; that was right, but I only learned it by running the generator twice.
7. **Use `go vet` per module before declaring the matrix green**; `go test` covers most of it, but the foreign breakage shows vet/test compile state is the real signal.

### What could I still improve?

1. **Unify the marker.** Two markers (`jsontext.Value` and `encodedPayload`) are defensible for compatibility but are a future split brain; either collapse them later or document the invariant prominently.
2. **Share the per-engine encode block.** Four engines now repeat a six-line `encode + wrap` idiom with slightly different context strings; an `EncodeStreamValuesErr([]any) ([]string, error)` helper in metaengine would remove the duplication and the message drift (`art-dupl` did not flag it at threshold 3, but the duplication is real).
3. **Make the read-side fallback explicit.** `DecodeStreamValue` hides corruption by design. A `DecodeStreamValueOk(s) (any, bool)` or a migration plan for the 11 call sites would let the engines distinguish "plain string row" from "corrupt row" without a breaking signature change.
4. **Publish the decision to consumers**, not just the reference docs: the `metaengine` module README's "Encoded Event Payloads" section is the main 500-char window a consumer reads; it now says the right thing, but `projectionadapter`'s own README (if any) and `system`'s docs still only imply it.
5. **Give the Corruption classification a test that walks the family**, not just the message (`errorfamily` classification through the `ApplyError` chain), so a future refactor cannot silently downgrade a deterministic decode failure to Transient.

---

## 7. f) Up to 50 things to get done next

Ordered roughly by blocking-ness then value. `[F]` = foreign/parallel-session owned.

**Unblock the push**

1. `[F]` Define (or remove the use of) `runSuffix` in `metaengine/pgengine/typed_wal_test.go:209`, then `GOWORK=off go vet ./...` in that module.
2. `[F]` Run the mysql/pg typed-WAL suites to green (`-short` at minimum).
3. Push `master` once HEAD compiles (`git push origin master`).
4. Decide whether `origin/agent/metaengine-codec-boundary-repair` should be deleted after the master push.
5. Re-word HEAD's commit message if the history is worth repairing (needs your approval; would require a force-push of the branch).

**Prove the risky parts of this change**

6. Replay test: `WithEventLog` + CBOR `ApplyEncoded` + `Verify` against a fresh memory store.
7. Replay test: `Backfill`/`DemoteEngine` catch-up with a `encodedPayload` in `EventInput`.
8. Test `ApplyEncodedRecord` with `EncodingUnknown` + CBOR bytes (sniff through the Record API).
9. Test a lying stamp (JSON stamp + CBOR bytes) and pin the "the stamp is trusted" contract.
10. Test the empty-payload and unknown-codec Corruption codes (assert `errors.Is` family, not just the message).
11. Test a `[]byte`-sample fold receiving raw CBOR bytes.
12. Table test `codecEncodingOf` for all four `record.Encoding` values.
13. Payload-shape matrix for `AutoDetect` with CBOR map/array/text/int/float/bool/null.
14. Assert the Corruption family survives the `ApplyError` chain (`errorfamily` classification test).

**Prevent the next regression**

15. Byte-level pin: engine rows + EventLog payload for a JSON `ApplyEncoded` call are unchanged by this commit (mirror `system/wal_raw_bytes_pin_test.go`).
16. Per-engine error-propagation test: `StreamAppend` with a `chan` value must fail, not write `""` (pg, duckdb, mysql, dgraph).
17. Add `art-dupl`/lint-friendly shared helper `EncodeStreamValuesErr` and migrate the four engines.
18. Consider a unified pipeline marker (collapse `jsontext.Value` + `encodedPayload`) with a replay-compat test.
19. Fuzz `encodePayload` with arbitrary bytes (must never panic; must return JSON or Corruption).

**Close the documented gaps**

20. Write the ADR for the codec boundary decision (supersede the codec silence in ADR-0062/0066).
21. Add the `metaengine/sqliteengine/encoding.go` `fmt.Sprintf("%v")` fallback to TODO_LIST (same silent-loss class).
22. Add read-side hardening plan for the 11 `DecodeStreamValue` call sites (TODO_LIST row).
23. Document the unexported-marker limitation on `EventInput.Payload`: external backfill from raw CBOR bytes must decode first.
24. Consider exporting an `EncodedPayload` constructor if external raw-CBOR replay is needed.
25. Add a `system`-level test that a CBOR event with no `ProjectionDecoder`/`EventDecoder` reaches a read model.
26. Check `projectionadapter.EventWithID` and example decoders for JSON-only assumptions.
27. Cross-link the new FAQ entry from the `time.Time` drift entry (and back).

**Measure instead of assert**

28. Benchmark `ApplyEncoded` JSON vs CBOR vs `Apply(struct)` (ns/op, allocs) and record the numbers in the plan or README.
29. Check whether the CBOR path's `reflect.New` per fold can reuse the sample type's converter (no action unless the numbers justify it).
30. Add a `benchkit` (or `metaengine/bench`) reference workload for encoded ingress.

**Finish the verification matrix**

31. `nix run .#verify-fast`.
32. `nix run .#verify` (or `scripts/verify-chain.sh`) once the foreign breakage is gone.
33. `nix run .#lint` repo-wide.
34. `nix run .#check-coverage` (metaengine delta).
35. `nix run .#check-api-stability` (the `-race` variant).
36. `nix run .#check-modsums` after the go-codec/error-family bump.
37. `nix run .#check-file-size` and `.#check-duplication` after the foreign fixes land (both currently red for unrelated files).
38. Run the `pgengine`/`mysqlengine` integration suites for the typed-WAL feature once it compiles (`nix run .#integration-pg`, `...#integration-mysql`).

**Tidy the record**

39. Add a CHANGELOG line for the `EncodeStreamValue` deprecation (removal at v5) if the Fixed entry is not enough.
40. Update the plan's §6 verification matrix to point at §9.4 rather than leaving the aspirational table as the only matrix.
41. Record the `error-family v0.11.0 → v0.12.0` bump (forced by go-codec) in the CHANGELOG or the module docs.
42. Append today's status link to `docs/status/README.md`.
43. Consider regenerating `docs/error-taxonomy.md` fully (it is stale for `ErrNoDatabase` rows) as its own small change.

**Wider codec hygiene (same class, outside this slice)**

44. Audit other `json.Marshal` error discards (`rg 'json.Marshal\(' | grep ', _ :='`) in metaengine.
45. Audit `json.Unmarshal` silent fallbacks in engine read paths.
46. Decide whether `metaengine` should expose an `Encoding`-aware `ApplyTyped[T]` for consumers who already know the type.
47. Consider whether `stream_codec` should store a per-row encoding stamp instead of assuming JSON (the ADR-0066 reversal question).
48. Add `go-codec` to the metaengine README's dependency list if one is maintained.
49. Check `cqrs-lint` for rules that assume JSON-only encoded ingress.
50. Add a short "codec contract" section to `metaengine/COOKBOOK.md` (the skill points consumers at it; it currently says nothing about encodings).

---

## 8. g) Three questions I cannot answer myself

1. **Who owns `metaengine/pgengine/typed_wal_test.go` (and the typed-WAL feature), and may I define the missing `runSuffix` helper?** I will not edit another session's in-flight test on a guess, and it is the only thing blocking `git push origin master`.
2. **Should the auto-commit daemon be paused (or is there a switch) while an authored commit is being prepared?** If not, the working practice must be "stage and commit immediately at each phase boundary", which changes how I sequence future work here.
3. **Is `metaengine` allowed to move to `go-error-family v0.12.0`?** It is forced by `go-codec v0.3.2`; the alternative is pinning go-codec to v0.3.1, which would put metaengine behind `event/` on the codec substrate. I picked alignment, but that is a dependency-policy call, not a code call.

---

## 9. Evidence appendix

### 9.1 RED before production changes

```
[FAIL] ApplyEncoded payload codecs [It] applies a CBOR payload with a sniffed encoding
  [corruption:metaengine.apply.encoded_decode] decode TaskCreated:
  json decode into TaskCreated: json: unable to unmarshal JSON number into Go time.Time within "/At"

[FAIL] TestAdapter_DefaultDecode_Codecs/json
  collection poisoned by fold panic:
  reflect: Call using map[string]interface {} as type projectionadapter_test.CounterIncremented

[FAIL] TestAdapter_DefaultDecode_Codecs/cbor
  [corruption:projectionadapter.decode_payload] jsontext: invalid character '\xa2' at start of value
```

The `time.Time` failure is the one that killed the planned transcode design: canonical CBOR (`EncOptions.Time = TimeUnixDynamic`) writes an untagged epoch integer, so schema-free JSON transcoding cannot round-trip it.

### 9.2 Final green matrix

| Gate | Command | Result |
| ---- | ------- | ------ |
| metaengine | `GOWORK=off go test ./... -count=1` | ok (full suite, 29-50 s) |
| projectionadapter | `GOWORK=off go test ./... -count=1` | ok |
| system | `GOWORK=off go test -short ./... -count=1` | ok |
| pg / duckdb / mysql / dgraph / sqlite / bigtable | `GOWORK=off go test -short ./... -count=1` | ok (before the foreign typed-WAL commit; pg is now red for that foreign file) |
| builds | `GOWORK=off go build ./...` x8 modules | ok |
| api surface | `go run . --update` + `go test -run TestEvery .` | ok, +2 exports |
| layers | `scripts/check-module-layers.sh` | ok |
| taxonomy | `scripts/check-error-taxonomy.sh` | ok (542 codes, pool=erraudit) |
| doc refs | `cmd/doc-check …` | ok (1349 refs) |
| recipes | `go test -run TestRecipes ./cmd/doc-check` | ok |
| formatting | `nix fmt` | applied (10 files changed, including 4 foreign formatting drifts left unstaged) |
| lint | `golangci-lint run ./...` in metaengine | 3 findings, all pre-existing in `keycodec/` |

### 9.3 Pre-existing reds observed, not touched

- `check-file-size`: `system/adapter_event.go` 351 lines, `metaengine/layout.go` 360 lines (uncommitted foreign edits at the time), `metaengine/sqliteengine/engine.go` 663 → 668 against its ratchet.
- `check-duplication`: 4 new clone groups, all outside this change (`system/adapter_{command,event}.go`, `system/wal_raw_bytes_pin_test.go` vs `queue/conformance/facthelpers.go`, a templ signature group, `metaengine/keycodec/escape.go`).
- `metaengine/pgengine/typed_wal_test.go`: build failure, foreign (see §5.1).

### 9.4 Local history as it actually happened

```
20a95af2c chore: auto-commit 55 changed file(s)   <- my final tree + foreign typed-WAL + fmt drift
d0e86a117 chore: auto-commit 53 changed file(s)   <- foreign dependency sweep
1271b2d74 chore: auto-commit 13 changed file(s)   <- my RED tests + intermediate normalize design
9beae27c5 chore: auto-commit 2 changed file(s)    <- my metaengine go.mod/go.sum
```

None of these carry an authored message; the intended `fix(metaengine): decode encoded applies with the payload's real codec` never committed. The full rationale lives in the plan (§9) and the CHANGELOG entry instead.
