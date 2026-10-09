# Session Review & Status — WAL Extraction Design Exploration

**Date:** 2026-10-09 02:47 (Friday)
**Session type:** Pure design exploration (zero code changes)
**Scope:** Whether fundamental metaengine/journal interfaces (append-only log / WAL) should be extracted into standalone generic module(s) — API design + data modeling focus.
**Verdict state:** Explored, NOT decided. No code, no ADR, no doc changes were made. This report is the session's only artifact.

---

## Session summary (what actually happened)

1. **Turn 1 — the question:** "Should fundamental metaengine interfaces (e.g. WAL) become their own generic module projects? Hardcore focus on API design AND data modeling?"
   - Loaded skills: `go-cqrs-lite`, `data-model-review`, `go-modularize`.
   - Research (sub-agent, exact signatures): `event/` journal contracts, `metaengine.StreamLogBackend`/`EventLog`, ADR-0126 generic cores (`LogStore[T,ID]`, `Inserter[T]`/`JournalReader[T]`, `AdapterCore[T]`), `record.Record` shape. Confirmed: **no `wal` module exists anywhere**.
   - Read ADR-0128 (sibling-extraction precedent) at source.
   - Delivered: verdict (idea sound; wrong extraction candidate — the gem is the ADR-0126 generic cores + capability-interface pattern, not `StreamLogBackend` which is `[]any`-typed), first-principles data model (invariants, axes of variation, hard-to-change decisions), a minimal `Log[T]` kernel + capability-interfaces API sketch, and Options A (in-repo Tier-0 `wal/` module) / B (sibling repo behind evidence gate) / C (standalone product — parked under fleet-first decision).
2. **Turn 2 — "what are we not considering?":** 11 blind spots in 4 groups; verified two facts in-repo first (no `Forwarder`/outbox code in `watermill/` — it is an upstream concept documented in the watermill skill; no `storage/conformance` package exists — the pattern is `storage/store_testsuite_test.go`). Key additions: module-vs-package payoff test, fixed-overhead ledger + v5-transition timing risk, make-vs-buy homework (Kafka offsets / etcd wal / BookKeeper), time axis, retention-vs-catch-up race, tamper-evidence/audit, decorator problem (ADR-0126 lesson), capability self-description + conformance, batch atomicity, naming ("WAL" over-promises), and the transactional-outbox synthesis as the potential second consumer / evidence-gate trigger.
3. **Turn 3 — this report.**

---

## a) FULLY DONE

1. **Skill activation per contract** — 3 skills turn 1, 2 skills this turn (`brutal-self-review`, `status-report`), loaded BEFORE acting.
2. **WAL/journal surface inventory** — exact interface signatures with file:line, all spot-verified this turn:
   - `event/store.go:110` `Journal`, `:129` `SeekableJournal`, `event/streaming_source.go:60` `StreamingJournal`
   - `metaengine/engine.go:468` `StreamLogBackend`, `metaengine/consistency.go:12` `EventLog` (a struct, not an interface)
   - `storage/memory/log_store.go:49` `LogStore[T,ID]` + `LogStoreConfig` policies, `storage/sql` `Inserter[T]`/`JournalReader[T]`, `system/adapter_core.go:20` `AdapterCore[T]`
   - `record/record.go` full field inventory
3. **Confirmed no `wal` module exists** (only Pebble-internal `WithDisableWAL` mentions).
4. **ADR-0128 read at source** — extraction precedent (codec/retry/idempotency/flightrecorder → sibling repos, motivated by dependency hygiene, not external adoption).
5. **First-principles data model** for an append-only log: essence (total order + immutability), 4 invariants (incl. the load-bearing missing-position semantics), 7 independent axes, hard-to-change list.
6. **API kernel sketch** — `Log[T]` (Append/Read/Head) + optional capabilities (Syncer, CASAppender, Truncater, Tailer) + error sentinels; explicitly keeps `record.Record` OUT.
7. **Options A/B/C with a recommendation** and the fleet-first scope challenge ("many people need it" is not a valid trigger under the 2026-10-08 consumer-scope decision).
8. **11 blind-spot analysis with in-repo verification** — including self-correcting the turn-1 recommendation (payoff test) and the outbox synthesis.
9. **Two fact-checks that corrected my own draft claims** (Forwarder absent in-repo; conformance is not a package).

## b) PARTIALLY DONE

