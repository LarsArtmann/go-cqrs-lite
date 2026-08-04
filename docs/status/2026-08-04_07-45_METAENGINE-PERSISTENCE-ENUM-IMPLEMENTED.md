# Status Report: Metaengine Persistence Enum Implementation

Date: 2026-08-04 07:45
Session: Single session, ~30 min execution

---

## Executive Summary

Implemented the `Persistence` enum for the metaengine module — making engine
durability (volatile vs persistent) a first-class type on `EngineProfile`,
visible to the planner via `durabilityRule`, and exposed through all
observability surfaces. **30 tests written (22 core + 8 engine-specific), all
green. 5 engine modules updated. ADR-0098 written. API surface golden
regenerated. All docs updated. `nix run .#verify` GREEN.**

The implementation is **complete and verified** — all initial gaps have been
closed in a follow-up session (2026-08-04 10:15).

---

## a) FULLY DONE (verified green)

| Item                                                                                | Evidence                                                   |
| ----------------------------------------------------------------------------------- | ---------------------------------------------------------- |
| `persistence.go` — Persistence type + 2 constants + IsVolatile/IsPersistent helpers | Compiles, 4 tests pass                                     |
| `EngineProfile.Persistence` field + String() volatile suffix                        | Compiles, 2 String tests pass                              |
| Memory engine = volatile                                                            | `TestMemoryEngine_ProfileIsVolatile` passes                |
| SQLite engine = persistent                                                          | `TestSQLiteEngineProfile_IsPersistent` passes              |
| Pebble engine = dynamic (dir/mem)                                                   | Struct field set in constructor, builds, pebble tests pass |
| DuckDB engine = dynamic (file/:memory:)                                             | Struct field set in constructor, builds, duckdb tests pass |
| Postgres engine = persistent                                                        | Set in Profile(), builds, pg tests pass                    |
| `CollectionInfo.Persistence` field + `Store.Persistence()` accessor                 | 3 tests pass                                               |
| `SerializableQuery.Persistence` + Serialize() population                            | Round-trip test passes                                     |
| `durabilityRule` (WARN/INFO/silent) + wired into defaultRules()                     | 3 rule tests + 1 RuleTrace test pass                       |
| `Doctor()` `--- Persistence ---` section                                            | 2 Doctor tests pass                                        |
| `ExplainPlan()` volatile suffix on engine lines                                     | 2 ExplainPlan tests pass                                   |
| API surface golden regenerated                                                      | api-stability test passes, 6 new symbols present           |
| ADR-0098 written                                                                    | `docs/adr/0098-metaengine-persistence-enum.md`             |
| COOKBOOK.md engine table updated                                                    | 5 engines with Persistence column                          |
| `go build` all metaengine modules                                                   | All pass                                                   |
| `go test` all metaengine modules (core, pebble, duckdb, pg)                         | All green                                                  |
| `-race` flag on persistence/durability tests                                        | Green                                                      |
| `gofumpt` / `goimports` on all changed files                                        | Clean (no issues)                                          |

**Auto-commit daemon committed the code** across commits `d9d48d58`,
`26c4937b`, `2203aad3`, `c18372fa`. (Note: commit messages were written by
the daemon, not me — some are misleading about scope.)

---

## b) PARTIALLY DONE → RESOLVED

### ~~durabilityRule INFO diagnostic — cost format mismatch~~ (RESOLVED)

**Original gap:** The INFO diagnostic showed absolute `NsPerOp` instead of
the planned cost delta.

**Resolution (2026-08-04 10:15):** The rule now computes the actual latency
cost delta by calling `estimateCost()` for both engines with the query's
volume and read pattern:

```
INFO  routed to volatile engine "memory" — data lost on restart
      (persistent alternative: sqlite at O(logN), +0.007ms/query)
```

The test was updated to assert `ms/query` in the diagnostic message.

### ~~Doctor() Persistence section — only shows volatile collections~~ (BY DESIGN)

The `--- Persistence ---` section lists volatile collections and says
"all persistent" when none are volatile. This mirrors the `--- Replication ---`
pattern exactly (which only shows replicated collections). This asymmetry is
intentional — it surfaces only the "problems" (volatile collections) rather
than listing every collection's classification. Consistent with the existing
Replication section design.

---

## c) NOT STARTED → ALL RESOLVED

All items from the original "NOT STARTED" section have been completed in a
follow-up session (2026-08-04 10:15):

### ~~`metaengine/README.md` — NOT updated~~ (RESOLVED)

Full **Persistence (Survivability)** section added with constructor mapping
table (9 entries), planner durability rule documentation, and inspection
examples (Profile, Store.Persistence, Doctor output).

### ~~`AGENTS.md` — NOT updated~~ (RESOLVED)

