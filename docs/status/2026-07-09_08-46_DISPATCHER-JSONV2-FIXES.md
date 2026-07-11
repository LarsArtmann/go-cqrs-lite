# Status Report: Dispatcher Middleware Fix + json/v2 Decode Fix + Golden Stability

**Date:** 2026-07-09 08:46
**Session scope:** Fix dispatcher middleware timing, fix json/v2 silent data corruption, fix catalog golden non-determinism, fix taskmanager deriver race, tidy workspace

---

## a) FULLY DONE ✅

### 1. json/v2 Silent Data Corruption — FIXED

**Root cause:** `encoding/json/v2` defaults to **case-sensitive** field matching. The repo's `JSONCodec` in `codec/json.go` imports `encoding/json/v2` unconditionally. Every event payload with lowercase JSON keys (e.g. `{"name":"Alice"}`) decoded into untagged Go structs (e.g. `struct{ Name string }`) silently produced zero values — no error, no warning.

**Proof of root cause:**

```
json/v2:  {"name":"Alice"}  →  Name=""   (no error)
json/v2:  {"Name":"Alice"}  →  Name="Alice"
```

**Fix:** Added `json.MatchCaseInsensitiveNames(true)` to `json.Unmarshal` calls in:

- `codec/json.go` — `JSONCodec.Decode`
- `codec/jsonv2_experiment.go` — `JSONCodecV2.Decode`

**Verification:** `TestDecodePayload_EncodingMatch` now passes. Full codec + event test suites green.

### 2. Dispatcher Middleware Timing — FIXED

**Root cause:** `dispatcher.Dispatcher[H,M].Register()` applied middleware at registration time via `d.middleware.Apply(handler, wrap)`, baking the wrapped handler into the map. `Dispatch()` returned the pre-wrapped handler without consulting the middleware chain. Any `Use()` call after `Register()` was silently ignored.

**This caused a real bug in taskmanager:** ALL middleware (recovery, logging, retry, OTel, idempotency) was silently bypassed because `registerHandlers()` ran before `setupFeatures()`.

**Fix:** Store raw handler + wrap function in `handlerEntry[H,M]`. Apply middleware at dispatch time in `getHandler()` using the current chain. Middleware can now be added in any order relative to `Register()`.

**Files changed:**

- `dispatcher/dispatcher.go` — new `handlerEntry` struct, `Register()` stores raw, `getHandler()` applies middleware
- `dispatcher/dispatcher_test.go` — 3 new regression tests:
  - `TestDispatcher_MiddlewareAppliedAtDispatchTime` — middleware added AFTER register still runs
  - `TestDispatcher_MiddlewareAddedBeforeAndAfter` — mixed ordering works
  - `TestDispatcher_MiddlewareChangeVisibleOnNextDispatch` — middleware added between dispatches is visible

**Verification:** All 21 dispatcher tests pass. command/, query/, event/ modules all green.

### 3. Catalog Golden Non-Determinism — FIXED

**Root cause:** `json.Marshal` in `encoding/json/v2` does NOT sort map keys by default (unlike v1). The catalog exporters use `map[string]Channel` and `map[string]any` fields. Different map iteration order across separate test binary runs produced different JSON output, making golden files impossible to match reliably.

**This was misdiagnosed in the prior session as a "golden file format difference."** It was actually non-determinism.

**Fix:** Added `json.Deterministic(true)` to all catalog JSON marshaling:

- `catalog/asyncapi/serde.go` — `Document.MarshalJSON`
- `catalog/openapi/serde.go` — `Document.MarshalJSON`
- `catalog/schema/reflect.go` — `ToJSON`, `ToAny`
- `catalog/eventcatalog/writer.go` — `writeExamples`, `writePackageJSON`
- `catalog/eventcatalog/writer_schemas_txt.go` — schema text writer

**Golden files regenerated:**

- `catalog/testdata/golden/asyncapi.json`
- `catalog/testdata/golden/asyncapi.yaml`
- `catalog/testdata/golden/openapi.json`
- `catalog/testdata/golden/package.json`

**Verification:** 3/3 full workspace runs with zero golden mismatches. Output is now deterministic across separate binary runs.

### 4. Taskmanager Deriver Race — FIXED