1. **The design exploration itself** — kernel sketch exists but is unvalidated (never compiled, never reviewed against the recipes harness), incomplete (11 blind spots identified but not incorporated into a revised sketch), and undecided (user has not chosen a direction).
2. **Evidence verification** — ADR-0128 verified at source; ADR-0151, ADR-0152, and the 2026-10-08 consumer-scope decision were cited from AGENTS.md summaries WITHOUT primary-source reads, while the recommendation leaned on their timing/scope.
3. **The go-modularize payoff test** — applied late (turn 2, after being challenged) and never executed on real data (no grep of cqrs-htmx/go-appkit for direct journal-position usage).
4. **Drift observation** — the suspected AGENTS.md "conformance is a subpackage" mismatch was noticed during turn-2 verification and mentioned only in passing; never flagged as a docs-drift finding, never resolved.
5. **Sub-agent citation trust** — turn-1 file:line citations came from the sub-agent report unverified; spot-checked only this turn (all correct; one imprecision: `LogStore` struct is L49, config L16 — reported as "L16 policy injection / L49 struct", actually fine).

## c) NOT STARTED

1. Any code change, module, or package (zero mutations this session by design).
2. ADR draft for the unified journal-position contract.
3. Consumer-usage analysis: does cqrs-htmx or go-appkit ever touch journal positions directly (bypassing event/system)? — the decisive input for module-vs-package.
4. Ecosystem contract comparison (Kafka offsets, etcd wal, tidwall/wal, BookKeeper).
5. TODO_LIST.md harvest from this report's section (f) — deliberately deferred: no decision taken yet, harvesting a brainstorm violates the docs-health anti-pattern (brainstorm ≠ commitment list). Also `TODO_LIST.md` was already modified-uncommitted at session start (not by me — left untouched).
6. md-go gate compliance for the API sketch (if the sketch ever lands in docs, fences must parse or carry `// skip-validate`).

## d) TOTALLY FUCKED UP

Nothing destructive — zero code/doc mutations; this report is the only write. Honesty demands two entries anyway:

1. **The turn-1 recommendation preceded its own litmus test.** I loaded `go-modularize` in turn 1, quoted its composability-payoff test, recommended Option A (new module) — and only ran the test when the user asked "what are we missing?" in turn 2. A new go.mod in a 98-module workspace one day after ADR-0152 froze topology is exactly the decision that deserved the test FIRST. Self-caught, but one turn late.
2. **Sat on a drift observation.** The AGENTS.md claim "conformance is a subpackage" (storage tier list) does not match reality (no such package found; only `store_testsuite_test.go` + one mention in `storage/memory/stream.go`). I noticed it mid-turn-2 and did not surface it as a finding. Owner-permission rule says fix-or-flag on sight; I under-flagged.

## e) WHAT WE SHOULD IMPROVE (session-derived, both process and design)

