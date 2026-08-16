# Status Report: Metadata Roundtrip Fix, Graph ADT, and CI Failure Triage

> **ARCHIVED 2026-08-11 — All work in this report is complete. Open items were resolved by later sessions, captured in TODO_LIST.md, or determined to be minor polish. Original content retained below for historical context.**

**Date:** 2026-08-10 18:49
**Session window:** ~16:50 – 18:49
**Starting point:** User provided a Phase 1 checklist with 7 sub-items covering Record consolidation (ADR-0111), graph ADT, branded-ID stamping, signing golden, metadata roundtrip, cqrs-lint findings, and `nix run .#verify`.

---

## Executive Summary

The status reports from earlier today (`16-14` and `16-15`) were **substantially stale** — they described failures that had already been fixed by the auto-commit daemon and prior session work. Of the 7 checklist items, **only 2 had real failures** when I ran the tests: the pebble/bbolt metadata roundtrip (ActorID lost in CBOR) and the cqrs-lint golden profile (V003 + C017 new findings). I fixed the metadata roundtrip root cause (a CBOR-vs-JSON marshaling incompatibility for `id.ActorID`), and the auto-commit daemon landed it. The cqrs-lint golden remains the sole outstanding test failure.

**The biggest lesson:** I should have run the tests FIRST before reading the status reports and forming a plan. Instead I spent research cycles investigating 7 issues, 5 of which were already resolved.

---

## a) FULLY DONE

### 1. Pebble/Bbolt Metadata Roundtrip Fix (root cause found + fixed + committed)

**Root cause:** `id.ActorID` is a struct with unexported fields (`kind`, `raw`) that implements `encoding/json.Marshaler` but NOT `cbor.Marshaler`. When `serializableEvent` was CBOR-encoded by fxamacker/cbor, reflection could not see the unexported fields, so ActorID was encoded as an empty map `{}` and decoded back as the zero value. This silently destroyed UserID/ActorID during serialization in both `storage/pebble` and `storage/bbolt`.

**Fix applied (both stores):** Introduced a `metadataPayload` type (`[]byte` wrapper) that stores event metadata as JSON bytes INSIDE the CBOR envelope. This ensures types implementing `json.Marshaler` (like `id.ActorID`) serialize correctly through the only code path they support. The `MarshalCBOR`/`UnmarshalCBOR` methods on `metadataPayload` include a backward-compatibility fallback: if the CBOR data doesn't decode as a byte string (legacy format where metadata was a CBOR map), it falls back to reflecting into `event.Metadata` and re-marshaling to JSON.

**Files changed:**

- `storage/pebble/serialization.go` — new `metadataPayload` type, updated `serializeEvent`/`deserializeEvent`/`serializableEvent`
- `storage/bbolt/serialization.go` — identical changes (cross-module duplicate, `//art-dupl:accept` tagged)
- `storage/pebble/cbor_test.go` — added ActorID assertion to `TestDeserializeEvent_CBORWithMetadata`, fixed `TestSerializeEvent_SmallerThanJSON` to include metadata in both CBOR and JSON comparison structs

**Committed by auto-commit daemon** in `74b5762e2` ("feat(metaengine): ship live-latency model P1/P2/P3, store stats, and metadata JSON encoding").

**Test verification:** `go test -tags "goexperiment.jsonv2" ./storage/pebble/ ./storage/bbolt/ -count=1 -timeout 300s` → **PASS** (all tests including `TestEventStore_MetadataRoundtrip`, `TestContract_MetadataRoundtrip`, `TestDeserializeEvent_CBORWithMetadata` with ActorID assertion).

### 2. Metaengine Graph ADT / Planner / Replication Tests (already fixed — verified)

The prior status report (`16-15`) claimed 15 failing Ginkgo specs for graph ADT and a replication RTT test failure. When I ran the tests, **all 145 specs passed**. The auto-commit daemon had already committed:

- `metaengine/memory_graph.go` — restored graph support to the memory engine (ADR-0113 post-fix)
- `metaengine/rule_replication_test.go` — updated RTT assertion to match new display format (`rtt=prior 5ms`)
- `metaengine/planner.go`, `metaengine/engine.go` — planner fixes

**Verification:** `go test -tags "goexperiment.jsonv2" ./metaengine/ -count=1 -timeout 300s` → **PASS** (145/145 specs, including `TestExplainPlan_ShowsReplicationForReplicatedEngine`).

