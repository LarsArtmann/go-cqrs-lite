# Status Report: WAL Unification Execution (Phases 1–4 of 7)

| Field     | Value                                      |
| --------- | ------------------------------------------ |
| Date      | 2026-08-14 14:59                           |
| Author    | Crush session (Lars-supervised)            |
| Scope     | `docs/planning/2026-08-14_11-27_WAL-UNIFICATION.md` + STORE-MIDDLEWARE-SIMPLIFICATION |
| Verdict   | **GREEN for Phases 1–4. Phases 5–7 NOT STARTED. One gate deliberately deferred.** |

## Executive Summary

Executed the first four phases of the WAL unification plan. Commands, Events,
and PersistedQueries now share one metadata generic (`metadata.Metadata[K]`),
one store-decoration mechanism (`event.DecorateStore`), one Record adapter set,
and one generic in-memory log core (`memory.LogStore[T, ID]`). All changed
modules and all 20 downstream consumers pass their test suites. Old APIs are
preserved as deprecated aliases/forwarders/shells per the no-external-breakage
constraint. The API-stability golden regen, ADR, doc sweep, and `nix run
.#verify` (Phase 7) have NOT run yet — that is the deliberate deferral.

## FULLY DONE (nothing left in these)

### Phase 1 — Metadata unification

- `metadata.Metadata[K ~string]` added as the canonical
  Tracing + `Custom map[K]string` shape with `Clone`/`Merge`/`WithCustom`/
  deprecated `EnsureCustom` methods.
- `metadata.CustomData[K]` converted to a **deprecated generic type alias** for
  `Metadata[K]` (Go 1.26 generic aliases). External `CustomData` users compile
  unchanged.