**Root cause:** The deriver projection auto-dispatches `task.assign` asynchronously (fire-and-forget goroutine) on `task.created`. The integration test dispatched `task.start` immediately after `task.created` was projected, without waiting for the async `task.assign` to commit. Under load, `task.start` and `task.assign` collided on the same aggregate → optimistic concurrency version conflict.

**Fix:** Added a `waitForView` check for `AssigneeID == defaultAssignee` before dispatching `task.start`, ensuring the deriver's async assign has been projected.

**Verification:** 10/10 individual runs pass. 3/3 full workspace parallel runs pass.

### 5. Stale Taskmanager Comment — REMOVED

The comment "Middleware must be configured BEFORE handlers are registered — the dispatcher applies middleware at registration time, not at dispatch time" in `example/taskmanager/setup.go` is no longer accurate after the dispatch-time fix. Removed.

### 6. go mod tidy Across All 49 Modules — DONE

Ran `go mod tidy -e` across all 49 modules. Key outcome:

- `command/go.mod`: `event/v4` demoted from direct to `// indirect` — confirms the extraction broke the compile dependency completely. Zero Go files in `command/` (not even tests) import `event/`.

### 7. Deterministic Placement Audit — DONE

All 7 `json.Deterministic(true)` calls are in `catalog/` (documentation/schema export paths). None are on hot data paths. The `docserver` HTTP handler correctly does NOT use Deterministic (per user instruction Q8).

---

## b) PARTIALLY DONE 🟡

### `check-module-layers.sh` still uses old 7-layer system

ADR-0046 defines the four-tier model, but `scripts/check-module-layers.sh` still enforces the old 7-layer system. The `metadata` module is not in the script at all. User deferred this ("ADR still needs refinement — not now").

### `metadata/` module has no dedicated unit tests

Tracing.Merge, CustomData.Clone, MergeCustomMaps edge cases. Identified but not addressed this session.

---

## c) NOT STARTED ⏸

| Item                                                    | Status          | Notes                                                                  |
| ------------------------------------------------------- | --------------- | ---------------------------------------------------------------------- |
| Consumer migration guide for id/ + metadata/ extraction | Not started     | "import id/ for AggregateRef, import metadata/ for Tracing/CustomData" |
| Storage/ split execution                                | Proposal exists | User said "I am preparing storage/ for more in the future"             |
| Event/ god module decomposition                         | Not started     | 9 concerns, 130+ exports                                               |
| metadata/ doc.go                                        | Not started     | Package documentation                                                  |
| Deprecated alias verification test                      | Not started     | Test that Deprecated warnings appear via staticcheck                   |

---

## d) TOTALLY FUCKED UP 💥 (Nothing this session)

No regressions introduced. All fixes verified across 65 test packages, 3 full workspace runs.

**Prior session note:** The sed-based `go.mod` editing that malformed 12 modules was the last major mistake — this session used `go mod tidy` instead, which worked cleanly.

---

## e) WHAT WE SHOULD IMPROVE

1. **Stop calling things "golden failures" without investigating root cause.** The prior session labeled 5 golden failures as "json/v2 format differences" and moved on. They were actually (a) silent data corruption from case-sensitive decode and (b) non-deterministic map ordering. Both were real bugs hiding behind the "golden" label.

2. **The dispatcher middleware-at-registration-time design survived through multiple reviews without anyone flagging it as a footgun.** The fix is simple and correct. Lesson: when a design requires a specific call ordering but doesn't enforce it, that's a bug in the design, not the callers.

3. **json/v2 migration needs more systematic testing.** The case-sensitivity change is a breaking behavioral difference from v1 that silently corrupts data. A migration checklist item should be: "verify all decode paths handle case-insensitivity correctly."

4. **Non-deterministic JSON output is invisible until it hits golden tests.** Every `json.Marshal` call with map fields should default to `Deterministic(true)` in library code where output might be compared or cached.

---

## f) Up to 50 Things to Get Done Next

### P0 — Immediate (blocks CI green or correctness)

1. ✅ ~~Fix json/v2 case-sensitive decode~~ — DONE
2. ✅ ~~Fix dispatcher middleware timing~~ — DONE
3. ✅ ~~Fix catalog golden non-determinism~~ — DONE
4. ✅ ~~Fix taskmanager deriver race~~ — DONE