### 3. Branded-ID Auto-Fold Stamping (already fixed — verified)

The prior status report claimed `TestAutoFold_RecordAware_Insert` and `TestIntegration_AutoInsert_ThroughAdapter` were failing due to a branded-ID stamping panic. When I ran both tests, **they passed immediately**. The `metaengine/record_stamp.go` code already calls `.String()` on branded ID types before reflect-stamping them into `string` fields. The fix was already applied.

### 4. Signing Golden Snapshot (regenerated — uncommitted)

The signing golden test (`TestGolden_HMACSignedEvent`) was failing because the snapshot still referenced `"userId": null` (pre-ADR-0111 field name) instead of the new `"actorId"` field and the new CommonMetadata timestamp fields (`clientCreatedAt`, `serverReceivedAt`, `serverStoredAt`, `schemaVersion`).

**Action taken:** Regenerated the snapshot via `UPDATE_SNAPS=true go test`. The updated `signing/testdata/golden/hmac-signed-metadata.snap` is in the working tree (uncommitted — 2 uncommitted files total).

**Verification:** `go test -tags "goexperiment.jsonv2" ./signing/ -count=1` → **PASS** (12/12 specs, 1 snapshot passed).

### 5. ADR-0111 Record Consolidation (already complete — verified)

Investigation confirmed that ADR-0111 Phases 1-4 are **all done at the type level**:

- Phase 1: `record.Record` + `record.CommonMetadata` exist in `record/record.go` ✅
- Phase 2: metaengine fold handlers receive typed `Record` ✅
- Phase 3: `event.Metadata` and `command.Metadata` both embed `record.CommonMetadata`; `metadata.Tracing` type is deleted ✅
- Phase 4: `event.Metadata` has no Tombstone field; tombstones are domain events (ADR-0114) ✅

**One stale doc found:** `metadata/doc.go` still documents the removed `Tracing` type and describes `CustomData` as embedding `Tracing`. This is a documentation-only issue, not a code issue.

---

## b) PARTIALLY DONE

### cqrs-lint Golden Profile (V003 + C017)

**Status:** Root cause identified, golden file updated in working tree, but test still fails.

The `TestIntegration_TaskmanagerExpectedFindings` test detects 2 new findings not in the golden profile:

- **V003**: `metaengine/sqliteengine/v4 is on v4.0.x — 3 minor versions behind latest (v4.3.x)` — the version pinning rule correctly detects that sqliteengine is behind other modules
- **C017**: `In-memory dead-letter store paired with persistent event store (sqlite) — lost on restart` — the correctness rule correctly detects a durability mismatch

The golden file `cmd/cqrs-lint/testdata/taskmanager_golden.txt` has been updated in the working tree (the `+` lines show both findings added), but the test still fails with "NEW finding not in golden profile." This suggests the test reads the golden from a different source (possibly a compiled profile, not the txt file), or the update was incomplete. Running with `CQRS_LINT_UPDATE_GOLDEN=1` would regenerate the canonical profile.

**These findings are legitimate** — they correctly identify real issues in the example/taskmanager project (version drift + durability mismatch). The fix is to accept them into the golden profile, not suppress them.

---

## c) NOT STARTED

The following items from the checklist were **not worked on** because they were either already done or outside the scope of this session's focus on test failures:

1. **`nix run .#verify` end-to-end** — NOT RUN. This is the full CI gate (build + vet + test + race + lint + doc-check + doc-assertions). I ran individual module tests but not the full nix pipeline.
2. **`nix fmt`** — NOT RUN. The formatting gate has not been executed.
3. **`nix run .#lint`** — NOT RUN.
4. **`nix run .#check-duplication`** — NOT RUN. The new `metadataPayload` type is a cross-module duplicate between pebble and bbolt (tagged with `//art-dupl:accept`).
5. **`nix run .#check-arch`** — NOT RUN.
6. **`metadata/doc.go` update** — The stale doc referencing the removed `Tracing` type was identified but not fixed.

---

## d) TOTALLY FUCKED UP

### Stale Status Reports Caused Massive Wasted Research

The two prior status reports (`16-14` and `16-15`) described 7 failing areas. When I actually ran the tests:

- 3 were **already fixed** (graph ADT, branded-ID stamping, replication RTT)
- 1 was **already fixed at type level** (ADR-0111 consolidation)
- 1 was **a real failure I fixed** (pebble/bbolt metadata roundtrip)
- 1 was **a golden regen** (signing — trivial)
- 1 is **still failing** (cqrs-lint golden)

I spent 3 parallel agent investigations researching issues that were already resolved. The status reports were point-in-time snapshots from a prior session, and the auto-commit daemon + other work had moved the codebase forward between sessions. **"Status reports are point-in-time, not living documents"** — this is even documented in the global AGENTS.md, and I still fell for it.

### The `metadataPayload` Approach Has a Hidden Cost

The fix stores metadata as a JSON byte string inside CBOR. This means:

- **Every event write** now does a JSON marshal of metadata + a CBOR marshal of the envelope (2 passes instead of 1)
- **Every event read** does a CBOR unmarshal + a JSON unmarshal of metadata (2 passes instead of 1)
- The CBOR envelope is slightly larger (JSON bytes are longer than native CBOR map encoding for simple fields)

The `TestSerializeEvent_SmallerThanJSON` test had to be updated to include metadata in BOTH structs for a fair comparison. This is acceptable — correctness (ActorID survives) beats micro-optimization — but it should be documented as a known tradeoff.

---

## e) WHAT WE SHOULD IMPROVE

### Process Improvements

1. **Run tests BEFORE reading status reports.** Always. The status report is a snapshot; the tests are ground truth. I should have run the full test suite first, identified the REAL failures, then read the reports for context on items that actually failed.
2. **Stop trusting auto-commit daemon messages.** The daemon commits sweep up in-flight work from multiple agents. Commit `74b5762e2` contains my pebble/bbolt fixes PLUS an entire live-latency model (probe.go, latency.go, engine_stats.go) that I didn't write. The commit message doesn't distinguish who did what.
3. **The `metadataPayload` duplication between pebble and bbolt should be extracted.** Both stores have identical `metadataPayload` types with identical `MarshalCBOR`/`UnmarshalCBOR`/`MarshalJSON`/`UnmarshalJSON` methods. This is tagged `//art-dupl:accept` but it's a prime candidate for extraction into `event/` or a shared `storage/encoding/` helper.
4. **`id.ActorID` should implement `cbor.Marshaler`.** The root cause was that ActorID only implements `json.Marshaler`. The real fix (architecturally) is for `id.ActorID` to also implement `cbor.Marshaler`/`cbor.Unmarshalmer` so it works with ANY encoder, not just JSON. The `metadataPayload` workaround is a store-level patch; a type-level fix would be more robust. However, this would add a `fxamacker/cbor` dependency to the `id/` module (currently zero-dep), which violates dep budgets.
5. **The cqrs-lint golden test should explain itself better.** The failure message says "not in golden profile" but the golden txt file has the entries. The test should clarify whether it reads from the txt file or a compiled profile.

### Code Quality

6. **`metadata/doc.go` is stale** — references deleted `Tracing` type. Should be updated to document `record.CommonMetadata` embedding.
7. **The backward-compat fallback in `UnmarshalCBOR`** does a reflection-based decode of legacy CBOR metadata maps. This path is untested for the ActorID case (legacy data written before the fix would have empty ActorID). A migration test would be valuable.
8. **No test for the metadata roundtrip with ALL CommonMetadata fields populated** — the existing test checks CorrelationID, CausationID, and ActorID, but not RequestID, ClientCreatedAt, ServerReceivedAt, ServerStoredAt, or SchemaVersion.

---

## f) Up to 50 Things to Get Done Next

### P0 — CI Gate (BLOCKING)

1. **Run `CQRS_LINT_UPDATE_GOLDEN=1` to regenerate cqrs-lint golden profile** — accept V003 + C017 findings as legitimate
2. **Run `nix fmt`** on the entire repo — `//nolint` directives may be displaced by golines reformatting
3. **Run `nix run .#lint`** and fix all findings
4. **Run `nix run .#check-duplication`** — update `.art-dupl-baseline.json` if the `metadataPayload` type triggers new clone detection
5. **Run `nix run .#check-arch`** — verify dependency budgets are not exceeded
6. **Run `nix run .#verify` end-to-end** — the full CI gate (build + vet + test + race + lint + doc-check + doc-assertions)
7. **Commit the 2 uncommitted golden files** (`signing/testdata/golden/hmac-signed-metadata.snap` + `cmd/cqrs-lint/testdata/taskmanager_golden.txt`)