- `command.Metadata` and `query.Metadata` converted from standalone structs to
  type aliases of `metadata.Metadata[MetadataKey]` (each with its module-local
  key type — ADR-0031's no-event-coupling and no-tombstone-leakage goals hold).
  Their duplicated method bodies deleted (~70 lines).
- metadata/command/query/event + all dependents tested green.
- **Deliberate deviation from plan:** task "event.Metadata embeds
  `metadata.Metadata[K]`" was **skipped**. Embedding would break external
  `event.Metadata{Tracing: ..., Custom: ...}` composite literals (field would
  move under the embedded struct name) — an external API break, which this
  effort explicitly forbids. Dedup win was ~2 field declarations. Event.Metadata
  stays a standalone struct.

### Phase 2 — Store middleware + transforms

- `event.SinkTransform` / `event.SourceTransform` / `event.DecorateStore(store,
  sinkT, sourceT)` added (`event/store_middleware.go`). The ONE place that
  forwards all interfaces: Store, Journal, SeekableJournal, BackwardsSource,
  **MultiSink** (new — the old encryptedStore silently lost this capability),
  io.Closer. Nil transforms = pass-through; both nil = returns inner unchanged.
  Panics on nil store (composition-time programmer error; error-returning
  constructors validate first).
- 4 new sentinels: `event.ErrInnerStoreNotJournal/NotSeekable/NotBackwards/
  NotMultiSink`.
- 10 DecorateStore tests (pass-through, sink/source application on every read
  path, read-error passthrough, optional interfaces, missing capabilities,
  closer, nil-store panic) — all pass.
- `encryption/store.go`: 241-line `encryptedStore` deleted. Replaced by
  `EncryptSinkTransform` + `DecryptSourceTransform`. `NewEncryptedStore` keeps
  its signature but now returns `event.Store` (was unexported `*encryptedStore`
  — non-breaking) built via DecorateStore.
- `encryption.ErrInnerStoreNot*` kept as **deprecated aliases** to event's
  sentinels so existing `errors.Is` checks keep matching.
- `EncryptMiddleware` now composes `EncryptSinkTransform` (single encrypt
  loop).
- `schema.UpcastSourceTransform(upcasters...)` added; 99-line `VersionedStore`
  struct deleted. `schema.VersionedStore` + `NewVersionedStore` kept as a
  **deprecated compat shell** (embedded decorated Store + Close forwarding) —
  same signature, strictly additive capabilities.
- All internal `NewVersionedStore` usages migrated to
  `event.DecorateStore(..., schema.UpcastSourceTransform(...))` (schema tests,
  golden/fuzz/benchmark, event BDD + example tests, decider doc).
  Nil-store fuzz tests + one compat-shell behavioral test kept against the
  deprecated constructor.
- `event.RejectingPublishMiddleware` / `event.RejectingHandlerMiddleware`
  added to `event/middleware.go`.
- `signing.RejectingPublishMiddleware` / `RejectingHandlerMiddleware` now
  **deprecated forwarders** to event's. `signing/multisig` migrated off them.
- encryption's unexported `rejecting*` helpers deleted.
- Known intentional behavior changes (documented here, ADR pending):
  unsupported-capability errors on wrapped stores now carry `event.*` codes
  instead of `encryption.*`; inner-store read errors through upcasting are no
  longer double-wrapped with `schema.versioned_*` codes (sentinel identity via
  `errors.Is` preserved).

### Phase 3 — Record adapter set completed

- `query.AsRecord(*PersistedQuery) record.Record` added (`query/asrecord.go`)
  with field mapping mirroring `command.AsRecord` (queries DO carry payload
  blobs, unlike commands; `ClientCreatedAt` ← `ReceivedAt`).
- `record/v4` dependency added to `query/go.mod` at existing tag v4.2.0.
- Tests cover field mapping, zero-tracing → empty strings, nil query.

### Phase 4 — Generic memory log core

- `storage/memory/log_store.go`: generic `LogStore[T any, ID comparable]` +
  `LogStoreConfig` (GetID, IsZeroID, ClosedErr, NewDupErr, NewNotFound,
  TrackStreams). Core owns lock discipline, global log, stream index, ID index,
  append, seek, clone.
- `MemoryStore`, `MemoryCommandStore`, `MemoryQueryStore` rewritten as thin
  wrappers embedding `*LogStore[...]` — public constructors, method signatures,
  interface assertions, error codes, and messages unchanged.
- Preserved divergent policies explicitly: event `ReadFrom` replays from
  beginning on missing position (safe re-replay) while command/query return
  empty; event Save uses version conflict, command/query use duplicate
  detection; query disables stream tracking.
- Command `AppendBatch` in-batch duplicate detection (`seen` map) preserved.
- Removed old per-store `withWriteLock`/`withReadLock`/`withCommandReadLock`/
  `withQueryReadLock` duplication.
- Coverage of the new core: 80–100% per function through the three stores'
  existing suites. **All 20 modules importing storage/memory test green.**

## PARTIALLY DONE (started, work remains)

~~**Phase 2 leftovers:** none code-wise, but the plan's Line-count/function-
size CI rules (max 350/file, 30/function) have not been machine-verified on
the rewritten files, and `nix fmt` (golines 120) has not run on the new
files.~~ Resolved later the same day — `nix fmt` run at `5c4dd294b`;
350-line limits enforced by every subsequent lint gate (76/76 clean since
`444be10a7`).

## NOT STARTED (planned, untouched) — ALL DONE 2026-08-14

- ~~**Phase 5 — SQL store dedup:** `storage/sql.Inserter[T]` generic~~ done at `44a8a895e` (Inserter shipped; deliberate task-29 deviation documented on the plan doc)
- ~~**Phase 6 — System adapter dedup:** `system.AdapterCore[T]`~~ done at `80d41da33` + `4e9c1190` (LoadStream migration closed it)
- ~~**Phase 7 — Ship:** ADR + golden regen + skill sweep + verify~~ done at `d0e0b682b` (ADR-0126); full verify GREEN 3× since (`5f2198189`)

## TOTALLY FUCKED UP (nothing here — honest close calls)

Nothing broken or knowingly wrong. Two near-misses caught and fixed in-flight,
recorded so future sessions don't re-trip:

1. My first `MemoryCommandStore.AppendBatch` rewrite dropped the in-batch
   `seen`-map duplicate check — duplicates inside one batch would have slipped
   through (store-level index only catches already-persisted IDs). Caught by
   reading the original before deleting; behavior restored.
2. Initial `LogStore` embedding used a value type while `NewLogStore` returns a
   pointer, and `WithReadLock` couldn't infer types from wrapper receivers.
   Fixed by embedding `*LogStore[T, ID]` and passing `s.LogStore` explicitly.
3. Test-design bug: a "bare" store built by embedding `fullStore` silently
   inherited its optional methods, which would have made the
   missing-capability tests vacuous. Rewritten as a genuinely minimal struct.

## WHAT WE SHOULD IMPROVE (beyond finishing 5–7)

1. **Regen API-stability golden the moment Phase 5–7 work resumes** — it is
   currently stale relative to the alias/type changes; `#verify` will fail
   until `cd cmd/api-stability && GOWORK=off go run main.go -update` runs.
2. Run `nix fmt` before adding any `//nolint` lines in the new files.
3. Add direct unit tests for `LogStore` error paths (nil `NewNotFound` guard,
  duplicate suffix formatting) — currently covered only indirectly.
4. Consider extracting the thrice-duplicated `brandedString` helper
   (event/command/query asrecord.go) into `record` (zero-dep-safe) if
   `check-duplication` flags the third copy.
5. `VersionedSeekableJournal` (schema) still hand-wraps a Journal+upcasters;
   a future `DecorateJournal` helper could kill that wrapper too.
6. Deprecation lint: enforce that internal code never references the new
   deprecated symbols (a cqrs-lint rule would prevent regression).
7. The `rejectSink` helper in encryption returns an unnamed func type so it
   doubles as both Sink and Source transform — fine, but a comment-based test
   would protect the assignability trick.

## UP TO 50 NEXT STEPS

1. ~~Regenerate API stability golden (`cmd/api-stability -update`).~~ done at `d0e0b682b`
2. ~~Run API stability meta-tests (`TestEveryGoModDirIsInModulesList` etc.).~~ done at `d0e0b682b`
3. ~~Write ADR-0118 (or next free): `metadata.Metadata[K]` unification; supersede ADR-0031 Decision 3.~~ done — ADR-0126, `d0e0b682b`
4. ~~Same ADR: record the `encryption.*` → `event.*` unsupported-capability error-code migration.~~ done (ADR-0126)
5. ~~Same ADR: document why `event.Metadata` does NOT embed `Metadata[K]` (composite-literal break).~~ done (ADR-0126; permanent for v4)
6. ~~Update ADR-0031 status note to point at the new ADR.~~ done at `d0e0b682b`
7. ~~Sweep `.agents/skills/go-cqrs-lite/references/*.md` for `NewVersionedStore`/`NewEncryptedStore` recommendations → DecorateStore + transforms.~~ done at `d0e0b682b` (recipe 2.7b added)
8. ~~Update `AGENTS.md` module map + internal contracts (store-middleware section).~~ done at `d0e0b682b` (contracts #16/#17)
9. ~~Run `cd cmd/doc-check && GOWORK=off go run . ../../SKILL.md ../../.agents/skills/go-cqrs-lite/references/*.md ../../AGENTS.md`.~~ done — 1020 refs, 0 warnings since `2e9a2fc28`
10. ~~Run `nix fmt`.~~ done at `5c4dd294b`
11. ~~Run `nix run .#verify` (expect golden + possibly lint findings; fix).~~ done — GREEN 3× since (`5f2198189`)
12. ~~Run `nix run .#check-arch` (dep budgets — query gained record dep).~~ done (green after the spaced-keys fix, `5127039da`)
13. ~~Run `nix run .#check-duplication` (new brandedString 3rd copy risk; update baseline only after judgment).~~ done — baseline re-pinned 92→97 at `875bb689b`
14. ~~Run `nix run .#check-coverage` (drift check).~~ done — gate repaired at `875bb689b`
15. ~~Verify 350-line file limit on all rewritten files.~~ done (76/76 lint clean)
16. ~~Read the three SQL stores (`storage/eventstore`, command/query SQL stores).~~ done at `44a8a895e`
17. ~~Design `storage/sql.Inserter[T]` (dialect placeholders, SQLite 999-param chunking reuse).~~ done at `44a8a895e`
18. ~~Implement `Inserter.Save` + `AppendBatch`.~~ done at `44a8a895e`
19. ~~Migrate SQL command store inserts to `Inserter[T]`.~~ done at `44a8a895e`
20. ~~Migrate SQL query store inserts to `Inserter[T]`.~~ done at `44a8a895e`
21. ~~Migrate SQL event store inserts to `Inserter[T]`.~~ **NOT-DO — documented deviation (task 29)**: event store keeps cached templates + batched multi-VALUES deliberately
22. ~~Run SQL store tests (SQLite) 3× with `-count=3 -race`.~~ done at `44a8a895e`
23. ~~Run PG/MySQL integration tests if inserts touched shared paths.~~ done (live-verified 2026-08-15, `4a95bd04d`)
24. ~~Read `system/adapter_event.go` / `adapter_command.go` / `adapter_query.go` + serial helpers.~~ done at `80d41da33`
25. ~~Design `system.AdapterCore[T]`.~~ done at `80d41da33`
26. ~~Extract `AdapterCore[T]` (backend, collection, serialize/deserialize).~~ done at `80d41da33`
27. ~~Migrate `EventAdapter` onto `AdapterCore`.~~ done at `80d41da33` + `4e9c1190`
28. ~~Migrate `CommandAdapter` onto `AdapterCore`.~~ done at `4e9c1190`
29. ~~Migrate `QueryAdapter` onto `AdapterCore`.~~ done at `80d41da33`
30. ~~Run system + integration module tests.~~ done (118 tests, 3× race)
31. ~~Add `event.DecorateStore` recipe to `.agents/skills/go-cqrs-lite/references/recipes.md`.~~ done at `d0e0b682b`
32. ~~Add `memory.LogStore` note to modules reference.~~ done at `d0e0b682b`
33. ~~Double-check no internal production code references deprecated symbols (`rg "schema.NewVersionedStore|signing.Rejecting" --type go` minus tests/docs).~~ done at `d0e0b682b`
34. ~~Add compat test asserting `encryption.ErrInnerStoreNotJournal` aliases `event.ErrInnerStoreNotJournal`.~~ done (compat_aliases_test.go, `d0e0b682b`)
35. ~~Re-run `go mod tidy` + standalone `GOWORK=off go build` per touched module (vulncheck parity).~~ done at `5c4dd294b`
36. ~~Check `query/go.mod` record dep moved out of `// indirect` after tidy.~~ done at `d0e0b682b`
37. ~~Soak-test env sanity run (`SOAK_SKIP_10M=1`) since memory store core changed.~~ done (verify gates since)
38. Consider `DecorateJournal` helper for `VersionedSeekableJournal`. ← **open — tracked in TODO_LIST → Code Quality**
39. ~~Update TODO_LIST.md: mark WAL-unification items 1–4 done, 5–7 open.~~ done
40. ~~Update FEATURES.md if it enumerates EncryptedStore/VersionedStore behavior.~~ done at `d0e0b682b`
41. ~~Re-read plan docs; annotate completed tasks to keep them honest.~~ done (EXECUTED banner, `875bb689b`)
42. ~~Race-run event + storage/memory suites (`-race -count=3`) after core change.~~ done (16-44 verification matrix)
43. Consider extracting `brandedString` into `record` (pending dup gate). ← **open — tracked in TODO_LIST → Code Quality**
44. ~~Bench sanity: `BenchmarkVersionedStore_Load` + memory store benchmarks.~~ done (verify race legs)
45. ~~Check `cmd/cqrs-lint` S010 still behaves (mentions NewEncryptedStore in suggestions).~~ done — S010/F005 de-staled at `d0e0b682b`
46. ~~Grep docs/ for stale `VersionedStore` narratives; fix wording.~~ done at `d0e0b682b`
47. ~~Delete stale plan checkboxes or convert remaining ones into TODO_LIST entries.~~ done (EXECUTED banner)
48. ~~Final `nix run .#verify` + tag decision per release process (only on request).~~ verify done; **tagging open — TODO_LIST → Release/Tagging**
49. ~~Scan for new gopls phantom errors after file rewrites (restart LSP if needed).~~ done (restarted in later sessions)
50. ~~Mark this report DONE in TODO_LIST when Phases 5–7 land.~~ done — WAL Unification closed; annotated + archived 2026-08-15

## QUESTIONS I CANNOT FIGURE OUT MYSELF (max 3)

1. ~~**event.Metadata embedding:** I skipped it because embedding breaks external
   `event.Metadata{Tracing: …}` literals.~~ Answered: permanent for v4 —
   documented in ADR-0126; revisit only as an explicit v5 break.
2. ~~**Deprecated shell lifetime:**~~ Answered: removed at v5 (ADR-0123 horizon;
   TODO_LIST Phase 8 entry).
3. ~~**Execution order:**~~ Moot — all phases executed; WAL Unification closed
   2026-08-15.

---

## Resolution (2026-08-15)

All 7 phases shipped (Phases 1–4 here, 5–7 in the successor report
`2026-08-14_16-44`, close-out in `2026-08-15_00-48`). All 50 next-steps closed:
47 shipped (hashes inline), 1 documented deviation (task 29), 2 deferred with
TODO_LIST pointers (DecorateJournal, brandedString). Full verify gate GREEN 3×
since. Archived.
