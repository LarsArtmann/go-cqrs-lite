# Status Report: Metaengine Scan-Family Deduplication

| | |
| --- | --- |
| **Date** | 2026-09-22, 23:49 CEST |
| **Session scope** | art-dupl clone elimination: `MapScanKeyValues` family (duckdb/sqlite/mysql/pg engines) + `MultiGroupedAggregate` family (duckdb/sqlite/pg) |
| **Trigger** | `art-dupl --sort total-tokens -t 7 --type-aware` → 1 clone group (`duckdbengine/planned_parity.go:43-75` vs `sqliteengine/planned_parity.go:42-74`) |
| **Tool** | art-dupl 0.7.0-81ce00b (system profile) |
| **Final state** | t7 type-aware: **0 shown clone groups** (was 1). Repo t3 gate: 74 → 69 new groups (all remaining pre-date this session) |
| **Working tree** | clean; all work absorbed by auto-commit daemon (no authored commits) |

## Executive Summary

The reported clone was the visible tip of a four-engine copy-paste family. Two rounds of
extraction moved the shared mechanics into `metaengine/scan.go` (the established home for
cross-engine scan plumbing), eliminating the reported group, two gate-unmasked satellites,
and a second family (`MultiGroupedAggregate`) that surfaced at the repo's own t3 threshold.
All seven touched module suites pass (incl. CGo duckdb and real-DB pg). The repo-wide
duplication gate was already red before this session (74 new groups vs the 2026-09-18
baseline); this session reduced it to 69 and left the re-pin/triage decision open.

---

## a) FULLY DONE

1. **Reported clone eliminated.** `MapScanKeyValues` scan loops extracted from
   duckdbengine + sqliteengine `planned_parity.go`; t7 type-aware now reports 0 shown
   groups (verified twice).
