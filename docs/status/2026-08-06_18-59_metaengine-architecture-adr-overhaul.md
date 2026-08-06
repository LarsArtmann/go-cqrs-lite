# Status Report: 2026-08-06 18:59 — Metaengine Architecture ADR Overhaul

## Context

This session began with a question about the metaengine's graph support, evolved into a request for Dgraph + Badger support, then uncovered a fundamental architectural disagreement: the zero-dependency boundary (ADR-0062) was wrong. The user corrected the course, and the session pivoted to a full ADR overhaul before any implementation work begins.

---

## a) FULLY DONE

### Amended ADRs (4 files edited in-place)

| ADR | File | What was done |
|-----|------|---------------|
| 0062 | `0062-metaengine-dependency-boundary.md` | Status → Amended. Added addendum: zero-dep boundary superseded, new rule = modules split by deployment concern |
| 0077 | `0077-metaengine-graph-reconciliation.md` | Status → Amended. Added addendum: graph/ API wins, GraphBackend deleted, GraphDriver implements Engine |
| 0046 | `0046-seven-tier-model.md` | Status → Amended. Added addendum: metaengine reclassified Tier 0 → Tier 3. Updated notable exceptions section |
| 0074, 0086, 0091 | `0074-pebble-engine.md`, `0086-metaengine-duckdb-engine.md`, `0091-sse-consolidation-decision.md` | Updated dep-boundary references from "zero-dependency core stays clean" to "deployment isolation" |

### New ADRs Written (7 files created)

| ADR | File | Decision |
|-----|------|---------|
| 0111 | `0111-record-type-extraction.md` | Extract `Record` type with `MetaData CommonMetadata` field. Shared base for Command + Event. Three timestamps: ClientCreatedAt, ServerReceivedAt, ServerStoredAt. Command adds nothing to Record. |
| 0112 | `0112-es-native-metaengine.md` | ES-native metaengine: planner depends on Record type, understands typed records, not `any` blobs |
| 0113 | `0113-delete-graphbackend.md` | Delete GraphBackend. graph.GraphDriver implements metaengine.Engine. Simple Edge folds auto-upgrade to MergeEdge |
| 0114 | `0114-tombstone-as-domain-event.md` | Tombstones are domain events (UserDeleted), not mutable metadata. Event stream is pure immutable |
| 0115 | `0115-sqlite-engine-extraction.md` | Move SQLite engine from metaengine core to metaengine/sqliteengine/ |
| 0116 | `0116-layered-auto-projection.md` | Layered auto-projection: 80% auto-generated from type inspection, 100% auto-routed |
| 0117 | `0117-command-lifecycle-as-events.md` | Command lifecycle (DLQ, retries) via event streams. No IntentStatus field. Dead-letter queues are projections |

### AGENTS.md Updated

- Tier model updated: metaengine moved from Tier 0 to Tier 3
- "metaengine/ is THE STRATEGIC FUTURE" callout rewritten to reference all new ADRs (0111-0117)
- ADR-0062 and ADR-0046 references updated

### Research Completed

- Full analysis of metaengine GraphBackend (interface, routing, 6 engine implementations, cost model, planner rules, adttest parity)
- Full analysis of graph/ module (GraphSink, GraphDriver, ReadableDriver, GraphProjection, Schema, MemoryDriver)
- Dgraph project research: v25.4.0, server-only (not embedded), gRPC client `dgo/v250`, no CGo for client, actively maintained by Istari Digital
- Badger project research: v4.9.6, pure Go embedded LSM, no CGo, actively maintained

---

## b) PARTIALLY DONE

### ADR Amendments — Partial Coverage

- **ADR-0087 (postgres engine)**: Checked for dep-boundary references, found none directly. Did NOT add an addendum noting the dep-boundary supersession. May need updating if it implicitly relies on the zero-dep pattern.

### ADR Cross-References

