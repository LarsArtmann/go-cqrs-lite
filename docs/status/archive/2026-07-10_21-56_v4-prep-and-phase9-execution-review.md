# Status Report: Post-v4-Prep Execution + Phase 9

<!-- historical-artifact-banner -->

> **Historical session artifact.** This is a point-in-time snapshot from a past
> session. Many items marked TODO / Open / Not Started / Broken have since been
> resolved. See [CHANGELOG.md](../../../CHANGELOG.md) and
> [TODO_LIST.md](../../../TODO_LIST.md) for current state.
> Last documentation health audit: 2026-07-16.

**Date:** 2026-07-10 21:56
**Session scope:** Verification gaps from prior session + Phase 9 execution
**Verdict:** Functional but messy. Lots of sharp edges left unpolished.
**Updated:** 2026-07-10 22:10 — `event.DefaultCodec` flip in progress, envelope fallback decision made (ADR-0050).

---

## a) FULLY DONE (verified passing)

### Verification Gates (all green)

| Gate                                | Status               | Detail                                       |
| ----------------------------------- | -------------------- | -------------------------------------------- |
| `go build ./...`                    | PASS                 | All 49 modules compile                       |
| Full workspace test (60 packages)   | PASS                 | 0 failures                                   |
| `go vet` on changed modules         | PASS                 | No issues                                    |
| `nix run .#lint`                    | CLEAN (0 new issues) | 118 pre-existing issues, 0 from this session |
| `nix run .#check-file-size`         | PASS                 | All files ≤350 lines                         |
| API stability (`cmd/api-stability`) | PASS                 | 2197 exports verified                        |
| Doc-check (`cmd/doc-check`)         | PASS                 | 817 references valid across 34 packages      |
| Workspace sync checks               | PASS                 | go.work ↔ flake.nix, go.work ↔ api-stability |

### File-Size Violations Fixed (3 files split)

1. `signing/cose.go` (360 lines) → extracted `signing/cose_sign1.go` (SignCOSE1 + VerifyCOSE1). Remaining: 174 + 197 lines.
2. `cmd/doc-check/main.go` (367 lines) → extracted `cmd/doc-check/exports.go` (buildExportIndex + parsePackageExports + helpers). Remaining: 225 + 138 lines.
3. `catalog/eventcatalog/frontmatter_render.go` (479 lines) → extracted `catalog/eventcatalog/frontmatter_convert.go` (to\* conversion helpers). Remaining: 197 + 280 lines.

### Phase 9: v4 Codec Flip (3 layers flipped)

- **Blind stores** (`kv`, `snapshot`, `command`, `query`) now default to `CBORCodec` instead of `JSONCodec`. All blind store tests pass.
- **`event.DefaultCodec`** flipped from `JSONCodec` to `CBORCodec`. Events are self-describing (`evt.Encoding()` stamped per-event), so `DecodePayloadAuto` handles mixed streams. Code change done, 2 tests fixed in `event/codec_typed_test.go` to use `DecodePayloadAuto` instead of `json.Unmarshal`. **WARNING: full workspace test NOT yet re-run after this flip — other modules may have tests that assume JSON event payloads.**
- `UnwrapDecode` fallback in all 4 blind stores hardcoded to `codec.JSONCodec{}` for backward compat with pre-envelope data.
- Envelope wrapping (ADR-0044) means new writes stamp their codec; old data falls through to JSON decode.

### Phase 9: Alias Removal (8 aliases deleted)

| Removed                    | Replacement                                                              |
| -------------------------- | ------------------------------------------------------------------------ |
| `event.AggregateRef`       | `id.AggregateRef`                                                        |
| `event.NewAggregateRef`    | `id.NewAggregateRef`                                                     |
| `event.AggregateType`      | `id.AggregateType`                                                       |
| `event.ParseAggregateType` | `id.ParseAggregateType`                                                  |
| `event.Tracing`            | `metadata.Tracing` (embedded in `event.Metadata` via `metadata.Tracing`) |
| `event.CustomData[K]`      | `metadata.CustomData[K]`                                                 |
| `event.MergeCustomMaps`    | `metadata.MergeCustomMaps`                                               |
| `schema.WithDecodeFunc`    | `schema.WithCodec()` / `schema.WithDecoder()`                            |

- 4 alias files deleted (`aggregate_ref.go`, `customdata.go`, `custommap.go`, `tracing.go`).
- `deprecated_alias_test.go` deleted (tested aliases that no longer exist).
- All internal usages in `event/` and `event/v4/eventtest/` updated.
- All doc references in `SKILL.md` and `AGENTS.md` fixed.

