# Status Report: Core Data Model v5 Plan — Foundation Tier (T05–T08) — 2026-08-22 05:37

**Session scope:** Resumed execution of `docs/planning/2026-08-22_03-52_core-data-model-v5-execution-plan.md` (25 tasks). This session: resolved the three "immediate" blockers from the previous session's report, then executed the entire additive foundation tier (T05 Cause, T06 Stamp, T07 Actor, T08 ID/Encoding) on top of the already-complete T04.

**Session commits (chronological):** `c4feaf031`+`dba330db4` (T05), `e217f19dd`+`2e879e897`+`b6784b8f1` (T06), `ebbe1ecd6`+`2e778cefe` (T07), `3eb1b4eb5` (T08), `a775466d7` (lint window), `e361f4091` (lint hygiene). Mixed authorship: the auto-commit daemon landed mid-task splits four times; every split was verified before I proceeded.

---

## a) Fully done (verified end-to-end)

1. **Blocker f2 — uncommitted ~58-module sweep: RESOLVED (not by me).** The daemon had committed it as `30405c219` (record/v4 local replaces for event/command/query) + `327c3d2f9` (go-sse v0.5.1 + toolchain 1.26.6 sweep). Tree was clean at session start; no rollback needed.
2. **Blocker — master standalone-green: VERIFIED.** `GOWORK=off go build + go test` green for record, event, command, query (repeated after every task).
3. **T15·a — clean exclusive `#verify-fast`: GREEN** (after fixing the environment, see d1/d2). Ran exclusively, no concurrent edits.
4. **T05 — `record.Cause`: SHIPPED.** `CauseKind` (None/Command/Timer/Event/**Unknown**), `Cause{Kind, ID}`, `IsZero`, `String` ("kind:id", mirrors ActorID wire form). `CommonMetadata.Cause` added; `CausationID` deprecated (removed in v5). Event bridge precedence: typed `Causation.CommandID` → `CauseCommand`; bare tracing chain → `CauseUnknown` (the honest "kind not discriminated" — deliberate extension of the review's 4-kind enum, mirrors `id.ActorUnknown`); else zero. Command/query bridges map tracing → `CauseUnknown`. Tests pin precedence + zero in all four modules. Both fields populated in lockstep until v5.
5. **T06 — `record.Stamp`: SHIPPED.** `Stamp{at, known}` (unexported fields — inconsistent states unconstructable), `NewStamp`, `IsZero`, `Time`, `String`, lossless JSON (`{"at":...}` / `null`, verified under encoding/json v1 AND v2). `CommonMetadata.Created/Received/Stored` added; the three `time.Time` timestamps deprecated (removed in v5). Event bridge stamps `Created` from `OccurredAt`; query bridge stamps `Received` from `ReceivedAt` — **the honest home** for the server-receive clock the old field parked in `ClientCreatedAt` (deviation from lockstep documented in the bridge comment; Created stays unknown for queries — no client clock exists).
6. **T07 — `record.Actor`: SHIPPED.** `ActorKind` mirror (Unknown/User/Bot/System/Service, uint8 + `String`), `Actor{Kind, Raw}`, `IsZero`, `String` — byte-identical wire form to `id.ActorID.PrefixedString`. `metadata.RecordActor(tracing)` converter: kind-discriminated `ActorID` wins; legacy `UserID` fallback is **upgraded** to `ActorUser` (the kind it always implicitly had — deliberate difference from `ActorString`'s bare string, pinned by test). metadata gained a `record/v4` dep (Tier 0→Tier 0; `#check-arch` green). `CommonMetadata.Actor` added; `ActorID` deprecated. All three bridges populate via `RecordActor`.
7. **T08 — `Record.ID` + `Record.Encoding`: SHIPPED.** Identity (EventID/CommandID/RequestID) no longer dropped by the bridges (P5). Event bridge fills `Encoding` from `evt.Encoding()` — mixed JSON+CBOR streams stay self-describing through Record-aware folds (tested with both codecs). **Deviation from plan/review:** `Encoding string` (the codec layer's self-describing form, same as the ADR-0044 envelope) instead of the sketched `uint8` — a numeric mapping would exist nowhere else in the ecosystem and would drift. Commands (no payload) and queries (envelope-wrapped) fill only `ID`.
8. **Deprecation-window lint plumbing.** The `SA1019 .*removed in v5.*` exclusion was path-scoped to the 2026-08-17 wave; extended to record|metadata|event|command|query|metaengine|commandlifecycle (text anchor still keeps all other deprecations loud). `commandlifecycle/projections` **migrated to canonical `Cause.ID`** (value-identical during lockstep) instead of suppressing; its module gained the standard local record replace (unpublished symbols) and its fixture now populates `Cause`. record/ joined the id/ `recvcheck` exclusion for the intentional mixed-receiver marshal pattern.
9. **Process artifacts:** api-stability golden regenerated after every task (4248 exports, +18 from this session); CHANGELOG `[Unreleased]` entries for all four features (changelog-symbols gate green — after one fix, see d3); `nix fmt` before every commit.

## b) Partially done

1. **Foundation-tier verify gate: 2 of 3 rounds RED→fixed, final re-run PENDING.** Post-T08 `#verify-fast` failed twice on lint (26 SA1019 window findings, then exhaustive/exhaustruct/goconst/nlreturn/recvcheck/wrapcheck on the new code — see d4). All fixes are committed (`a775466d7`, `e361f4091`) and record+metadata lint clean standalone, but **the full exclusive verify-fast has NOT been re-run on the final state** — that is the single most important open verification.
2. **T09 (metadata capability interface): RESEARCHED, not implemented.** Survey result: the only production duck-typed sites are the two identical inline `metadatable` declarations in `query/audit.go:86,114` (+ one inline `Payload()` assertion at :101). `Command`/`Query` interfaces are minimal by design; `*BasicCommand`/`*BasicQuery` both already implement `Metadata()`. TODO_LIST already records the owner decision (capability now; interface growth BREAKS hand-rolled impls → v5 cut). Implementation is scoped: `MetadataCarrier` (+ `PayloadCarrier`) in query/, `MetadataCarrier` in command/, adopt in audit.go, comparison-table memo.

## c) Not started (from this session's scope)

- **T09** implementation (see b2), **T10** (decider `ExecuteRef`/`LoadRef`), **T11** (scheduling `TimerID` + `Timer.Actor id.ActorID`), **T12** (`record.Type` consolidation), **T24** (post-landing sweep: doc-check skill refs, consumer-pin matrix, meta-tests), and the entire long tail (T16–T23, T25). TODO_LIST "Core Data Model v4.x/v5" section carries all of them with checkboxes.

## d) What I totally fucked up (honest log)

1. **Repeated a documented mistake: chained commit after a gate with `;` instead of `&&`.** The T06 CHANGELOG commit ran although the changelog-symbols gate had FAILED (exit 1). Consequence: one broken commit (`2e879e897`) claiming a green entry it didn't have; fixed immediately in `b6784b8f1`. This is the exact hazard written in my own hygiene rules — I typed `;` out of habit.
2. **/tmp exhaustion found late.** First verify-fast died with "no space left" (48G tmpfs at 100% — a stale 27G `/tmp/gocache` plus 6 duplicate caches). `go clean -cache` alone failed because `GOMODCACHE` still pointed at the corrupted `/mnt/buildcache` — the env override must include GOMODCACHE for ANY go command, even `go clean`. 35G freed. Should have checked `df /tmp` before the first gate, not after.
3. **Three of my CHANGELOG symbol citations were prose, not symbols** (`evt.OccurredAt`, `q.ReceivedAt`, `codec.Encoding`) — the honesty gate parses dotted identifiers as pkg.Symbol. Two rounds of rewording. Rule for next time: no `x.Y` in Unreleased Added/Changed text unless X is a repo package.
4. **Shipped the foundation tier with lint findings twice.** I ran module tests but only ran lint at the tier gate — 26+9 findings across two verify rounds. Cause: `golangci-lint` wasn't in my per-task loop (module tests ≠ lint). Per-task `golangci-lint run` on touched modules is ~10s each and would have caught all of it pre-commit.
5. **The daemon swallowed my final commit.** Between my last edit and my commit, the daemon committed the identical content (`e361f4091`) — my commit found a clean tree and failed with exit 1. Not data loss, but it means the final lint-fix state was committed by the daemon's hook path, and I have NOT verified that commit's pre-commit gates actually ran.
6. **Assumed, didn't ask (again).** g1/g2/g3 from the previous session: I proceeded on stated assumptions (g2 = one commit per task — held; g3 = never push — held; g1 = capability + comparison table — still unconfirmed).

## e) What I can do better

- **Add `golangci-lint run` (touched modules only) to the per-task loop**, before every commit. Module tests are not a lint gate.
- **Commit in smaller stages** (per module green, not per task) — four daemon splits this session; smaller stages make each split harmless and reviewable.
- **Check `df /tmp` + export the full env block (incl. GOMODCACHE) at session start**, before the first gate, not after the first ENOSPC.
- **Never type `;` before `git commit` after a gate.** Use `gate && commit` or run gates as separate tool calls.
- **Tick the TODO_LIST checkboxes in the same wave as each task's commit** — T05–T08 boxes are still unticked (deferred to next session's first action).

