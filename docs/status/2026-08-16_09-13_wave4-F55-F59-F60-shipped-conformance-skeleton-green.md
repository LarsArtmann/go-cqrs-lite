# Status Report — Perf Pareto Wave 4: F55 + F59 + F60 Shipped (09:13)

> Session 08:19 → 09:13, 2026-08-16. Continued execution of
> `docs/planning/2026-08-16_03-18_PERF-PARETO-SAFETY-FIRST-EXECUTION.md`.
> Queue on entry: F55 → F59 → F60 → F46 → F47-F49 → final verify.
> First action: corrected the todo list (F54 was already complete, evidence
> in the 08:19 report).

---

## a) FULLY DONE (verified green this session)

### F55 — `event.DecorateJournal` (ADR-0126 completion) ✅

The headline item. The TODO said "schema upcasting path still hand-wraps
Journal+upcasters; the DecorateStore-equivalent for journals is the missing
piece of ADR-0126." Research found the hand-wrapper was worse than
incomplete: **`VersionedSeekableJournal` silently dropped `StreamingJournal`**
(ReadStream/ReadStreamFrom) — 4 backends implement it (bbolt, pebble, memory,
eventstore).

- `event/journal_middleware.go` (NEW): `DecorateJournal(journal, sourceT)` —
  preserves Journal + SeekableJournal + StreamingJournal + io.Closer.
  Streaming reads apply the transform per 128-event chunk
  (`transformingIterator`), keeping memory bounded. New sentinel
  `event.ErrInnerStoreNotStreaming` for inner journals lacking streaming.
- `event/journal_middleware_test.go` (NEW): 13 tests — nil passthrough,
  panic-on-nil, transform on ReadAll/ReadFrom, chunk-boundary streaming
  (2×128+7 events), filtering transform, not-streaming/not-seekable
  rejections, transform error, error passthrough, Close delegation. All
  pass, also with `-race`.
- `schema/versioned_journal.go`: `NewVersionedSeekableJournal` rewritten as a
  **deprecated shell** over `DecorateJournal` + `UpcastSourceTransform` (same
  pattern as the `VersionedStore` shell; removal at v5). Keeps its
  error-wrapping surface; adds a defensive seekable-assert using the static
  `event.ErrInnerStoreNotSeekable` sentinel.
- `schema/versioned_journal_test.go`: +1 equivalence test (shell vs canonical
  produce identical upcasted reads).
- `projectionhost/versioned_journal_integration_test.go`: converted to the
  canonical form (`DecorateJournal` + `UpcastSourceTransform` through
  `projectionhost.New`) — proves the real-world composition.
- **Standalone-build repair (discovered mid-task)**: `event` GOWORK=off was
  broken since wave 3 — `asrecord.go` uses `metadata.BrandedString` (untagged
  commit; `metadata/v4.5.0` predates it). Fixed: pin remains v4.5.0-tagged
  but with local `replace ../metadata`; `schema` gained `event` + `metadata`
  replaces (documented replace-cascade gotcha applied). All three replaces
  drop once metadata v4.5.1+ and event are tagged — fed to the replace-drop
  sweep.
- Golden regen: 4111 → 4120 exports (+`DecorateJournal`,
  +`ErrInnerStoreNotStreaming`, +iterator methods).
- Docs: AGENTS.md contract 16 now covers journal wrapping; skill `core.md`
  recipe + decision matrix + `modules.md` updated to canonical form;
  CHANGELOG entry; TODO_LIST flipped `[x]`.
- Gates: tests + race green (event/schema/projectionhost), lint **0 issues**,
  doc-check 877 refs valid.

### F59 — Seq-carrying journal reads design ✅ (design-only per plan)