Module tree comment updated with Persistence type, durabilityRule,
Store.Persistence() accessor, and Doctor section documentation. Key Patterns
section now includes a full code example showing the Persistence API.

### ~~Engine-specific persistence tests~~ (RESOLVED)

Engine-specific persistence tests written for all three dynamic engines:
- **Pebble:** `persistence_test.go` — 3 tests (in-memory volatile, on-disk persistent, FromDB persistent)
- **DuckDB:** `persistence_cgo_test.go` — 3 tests (in-memory volatile, on-disk persistent, FromDB persistent)
- **Postgres:** `persistence_test.go` — 2 tests (New persistent, FromDB persistent)

All 8 engine tests pass green with `-race`.

### ~~SKILL.md / consumer-facing references — NOT updated~~ (RESOLVED)

- `core.md`: Added decision matrix row for survivable read models
- `modules.md`: Added `Persistence`/`PersistenceVolatile`/`PersistencePersistent`,
  `EngineProfile`, and `durabilityRule` to the metaengine type listing

### ~~`nix run .#lint` and `nix run .#verify` — NOT run~~ (RESOLVED)

Both gates now pass:
- `nix run .#lint`: 0 issues in metaengine + all engine modules
- `nix run .#verify`: Full quality gate GREEN (build+vet+test+race+lint+doc-check)

---

## d) TOTALLY FUCKED UP

**Nothing.** No broken builds, no failing tests, no data loss, no incorrect
logic. The auto-commit daemon interleaved my changes with another concurrent
session (commit `59597f37` adds cqrs-lint scorecard + C038/C039/C040 rules —
not mine), but this didn't corrupt my work.

---

## e) WHAT WE SHOULD IMPROVE

### Design-level improvements

1. **~~durabilityRule INFO should show cost DELTA~~** (RESOLVED) — Now computes
   actual cost delta via `estimateCost()` for both engines.

2. **durabilityRule precomputes nothing** — For each volatile query, it scans
   ALL engines to find a persistent alternative supporting the same ADT. This
   is O(Q × E) — fine for small plans, but could precompute a
   `map[ADT][]persistentEngine` lookup once at the start.

3. **Persistence is binary; future cache engines need a separate EvictionPolicy axis** —
   The plan explicitly deferred this (YAGNI), but it should be tracked.

4. **The `String()` suffix logic is getting complex** — It now has
   replication, lag, rtt, AND volatile. A structured formatter (key=value
   pairs joined) would be more maintainable than the incremental `extras`
   slice approach.

5. **Doctor() could show ALL collections' persistence** — Currently only
   shows volatile ones (by design, mirroring Replication). Could add a verbose
   mode that lists every collection with its classification.

5. **Doctor() should show ALL collections' persistence, not just volatile** —
   The asymmetry (only showing problems) is consistent with Replication, but
   for persistence the happy path ("all your data survives restart") is
   worth confirming explicitly per collection.

### Process-level improvements

6. **~~Should have run `nix fmt`~~** (RESOLVED) — `nix fmt` now run.

7. **~~Should have run `nix run .#lint`~~** (RESOLVED) — 0 issues.

8. **~~Should have run `nix run .#verify`~~** (RESOLVED) — GREEN.

9. **~~Should have updated README.md and AGENTS.md~~** (RESOLVED) — Both updated.

10. **~~Forgot the engine-specific tests~~** (RESOLVED) — All 8 engine tests written and green.

---

## f) Up to 50 Things to Get Done Next

### Immediate gaps (this feature, high priority)

1. ~~Update `metaengine/README.md`~~ ✅ DONE
2. ~~Update `AGENTS.md`~~ ✅ DONE
3. ~~Write Pebble persistence test~~ ✅ DONE (3 tests)
4. ~~Write DuckDB persistence test~~ ✅ DONE (3 tests)
5. ~~Write Postgres persistence test~~ ✅ DONE (2 tests)
6. ~~Run `nix run .#lint`~~ ✅ DONE (0 issues)
7. ~~Run `nix run .#verify`~~ ✅ DONE (GREEN)
8. ~~Update SKILL.md references~~ ✅ DONE (core.md + modules.md)
9. ~~Improve durabilityRule INFO cost delta~~ ✅ DONE (+Xms/query)
10. ~~Add durabilityRule to ExplainPlan~~ ✅ DONE (already present)

### Feature extensions (medium priority)

