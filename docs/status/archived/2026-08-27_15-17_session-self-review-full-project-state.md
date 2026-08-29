# Status Report: Session Self-Review + Full Project State — 2026-08-27 15:17

**Session under review:** the 2026-08-22 execution run that completed the
core-data-model v5 execution plan (T16–T23 + f24, plan 25/25). This report
covers what that session fully did, where it stumbled, what it left open,
and the current repo state as of 2026-08-27 15:17 — including working-tree
facts I noticed but did NOT touch.

**Current repo state (verified just now):** HEAD `9685a9810` (the session's
last commit), tree has **4 modified files I did not author** (uncommitted:
`storage/bbolt/command_serialization.go`, `query_serialization.go`,
`storage/turso/indexing/policy.go`, `policy_test.go` — they implement
exactly my TODO follow-ups E3+E9), plus **3 untracked planning artifacts
from 2026-08-27 15:10** (`docs/planning/2026-08-27_15-10_DEEP-FULL-CODE-REVIEW-PARETO.html/.d2/.svg`). Local master is **18 commits ahead of origin**.

---

## a) FULLY DONE (this session, verified end-to-end)

1. **Answer-gate resolved with owner** (question tool): KEEP-BOTH identity shapes confirmed; push-master approved (executed for the then-state); T17-next confirmed.
2. **T17** — `snapshot.NewSnapshot` validating constructor + `Validate` + `Ref` + `ErrInvalidSnapshot` + `Encoding record.Encoding` stamp; TypedStore + decider stamp on save; `SaveSnapshot` deprecated; internal caller migrated. Tests+lint+golden+CHANGELOG+skill row. (`1a7be0816`)
3. **T18** — per-backend snapshot wire audit table in TODO_LIST §v5 (keep-old-tags-until-v5 decision); pebble+bbolt now PERSIST the encoding stamp (additive CBOR `omitempty`, old rows decode as Unknown, roundtrip tests). (`59799970e`)
4. **T19** — `docs/planning/v5-deprecation-sweep.md`: 42 aggregate-vocabulary symbols across 6 modules, 5 record-bridge fields, tombstone API, wire tags + stale error-code inventory, deprecated-module list, execution rules. (`f464b7936`)
5. **T20** — tombstone migration doc verified against shipped APIs; REAL gap found and fixed: OnTombstone/OnRebirth had zero trigger coverage anywhere → new `stack/materialize_tombstone_bridge_test.go` pins the full StatusMiddleware→mark→handler chain; v5 deletion pre-reqs listed. (`463bb525e`)
6. **T21** — extended data-model review (`docs/reviews/2026-08-22_extended-data-model-review.md`): 15 findings E1–E15 + verified backend capability matrix; verdict: no any-typed values anywhere, the disease is cross-backend drift; pebble lying-doc fixed on the spot. (`9b863467e`)
7. **T22** — naming decision recorded as plan Appendix D: trio keeps adjective-bearing names; REAL smell found (`record.StreamRef` string `/`-form vs `id.StreamRef` struct `:`-form — same name, different concept + separator) → v5 rename `record.StreamKey` queued in the sweep; ActorKind mirror re-confirmed accepted. (`8e3f701fe`)
8. **T23** — data-model-review skill fixed in `~/.config/crush/skills/`: output path divergence (`docs/brainstorming`→`docs/reviews`) + two new Step-5 rules (read prior reports; copy the template, never transcribe). (`9b562f7c6` tick)
9. **T16** — core review HTML polish: anchors 0 broken/0 orphan (Amendments TOC entry added), CSS diff vs kit = zero token drift, Related Reviews + Next Skills sections added. (`fc2b18fe2`)
10. **f24** — first real PG execution post-TimerID caught TWO real breaks: (a) `-tags=integration` test file compared TimerID as untyped string — invisible to every normal gate; (b) benchkit + projectionhost stranded on old storage pins. Both fixed; scheduling/sqlstore ran green on PostgreSQL. Stranded-pin sweep over all scheduling v4.3.0 consumers. (`043467885`)
11. **Final gate GREEN** — `#verify-fast` fully green after three en-route fixes (dep budget 10→11 for pebble/bbolt with rationale; art-dupl annotations on the intentional conversion-helper mirrors; /tmp tmpfs reclaim via trash-empty). Session report committed (`cba8122cc`).
12. **Plan progress: 25/25 tasks across all sessions. The core-data-model v5 execution plan is COMPLETE.**

## b) PARTIALLY DONE