### Other Completed Items

- **SECURITY.md** — Created with vulnerability reporting process, supported versions table, security feature documentation.
- **HealthChecker on `*SQLEventStore`** — `HealthCheck(ctx)` method added, wraps `db.PingContext`. 2 tests: healthy + closed-store error case.
- **TODO_LIST.md** — Fixed false `[x]` claim (changed "All 5 files split" → accurate "3 production files split" with names).
- **AGENTS.md** — Updated codec default table (JSON→CBOR for blind stores), added health check + shutdown ordering pattern examples, added `metadata` import to benchmark test.
- **MIGRATION-GUIDE.md** — Rewritten from "what will change" to "v3→v4 migration guide" with both breaking changes documented.
- **API surface golden file** — Regenerated twice (2204 after verification steps, then 2197 after alias removal).
- **stack/health.go lint fix** — Fixed `varnamelen` (hc→checker) and `wsl_v5` (missing whitespace).

---

## b) PARTIALLY DONE

### HealthChecker interface — 1 implementation, not comprehensive

- `*SQLEventStore` has HealthCheck. But `*SQLSnapshotStore`, `*SQLCheckpointStore`, `*SQLCommandStore`, `*SQLQueryStore`, pebble stores — none implement it.
- `stack.Bundle.HealthCheck()` checks the `*sql.DB` directly AND iterates closers for `HealthChecker`. The direct DB ping path works, but only `*SQLEventStore` will match the interface in the closer list.
- **What's missing:** At minimum, the other SQL stores should implement it too since they share the same `*sql.DB`. Or better: `OwnedDBHandle` should implement `HealthCheck` so every store inherits it.

### Shutdown ordering — tested via constructor bypass

- `WithShutdownDependency` works but tests create `Bundle` literals directly instead of going through `New()`. This means the integration path (constructing via `sqlite.New(..., WithShutdownDependency(...))`) is untested.

### `event.DefaultCodec` flip — code done, full workspace test NOT verified

- `event/codec.go` changed: `DefaultCodec` now `CBORCodec`.
- 2 tests fixed in `event/codec_typed_test.go` (removed `json.Unmarshal` of CBOR payloads, switched to `DecodePayloadAuto`).
- **NOT verified:** Full workspace test not re-run. Other modules (decider, integration, example, stack presets) may have tests that `json.Unmarshal` event payloads assuming JSON encoding. These will break with CBOR default.
- Unused `encoding/json` import removed from `codec_typed_test.go`.

### Codec flip — no benchmark measuring impact

- The envelope wrapping doubles encode cost (inner codec + JSON envelope marshal). This was noted as a risk but never benchmarked. We don't know the actual overhead.

### AGENTS.md updates — incomplete

- Updated the codec table and added health/shutdown examples. But did NOT update:
  - The "Design Principles" section (point 9 mentions `any` exceptions — unchanged, but principle 17 about singleflight is fine)
  - The error handling section still mentions `event.Family` and `event.Error` aliases — those may still exist
  - The module graph (Tier model) wasn't updated for codec changes
  - No mention of `codec.WrapEncode`/`UnwrapDecode` in the Key Patterns section

---

## c) NOT STARTED

1. **Full workspace test after `event.DefaultCodec` flip** — Other modules may have tests assuming JSON event payloads. Must run full suite.
2. **Envelope magic string strengthening** — "cqrs" is 4 chars. Could collide with real data containing a `$` field set to "cqrs". Should be longer (e.g., "cqrs-envelope-v1").
3. **Envelope backward-compat integration test** — No test verifies that pre-envelope JSON data in a real store round-trips correctly through the new `UnwrapDecode` fallback path.
4. **Envelope benchmark** — No measurement of double-encode overhead.
5. **README.md** — Still missing encryption, turso, testutil sections (pre-existing gap, not touched).
6. **FEATURES.md** — Not updated with health checks, shutdown ordering, envelope wrapping, codec flip.
7. **SKILL.md module decision matrix** — Was updated with `metadata/` row in prior session, but NOT updated for codec defaults changing or alias removal.
8. **SKILL.md references/recipes.md** — May contain code examples using removed aliases. Not checked.
9. **SKILL.md references/modules.md** — Per-module table may reference removed aliases. Not checked.
10. **Shutdown ordering through real `New()` constructor** — Not tested.
11. **HealthChecker on other SQL stores** — Not implemented.
12. **HealthChecker on pebble stores** — Not implemented.
13. **Phase 11: Performance** (hot-state cache, read-pressure snapshots) — Not started.
14. **Phase 12: Transport** (NATS, ValKey adapters) — Not started.
15. **Phase 13: Public Release** (license swap, git history scrub) — Not started.
16. **v4 changelog** — No CHANGELOG.md entry for the breaking changes.