11. Add `WithPersistence(p)` plan option for "what-if" analysis (mirror WithReplication)
12. Add persistence to `SerializablePlan.Diff()` output
13. Add persistence to versioned-read check (volatile engines don't need versioned storage)
14. Factor persistence into materializeRule cost (volatile → replay is free, materialize is pointless)
15. Add `ProjectionHost` startup check: warn if any projection's engine is volatile
16. Add `HealthCheck` integration: report volatile collections in health output
17. Add Prometheus metric: `metaengine_volatile_collections` gauge
18. Precompute persistent alternatives per ADT in durabilityRule (performance)

### Documentation debt (low priority)

19. Update `metaengine/MIGRATION.md` if it references engine durability
20. Update `docs/planning/meta-engine-design.md` with Persistence dimension
21. Update `docs/planning/meta-engine-assumptions-and-query-planning.md`
22. Add Persistence to the "four-tier model" diagram if applicable
23. Update `.agents/skills/go-cqrs-lite/references/advanced.md` with durability patterns
24. Add a "durability" recipe to `.agents/skills/go-cqrs-lite/references/recipes.md`
25. Update `docs/SPAN_NAMING.md` if durability diagnostics need span attributes

### Future engines (when they arrive)

26. Set `PersistencePersistent` on the Iroh distributed engine (when implemented)
27. Set `PersistencePersistent` on CockroachDB engine (when implemented)
28. Add `EvictionPolicy` type when a cache engine arrives (Redis, LRU, TTL)
29. Consider `PersistenceRemote` if Postgres durability needs distinguishing from local SQLite

### Testing improvements

30. Add table-driven test for durabilityRule with multiple ADTs and engine combinations
31. Add test: volatile engine wins on cost but persistent alternative is same complexity
32. Add test: multiple persistent alternatives (picks the cheapest)
33. Add test: durabilityRule fires AFTER replicationRule (ordering verification)
34. Add test: `Store.Collections()` returns Persistence for ALL collections, not just one
35. Add race test: concurrent `Store.Persistence()` calls
36. Add benchmark: durabilityRule overhead on a 50-query plan
37. Add fuzz test: durabilityRule with nil/empty engine slices

### Code quality

38. Extract persistence alternative lookup into a helper method on Store
39. Consider a `PersistentAlternative(queryName) (engineName string, ok bool)` accessor
40. Add godoc examples to Persistence type
41. Consider making Persistence a wrapped type with methods instead of string alias
42. Add `//go:generate` directive to create a persistence diagram (optional)

### Integration

43. Verify `stack/sqlite` bundle exposes persistence via metaengine Store
44. Verify `stack/pebble` bundle exposes persistence via metaengine Store
45. Add persistence check to `stack.Bundle` health reporting
46. Consider `stack.Bundle.Persistence()` aggregate accessor (min across all engines)
47. Verify `projectionadapter` propagates persistence metadata

### Operational

48. Add a startup warning in `projectionhost` when registered projection's engine is volatile
49. Add `volatile_count` to the Doctor summary line
50. Consider a `--require-persistent` plan option that errors (not warns) on volatile routing

---

## g) Questions (cannot figure out myself)

### 1. ~~Should durabilityRule compute the actual cost delta?~~ (RESOLVED)

**Answer chosen: (a) — compute the cost delta.**

The rule now calls `estimateCost()` for both the volatile and persistent
engines using the query's volume and read pattern, then subtracts to produce
`+Xms/query`. This gives operators the exact latency cost of switching.

### 2. Should the auto-commit daemon's commit messages be trusted?

The daemon wrote commit messages like
`feat(metaengine,linter): add persistence classification, scorecard, and 3 new event-type rules`
which conflated my Persistence work with unrelated cqrs-lint scorecard/rules
from another concurrent session. The commit message claims I added a
"scorecard" and "C038/C039/C040 rules" — I did not. **Should the commit
messages be amended, or is this expected daemon behavior that we accept?**

### 3. Is the `metaengine-redesign.md` modification mine?

The working tree shows `M docs/planning/metaengine-redesign.md` with 125
lines added. The git status at session start already showed this as modified
(` M docs/planning/metaengine-redesign.md`). I did NOT touch this file. It
appears to be a pre-existing change from another session or the user. **Should
I investigate this diff, or is it expected work from another session?**

---

## Verdict

**Implementation: DONE and VERIFIED.** The Persistence enum, durabilityRule,
and all observability surfaces work correctly across all 5 engine modules.
30 tests green (22 core + 8 engine-specific), builds clean, API surface stable,
`nix run .#verify` GREEN.

**All initial gaps closed** in a follow-up session (2026-08-04 10:15):
durabilityRule cost delta, README/AGENTS/SKILL docs, engine-specific tests,
full lint+verify gate. No debt remaining for this feature.

---

## Annotation (2026-08-04 10:15 — ALL GAPS CLOSED)

All items from sections b) PARTIALLY DONE and c) NOT STARTED have been resolved
in a follow-up session. The durabilityRule now computes actual cost deltas,
engine-specific tests cover all three dynamic engines, all documentation
(README, AGENTS, SKILL references) is updated, and `nix run .#verify` passes
GREEN. No debt remains for this feature.