### P1 — High Value (next 1-2 sessions)

5. Add unit tests for `metadata/` package (Tracing.Merge, CustomData.Clone, MergeCustomMaps)
6. Write consumer migration guide for id/ + metadata/ extraction
7. Add `metadata/` to `scripts/check-module-layers.sh`
8. Add `metadata/ doc.go` package documentation
9. Add deprecated alias verification test (staticcheck SA1019 compliance)
10. Update ADR-0046 four-tier model (user said it needs refinement)
11. Update AGENTS.md with json/v2 case-insensitivity note
12. Update AGENTS.md with dispatch-time middleware note (remove old "must Use before Register" language)
13. Update SKILL.md (consumer guide) with dispatcher middleware ordering freedom

### P2 — Medium Value (next 2-4 sessions)

14. Event/ god module decomposition: extract tombstone detection → `tombstone/`
15. Event/ god module decomposition: extract command causality enrichment
16. Event/ god module decomposition: extract replay mode / processing mode
17. Event/ god module decomposition: extract arena allocation
18. Storage/ split execution (proposal exists at `docs/planning/2026-07-09_STORAGE-SPLIT-PROPOSAL.md`)
19. Add integration test verifying dispatcher middleware runs in command/ and query/ (not just core dispatcher/)
20. Add benchmark for dispatch-time vs registration-time middleware overhead
21. Review all other `json.Marshal` calls across the repo for missing `Deterministic(true)` where output is compared/cached
22. Review all other `json.Unmarshal` calls across the repo for missing `MatchCaseInsensitiveNames(true)`
23. Pebble store multi-process support
24. Distributed event bus (Redis/NATS) via Watermill backend

### P3 — Polish & Cleanup

25. Remove `codec/jsonv2_experiment.go` — `JSONCodecV2` is redundant now that `JSONCodec` itself uses json/v2
26. Add race detector run to CI (`-race` flag)
27. Add coverage report for `metadata/` module
28. Run `nix run .#lint` and fix any new lint issues from this session's changes
29. Update `docs/SPAN_NAMING.md` if dispatcher span names changed
30. Add ADR for dispatch-time middleware decision
31. Add ADR for json/v2 case-insensitive decode decision
32. Add ADR for json/v2 Deterministic output in catalog exports
33. Verify `nix run .#check-layers` passes with metadata/ module
34. Add `metadata/` to the module list in AGENTS.md Quick Reference table
35. Update the module graph in AGENTS.md to show metadata/ at Tier 0

### P4 — Future Consideration

36. GraphQL transport module (was "HARD NO" — revisit if needed)
37. Outbox pattern (ADR-0016 is Declined — use Watermill)
38. Pebble backup automation
39. Multi-DB Postgres preset testing
40. gRPC transport streaming support
41. SSE broker authentication
42. Relational projection MySQL dialect support
43. Graph projection Neo4j driver
44. OpenTelemetry collector integration
45. Prometheus Grafana dashboard templates
46. EventCatalog CI/CD deployment
47. AsyncAPI code generation
48. OpenAPI client SDK generation
49. Versioned migration tooling for event schema evolution
50. Multi-region replication strategy

---

## g) Top 2 Questions I Cannot Answer Myself

**Q1: Should `MatchCaseInsensitiveNames(true)` be the default for ALL json/v2 decode paths in the library, or should consumers opt in?**

Currently it's only on `JSONCodec.Decode`. But other modules might decode JSON directly (e.g., `event/arena_experiment.go`, `storage/` SQL scan paths, `transport/http/` SSE handlers). I don't know whether any of those rely on case-sensitive matching, and a blanket audit would touch dozens of files. Should I do the audit, or is the codec layer sufficient?

**Q2: The dispatcher now applies middleware on every `Dispatch()` call (function composition per call). Is the overhead measurable in the stack/bench benchmarks, and if so, should we cache the wrapped handler and invalidate on `Use()`?**

The current approach wraps on every `getHandler()` call. An optimization would cache the result and invalidate when `Use()` is called. But this adds complexity. I can't tell if the overhead matters without running `stack/bench` — and I'm not sure the benchmark covers middleware-heavy dispatch paths.