- `docs/planning/SEQ-CARRYING-JOURNAL-READS.md` (NEW): full design for
  `StreamLogEntry{Seq,Value}` + `SeqSeekableStreamLog` capability interface.
  Grounded in verified code facts: per-engine OFFSET-vs-seek table (sqlite
  OFFSET at `stream_log.go:59`, pebble O(log n) seek, bbolt Seek, memory
  linear), adapter costs (`AdapterCore.ReadFromAfter` O(N) ID resolution;
  `EventAdapter.seqCache` position arithmetic at `adapter_event.go:247` and
  cold-miss full `JournalReadAll`). Includes the cursor-safety argument
  (why raw seq is safe as an opaque token but unsafe as a position),
  per-engine token sources (all SQL engines already have the
  `(collection, seq)` index — zero migration), adapter integration,
  O(N²/P) → O(N + (N/P)log N) impact analysis, 5-step rollout with
  conformance tests first, 4 rejected alternatives, 3 risks (DuckDB seq
  semantics, Dgraph, token leakage across engine migration).
- TODO_LIST item annotated with the design pointer; implementation stays
  open (Effort M).

### F60 — Engine capability conformance skeleton ✅

- `metaengine/adttest/conformance.go` (NEW): `RunCapabilityConformance`
  (test harness) + `AuditCapability` (plumbing-free, for reports/Doctor) +
  `CapabilityTable` + `KnownGaps`. Three rules: over-declaration (declared
  native, no backend), under-declaration (backend present, undeclared),
  DegradedADTs ⊆ Supports. Structural check only — behavioral parity stays
  in RunMatrix.
- `metaengine/adttest/conformance_test.go` (NEW): violation-path tests with
  a `lyingEngine` fake (one violation per rule + exact count + KnownGaps
  suppression + memory-engine negative control).
- Wired into **10 places**: root `metaengine/adt_matrix_test.go` (memory,
  sqlite) + all 9 engine modules (pebble, bbolt, badger, pg, mysql, duckdb,
  dgraph, turso, iroh). Vet-OK everywhere; ~~**ran green** on
  memory/sqlite/pebble/bbolt/badger/iroh/pg/duckdb (pg/duckdb via their
  live/testcontainer paths).~~ CORRECTION (2026-08-16): iroh was later found RED
  by the conformance suite (11-33 report) and fixed (12-39) — the 09:13 "iroh green"
  observation was module-suite green, not conformance green.
- **Finding**: the historic "6 engines over-declare" issue is ALREADY
  neutralized — every unimplemented ADT on pg/duckdb is marked degraded, not
  claimed native. The skeleton now enforces that permanently; drift fails CI.

### Session hygiene

- Todo list corrected at session start (F54 → completed).
- `metaengine` root + `adttest` full suites green (soak-skips per AGENTS.md).
- Golden regenerated after every export-affecting change (same-edit rule);
  final count 4120.

## b) PARTIALLY DONE

