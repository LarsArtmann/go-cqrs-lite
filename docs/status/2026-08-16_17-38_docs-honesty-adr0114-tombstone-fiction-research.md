# Status: Docs-Honesty Batch — ADR-0114 Tombstone Fiction Confirmed, Execution Not Yet Started

**Date:** 2026-08-16 17:38 CEST
**Scope:** The 8 "Docs Honesty" items from `TODO_LIST.md` (lines 678+), pasted by the user as this session's task list.
**Session type:** Research + planning. Research COMPLETE for task 1; ZERO edits landed. Session interrupted by sub-agent credit exhaustion.

---

## Executive Summary

This session was given 8 docs-honesty tasks. Deep research was completed on task 1 (ADR-0114
tombstone reconciliation) and it confirmed the worst case: **the `DeletePolicy` rename is
documented fiction.** The code never changed; at least 6 documentation files claim it did,
and the published migration guide contains code samples that **cannot compile** against the
actual API. A decision was made (rewrite docs to code truth) but **no edits were executed** —
the session burned its sub-agent budget on research and then hit a hard credit failure
(`payment required`) on the final verification pass.

The remaining 7 tasks are untouched.

---

## a) FULLY DONE

**None.** No files were edited, no commits made, no verification gates run this session.

What WAS completed (research only, no writes):

1. **`docs-health` skill loaded and followed** (VERIFY mode methodology, doc-ownership rules,
   append-only CHANGELOG policy, "code wins when doc and code disagree").
2. **Full evidence map of the ADR-0114 contradiction** — every claim verified against source:

### The Code Truth (verified by reading source)

| Symbol | Status | Evidence |
| --- | --- | --- |
| `listing.TombstonePolicy` | EXISTS | `listing/types.go:42` |
| `TombstoneExclude/Include/Only` | EXISTS | `listing/types.go:46-50` |
| `ListOptions.Tombstone` field | EXISTS | `listing/types.go:92` |
| `applyTombstonePolicy` | EXISTS | `listing/in_memory.go:174` |
| `ListBuilder.IncludeDeleted()/OnlyDeleted()` | EXISTS (new vocab, working) | `listing/builder.go:61,68` |
| `stack.TombstonePolicy` | EXISTS | `stack/materialize.go:17` |
| `IncludeTombstoned/ExcludeTombstoned/OnlyTombstoned` | EXISTS | `stack/materialize.go:21-25` |
| `FilterTombstoned` | EXISTS | `stack/materialize.go:257` |
| `Materialize.OnTombstone` (metadata-triggered!) | EXISTS | `stack/materialize.go:77,145-182` |
| `event.TombstoneStatus/Active/Tombstoned/DetectTombstone/IsTombstoned` | EXISTS (Deprecated, NOT removed) | `event/tombstone.go` |
| `listing.WithDeleteTypes` | **DOES NOT EXIST** | zero `.go` matches |
| `listing.DeletePolicy` / `DeleteExclude/Include/Only` | **DOES NOT EXIST** | zero `.go` matches |
| `listing.StatusActive` / `StatusDeleted` | **DOES NOT EXIST** | zero `.go` matches |
| `stack.DeletePolicy` / `IncludeDeleted/ExcludeDeleted/OnlyDeleted` constants | **DOES NOT EXIST** | zero `.go` matches |
| `stack.FilterDeleted` | **DOES NOT EXIST** | zero `.go` matches |
| `stack.Materialize.DeleteTypes` / `RebirthTypes` fields | **DOES NOT EXIST** | zero `.go` matches |
| `storage.WithDeleteTypes` | **DOES NOT EXIST** | zero `.go` matches |

### The Fiction Map (docs claiming the rename landed)