### P1 — Correctness & Robustness

8. **Add a comprehensive metadata roundtrip test** that populates ALL `CommonMetadata` fields (CorrelationID, CausationID, ActorID, RequestID, ClientCreatedAt, ServerReceivedAt, ServerStoredAt, SchemaVersion) and verifies roundtrip through pebble + bbolt
9. **Add a legacy CBOR backward-compat test** that decodes pre-fix CBOR data (where metadata was a CBOR map) and verifies the fallback path works
10. **Fix `metadata/doc.go`** — remove references to deleted `Tracing` type, document `record.CommonMetadata` embedding
11. **Consider implementing `cbor.Marshaler` on `id.ActorID`** (or document why the `metadataPayload` workaround is preferred to avoid adding cbor dep to `id/`)
12. **Run the bbolt backup test suite** (`storage/backuptest/`) to verify the serialization change doesn't break backup/restore
13. **Run the pebble backup test suite** — same concern
14. **Verify the `storage/eventstore/` SQL stores don't have the same ActorID bug** — SQL stores may serialize metadata differently (as JSON text column, which should be fine, but verify)
15. **Run `-race` on all affected modules** — `go test -tags "goexperiment.jsonv2" -race ./storage/pebble/ ./storage/bbolt/`

### P2 — Documentation

16. **Update AGENTS.md Gotchas section** with the CBOR/JSON marshaling incompatibility for types with unexported fields
17. **Document the `metadataPayload` pattern** in the storage skill references
18. **Update TODO_LIST.md** — mark the metadata roundtrip item as done
19. **Update CHANGELOG.md** — document the metadata serialization fix
20. **Write an ADR** for the metadata-in-CBOR-as-JSON decision (or amend ADR-0111)

### P3 — Architecture / Refactoring

21. **Extract `metadataPayload` into a shared package** (`event/metadata_payload.go` or `storage/encoding/`) to eliminate the pebble/bbolt duplication
22. **Audit ALL types in `record.CommonMetadata` and `event.Metadata`** for CBOR compatibility — any struct with unexported fields implementing only `json.Marshaler` has the same bug
23. **Consider a `cbor.Marshaler` interface check** in a test that fails if any type in `id/` or `record/` implements `json.Marshaler` but not `cbor.Marshaler`
24. **Review the metaengine live-latency model** (probe.go, latency.go, engine_stats.go) that was committed by the daemon — understand what it does and whether it needs tests
25. **Run `nix run .#vulncheck`** — per-module standalone build check

### P4 — Test Coverage Gaps (from prior session reports)

26. **bboltengine: persistence_test.go** — test data survives close/reopen
27. **bboltengine: restart_safety_test.go** — test seq counter seeding on restart
28. **bboltengine: calibration_bench_test.go** — benchmark cost model calibration
29. **mysqlengine: stream_log_test.go** — test StreamLogBackend append/scan
30. **mysqlengine: pushdown_test.go** — test JSON path pushdown queries
31. **mysqlengine: explain.go** — add ExplainableScan/ExplainableAggregate
32. **tursoengine: record_stamp_test.go** — test Record stamping
33. **tursoengine: soak_autocrud_test.go** — soak test
34. **tursoengine: healthcheck_test.go** — health check test

### P5 — Polish