- New ADRs (0111-0117) reference each other and the amended ADRs, but the **amended ADRs do not all reference the new ADRs**. For example, ADR-0062's addendum mentions ADR-0111 and ADR-0115 but not ADR-0112, 0113, 0114, 0116, or 0117. A reverse-reference sweep is needed.

### AGENTS.md Module List

- The tier model block was updated, but the **module list** at the top of AGENTS.md was NOT updated with new planned modules (badgerengine, dgraphengine, sqliteengine, graphadapter). The "Modules" row in the Quick Reference table still lists the old structure.

---

## c) NOT STARTED

### No Code Changes Made

This was intentional — the user explicitly said "fix ADRs before ANYTHING ELSE." Zero Go files were modified. The following implementation work is entirely unstarted:

1. **Record type extraction** (ADR-0111) — no `Record` struct, no `CommonMetadata`, no shared module
2. **ES-native metaengine** (ADR-0112) — fold handlers still receive `any`, not `Record`
3. **GraphBackend deletion** (ADR-0113) — GraphBackend still exists in engine.go:394-397
4. **Tombstone removal** (ADR-0114) — DetectTombstone, MarkTombstone, TombstoneStatus still in event/
5. **SQLite extraction** (ADR-0115) — sqlite_engine.go still in metaengine core
6. **Auto-projection** (ADR-0116) — no type inspection, no auto-generation
7. **Command lifecycle** (ADR-0117) — no lifecycle event streams, no DLQ projection

### No New Modules Created

- `metaengine/badgerengine/` — not created
- `metaengine/dgraphengine/` — not created
- `metaengine/sqliteengine/` — not created
- `metaengine/graphadapter/` — not created

### Dgraph/Badger Engine ADRs Not Written

- No ADR for the Badger engine implementation
- No ADR for the Dgraph engine implementation
- These need their own ADRs before implementation

### No go.work / go.mod Changes

- No modules added or removed from go.work
- No dependency changes

### No Test Changes

- No tests written or modified
- No adttest harness updates

---

## d) TOTALLY FUCKED UP

### Nothing is Irreparably Broken

No code was changed, so nothing is broken. The ADRs are documentation-only.

### However — Things I Got Wrong During the Session

1. **Initial Dgraph rejection was wrong.** I told the user Dgraph was "evaluated and skipped" and cited a status report from earlier that day as if it was authoritative. The user then pointed out that the metaengine doesn't require an embedded-library model (Postgres is already a server). I was anchored on the existing ADR-0062 zero-dep argument and failed to think critically.

2. **Initial graph merge rejection was wrong.** I cited ADR-0077 as if an Accepted ADR is immutable truth. I used the zero-dep boundary (ADR-0062) as the "hard blocker" without questioning whether the boundary itself was correct. The user had to correct me.

3. **IntentStatus proposal was bad.** I proposed a 5-state IntentStatus enum on Commands (Pending → Accepted → Rejected → Completed → Failed). The user immediately caught this as wrong: commands are immutable intents, not state machines. Lifecycle should be events. This was corrected and encoded in ADR-0117.

4. **ServerCommittedAt was a bad name.** The user pointed out we can't know when the DB "actually committed" — we can only know what the DB told us. Renamed to ServerStoredAt in ADR-0111.

5. **ADR-0046 tier diagram not updated.** The Mermaid diagram still shows metaengine in Tier 0. Only the text and tables were updated. The diagram is now inconsistent with the addendum.

---

## e) WHAT WE SHOULD IMPROVE

### ADR Quality

1. **ADR-0046 Mermaid diagram is stale.** The diagram still shows `metaengine["metaengine/"]` in the Tier 0 subgraph. Needs to move to Tier 3 subgraph and update the `metaengine --> dedup` edge.
2. **Reverse-reference sweep.** Amended ADRs should cross-reference the new ADRs that supersede their assumptions, and vice versa.
3. **ADR numbering conflict.** Two files share number 0100 (`0100-readcosts...` and `0100-redesign-scope...`). The readcosts file internally says "ADR-0099". This pre-exists this session but should be fixed.