- **Master push**: the session-start push (owner-approved) published up to `5d3ce030d`; the session's own 13 commits (+5 daemon) after that are **local only** (18 ahead of origin). Approval scope was read conservatively.
- **Working-tree follow-ups (NOT mine)**: someone began executing my report §f items — bbolt error-family parity (E3) and turso Policy nil-map guard (E9) sit **uncommitted** in the tree with clean-looking diffs. Untouched by me; unverified/uncommitted status unknown.

## c) NOT STARTED (known, deliberate — out of session scope)

1. **The v5 deletion wave itself** — executing `v5-deprecation-sweep.md` (the plan deliberately prepared, not executed).
2. **Snapshot release tag wave** — snapshot/decider/pebble/bbolt carry unpublished symbols behind local `replace` directives until tagged.
3. **PG integration test isolation fix** (filed: shared `cqrs_test` DSN bleeds global journal across tests — pre-existing, reproduced twice).
4. **system.ShutdownDependency name validation** (E10).
5. **system.Execute ref-form wrapper** (v5, noted out-of-scope).
6. Whatever the **2026-08-27 15:10 deep-full-code-review pareto artifact** initiates — appeared after the session; unread by me.

## d) TOTALLY FUCKED UP (honest log, this session)

1. **Fell into the documented `/`-in-log-path trap AGAIN** (storage module build loop) — phantom FAILs from failed redirects; the exact lesson was written in the 14:10 report of the SAME project. Two calls burned.
2. **T22 appendix append blunder** — appended a duplicate T22 section at the plan file END instead of editing the body section in place; needed surgical removal.
3. **Wrote test code against constructors I'd never opened** (`memory.NewSnapshotStore` doesn't exist; `idtest` import path guessed wrong) — two compile round-trips that grepping first would have killed.
4. **govet non-constant format string** in `helper.go` — wrote string concatenation where a format string belongs; caught by tests, not by me.
5. **art-dupl directive invented, not copied**: unspaced `//art-dupl:accept` + >120-char line — violating BOTH the repo's spaced convention and gofumpt. Two lint round-trips. Should have grepped an existing directive first.
6. **Five verify-fast runs to reach GREEN**: budget violation (T17/T18 promoted record to a direct dep — predictable), then dupl gate, then lint form. Root cause: I committed per-task WITHOUT running the cheap gates (`#check-arch`, `#check-duplication`) that would have caught each issue in seconds at commit time.
7. **Never checked disk across heavy runs** — two full PG suites + three verify runs filled the 48G tmpfs; the DuckDB static link died with "No space left on device" mid-gate. Fix (trash-empty, +11G) was documented in AGENTS all along.
8. **Wrong replace depth** (`../snapshot` instead of `../../snapshot`) for nested storage modules — one round-trip each.

## e) WHAT WE SHOULD IMPROVE (process, derived from d)