35. **Run `cd cmd/api-stability && GOWORK=off go run main.go -update`** — regenerate API surface golden (may have drifted)
36. **Run `cd cmd/doc-check && GOWORK=off go run . ../../SKILL.md ../../.agents/skills/go-cqrs-lite/references/*.md ../../AGENTS.md`** — verify docs
37. **Update SKILL.md module map** with any new modules
38. **Update `.agents/skills/go-cqrs-lite/references/modules.md`** with new entries
39. **Run `go mod tidy` on all affected modules** to ensure go.sum files are clean
40. **Check if the `encoding/json/v2` migration** affected any other serialization paths (SQL stores, snapshot stores, command stores, query stores)
41. **Add a cross-store serialization contract test** that ALL stores (memory, pebble, bbolt, SQL) preserve ALL metadata fields
42. **Review the `isCBOR` detection function** — it checks for major type 5 (0xa0-0xbf), but a CBOR byte string (major type 2) starts with 0x40-0x5b. The metadataPayload bytes inside the envelope are stored as CBOR byte strings. Verify the top-level detection still works.
43. **Consider whether `metadataPayload.MarshalCBOR` should use `codec.CBOREncMode()`** directly instead of the local `marshalCBOR` wrapper (consistency)
44. **Audit the `UnmarshalCBOR` fallback path** — if `unmarshalCBOR(data, &jsonBytes)` succeeds but the result isn't valid JSON, the fallback to struct reflection is skipped. Add error handling.
45. **Run the full integration test suite** (`nix run .#test-integration`) to verify pebble/bbolt serialization with real workloads
46. **Check if the `graphadapter` engine needs the same metadata serialization fix** (it wraps MemoryDriver, probably doesn't serialize events)
47. **Verify the `storage/migrations/` embedded SQL** doesn't have metadata-related issues
48. **Run `scripts/check-coverage.sh`** to check for coverage drift in the modified files
49. **Review whether the `metadataPayload` type should implement `TextMarshaler`/`TextUnmarshaler`** for SQL text columns
50. **Consider a fuzz test** for the CBOR serialize/deserialize roundtrip with random metadata field combinations

---

## g) Questions I CANNOT Answer Myself

1. **Should `id.ActorID` implement `cbor.Marshaler`?** This would fix the root cause for ALL stores (not just pebble/bbolt), but it adds a `fxamacker/cbor` dependency to the `id/` module, which is currently zero-dependency and Tier 0. The alternative is the per-store `metadataPayload` workaround I implemented. Which approach do you prefer — type-level fix (adds dep) or store-level workaround (duplication)?

2. **Are the V003 and C017 cqrs-lint findings legitimate issues to fix in `example/taskmanager`, or should they be accepted into the golden profile as-is?** V003 flags sqliteengine version drift (v4.0.x vs v4.3.x). C017 flags an in-memory dead-letter store paired with a persistent event store. Both seem like real issues in the example code, but fixing them may require tagging new module versions or restructuring the example.

3. **Should I run `nix run .#verify` now (may take 3-4 min), or wait until the cqrs-lint golden is resolved?** The golden update is a 1-line fix (`CQRS_LINT_UPDATE_GOLDEN=1`) but I stopped because you asked for a status report. Should I continue fixing the golden first, or leave everything as-is for your review?

---

## Commits This Session

| Commit      | Description                                                                                 | Author                                                                                        |
| ----------- | ------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------- |
| `74b5762e2` | feat(metaengine): ship live-latency model P1/P2/P3, store stats, and metadata JSON encoding | Auto-commit daemon (includes my pebble/bbolt serialization fix + unrelated live-latency work) |
| `02a05a163` | chore: refresh deps, fix go.mod ordering, and tidy formatting                               | Auto-commit daemon                                                                            |

## Uncommitted Changes (2 files)

| File                                                | Change                                                     |
| --------------------------------------------------- | ---------------------------------------------------------- |
| `signing/testdata/golden/hmac-signed-metadata.snap` | Regenerated: `userId→actorId` + new timestamp fields       |
| `cmd/cqrs-lint/testdata/taskmanager_golden.txt`     | Updated: added V003 + C017 findings (but test still fails) |

---

## Test Results Summary

| Module                          | Status     | Notes                                                            |
| ------------------------------- | ---------- | ---------------------------------------------------------------- |
| `storage/pebble/`               | ✅ PASS    | All tests including metadata roundtrip + ActorID assertion       |
| `storage/bbolt/`                | ✅ PASS    | All tests including metadata roundtrip                           |
| `metaengine/`                   | ✅ PASS    | 145/145 Ginkgo specs (graph ADT, replication, planner all green) |
| `signing/`                      | ✅ PASS    | 12/12 specs, 1 golden snapshot passed                            |
| `metaengine/projectionadapter/` | ✅ PASS    | Auto-insert through adapter works                                |
| `cmd/cqrs-lint/pkg/rules/`      | ❌ FAIL    | V003 + C017 not in golden profile (golden update incomplete)     |
| `nix run .#verify`              | ⏳ NOT RUN | Full CI gate not executed                                        |