### Missing ADRs

4. **No Badger engine ADR.** ADR for why Badger, what ADTs it covers, cost profile.
5. **No Dgraph engine ADR.** ADR for why Dgraph, how the gRPC client maps to GraphDriver, deployment model.
6. **No ADR for the shared module home.** ADR-0111 says "new shared module or expand metadata/" but doesn't decide which. This is a real open question.
7. **No ADR for how graph.GraphDriver implements metaengine.Engine.** ADR-0113 says "GraphDriver becomes an Engine" but the GraphDriver interface (RunInTx + Close) doesn't match the Engine interface (Profile + Close). An adapter or interface reconciliation is needed.

### Conceptual Gaps

8. **Record type doesn't define StreamRef.** ADR-0111 references `StreamRef` but doesn't define it. Is it the existing `id.StreamID`? A new type? A string?
9. **Record.Payload is `[]byte`** but the codec system (JSON/CBOR) is not mentioned. How does Record relate to codec/?
10. **"MetaData" vs "metadata" naming.** The field is `MetaData` (PascalCase), the type is `CommonMetadata`. The existing module is `metadata/`. Inconsistency may cause confusion.
11. **Command CausationID semantics unclear.** ADR-0111 says CausationID = "what caused this record." For events, this is the command ID. For commands, the user asked "what is causation for a Command? HTTP request ID? Actor ID?" — this was never fully resolved. The ADR says "command ID, request ID" ambiguously.
12. **Auto-projection type inspection mechanism undefined.** ADR-0116 says "inspect event struct fields" but doesn't specify whether this is reflection (runtime) or code generation (cqrs-gen, compile time). Both are mentioned as complementary but neither is chosen as the primary mechanism.

### Process Improvements

13. **Should have run doc-check.** The AGENTS.md references ADRs by number. Changing tier assignments and adding new ADRs may have broken cross-references that `cmd/doc-check` would catch.
14. **Should have verified ADR file naming.** New ADRs 0111-0117 follow the existing naming convention, but this was not verified against any naming standard.

---

## f) Up to 50 Things We Should Get Done Next

### Phase 0: ADR Polish (before any code)

1. Fix ADR-0046 Mermaid diagram — move metaengine to Tier 3 subgraph
2. Fix ADR-0100 numbering conflict (two files share number 0100)
3. Do a reverse-reference sweep: every amended ADR → new ADRs, and vice versa
4. Write ADR for Badger engine (badgerengine)
5. Write ADR for Dgraph engine (dgraphengine)
6. Write ADR deciding where Record/CommonMetadata lives (new module vs expand metadata/)
7. Write ADR for GraphDriver → Engine interface reconciliation (adapter vs interface change)
8. Define StreamRef in ADR-0111 (is it id.StreamID? new type?)
9. Resolve Command CausationID semantics (HTTP request ID? Actor ID? Both?)
10. Resolve auto-projection primary mechanism (reflection vs code-gen)
11. Run `cmd/doc-check` to verify all ADR cross-references are valid
12. Update AGENTS.md Quick Reference "Modules" row to include planned new modules
13. Update AGENTS.md module structure tree with planned modules

### Phase 1: Record Type Extraction (ADR-0111)

14. Decide module home for Record/CommonMetadata (expand metadata/ or new record/ module)
15. Define `Record` struct with `MetaData CommonMetadata`
16. Define `CommonMetadata` struct (CorrelationID, CausationID, ActorID, 3 timestamps, SchemaVersion)
17. Define `StreamRef` type
18. Make `event.Event` embed or alias `Record`
19. Make `command.Command` embed or alias `Record`
20. Update event.Metadata → remove Tombstone (ADR-0114)
21. Update command.Metadata → use CommonMetadata
22. Update metadata/ module to export CommonMetadata
23. Write tests for Record encoding/decoding (JSON + CBOR round-trip)
24. Update go.work if new module created