## f) Next steps (priority order, first 25)

1. **Re-run the exclusive `#verify-fast` on current HEAD (`e361f4091`)** — the pending foundation-tier gate. Everything else waits on this.
2. If green: tick TODO_LIST checkboxes for T05–T08 items in "Core Data Model v4.x/v5".
3. **T09·c/d**: `query.MetadataCarrier` (+ `PayloadCarrier`) and `command.MetadataCarrier` capability interfaces; adopt in `query/audit.go` (replaces both inline `metadatable` declarations + the inline `Payload` assertion).
4. **T09·b/e**: comparison-table memo (capability vs v5 interface growth — pending g1 confirmation) as plan Appendix C or ADR-0123 addendum; tests + golden + CHANGELOG.
5. **T10·a/b**: `Decider.ExecuteRef`/`LoadRef` additive variants; pair forms get `Deprecated: removed in v5`. **Ref type needs the owner's call (see g2).**
6. T10·c: migrate internal callers (scenario/, examples) to ref forms.
7. T10·d: tests + golden + CHANGELOG + verify-fast.
8. **T11·a**: `TimerMarker` phantom + `TimerID = id.Of[TimerMarker]` in scheduling.
9. T11·b: `Timer.Actor string` → `id.ActorID`; keep JSON wire compat (PrefixedString round-trip; check sqlstore/memory persisted forms first).
10. T11·c: add `id/v4` dep to scheduling go.mod (+ local replace); run `#check-arch`.
11. T11·d/e: fix store signatures + tests; golden + CHANGELOG; consumer-pin sweep for scheduling consumers.
12. **T12·a**: `record.Type` + shared `ParseType`/`IsZero` (parametrized error) in record/.
13. T12·b/c: `type Type = record.Type` aliases in event/command/query; deprecate local ParseType duplicates as thin wrappers.
14. T12·d: golden + verify-fast.
15. **T24·a**: api-stability meta-tests (`TestEvery*`) — should be green from per-task regens; confirm.
16. T24·b: doc-check over SKILL.md + `.agents/skills/go-cqrs-lite/references/*.md` + AGENTS.md — **must also document the new record API** (Cause/Stamp/Actor/ID/Encoding + NewStreamRefOrZero from last session) in the skill references, not just pass the checker.
17. T24·c: consumer-pin sweep — bump every consumer pinning `record/v4 v4.2.0`/`metadata/v4`/`id/v4` to the next tags once cut; GOWORK=off build matrix.
18. **Tag `record/v4`, `metadata/v4`** (and event/command/query if warranted) so the local replaces in event/command/query/metadata/commandlifecycle/projections can be dropped at the next pre-tag sweep — six modules now carry unpublished-symbol replaces.
19. T16: report polish leftovers (anchor integrity script, CSS diff) on the core review HTML.
20. T17: `snapshot.NewSnapshot` constructor + codec stamp (envelope style).
21. T18: snapshot wire-tag migration audit across pebble/bbolt/sql/memory.
22. T19: deprecation census artifact → `docs/planning/v5-deprecation-sweep.md` (now includes the six new record-bridge deprecations).
23. T20: tombstone v5 deletion prep (verify migration doc + StatusMiddleware bridge tests).
24. T21: extended data-model review of storage/*, system/, stack/, watermill/, middleware/ (new review file, cross-linked).
25. T22 (naming review of Stream/StreamRef/Cause/Stamp/Actor vocabulary) + T23 (upstream skill fixes) + T25 (AGENTS.md memory update with the decided conventions).

## g) Questions (max 3)

1. **T09 memo placement (g1, still open from last session):** my reading of your "Table view comparison" answer is *add a comparison table (capability-interface now vs v5 interface growth) to the T09 decision memo*. Proceed with the capability interfaces now and put the table in the plan's Appendix C? (The interface-grows-at-v5 half is already recorded as decided in TODO_LIST.)
2. **T10 ref type:** the plan says `Decider.ExecuteRef/LoadRef(ctx, ref, …)` but never names `ref`. Both shapes are entrenched: `record.StreamRef` (validated string — the Record-interchange convention this whole tier just built) vs `id.StreamRef` (struct — the storage-layer convention: pebble/snapshot/commandlifecycle/system take it as parameter type). Which one becomes the decider hot-path identity?
3. **T08 Encoding type veto check:** I shipped `Record.Encoding string` (codec layer's self-describing form, matches the ADR-0044 envelope) instead of the review's sketched `uint8`. Rationale: no numeric mapping exists anywhere in the ecosystem; a parallel one would drift. Confirm or veto — veto means a mechanical type swap before any tag is cut.

---

*State at pause: tree clean, HEAD `e361f4091`, foundation tier (T04–T08) fully shipped and committed, final tier gate re-run pending (f1). No push performed. Plan progress: 10 of 25 tasks complete (T01–T08, T13–T15), T09 researched.*