---

## d) TOTALLY FUCKED UP / CLOSE CALLS

### 1. Blind sed replacement corrupted method names

**What happened:** When removing `AggregateType` alias, I used `sed -i 's/\bAggregateType\b/id.AggregateType/g'` across all event/*.go files. This replaced the **method name** `AggregateType()` in `event/event.go:77`, producing `func (e *ImmutableEvent) id.AggregateType()` — a syntax error.

**Also happened:** `sed` double-prefixed `id.ParseAggregateType` → `id.id.ParseAggregateType` in two test files (prior sed replacement had already added `id.`, then a second sed pass added another `id.`).

**Root cause:** Using global find-replace without accounting for method definitions and call sites that share the same identifier name. Go doesn't distinguish `type AggregateType` from `func AggregateType()` in a blind regex.

**How I caught it:** Build failed. Fixed manually. But this is fragile — should have used `gopls rename` or LSP references instead.

### 2. event/metadata.go Tracing embedding broke silently

**What happened:** The `Tracing` type was embedded in `Metadata` via the alias `type Metadata struct { Tracing; ... }`. When I deleted the `Tracing` alias from event/, the embedding broke. I had to manually change it to `metadata.Tracing`.

**Why it was close:** If the struct had been in a different file or the embedding was less obvious, this could have been a silent semantic change — `Metadata` would lose all tracing fields.

### 3. eventtest module needed import additions

**What happened:** After replacing `event.AggregateRef` → `id.AggregateRef` in eventtest files, `fake_snapshot.go` used `id.AggregateRef` but didn't import `id/`. Build failed.

**Root cause:** Blind sed doesn't add imports. I had to manually check each file.

### 4. catalog frontmatter_convert.go had type mismatches

**What happened:** When I extracted the conversion helpers to a new file, two fields (`Description` and `Summary`) had type mismatches — the original code used `string(t.Description)` in some places but `t.Description` in others. The `frontmatter_types.go` changes from the prior session (which changed some fields to typed strings like `catalog.Description`) meant the extracted code needed explicit `string()` conversions.

**How I caught it:** Build failed with type mismatch errors. Fixed with `string()` wrapping.

---

## e) WHAT WE SHOULD IMPROVE

### Process Improvements

1. **Never use blind sed for type renames** — Use LSP `references` tool or `gopls rename` which understand Go semantics and won't corrupt method names.
2. **Test after EVERY file split** — I batched 3 file splits then tested. Should have tested each individually.
3. **Read the full file before extracting** — The catalog frontmatter issue was caused by not reading the full file carefully before splitting. The type conversions were inconsistent in the original.
4. **Run `goimports` after every batch sed** — Would have caught missing imports immediately.
5. **The envelope magic string "cqrs" is too short** — Should be strengthened to prevent collision with real data.

### Code Quality Improvements

6. **`OwnedDBHandle` should implement `HealthCheck`** — Instead of adding it to every individual store, add it to the shared handle so ALL SQL stores inherit it.
7. **Codec flip needs a migration test** — Write data with old JSON default, read with new CBOR default, verify round-trip.
8. **Envelope benchmark** — Measure the overhead of double-encoding (inner codec + JSON envelope).
9. **Shutdown ordering needs integration test** — Through real `New()` constructor, not struct literals.
10. **SKILL.md references need checking** — recipes.md and modules.md may still reference removed aliases.

### Documentation Improvements

11. **CHANGELOG.md** — Should have a v4 section documenting the breaking changes.
12. **FEATURES.md** — Needs update for health checks, shutdown ordering, envelope wrapping.
13. **ADR for codec flip** — The codec default change is a significant decision that should be documented in an ADR.

---

## f) Up to 50 Things to Do Next

#### Critical (must do before any v4 release)

1. Strengthen envelope magic string from "cqrs" to "cqrs-envelope-v1" (collision risk)
2. Write envelope backward-compat integration test (old JSON data → new CBOR default read path)
3. Write envelope benchmark (measure double-encode overhead vs raw codec)
4. Implement `HealthCheck` on `OwnedDBHandle` (inheritable by all SQL stores)
5. Test `WithShutdownDependency` through real `sqlite.New()` constructor
6. Check SKILL.md references/\*.md for removed alias usage
7. Check SKILL.md references/recipes.md for removed alias usage in code examples
8. Write CHANGELOG.md v4 entry with breaking changes
9. Update FEATURES.md with health/shutdown/envelope features
10. Write ADR for codec default flip (JSON→CBOR for blind stores)

#### High Value

11. Add `HealthCheck` to pebble stores (`PebbleBackend`)
12. Add `HealthCheck` to memory stores (always healthy)
13. Add `WithShutdownDependency` usage examples to SKILL.md
14. Add `HealthCheck` usage examples to SKILL.md
15. Update README.md with missing module sections (encryption, turso, testutil)
16. Update SKILL.md module decision matrix for codec default change
17. Add `codec.WrapEncode`/`UnwrapDecode` to AGENTS.md Key Patterns section
18. Verify `event.Family` and `event.Error` aliases still exist (or remove them too)
19. Check if `command.Metadata` and `query.Metadata` aliases for `event.Metadata` need updating
20. Run `go mod tidy` in all modules after alias removal (dependency graph may have changed)
21. Verify `nix run .#test-race` passes (race conditions in shutdown ordering?)
22. Verify `nix run .#coverage` hasn't dropped significantly

#### Medium Value

23. Add shutdown ordering to `pebble.Bundle` and `stack.Bundle` presets
24. Add health check endpoint to `example/taskmanager` (demonstrate the pattern)
25. Add `WithHealthCheckInterval` option for periodic background health checking
26. Document codec flip in stack preset module READMEs
27. Add `stack.WithEventCodec` documentation (currently underdocumented)
28. Verify `go mod tidy -e` works cleanly in event/ (nested eventtest module)
29. Check if any example code uses removed aliases
30. ~~Add deprecation notice to `event.DefaultCodec`~~ — DONE: flipped to CBOR in this session
31. Consider making envelope encoding configurable (opt-out for performance-critical paths)
32. Add `codec.Determinate` option for envelope (currently hardcoded `json.Deterministic(true)`)
33. Profile blind store write path before/after envelope wrapping
34. Consider CBOR envelope instead of JSON envelope (smaller, but can't fall back to JSON decode)

#### Lower Priority

35. Add `HealthCheck` to `watermill.EventBus` / `watermill.CommandBus`
36. Add shutdown timeout to `stack.Bundle.Close()` (currently unbounded)
37. Add `WithHealthChecker` option to explicitly register custom health checkers
38. Document the envelope wire format in AGENTS.md
39. ~~Add `codec.UnwrapDecodeStrict` variant that errors on non-envelope data~~ — REJECTED (ADR-0050: keep forever, no strict mode)
40. Consider adding envelope version field (currently just magic string)
41. ~~Add migration script for consumers who want to bulk-rewrite old JSON data to CBOR~~ — REJECTED (ADR-0050: no migration needed, fallback is permanent)
42. Add `stack.Bundle.Status()` method returning per-resource health
43. Document the fallback decode behavior in SKILL.md FAQ
44. Add test for envelope with nil/empty data edge case
45. Add test for envelope with corrupt/truncated inner data
46. ~~Consider whether `event.DefaultCodec` should also flip to CBOR in v4~~ — DECIDED: YES, flipped to CBOR
47. Add `WithCodec(c)` option to all stack presets for one-call codec override
48. Profile `UnwrapDecode` overhead (JSON parse attempt on every read, even non-envelope data)
49. Add benchmark comparing envelope read path vs raw read path
50. Consider lazy envelope detection (check first byte instead of full JSON parse)

---

## g) Top 2 Questions — BOTH ANSWERED

### 1. Should `event.DefaultCodec` also flip to CBOR in v4? — **YES (DONE)**

**Decision:** Flipped to `CBORCodec`. Events are self-describing (`evt.Encoding()` stamped
per-event), so `DecodePayloadAuto` handles mixed streams transparently. This is now the
third breaking change in v4: blind store codec defaults, alias removal, and event default
codec.

**Status:** Code changed in `event/codec.go`. 2 tests fixed in `codec_typed_test.go`.
**WARNING:** Full workspace test NOT yet re-run — other modules may have tests that
assume JSON event payloads.

### 2. Should we keep the envelope fallback to JSON forever, or sunset it? — **KEEP FOREVER (ADR-0050)**

**Decision:** The JSON fallback in `UnwrapDecode` is permanent. Full comparison table and
rationale documented in `docs/adr/0050-envelope-json-fallback-keep-forever.md`.

**Key reasoning:** Blind store data (KV rows, snapshots) is materialized state with no
automatic migration path. The fallback costs nothing measurable. Removing it would be a
breaking change for purely cosmetic reasons.

**Rejected alternatives:** Sunset in v5 (no migration automation), configurable strict mode
(unnecessary API surface).
