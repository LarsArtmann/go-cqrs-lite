# Status Report — testfixtures Research Session (2026-10-17 08:23 CEST)

> **Scope:** This report covers ONLY this session's run: the
> `github.com/go-testfixtures/testfixtures` research question ("what can we
> learn?"), its synthesis, the TODO_LIST harvest, and things *noticed* along
> the way. No unrelated repo-wide research was performed (per instruction).
> Format: `.md` at operator's explicit demand (HTML is otherwise canonical for
> status reports — override flagged).

---

## Session summary

One research task, executed end-to-end:

1. Loaded `verify-external-claims` skill; verified testfixtures facts from
   primary sources: GitHub API metadata (v3, MIT, 1,238 stars, active since
   2016-04-05, last push 2026-10-15), README, and **11 source artifacts read
   directly** (`testfixtures.go`, `helper.go`, `postgresql.go`, `dump.go`,
   `time.go`, `bytes.go`, `dbtests/containers_test.go`, `dbtests/examples.md`,
   `.github/workflows/test.yml`, repo tree, `dbtests/testdata/` layout).
2. Grounded the comparison in OUR repo: `docs/agents/gotchas-testing.md`
   (full read), `testutil/pgtestcontainer/pgtestcontainer.go` (full read),
   repo-wide `fixture|Fixture` grep, journal-dump-affordance grep
   (confirmed: **no journal→file dump/replay path exists** — only benchkit
   stream discovery).