2. **Full family deduplicated (4 engines).** mysql `backfill.go` + pg `backfill.go` had the
   identical pattern (found at t3 gate exposure, not in the user's t7 report); both shrank
   to 3-line bodies delegating to core.
3. **`MultiGroupedAggregate` family deduplicated (3 engines).** duckdb `aggregations.go`,
   sqlite `aggregations_grouped.go`, pg `aggregations.go` — identical 34-line drain loops
   → `metaengine.ScanGroupedAggregates` + unexported `scanGroupedRow` (split to respect the
   30-line function cap).
4. **New shared core API, deliberately minimal:** `ScanKeyValuesPage`, `ScanLimit`,
   `CursorArg`, `ScanGroupedAggregates`. Error strings preserved byte-identically via the
   established `label` prefix pattern; `ScanJSONKeyValues` was unexported before any
   release (no API break).
5. **Intentional clones annotated, not merged.** badger/bbolt/pebble
   `register_durability_test.go` trio got `//art-dupl:accept` (rule-19 pattern:
   dep-isolated engine modules each need their own registration test).
6. **All tests green** (`GOWORK=off go test ./... -count=1`): metaengine core,
   sqliteengine (3.2s), pgengine (17.6s, real-DB backfill test), duckdbengine (CGo,
   117–125s), badger (1.4s), bbolt (493s), pebble (1.9s). Targeted parity tests
   (`TestDuck/SQLite_MapScanKeyValues_Paged`, `TestPgBackfillPlannedCollection`) pass.
7. **Gates run:** api-stability golden regenerated + `TestEvery` meta-test; doc-check
   (1200 refs valid); check-file-size (scan.go 296/350, no new offenders from my files);
   BuildFlow `format` + golangci-lint fan-out (all touched modules clean).
8. **Final API surface verified:** exactly `CursorArg`, `ScanGroupedAggregates`,
   `ScanKeyValuesPage`, `ScanLimit` added; `ScanJSONKeyValues` never shipped.
9. **Pre-existing red gates attributed, not blindly "fixed":** cqrs-lint 10 errors
   (A001/A008 in command/event/query/catalog/turso — files untouched this session);
   file-size offender `catalog/eventcatalog/frontmatter_convert.go` (370 lines) created by
   a **parallel session** at 23:36, not by me.

## b) PARTIALLY DONE

1. **Repo duplication health.** Session net: 74 → 69 new groups at t3. The reported family
   is fully clean, but the gate remains red on pre-existing drift (baseline pinned
   2026-09-18, 60 groups). Re-pin vs triage decision deliberately left to owner.
2. **Repo lint health.** All my modules clean; systemtest/example warnings and the
   eventcatalog `too many errors (typecheck)` belong to other/pre-existing work.
3. **Root-cause understanding of the scan-pattern spread.** Confirmed for 4 of 5 SQL
   engines (duck/sqlite/mysql/pg); **turso never checked** — assumption, not verification.
4. **art-dupl output anomaly.** Mid-session the same binary+flags switched formats
   (`Found total 0 clone groups.` → `Detected 317, 0 shown (65 non-actionable, 252 filtered
   suppressed)`) and file count 2991→2992. End state is verified repeatedly, but the
   intermediate discrepancy was never root-caused.

## c) NOT STARTED

1. CHANGELOG `[Unreleased]` entries for the 4 new exports (+ `check-changelog-symbols` run).
2. SKILL.md / `references/*.md` documentation of the new helpers as the canonical
   engine-authoring pattern (doc-check passes only because nothing references them yet).
3. Triaging the 69 remaining t3 groups (harmful vs accept vs dedupe) into a decision list.
4. Turso (and any non-Postgres/MySQL dialect) sweep for the same scan/aggregate pattern.
5. AGENTS.md internal-contract entry: "engine scan/aggregate loops must use metaengine
   scan helpers" (convention discovered by this session, not yet written down).
6. Authored commits — everything landed as `chore: auto-commit` daemon noise; one commit
   (4bd4e1670) even mixes my annotations with the parallel session's eventcatalog work.
7. CI gate hardening: `check-duplication` silently **SKIPS** when `art-dupl` is not in
   PATH (flake comment admits it) — soft gate.
8. TODO_LIST entries for the follow-up campaign.

## d) TOTALLY FUCKED UP

1. **False mid-session verification.** After round 1 I reported "t7: 0 clone groups" as
   proof of done — yet the durability-test trio and the aggregations pair (both
   pre-existing, both shown at t7 later) existed at that moment too. Same stated binary,
   different verdicts; I never explained it. If I had stopped there, the session would
   have ended on a false green. The FINAL state is solid; the intermediate claim was not.
2. **Ignored evidence I already had.** My very first `rg -l MapScanKeyValues` returned
   `mysqlengine/backfill.go` and `pgengine/backfill.go` — same function, same doc comment —
   and I consciously scoped them out because "the report only listed 2 clones". The gate
   then forced round 2. One judgment call up front would have saved a full
   rebuild-retest-golden cycle (~15 min) and an API churn round.
3. **Unnecessary API churn.** Round 1 exported `ScanJSONKeyValues`, regenerated the golden,
   then round 2 unexported it. Harmless (unreleased) but sloppy sequencing — the family
   shape should have been designed before the first edit.
4. **History pollution.** No authored commits at phase boundaries (AGENTS.md explicitly
   warns the daemon will absorb mid-phase edits); my work is now spread across six
   `chore:` commits, one of which interleaves with a parallel session's unrelated changes.
   Bisecting this later will be unpleasant.

## e) WHAT WE SHOULD IMPROVE

1. **Run the repo's own gate FIRST**, not just the user's ad-hoc flags — the contract is
   `check-duplication` (t3 semantic + baseline), and its threshold unmasked two extra
   groups that t7 hid.
2. **Treat identical function names as family, immediately.** A grep hit with the same
   signature + doc comment in a sibling module is a clone candidate regardless of what the
   detector's current threshold reports.
3. **Never accept a single tool run as "clean"** — re-run with fresh eyes/flags before
   claiming zero; investigate any output-format or count anomaly before building on it.
4. **Check gate sensitivity before extracting:** the t3 gate saw smaller clones behind the
   t7 group (AGENTS.md §14 documents this unmasking behavior — should have been applied
   proactively).
5. **Commit authored work at phase boundaries** (source `scripts/go-env.sh`, `git add` the
   specific files, commit) so the daemon absorbs nothing and history stays attributable.
6. **Snapshot `git log` freshness before whole-repo gates** (file-size, duplication, lint
   fan-out) — a parallel session's in-flight commits otherwise get attributed to you.
7. **Design the target API shape before the first edit** when a clone might be a family
   (drain primitive vs page wrapper vs cursor/limit utils were all needed).
8. **Make the duplication gate deterministic and nix-provisioned** instead of
   PATH-lookup-and-skip.

## f) NEXT (up to 50)

**Duplication follow-up**
1. Decide baseline policy: wholesale re-pin the 69 vs triage-first (blocks CI green).
2. Triage the 69 groups into harmful / accept-annotate / dedupe-now buckets; write the
   list to `docs/planning/` or TODO_LIST.
3. Dedupe `mysqlengine/vector.go:54-74` vs `sqliteengine/vector.go:88-108` (same
   `sql.ErrNoRows` vector-read shape, 21 lines each).
4. Same for badger vs pebble `seedSeqCounters` pair (`engine.go:120-130/156-167`).
5. Same for bbolt `store.go:148-154` vs pebble `command_store.go:158-164` (empty-batch guard).
6. Same for pebble `layout_planner.go` internal pair (134-140 vs 182-188).
7. Same for `typed_reader.go` internal pair (68-74 vs 82-88).
8. Evaluate the `rule_degraded_adt/rule_durability/rule_replication` trio
   (meta, ok := ctx.Store.queries…) for a shared rule-prelude helper.
9. Evaluate `fold_inference.go` vs `infer_named.go` pair (14-16 lines).
10. Evaluate the graph BFS trio (`memory_graph.go` / `sqliteengine/graph.go` /
    `sqliteengine/graph_undirected.go`).
11. Evaluate the `PlannedTables` sort/count trio (duckdb/sqlite/pg) for a core helper.
12. Evaluate iroh transport pair (`loopback/transport.go` vs `quic/transport.go`).
13. Sweep for stale `//art-dupl:accept` directives pointing at since-fixed clones.
14. Decide whether benchkit `phases_projection.go` internal triple warrants a loop helper
    or an accept.
15. Re-pin the baseline on a committed tree once triage lands (regen is legitimate after
    structural shifts; annotation-first for intentional clones).

**Docs & conventions**
16. Document `ScanKeyValuesPage`/`ScanGroupedAggregates`/`ScanLimit`/`CursorArg` in
    SKILL.md references (recipes/modules) as the canonical SQL-engine pattern.
17. Add AGENTS.md internal contract: new engine scan/aggregate reads must delegate to the
    metaengine scan helpers.
18. CHANGELOG `[Unreleased]` entries + run `check-changelog-symbols`.
19. Update the `KeyScanBackend` interface doc to state the limit-normalization contract
    (≤0 → 500) now enforced in core.
20. Note the label-prefix error convention for engine reads in `references/faq.md` if absent.

**Verification debt**
21. Run full `nix run .#verify` once the parallel session's eventcatalog work lands.
22. Run `nix run .#verify-ci` (GOWORK=off per-module matrix parity).
23. Run `nix run .#check-arch` and `nix run .#check-coverage` after the move.
24. Run `nix run .#check-error-taxonomy` (new `fmt.Errorf` wraps introduced no new codes — confirm).
25. Add a shared `adttest`-style KeyScanBackend conformance harness (paging, cursor="" first
    page, hasMore-on-exact-multiple edge) run by all 4 engines.
26. Add a test pinning `ScanGroupedAggregates` row ordering across engines.
27. Investigate the art-dupl 0.7.0 output-format switch mid-session (PATH resolution?
    profile activation? corpus 2991→2992 file?).
28. Reconcile the 23:02 "2 new" vs later "72 new" gate readings on near-identical trees;
    consider `BUILDFLOW_NO_RESULT_CACHE=1` for gate runs.
29. Make `check-duplication` fail hard (not SKIP) when art-dupl is absent; provision
    art-dupl from the flake so CI and local use the same version.
30. Investigate bbolt's 493s module suite (slowest by far; soak markers?).

**Cross-engine consistency (bugs-adjacent, found this session)**
31. Decide: sqlite/mysql `MapScanKeyValues` used raw `e.db` while duck/pg used `e.conn(ctx)`
    (tx-aware). I preserved original behavior — but is bypassing active transactions in
    backfill reads intentional? Document or unify.
32. Sweep all engine read paths for the same `e.db` vs `e.conn(ctx)` inconsistency.
33. Verify turso engine for the scan/aggregate pattern (c.4 — the one engine never checked).
34. Confirm `value::text` (pg) vs `VARCHAR` (duck) vs backtick quoting (mysql) dialect
    matrix is covered by tests for NULL/empty/unicode keys.

**Parallel-session & repo hygiene**
35. Coordinate with the eventcatalog session: their `frontmatter_convert.go` is a 370-line
    file-size ratchet offender; split or baseline it (owner: them).
36. Re-run `nix run .#check-eventcatalog` + `check-file-size` after their work lands.
37. Fix `eventcatalog/exporter_message.go` typecheck errors ("too many errors" in lint).
38. Fix systemtest lint warnings (gocyclo `integration_lifecycle_test.go:26`,
    nestif `testdsn_test.go:69`, prealloc `sqlite_wiring_test.go:321`).
39. Fix `example/metaengine-quickstart` warnings (errcheck RemoveAll/Close, duplicate godoc,
    magic number 0o600).
40. Triage the 10 pre-existing cqrs-lint errors (A001 command/event/query store+event.go,
    A008 catalog/types_phantom.go, turso advisor) — baseline or fix.
41. Adopt authored-commit-at-phase-boundary for future sessions (avoid daemon absorption).

**Linter/tooling ideas spawned by this session**
42. cqrs-lint rule candidate: flag `json.Unmarshal`-based scan loops inside
    `metaengine/*engine` modules that bypass the shared helpers.
43. Evaluate art-dupl annotation format stability across versions (accept directives are
    load-bearing; a 0.7.x format change would silently un-suppress groups).
44. Add the 4 new exports to the api-stability "high-risk aggregate exports" watchlist if
    their signatures ever grow (they're consumed by 4 engine modules).
45. Consider a `KeyScanBackend` page-size constant export (500 is now encoded in
    `ScanLimit`; engines' doc comments repeat the number).

**Nice-to-have**
46. Benchmark the helper path (per-row `raws`/`scanTargets` allocs) in benchkit sweep.
47. Showcase the shared helpers in `example/metaengine-quickstart` (examples are demos).
48. Short ADR or planning note: "engine-tier shared plumbing lives in metaengine core"
    (formalizes the QuoteIdent/DeferClose/scan-helpers precedent).
49. Update `docs/agents/module-map.md` engine rows if the helper contract changes
    maintenance notes.
50. After everything: rerun `art-dupl --sort total-tokens -t 7 --type-aware` +
    `nix run .#check-duplication` as the closing gate pair, and record results in
    TODO_LIST.

## g) QUESTIONS (cannot answer myself)

1. **Baseline policy for the 69 remaining t3 groups:** wholesale re-pin now (accepting
   ~3 days of unreviewed drift into the golden), triage-first with the gate left red, or
   re-pin + schedule a dedup campaign against the pinned list? This blocks CI-green and I
   won't pick it unilaterally.
2. **Transaction routing in backfill reads:** sqlite/mysql `MapScanKeyValues` read via raw
   `e.db` while duck/pg route through tx-aware `e.conn(ctx)`. I preserved the original
   split. Is bypassing an active transaction during planned-table backfill intentional
   (e.g. backfill must see committed base rows), or should all engines route via `conn(ctx)`?
3. **Contract status of the new helpers:** should `ScanKeyValuesPage`/`ScanGroupedAggregates`
   become the documented, mandatory pattern for engine authors (SKILL.md + AGENTS.md
   convention + possible cqrs-lint rule), with turso swept for compliance — or are they
   opportunistic internal helpers only?

---

*Report generated 2026-09-22 23:49 CEST. All session work: metaengine/scan.go + 7 engine
modules; final t7 = 0 shown; t3 gate 74→69 (pre-existing drift documented above).*