### Phase 2: Tombstone Removal (ADR-0114)

25. Remove `event.DetectTombstone`, `event.MarkTombstone`, `event.TombstoneStatus`
26. Update `listing/` module (tombstone detection, StatusMiddleware)
27. Update projection handlers that rely on tombstone metadata
28. Update example/taskmanager tombstone usage
29. Update tests that assert tombstone behavior

### Phase 3: ES-Native Metaengine (ADR-0112)

30. Make metaengine go.mod depend on Record type module
31. Change fold handler signatures from `any` to `Record`
32. Update `store.go` graph routing (will change again in Phase 4)
33. Update `execute.go` graph routing
34. Update `fold_classify.go` to work with Record types
35. Update all fold implementations (insertFold, edgeFold, counterFold, etc.)
36. Update adttest harness to use Record types
37. Update metaengine tests for Record-typed folds

### Phase 4: GraphBackend Deletion (ADR-0113)

38. Delete GraphBackend interface from engine.go
39. Delete graph implementations from memory_engine.go, sqlite_engine.go, pebbleengine
40. Create metaengine/graphadapter/ (GraphDriver → Engine adapter)
41. Make graph.MemoryDriver available as a metaengine Engine
42. Update planner to route ADTGraph to GraphDriver engines
43. Update adttest graph scenario to use GraphDriver
44. Update cross-engine parity tests

### Phase 5: SQLite Extraction (ADR-0115)

45. Create metaengine/sqliteengine/ module
46. Move sqlite_engine.go, sqlite_backends.go to sqliteengine/
47. Create go.mod for sqliteengine/
48. Update go.work
49. Update all imports from metaengine to metaengine/sqliteengine
50. Update adttest to import SQLite from new location

### Phase 6: New Engines (after architecture is stable)

- (Beyond 50 items) Implement badgerengine following pebbleengine pattern
- (Beyond 50 items) Implement dgraphengine following pgengine pattern (gRPC client)
- (Beyond 50 items) Implement auto-projection (ADR-0116)
- (Beyond 50 items) Implement command lifecycle event streams (ADR-0117)

---

## g) Questions I Cannot Answer Myself

### 1. Where should Record/CommonMetadata live?

ADR-0111 says "new shared module or expand metadata/" but this is a real architectural decision I cannot make. Options:
- **Expand `metadata/`**: already exists, event/ and command/ already depend on it. But metadata/ currently holds Tracing + CustomData — adding Record changes its scope.
- **New `record/` module**: clean start, but adds a module to the workspace and requires event/, command/, metaengine/ to all depend on it.
- **Put it in `event/`**: rejected by the user ("event/ IS the base hard no!").

The choice affects the dependency graph and tier assignments. I need your call.

### 2. What is CausationID for a Command?

You asked this during the session and it was never fully resolved. For Events, CausationID = the Command that produced it. For Commands:
- Is it the HTTP request ID that triggered the command?
- Is it the Actor ID of the user who initiated it?
- Is it a parent Command ID (for saga-derived commands)?
- Is it all of the above (and we need a more nuanced causation model)?
- Or should CausationID be optional/empty for commands (since nothing "caused" the intent — it just appeared)?

This affects the CommonMetadata shape in ADR-0111.

### 3. Should the ADR amendments also update the design docs?

The metaengine has canonical design docs that the ADRs reference:
- `docs/planning/meta-engine-project-definition.md`
- `docs/planning/meta-engine-design.md`
- `docs/planning/meta-engine-assumptions-and-query-planning.md`

These likely describe the zero-dependency architecture and the generic planner vision. Should they be updated/amended in the same pass, or left for the implementation phase? Updating them now keeps docs consistent; leaving them risks the next session reading stale design docs and re-introducing the old assumptions.