~~- **Final full verify** — NOT run yet. All per-module gates green
(tests/race/lint/golden/doc-check where touched), but the end-to-end
`nix run .#verify` (build + vet + test + race + lint + doc-check +
doc-assertions, all 82 modules) is still owed before any "wave 4 GREEN"
claim. Also un-run: `#check-coverage` (schema/event coverage moved),
`#check-duplication` (new decorator/shell code), `#check-arch` (no new
deps added, should pass).~~ done — full `#verify` GREEN (2026-08-16 13:15 run #4) incl. race + doc-check + doc-assertions; `#check-coverage` + `#check-duplication` also EXIT=0

~~- **mysql / dgraph / turso conformance tests** — wired and vet-clean but not
executed locally (need running servers; they skip cleanly without them, as
their ADT matrices already do). Will run under `#test-integration`.~~ partial — iroh conformance was found RED later that day (11-33) and fixed (12-39); mysql/dgraph/turso live conformance runs still not recorded

## c) NOT STARTED

~~- **F46** — go-codec `UnwrapDecode` first-byte sniff (external sibling repo
`/home/lars/projects/go-codec`).~~ done — go-codec `autodetect.go` first-byte sniff shipped

~~- **F47-F49** — benchstat baselines for the 5 BENCHMARKS.md ledger paths;
false-sharing measure-then-pad for worker counters / multiSeqCounter /
SSEReplay.seq @-cpu=16,32.~~ done — benchstat baselines + all three measure-then-pad decisions recorded in `docs/BENCHMARKS.md`

~~- Report addendum for the 08:19 HTML report (this file supersedes that
need for wave-4 content).~~ moot — no 08-19 file exists in docs/status/ (this report superseded it)

## d) TOTALLY FUCKED UP (owned, all caught + fixed in-session)

1. **Dead-append bug in conformance.go (SA4010)** — `violations, notes =
   degradedSubsetViolations(...)` OVERWROTE the rule-1/2 violations appended
   in the ADT loop. My first test run passed because only conformant engines
   were exercised (violations empty, overwrite harmless). The linter caught
   what my tests couldn't. Fixed with append-merge + added the `lyingEngine`
   violation-path tests that would have caught it. **Lesson: a validator's
   tests must exercise the failing path, not just the happy one.**
2. **Fragile first draft of the violation test** — I initially faked the
   entire `testing.TB` interface with a recorder struct (type-asserting
   `args[0].(string)`). Self-caught as brittle; replaced by exporting
   `AuditCapability` (no test plumbing) — which is also more useful
   (Doctor/report integration).
3. **Two sloppy edits mid-F55**: one edit dropped a `for` loop opener
   (repaired); one `multiedit` hit mtime drift from the formatter daemon
   (re-read, re-applied). No committed damage.

## e) WHAT WE SHOULD IMPROVE (observations from this session)

1. **Conformance for `DegradedADTs` honesty, not just presence** — the
   skeleton checks declared-vs-implemented structurally; it does NOT check
   that a "degraded" complexity is honest (e.g. Log on pg is actually
   emulated how?). Future: behavioral cost probes, not just structure.
2. **`StreamLogEntry` design should land soon** — the OFFSET pathology it
   removes is the same class as the wave-3 fix that bought 285x. The design
   is ready; implementation is one query per SQL engine.
3. **Replace-directive debt is growing** — now 5 local replaces exist purely
   because wave-3/4 code is untagged (system/integration duckdb, event→metadata,
   schema→event+metadata). Every one is documented droppable-on-tag, but the
   sweep cost compounds. Tagging would delete all of them at once.
4. **`t.Log`-based tables** — conformance renders its table via t.Log; CI
   hides it unless verbose. The exported `AuditCapability` should be wired
   into `Doctor` output so operators see declared-vs-implemented drift
   without reading test logs.
5. **Test the tester earlier** — I wrote ~300 lines of validator before any
   failing-path test. The dead-append bug survived exactly that gap.

## f) NEXT (up to 50, priority order)

~~1. Run `nix run .#verify` end-to-end; fix anything it catches.~~ done — 13-15 run #4 GREEN end-to-end

~~2. Run `nix run .#check-coverage`; re-baseline if schema/event coverage moved
(tests were added — expect upward drift, fix by re-baselining UP).~~ done — `#check-coverage` EXIT=0

~~3. Run `nix run .#check-duplication`; baseline if decorator/shell similarity
flags.~~ done — `#check-duplication` EXIT=0

~~4. F46: go-codec `UnwrapDecode` first-byte sniff (external repo; load its
AGENTS.md first; run its verify gate).~~ done — go-codec `autodetect.go`

~~5. F47: contention benches @-cpu=16,32 for workloadMeter; commit benchstat
baselines for the 5 BENCHMARKS.md ledger paths.~~ done — `-cpu 4,8,16,32` ledger entries + benchstat in benchkit

~~6. F48: worker-counter false-sharing measurement @32P; pad only if >10%;
record decision.~~ done — NO PAD, measured (BENCHMARKS.md:35)

~~7. F49: multiSeqCounter + SSEReplay.seq same protocol.~~ done — multiSeqCounter padded, SSEReplay NO PAD (BENCHMARKS.md:34,36)

~~8. Annotate the 08:19 HTML report (or write a short addendum pointing here).~~ moot — no 08-19 report exists on disk

~~9. Wire `AuditCapability` into metaengine `Doctor` output (declared-vs-
implemented section).~~ done — `AuditCapability` shipped into Doctor (11-33, `30711eb79b`)

~~10. Run conformance under `nix run .#test-integration` so mysql/dgraph/turso
rows are exercised for real.~~ partial — iroh exercised (RED→fixed, 11-33/12-39); mysql/dgraph/turso live runs not recorded

~~11. Implement F59 rollout step 1: `SeqSeekableStreamLog` + conformance tests
in metaengine (design doc §5).~~ done — `a1334d8c5` (all engines), flipped IMPLEMENTED (`f2bbf4621`)

~~12. F59 step 2: sqlite/turso `seq > ?` journal query.~~ done — sqlite/turso `seq > ?` shipped in `a1334d8c5`

~~13. F59 step 3: pg/mysql/duckdb `seq > ?` journal query.~~ done — pg/mysql/duckdb shipped in `a1334d8c5`

~~14. F59 step 4: pebble/bbolt key-decoded seq + memory `sort.Search`.~~ done — pebble/bbolt key-decoded seq + memory `sort.Search` shipped

~~15. F59 step 5: EventAdapter token-backed seqCache + benchkit drain bench +
BENCHMARKS.md row.~~ done — EventAdapter seqCache + bench files landed 2026-08-16 (`metaengine/sqliteengine/stream_log_bench_test.go`, bench recalibration)

~~16. Verify F56/F57/F58 status (singleflight leader-ctx, turso DSN leak,
Close() leak) — they were not in my resumed queue; confirm whether an
earlier wave shipped them before claiming wave 4 complete.~~ confirmed — F56 `9541df676` (singleflight), F57 `921147a01` (turso DSN), F58 `9541df676` (Close leaks)

~~17. Harvest this report into TODO_LIST (docs-health discipline).~~ done — TODO_LIST carries the wave-4 release batch + capability/seq-carrying items

~~18. DuckDB `AggregateReader` real pushdown (TODO_LIST; highest-leverage
DuckDB item).~~ open — DuckDB AggregateReader pushdown still TODO_LIST-standing

~~19. Dgraph engine hardening (`Transactional` or explicit rejection).~~ open — Dgraph Transactional hardening not done

~~20. `brandedString` extraction decision (record/ or drop).~~ done — extracted as `metadata/ids.go` (`BrandedString`/`ActorString`)

~~21. Per-module CHANGELOG policy decision.~~ done in practice — root CHANGELOG `[2026-08-16 module releases]` carries per-module entries

~~22. One bench system consolidation (delete redundant harnesses).~~ open — benchkit + metaengine/bench + stack/bench all still exist

## g) QUESTIONS (cannot resolve myself)

**Q1 — Tagging authorization (BLOCKING ~6 modules + replace-debt deletion).**
Still open from the 08:19 report §g, now EXPANDED: wave-4 added
`event` (DecorateJournal + sentinel + tests) and depends on
`metadata` (BrandedString, untagged). Full set wanting tags:
`storage/bbolt`, `metaengine/pgengine`, `stack/pebble`, `metaengine`,
`event`, and `metadata` (v4.5.1+). Tagging all of them deletes 5 local
replace directives in one stroke and un-breaks GOWORK=off standalone builds
without them. May I cut these tags (per `scripts/tag-release.sh`, annotated,
monotonic semver+ancestry), or do you want to review the CHANGELOG entries
first?

**Q2 — The other session's stranded commits (`092b5e8a8`, `4907b6afc`).**
Unchanged from the 08:19 report: cherry-pick, leave, or discard? I have not
touched them.

**Q3 — Pin-staleness policy for the F53 pin-drift meta-test.**
The test currently hard-fails only on unresolvable pins (0 today);
staleness (16 pins behind latest, all replace-governed eventtest v0.3.0
lines) is warning-only, gated on this answer: should stale-but-valid pins
FAIL CI (forces a sweep every release), or stay warnings until the
replace-drop sweep lands? This is your ROADMAP Open Question, not mine to
decide.

---

**Bottom line**: F55/F59/F60 all shipped and green at module level; one real
bug in my own validator caught by lint and fixed with proper failing-path
tests; wave 4 execution queue is now F46 + F47-F49 + the full verify gate;
tagging remains the single user decision gating the largest cleanup.