3. Delivered the synthesis in chat (adopt / borrow / validate / don't-copy),
   every external claim primary-sourced.
4. Harvested 2 actionable rows into `TODO_LIST.md` §"Journal fixtures
   (testfixtures lessons, harvested 2026-10-17)" + section-index entry.
5. During THIS review, retroactively verified: no duplicate backlog row
   existed (grepped TODO_LIST for journal-dump/replay/scenario-from-production
   patterns), no existing testing-isolation coverage in skill references
   (grepped recipes/faq/modules for isolation/parallel/txdb/template-DB), and
   the `DangerousSkipTestDatabaseCheck`/`DangerousSkipCleanupFixtureTables`
   option names exist in source (`testfixtures.go:280-292`) — previously
   README-only.

---

## a) FULLY DONE

| # | Item | Evidence |
|---|------|----------|
| 1 | Primary-source verification of testfixtures (metadata, README, 11 source files) | working copies in `/tmp/testfixtures-src/`, fetched 2026-10-17 |
| 2 | Our-side grounding (testing conventions, PG harness internals, fixture-landscape grep, dump-affordance absence) | `docs/agents/gotchas-testing.md`, `testutil/pgtestcontainer/pgtestcontainer.go:1-200` |
| 3 | Synthesis delivered with verified claims + file:line citations | chat answer, this session |
| 4 | TODO_LIST harvest: new section + 2 rows (`dump⇄load` 🔥 Effort L; isolation-strategy doc, Effort S) + index entry | `TODO_LIST.md:81,128-152` |
| 5 | Retroactive duplicate-checks + claim upgrade (Dangerous\* names in source) | this session's review run |

## b) PARTIALLY DONE

| # | Item | What's missing |
|---|------|----------------|
| 1 | testfixtures deep-dive (~80% coverage) | NOT read: `CHANGELOG.md` (9-year evolution), `cmd/testfixtures/` CLI, `dbtests/common_test.go` (suite structure), `sqlite.go`/`mysql.go` internals, `shared/shared.go` (`Queryable`), `mocks_for_test.go`, `clickhouse.go`/`sqlserver.go`/`spanner.go` bodies |
| 2 | Both TODO rows harvested but 0% executed | by design (research session) — implementation untouched |
| 3 | Line-number citations in the chat answer came from raw master working copies | accurate, but not re-checked against upstream canonical numbering (they are the same files; low risk) |

## c) NOT STARTED

1. TODO row 1: journal dump⇄load round-trip (design + implementation).
2. TODO row 2: consumer-facing test-isolation strategy doc.
3. Skill-reference updates — none needed (no API surface changed this session).
4. Comparative write-up vs. the alternatives they document (factory-go, fixtory, txdb, dbcleaner) — not examined at all.

## d) TOTALLY FUCKED UP!

**Nothing in the repo is broken by this session** — zero code changes, one
doc-ledger edit (verified applied). But honest process failures, zero damage
this time (loaded dice — the checks I did late happened to come back clean):

1. **Wrong order of operations on duplicate-checking.** I wrote the TODO_LIST
   section and recommended the isolation doc BEFORE grepping the ledger and
   the skill references for existing coverage. Both retro-checks passed, but
   the protocol is backwards: ledger-first, then write. Had there been an
   existing row, I'd have created a backlog split brain.
2. **README-only sourcing for exact exported names.** The `Dangerous*` option
   names were cited from the README; source verification happened only during
   this review. Exact identifiers belong to source, not docs.
3. **New top-level TODO section instead of thematic placement.** The dump⇄load
   row arguably belongs under an existing systemscenario/metaengine section;
   I added a fresh dated section. Taxonomy sprawl risk (the Declined/sections
   list keeps growing).
4. **Deliberately did not load the `go-cqrs-lite` SKILL.md** for a task whose
   output shapes repo recommendations. Defensible (AGENTS.md carried the
   needed context; the skill targets API usage) but borderline — recorded as
   a judgment call, not compliance.

**Noticed, NOT investigated (out of session scope, unowned):**

- `example/taskmanager/` shows **7 LSP compile errors at committed HEAD**
  (`undefined: servertiming` ×4, `undefined: systemRegisterQuery`,
  `undefined: idStreamID`, `undefined: parseIDString`) plus a `go mod tidy`
  warning (`httputil/server_timing` unused). Files are committed-clean (last
  touches were auto-commit daemon). Either the daemon absorbed a broken
  mid-edit state (the documented failure mode) or LSP staleness — **not
  build-verified** because the instruction forbade unrelated research.
- `catalog/architecture/{html,model,widgets}.go` — untracked files from a
  parallel session; untouched by me, correctly so.
  > **RESOLVED 2026-10-17 (owner session):** the parallel session was the
  > `catalog/architecture` implementation; the files landed in `8e39faad5`
  > with ADR-0159, recipe §2.48 and the api-stability golden rows. No action
  > owed.

## e) WHAT WE SHOULD IMPROVE (session-process lessons)

1. **Ledger-first protocol for research sessions:** grep `TODO_LIST.md` +
   skill references for thematic overlap BEFORE writing recommendations or
   harvesting rows. (This session: done backwards, got lucky.)
2. **Source-first for identifiers:** any exact exported name / flag / option
   cited in an answer gets verified in source even when the README shows it.
3. **Coverage manifest:** research answers should state which upstream files
   were read vs. skipped, so a later session can resume the dive cheaply
   (this report's §b-1 does that retroactively).
4. **Refuse list padding:** section (f) below is capped at 30 honest,
   session-derived items — padding to 50 would have required the unrelated
   repo-wide research the instruction forbade.
5. **Harvest routing already correct here:** the 2 Pareto-grade items went
   into TODO_LIST during the session (pre-report); the remaining (f) items
   stay report-scoped deliberately — bulk-harvesting 30 brainstorm items
   would violate docs-health routing rigor.

## f) Next things (30 honest items; cap explained in §e-4)

**Cluster A — execute TODO row 1: journal dump⇄load (🔥 Pareto)**

1. Decide file format doctrine (see question g-3) — typed generated vs
   self-describing envelope.
2. Decide train: v4.x additive (`systemscenario`/`eventtest` surface) vs v5
   core (ADR-0152 topology) (see question g-2).
3. ADR: scope = journal segments or whole streams; PII/redaction stance;
   versioning of the fixture format.
4. Design the dump side: `Dump(journal, filter) → file(s)` — reuse
   `Journal.ReadAll`/`ReadStreamFrom` shapes; check `flightrecorder`
   (external repo) for capture-format overlap first — do NOT reinvent.
5. Design the load side: file → `systemscenario.Given` seeds AND/OR direct
   `AppendBatch` into a fresh engine (both entry points?).
6. Blast-radius guard: default refusal on non-test targets (their
   `EnsureTestDatabase` analog — DB name/stream prefix heuristic or explicit
   opt-in marker), `Dangerous`-prefixed escape options.
7. Watermark-based skip: if load target already has journal ≥ fixture
   watermark, skip re-append (their checksum-skip analog).
8. Parse-once/load-many: compile fixture file once into immutable event
   batch; reuse across scenarios (events are immutable — sharing safe, same
   contract as load coalescing).
9. Escape hatch audit: any smart coercion (time parsing, hex bytes) needs a
   RAW/explicit opt-out — steal the lesson, not the stringly format.
10. Rich failure context: load errors must carry file, record index, stream,
    SQL-equivalent context (their `InsertError` pattern).
11. Round-trip property test: dump(load(f)) == f across JSON+CBOR codecs.
12. Conformance: fixture replay equivalence vs live-append equivalence on ≥3
    engines (sqlite, pg, memory).
13. Consumer docs: recipes.md section + FAQ entry once shipped.

**Cluster B — execute TODO row 2: test-isolation strategy doc (S)**

14. Draft `dbtests/examples.md`-style matrix: sequential shared / per-test
    disposable DB / txdb / PG template DBs — each with pros/cons + runnable
    example against `system.New`.
15. Add the ES-native 5th strategy: reset-by-replay (fresh engine + journal
    CatchUp) — our differentiator, they cannot have it.
16. Verify the txdb DDL caveat against OUR engines (CREATE TABLE at New — PG
    transactional DDL works, MySQL does not; write the honest engine matrix).
17. Home decision: skill `references/recipes.md` §new vs
    `docs/testing-isolation.md` (doc-check scan implications either way).
18. Cross-link from cqrs-htmx companion docs (fleet-first audience).

**Cluster C — harden this session's claims (small)**

19. Build-verify `example/taskmanager` (or confirm LCP staleness) — settles
    the noticed-diagnostics question mechanically.
20. Read testfixtures `CHANGELOG.md` — 9 years of evolution data; check for
    withdrawn features we were about to admire.
21. Read `dbtests/common_test.go` — their suite structure may hold harness
    lessons the README doesn't.
22. Re-pin line-number citations against upstream if any get quoted further.

**Cluster D — noticed-item triage (needs ownership answer first)**

23. If taskmanager breakage is real: fix or revert the broken committed
    state; consider a `#build-examples` CI leg reaction.
24. `catalog/architecture/*.go` untracked files: parallel session owns; do
    nothing until it lands.

**Cluster E — optional research extensions (ROADMAP fuel, not commitments)**

25. Compare with `fixtory` (go:generate typed factories) — relevant if g-3
    answers "typed generated".
26. Compare with `pgtestdb` internals — template-DB clone cost model (we
    dismissed for internal use on reasoning, not measurement).
27. Their `shared.Queryable` (tx-or-db unification) vs our
    `storage/sql` tx handling — is there a leak we papered over?
28. Their errgroup-parallel dialect `init()` — applicable to our engine
    constructor metadata probes (turso version probe, pg capabilities)?
29. Their pinned-action-SHA CI hygiene vs our Nix-based CI — delta note only.
30. Their README "Alternatives" section pattern — worth an honest
    alternatives section in OUR README for fleet onboarding? (first-party
    fleet scope may make it moot).

## g) Questions I cannot figure out myself

1. **taskmanager ownership:** `example/taskmanager/` carries 7 LSP compile
   errors at committed HEAD (undefined: `servertiming` et al.) — is that a
   parallel session's in-flight state to leave alone, or should the next
   session build-verify and triage/fix it?
2. **Train choice for dump⇄load:** v4.x additive feature on
   `systemscenario`/`eventtest`, or fold into the v5 core design
   (ADR-0152)? This is a product/scope call that shapes the ADR.
3. **Fixture format doctrine:** typed generated Go (fixtory-style,
   compile-checked, no runtime parsing) vs self-describing runtime envelope
   (JSON/CBOR, ADR-0044-style, language-agnostic, PII-redactable)? Both
   defensible for the fleet; which doctrine should the ADR commit to?

---

*Point-in-time snapshot. The 2 Pareto-grade actionables are already in
TODO_LIST (harvested pre-report); remaining (f) items are intentionally NOT
bulk-harvested (routing rigor). Auto-commit daemon will absorb this file;
no manual commit per harness contract.*