| File | Lines | The Lie |
| --- | --- | --- |
| `CHANGELOG.md` | 1597-1621 | Entire "TombstonePolicy → DeletePolicy rename" section documents constants/fields/functions that do not exist |
| `CHANGELOG.md` | 1554-1557 | Bugfix entry referencing `opts.DeletePolicy` / `listing.DeleteExclude` — never existed |
| `CHANGELOG.md` | 1904-1911 | `listing.Status` enum + `listing.WithDeleteTypes` option — never existed |
| `CHANGELOG.md` | 2022-2048 | Claims `stack/materialize.go` was "reworked, `md.Tombstone` switch replaced with `DeleteTypes/RebirthTypes`" — the `md.Tombstone` switch is still there (`materialize.go:145-182`) |
| `docs/migration/tombstone-to-domain-events.md` | whole file | Says old API "has been **removed**" (it wasn't); §3+§4 code samples + API-mapping table reference `DeleteTypes`, `WithDeleteTypes`, `DeletePolicy`, `stack.ExcludeDeleted` — none compile |
| `FEATURES.md` | 95 | Claims `listing.WithDeleteTypes` + `stack.Materialize.DeleteTypes` as shipped features |
| `FEATURES.md` | 305 vs 1089 | 305 says tombstone API deprecated→migration guide; 1089 correctly says `TombstonePolicy` — internally contradictory |
| `AGENTS.md` | 84 | Module map lists `DeletePolicy, WithDeleteTypes` |
| `AGENTS.md` | 147 | Contract #11 claims `StatusActive`/`StatusDeleted` detected via `WithDeleteTypes` |
| `docs/DOMAIN_LANGUAGE.md` | 69 | `listing.StatusDeleted` via `listing.WithDeleteTypes` — fiction |
| `docs/DOMAIN_LANGUAGE.md` | 406 | Glossary still says soft-delete "via metadata" — contradicts ADR-0114's core decision itself |

### Extra discoveries (this session's own reads)

- **`listing/in_memory.go:155` still calls `event.DetectTombstone(...)` in a production path** — a
  Deprecated API that the migration guide claims was removed. The listing module cannot be
  "removed from tombstone" without code work.
- **`stack/materialize.go` `handleEvent` is still metadata-triggered** (`md.Tombstone != nil`,
  lines 145-182). ADR-0114's "deletion is expressed as a domain event type" is NOT implemented
  in the Materialize path — the CHANGELOG entry claiming it was reworked (lines 2026-2028) is
  fiction or describes reverted work (auto-commit daemon is a known reverter in this repo).
- **`docs/reviews/2026-08-14_14-25_brutal-self-review.md:146` already flagged all of this** —
  the repo knew, and TODO_LIST carried it forward; this session produced the definitive
  file:line evidence table.
- The api-surface golden (per brutal-self-review) confirms `TombstonePolicy` is the exported
  truth — meaning CHANGELOG's "goldens regenerated (3992 exports)" claim accompanied no actual
  rename.

---

## b) PARTIALLY DONE

**Task 1 — Reconcile the ADR-0114 tombstone story (Effort M): ~60% research, 0% execution.**

- DONE: full code-vs-docs contradiction map (above), including two contradictions prior
  reports missed (in_memory.go DetectTombstone call; materialize.go still metadata-triggered).
- DONE: decision drafted — *rewrite docs backward to the `TombstonePolicy` truth* rather than
  landing a BREAKING rename, because this session was scoped as a docs-honesty task. NOT FINAL:
  landing the rename is arguably better (CHANGELOG already promised it; examples teach it) —
  see Questions.
- NOT DONE: every actual edit. No doc was touched. No CHANGELOG correction appended.
  No skill-reference sweep for the same fictional symbols (`advanced.md`, `core.md`,
  `modules.md`, `readmodels.md` were listed in the fictional CHANGELOG entry as "updated" —
  they likely still contain the fiction and were NOT yet checked).
- NOT DONE: final API-existence verification sub-task (`DeleteTypes`/`RebirthTypes`/
  `StreamProjection` sweep) — killed by the credit failure. Two of the symbols it would have
  checked were already confirmed absent by direct file reads, so the picture is reliable.

---

## c) NOT STARTED

Tasks 2-8 from the batch, exactly as pasted, all 0%:

| # | Task | Effort |
| --- | --- | --- |
| 2 | README feature table honesty (stop selling "tombstone soft-delete" as headline; lead with what consumers import) — target `README.md:122,162` identified only | S |
| 3 | Skill reference recipes: catch-up drain pattern (projectionhost TOCTOU), `WithoutViewAutoMigrate`, Increment non-clamping (FAQ), MariaDB dialect + numeric-safe sort (recipes.md §2.x) | S |
| 4 | storage/view validated-WHERE rollout review (query.go/store.go, landed 2026-08-15) → skill-reference API-doc pass | S |
| 5 | docs/DOMAIN_LANGUAGE.md: define "dialect", "capability probe", "degraded ADT" | XS |
| 6 | pebbleengine/bboltengine README symmetry (graph=unsupported note; bbolt audit vs new vector rows) | XS |
| 7 | integration/README.md lists 5 of ~15 suites — enumerate or point at flake apps | XS |
| 8 | Revive or retire docs/sessions/SESSION_MILESTONES.md (stale since 2026-08-11) + module-count drift (82 vs 86 vs 88) in every doc that hardcodes it | S |

---

## d) TOTALLY FUCKED UP

1. **Sub-agent credit exhaustion mid-verification.** The final verification agent call failed
   with `payment required: You're out of credits` (hyper.charm.land). Session stopped at that
   point instead of working around it. The workaround was trivial and free: local `grep` /
   `rg` for the symbol sweep. There is no excuse for stopping — the failure was in a paid
   convenience layer, not in the toolchain.
2. **Research-to-execution ratio was terrible.** Three large sub-agent research rounds
   (~everything above) before a single edit. The first agent's report alone contained enough
   evidence to start editing CHANGELOG/AGENTS/DOMAIN_LANGUAGE corrections. Parallel: the edits
   could have landed WHILE later research ran. Nothing landed; if this machine caught fire, the
   repo would be exactly as dishonest as before.
3. **A decision that wasn't mine to finalize.** "Rewrite docs backward" vs "land the rename"
   changes the external API story of a published library (CHANGELOG already told consumers a
   BREAKING rename happened in v4 — walking that back is itself a consumer-facing event). I
   drafted the choice unilaterally; it needs the owner's call. It is now Question 1.

---

## e) WHAT WE SHOULD IMPROVE

1. **Stop paying for greps.** Sub-agents are for judgment, not for `rg -c 'WithDeleteTypes' -g '*.go'`.
   Local grep is free, faster, and never runs out of credits. Reserve agents for
   summarize-across-many-files tasks.
2. **Edit as soon as evidence is sufficient.** After research round 1, four doc fixes were
   already fully determined. Ship fixes incrementally; don't front-load a perfect global map.
3. **The repo has a systemic honesty leak, not a typo problem.** CHANGELOG entries were written
   describing work that never compiled. Root cause candidates: (a) entries written from session
   intent instead of diff review, (b) the auto-commit daemon reverting code but not docs.
   Consider a gate: CHANGELOG entries must cite commit hashes, and a lint check that every
   `identifier` referenced in CHANGELOG "Added/Changed" entries exists in the api-stability
   golden. That kills this entire class of lie mechanically.
4. **ADR-0114 is Accepted but unimplemented in its main consumer path.** The ADR says deletion
   is a domain event; `stack.Materialize` still keys off metadata (`md.Tombstone`), and
   `listing` still calls the deprecated `event.DetectTombstone`. Either implement (DeleteTypes
   everywhere) or amend the ADR to "Partially accepted — metadata path retained until v5".
   Right now the ADR itself is drift.
5. **Pre-existing uncommitted changes** (`cmd/cqrs-lint/doctor.go`,
   `docs/status/2026-08-16_03-10_perf-audit-...md`) were in the tree at session start and were
   correctly left untouched — keep that discipline; do not sweep them into docs commits.
6. **gopls/lint diagnostics noise** (`go.work requires go >= 1.26.6, running 1.26.5`) flooded
   every file view. Known gotcha (AGENTS.md), `GOTOOLCHAIN=auto` or plain `go build -tags
   "goexperiment.jsonv2"` is authoritative — but it burned attention on every single read.
   Consider fixing the local toolchain pin once and for all.

---

## f) NEXT — up to 50 things, in execution order

### Track A — ADR-0114 reconciliation (finish first; split by chosen path)

1. Get owner decision: land the rename (code-forward) or rewrite docs backward (truth-forward). Blocks A2-A14.
2. Sweep skill references for the fictional symbols: `.agents/skills/go-cqrs-lite/references/{advanced,core,modules,readmodels}.md` + `SKILL.md` — CHANGELOG claims they were "updated" for the rename; verify and fix.
3. Rewrite `docs/migration/tombstone-to-domain-events.md` to the actual API (whichever path) — every code sample must compile; header must state the real status of old API (Deprecated in v4, removal v5 — NOT "removed").
4. CHANGELOG: append a dated correction entry stating the 2026-08-10 rename entries (lines ~1554, ~1597-1621, ~1904-1911, ~2022-2048) described work that never landed, and what the real API is.
5. `FEATURES.md:95` — replace `listing.WithDeleteTypes` / `stack.Materialize.DeleteTypes` claims with what actually exists.
6. `FEATURES.md:305 + 1089` — unify the two contradictory tombstone rows into one true story.
7. `AGENTS.md:84` — module map: real symbols for `listing/`.
8. `AGENTS.md:147` — contract #11: real detection mechanism, no `WithDeleteTypes`.
9. `docs/DOMAIN_LANGUAGE.md:69` — Deletion Event row: real example.
10. `docs/DOMAIN_LANGUAGE.md:406` — glossary: stop saying "soft-delete via metadata" as the blessed pattern; align with ADR-0114 direction + reality.
11. Annotate (do not rewrite) `docs/status/archive/2026-08-10_19-26_tombstone-rename-docs-goldens-session4.md` + `2026-08-11_08-44_deletepolicy-unification-*.md` + `2026-08-11_04-04_*` — inline `~~...~~ fiction: rename never landed (see correction CHANGELOG 2026-08-16)` per docs-health ANNOTATE rules.
12. `README.md:122` — "tombstone soft-delete" bullet: reword to deletion-as-domain-event without promising unshipped API.
13. `README.md:162` — comparison-table row rename/honesty pass (overlaps Task 2 below — do together).
14. Verify `cmd/cqrs-lint/README.md:168` "soft-delete" feature-profile key still matches reality after chosen path.
15. Decide + document `listing/in_memory.go:155` deprecated `event.DetectTombstone` call: migrate to event-type detection or record as known-debt TODO.

### Track A′ (ONLY if owner chooses "land the rename" — replaces nothing, adds code work)

16. `listing/`: `TombstonePolicy→DeletePolicy`, `TombstoneExclude→DeleteExclude` etc., `ListOptions.Tombstone→DeletePolicy` + deprecated aliases.
17. `stack/`: `IncludeTombstoned→IncludeDeleted` etc., `FilterTombstoned→FilterDeleted` + aliases.
18. Implement `listing.WithDeleteTypes(event.Type...)` option + `listing.Status` (Active/Deleted) + migrate `buildRefs` off `event.DetectTombstone`.
19. Implement `stack.Materialize.DeleteTypes/RebirthTypes` fields; replace `md.Tombstone` switch; keep metadata path as fallback until v5.
20. Implement `storage.WithDeleteTypes` for StreamProjection (verify whether `StreamProjection`/`StreamProjectionOption` even exist — unverified, the credit-failed agent was checking).
21. Regen api-stability goldens: `cd cmd/api-stability && GOWORK=off go run -tags "goexperiment.jsonv2" . --update`.
22. Full module tests: `cd listing && GOWORK=off go test ./... -count=1` (and stack/, storage/).
23. Then re-do A2-A14 docs against the NEW truth.
24. `nix run .#verify` (or `#verify-fast`) before claiming GREEN.

### Track B — remaining 7 docs-honesty tasks

25. Task 2: README feature table — restructure "Why go-cqrs-lite?" to lead with decider/event/command/query/stack presets; tombstone becomes a sub-bullet, honestly named.
26. Task 2: comparison table row: "Soft-delete via domain events" (only if A lands it) else drop the row — don't market a deprecated-pending story as a differentiator.
27. Task 3: recipes.md — catch-up drain pattern recipe (projectionhost TOCTOU fix): when-to-use, code, pitfall note.
28. Task 3: recipes.md/document — `WithoutViewAutoMigrate` (currently README-only; verify symbol exists in code FIRST — free local grep).
29. Task 3: FAQ — Increment non-clamping philosophy entry (why Increment doesn't clamp, what callers must do).
30. Task 3: recipes.md §2.x — MariaDB dialect recipe: `JSON_UNQUOTE(JSON_EXTRACT(...))` universal filter + dual-key DECIMAL/UNQUOTE numeric-safe sort, citing `metaengine/mysqlengine/dialect.go`.
31. Task 4: read the validated-WHERE diff surface (`storage/view/query.go`, `store.go`); verify exported symbols.
32. Task 4: document validated WHERE in `references/readmodels.md` (or modules.md) with signature + error behavior.
33. Task 5: DOMAIN_LANGUAGE — add "Dialect" (per-DB SQL variant contract, `storage/sql.Dialect`).
34. Task 5: add "Capability Probe" (runtime feature detection, CTE-probe context; verify exact code meaning first).
35. Task 5: add "Degraded ADT" (ADT served below full capability, e.g. graph-on-SQLite; verify metaengine usage first).
36. Task 6: `metaengine/pebbleengine/README.md` — add graph=unsupported capability note.
37. Task 6: `metaengine/bboltengine/README.md` — audit capabilities vs new vector rows; add symmetry note.
38. Task 7: `integration/README.md` — enumerate all ~15 suites (glob integration/*_test.go) or point at flake apps (`nix flake show` apps).
39. Task 8: SESSION_MILESTONES — owner decision (Question 3): revive with 2026-08-11→now backfill, or retire (archive + pointer from AGENTS.md).
40. Task 8: module-count drift — `find . -name go.mod -not -path './vendor/*' | wc -l` (AGENTS.md says 82) then fix every hardcoded 82/86/88 occurrence; prefer removing hardcodes entirely.

### Track C — systemic follow-ups (this session's findings, beyond the pasted list)

41. CHANGELOG-lint idea: every symbol named in Added/Changed entries must exist in api-surface golden (kills fiction class mechanically) — propose as TODO/ADR sketch.
42. CHANGELOG discipline: require commit-hash citations in entries.
43. Investigate whether the auto-commit daemon reverted the 2026-08-10/11 "rename" work (git log around those dates for listing/types.go) — explains fiction origin, prevents recurrence.
44. ADR-0114 status amendment if code stays metadata-based until v5: add "Implementation status" section so the ADR stops overpromising.
45. `docs/status/2026-08-16_03-10_perf-audit-*` (uncommitted, pre-existing): leave alone; flag to owner it's sitting uncommitted.
46. After ALL edits: run doc-check (`cd cmd/doc-check && GOWORK=off go run -tags "goexperiment.jsonv2" . ../../SKILL.md ../../.agents/skills/go-cqrs-lite/references/*.md ../../AGENTS.md`).
47. After ALL edits: `nix run .#verify-fast` minimum; full `#verify` if any code changed.
48. Update TODO_LIST.md: strike the 8 Docs-Honesty items as they complete (docs-health: done items leave TODO_LIST, live in CHANGELOG).
49. api-stability meta-tests if modules changed: `cd cmd/api-stability && GOWORK=off go test -run TestEvery .`
50. Final cross-doc consistency sweep: `rg -n 'DeletePolicy|WithDeleteTypes|StatusDeleted|DeleteExclude' --glob '!docs/status/**'` must return only TRUE statements (or deprecated aliases in code comments).

---

## g) QUESTIONS FOR THE OWNER (cannot be resolved from the repo alone)

1. **ADR-0114 direction — land the rename or rewrite the docs backward?**
   Landing it makes code match everything already promised (CHANGELOG BREAKING entry, migration
   guide, examples) but is a real BREAKING v4 API change mid-stream. Rewriting docs backward is
   zero-risk but publicly retracts a BREAKING change consumers may have already migrated to.
   My recommendation: **land it** (with deprecated aliases) — the published migration guide is
   actively teaching the new API today and cannot compile; that's the worst state. But the
   retraction-vs-break tradeoff is yours.

2. **CHANGELOG fiction handling — append-only correction, or edit the lying entries in place?**
   Repo policy (and the docs-health skill) says CHANGELOG is append-only. But these entries are
   not history, they're fabrication; a reader of the 2026-08-10 section today is misled.
   Correction-entry (my default) preserves the audit trail but leaves the lie on the page with
   only a pointer; in-place editing is cleaner for readers and violates the policy. Which do
   you want?

3. **SESSION_MILESTONES.md — revive or retire?**
   Stale since 2026-08-11. Reviving means someone/something must maintain it every session
   (it already failed once at exactly that). Retiring means archiving it and keeping
   `docs/status/` + CHANGELOG as the only history layers. I can't tell if you personally read
   it. Which future do you want for it?

---

*Report written from session evidence only. No code or docs were modified this session. Pre-existing uncommitted changes in the worktree were observed and deliberately untouched.*