1. **Run the cheap gates BEFORE every commit** (`check-arch`, `#check-duplication`, `check-changelog-symbols`), not only per-wave — would have collapsed 5 verify runs into ~1-2.
2. **Module-loop boilerplate**: flat log filenames (`tr / -`) in the FIRST loop version; `df -h /tmp` check between heavy nix runs.
3. **Copy, don't invent**: directives, doc-comment forms, and constructor names — grep an existing example before writing new ones.
4. **Add a compile-only CI pass with `-tags=integration`** — the f24 bug class (tag-gated files rotting invisibly) is structural; every gate compiles default tags only. One `go vet -tags=integration ./...` job kills the whole class.
5. **Pre-flight dep-budget check when promoting an indirect dep to direct** (record in pebble/bbolt was foreseeable at edit time from contract #3).
6. **Push policy needs one explicit rule** — "push each green session-end" vs "push only on request" — so approvals don't need re-interpretation (see g2).

## f) NEXT — up to 50 things, priority-ordered

**Blocking/unblocking (do first):**

1. Decide ownership of the 4 uncommitted working-tree files (E3+E9 work) — verify + commit or discard (g1).
2. Read the 2026-08-27 15:10 DEEP-FULL-CODE-REVIEW-PARETO artifact — it likely re-prioritizes everything below (g3).
3. Push the 18 local commits (owner call — g2).
4. Snapshot tag wave: tag snapshot→decider→storage→pebble→bbolt in dependency order (push-interleave protocol), drop the 3 local replaces, standalone-build matrix over consumers.
5. PG integration isolation fix (per-test DB under explicit DSN) — S.

**Extended-review follow-ups (S, each independent):**
6. bbolt error-family parity — E3 (see also working tree, g1).
7. turso.Policy nil-map write guards — E9 (same).
8. system.ShutdownDependency name validation vs Engines — E10.
9. Document handle-ownership asymmetry in storage/eventstore (E14).
10. Document the watermill protocol metadata key-set as a protocol version (E12).

**v5 cut wave (from v5-deprecation-sweep.md, per its execution rules):**
11. Delete id's 12 Aggregate aliases.
12. Delete event's 8 Aggregate aliases + 2 methods.
13. Delete command's 6 Aggregate aliases.
14. Delete query's 2 error aliases.
15. Delete listing's 5 Aggregate aliases.
16. Delete misc singles (command StreamRef alias, metadata EnsureCustom/CustomData, snapshot SaveSnapshot, 3 ParseType wrappers, decider pair forms).
17. Delete record-bridge Deprecated fields (CausationID, ActorID, ClientCreatedAt, ServerReceivedAt, StoredAt) after lockstep period.
18. Delete tombstone metadata API (needs: listing type-driven status first).
19. listing: type-driven status replacing DetectTombstone call at in_memory.go:155.
20. Migrate example/taskmanager off OnTombstone.
21. Rename `record.StreamRef` → `record.StreamKey` (+ deprecated alias one major).
22. Converge stream-key separator (`/` vs `:`) + migration guide entry.
23. Event-envelope `Encoding string` → `record.Encoding` in pebble+bbolt (E1).
24. Rename watermill RetryConfig (E7) + add Validate.
25. Typed `Kind` enum for MessageAdapter/DeadLetterEntry (E8).
26. `AdapterCore.Encode` gains error return (E11).
27. SQLTimerStore phantom type param (E13) — redesign or document.
28. Middleware signature unification assessment (E15).
29. Merge/bridge middleware Option vs BundleOption (E6).
29b. Batch error-code renames (aggregate→stream vocabulary) + changelog note.
30. snapshot.Snapshot JSON tags → honest stream names + pebble dual-read migration.
31. pebble serializableSnapshot tag rename (aggregate_id→stream_id) with legacy read path.
32. bbolt command_serialization tag rename.
33. SQL snapshots table column renames (ALTER per dialect) + migrations.
34. benchkit `aggregates` JSON key rename (output contract break — decide).
35. Delete stack Bundle/Materialize/RunProjections + 8 presets.
36. Delete storage/view + storage/relational.
37. Delete transport/http + transport/grpc (tag final patches first).
38. Delete schema.VersionedStore shells, signing.Rejecting*, encryption.ErrInnerStoreNot*.
39. Delete storage/sql.BuildWhereClause.
40. `record.NewStreamRef` validating v5 signature change + call-site migration.
41. v5 migration guide (stack presets → system, v1 tiers, transport, manual views).
42. system.Execute ref-form wrapper decision.

**Hardening/quality:**
43. CI compile-only pass with `-tags=integration` (kills the f24 bug class).
44. `df /tmp` guard or GOTMPDIR hygiene doc for long gate sessions.
45. Extend cqrs-lint with a rule: flag string comparisons against branded-ID types (would have caught f24 at write time).
46. Consider per-test-database helper for ALL integration suites (same class as #5).
47. Update AGENTS.md with the art-dupl spaced-directive + cheap-gates-before-commit conventions (e is only in this report today).
48. Re-run data-model-review skill after the v5 cut (per Next Skills section of the core review).
49. Sweep example/getting-started + integration old storage pins during the next wave (internally consistent today, latently fragile).
50. docs-health pass: TODO_LIST pruning of completed plan sections now that the plan is 25/25.

## g) Questions (3, cannot figure out myself)

1. **The 4 uncommitted working-tree files** (bbolt serialization + turso policy — exactly my TODO E3/E9): another agent session's in-flight work, or yours? Commit-and-verify them, leave them, or discard?
2. **Push policy**: the session-start approval published up to `5d3ce030d`; 18 commits have accumulated since (incl. all of the session's shipped work). Should "push master" now mean every green session-end, or only on explicit request?
3. **The 2026-08-27 15:10 DEEP-FULL-CODE-REVIEW-PARETO artifact** (untracked, 7 min old): is a new review/planning cycle running in parallel, and does its outcome supersede or complement the v5-deprecation-sweep as the next major wave?

---

_Reported 2026-08-27 15:17 from repo state HEAD `9685a9810`, tree NOT clean (4 foreign modified files + 3 untracked planning artifacts), master 18 ahead of origin. No files were modified in the making of this report. WAITING FOR INSTRUCTIONS._