**Process (mine):**
1. Apply a loaded skill's litmus tests BEFORE recommending, not when challenged.
2. When a recommendation hinges on an ADR's content/timing, read the ADR at source; AGENTS.md summaries compress exactly the details that matter at decision time.
3. Surface noticed drift immediately and explicitly (fix-or-flag, on sight).
4. Lead breadth-answers with the load-bearing items; appendix the rest (turn 2's 11 items were right in content, suboptimally ranked).
5. Close the loop: report (f) sections are HARVEST input — queue it explicitly instead of leaving it entombed.

**Design (the exploration's own output, restated as improvements):**
6. One typed position contract should replace the THREE homes of dangling-cursor semantics (event/store.go:122-128 comment, metaengine/seq_seek.go gap tolerance, AdapterCore "unknown ID → 0").
7. The missing-position policy should be a NAMED policy, not a bool (`fromStartWhenMissing`).
8. `StreamLogBackend`'s `[]any` values violate the repo's own strong-types contract (#12) — v5 candidate.
9. Any `Log[T]`-style API must ship the capability-preserving decorator (`wal.Decorate`) from day one — ADR-0126's core lesson, initially missing from my own sketch.
10. Do not call it "wal" unless we mean the recovery protocol; "journal" is the honest in-repo vocabulary.

**Brutal-self-review answers (condensed):** Forgot: payoff test first, primary-source ADR reads, immediate drift flag. Stupid-but-done: recommending before testing. Better: verification depth proportional to recommendation weight. Still improve: items 1–5 above. Lied: no — near-misses disclosed in (b)(2), (d). Ghost systems: none created; none found (EventLog is wired via `WithEventLog`). Split brains: none created; one pre-existing one DOCUMENTED (the contract triplication — the session's core finding). Removed-useful: nothing removed. Tests: N/A (no code); the sketch's future "tests" are the recipes-compile harness + store-test-suite conformance pattern.

## f) Up to 50 things to get done next (35 real; ranked, session-derived)

**Decision homework (before any greenlight):**
1. Enumerate direct journal-position consumers in cqrs-htmx + go-appkit (`rg "ReadFrom|afterEventID|afterSeq"`) → module-vs-package verdict.
2. Read ADR-0152 at source (topology + wave mechanics the timing claim leaned on).
3. Read ADR-0151 at source (evidence-gate precedent invoked twice).
4. 1-page contract comparison: Kafka offsets / etcd wal / tidwall/wal / BookKeeper vs our three homes.
5. Decide the outbox question: must the contract express "append atomically with business write"? (the named-second-consumer gate test).
6. Decide the name (`wal` vs `journal` vs `appendlog`) — run naming review when drafting.
7. Timing decision: ride the v5 core cut vs. defer until after v5 lands.

**Design work (if greenlit):**
8. Draft ADR "unified journal position contract" with the three homes' semantics side by side.
9. Decide gapless vs gappy positions (input: `SeqSeekableStreamLog` gap tolerance).
10. Replace `fromStartWhenMissing` bool with a named missing-position policy type.
11. Define Time as a first-class axis (Position vs Time authority; time-bounded reads; time-travel semantics already tested in `event_store_timetravel_test.go`).
12. Retention contract: `TruncateBefore` + `ErrPruned` + defined catch-up recovery (the retention-vs-offline-subscriber race).
13. GDPR axis: per-tenant-key cryptographic erasure vs append-only tension — decide if in scope.
14. Tamper-evidence capability sketch (hash-chaining, verify, actor attribution) — gated on question g3.
15. Idempotent append: where `dedup/` (ring) plugs into the contract.
16. Entry schema evolution: upcast story for generic `T` (`schema.UpcastSourceTransform` precedent).
17. Batch atomicity + multi-writer serialization point (SaveMultiBatch/MultiSink precedent).
18. Tailer backpressure policy (bounded channel; drop-old vs block — ServeSSE ring precedent).
19. Error sentinel contract + errorfamily mapping (`ErrClosed`/`ErrStalePosition`/`ErrPruned`).
20. Capability self-description as data (`Capabilities()`/`Profile()` pattern alignment).
21. Capability-preserving `wal.Decorate` (ADR-0126 lesson).
22. Conformance suite: extend the store-test-suite pattern to pin the capability matrix per implementation.
23. Namespacing decision: collection as first-class method arg vs. per-collection Log instances.
24. OTel story for a zero-dep Tier-0 module (span-name string injection pattern from `Inserter`/`JournalReader`).
25. Zero-alloc hot-path audit of the kernel (repo contract #7).
26. `record.Record` exclusion boundary: event/↔wal adapter design.
27. Revise the kernel sketch to incorporate ALL 11 blind spots (current sketch predates them).
28. Compile-check any Go that lands in docs (md-go gate; recipes catalog rules).

**Repo hygiene noticed this session (unrelated to the design, fix-or-flag):**
29. Resolve the AGENTS.md "conformance is a subpackage" claim (suspected drift; two checks done, no package found).
30. Promote the dangling-cursor contract out of the event/store.go:122-128 comment into typed/doc form wherever it lands.
31. Flag `StreamLogBackend []any` as a v5 typing candidate.
32. Check the pre-existing uncommitted `TODO_LIST.md` modification (not authored this session) before any future harvest, to avoid clobbering concurrent work.

**Process improvements (mine, carry forward):**
33. Litmus-test-before-recommend rule (see (d)1).
34. Primary-source ADR reads when recommendations hinge on them.
35. Run docs-health HARVEST from this report's (f) once the user takes the extract-or-not decision.

## g) Questions I cannot figure out myself (max 3)

1. **Is there a planned fleet consumer that needs an append-only log WITHOUT the CQRS module graph** (e.g. an audit trail in go-appkit/monitor365, a non-CQRS project)? This single fact decides module-vs-internal-package and Option A vs B — I cannot see your project pipeline intentions.
2. **Should the unified journal-position contract ride the v5 core cut** (ADR-0152 waves, in motion now) **or wait until after v5 lands?** This is a sequencing/taste call with real risk on both sides (mid-transition insertion vs. another breaking wave later).
3. **Does audit-grade, tamper-evident logging (hash-chained, non-repudiable) exist anywhere in the fleet roadmap?** It decides whether chain-integrity is a day-one capability or YAGNI — retrofitting chain hashes into a live log is impossible.

---

**Report status:** written 2026-10-09 02:47. Not committed (harness forbids unprompted commits; auto-commit daemon will absorb). HARVEST deferred pending decision + the 3 answers above. **Waiting for instructions.**
