# Status Report: Core Data Model Plan Execution (T01–T15 wave 1)

**Date:** 2026-08-22 04:53 CEST
**Scope of this report:** ONLY this session's run — executing the Pareto plan
(`docs/planning/2026-08-22_03-52_core-data-model-v5-execution-plan.md`) from
T03 through T04, plus the T01 owner checkpoint and the T13/T15 hygiene tier.
No research beyond what the plan itself ordered.

---

## a) FULLY DONE

1. **T03 — metaengine review de-dupe** (commit `d3c711023`). Read
   `2026-08-01_metaengine-data-model.html` end-to-end (full text extraction,
   2049 lines). Verdict: **zero overlapping findings** — that review covered
   metaengine internals (Fold god-struct, Store god-object, queryRuntime
   twin), this one the core type system. Two coordination facts recorded in
   a cross-link annotation added to that file: both reviews use `P1–P12` for
   DIFFERENT axes (anti-pattern classes vs findings), and metaengine's
   deferred M2 (Delta wrapper "until major version bump") rides the same v5
   wave. Div balance re-validated (127/127). The reciprocal link in the core
   review landed with T02.
2. **T14 — metaengine ripple sizing** (commit `e7bd90b0a`, plan Appendix A).
   Swept 98 files: all `record.Record` literals in metaengine are keyed
   (additive fields compile clean), no production Record JSON serialization,
   no go-snaps goldens; the real v5 exposure is `record_stamp.go`'s 7 field
   getters + exactly 3 production `NewStreamRef` call sites (the three
   bridges). Zero production `.Split()`/`.Validate()` callers found.
3. **T01·a+b — decision memo** (commit `de42441ca`, plan Appendix B).
   Extracted the exact recorded text (finding: ADR-0123 itself never names
   `NewStreamRef` — the decision lives in TODO_LIST §v5 + the record.go
   NOTE). Key new research fact: `id.StreamRef` is entrenched as the
   STORAGE-layer ref (pebble/snapshot/commandlifecycle/system), so a
   record-side struct would add a third identity shape, not remove one.
4. **T01·c — OWNER CHECKPOINT executed.** Decision received via structured
   question: **Option B — the string `record.StreamRef` survives v5; the
   validating constructor stands; the struct proposal is rejected.**
5. **T01·d+e — decision recorded.** TODO_LIST §v5 `NewStreamRef` item
   owner-confirmed with memo pointer; plan Appendix B status flipped to
   DECIDED (landed via auto-commit daemon as `a16ebf924`/`94a3b3266`).
6. **T02 — published review corrected** (commits `b8da8c491`, `904502e8d`).
   In-place fixes: Roadmap Step 1 dual-Version proposal replaced (int64 +
   validate now, uint64 swap at v5); Step 2 reclassified as **breaking for
   hand-rolled implementations** with the v4.x capability-interface path;
   Step 6 keeps the string type; Decision Log rows 1 ("Stream shape") and 7
   ("Command interface growth") amended. New "Amendments (2026-08-22)"
   section records the owner decision + rationale, the de-dupe verdict, and
   adds Related-reviews + Next-skills sections (also closes T16·c). HTML
   validated: 143/143 divs, all anchors resolve, 0 external refs, clean tag
   stack.
7. **T13 — harvest** (commit `c9dc8f1d1`). New TODO_LIST section "Core Data
   Model v4.x/v5" with the 19 open tasks; duplicate cross-check clean (the
   v5 breaking-cut item and the new v4.x adoption item are complementary).
8. **T15·b+c — treefmt/HTML gate check.** treefmt config has NO html
   formatter → HTML reports are simply not covered → the CI
   `--fail-on-change` gate cannot touch them. No action needed (adding an
   HTML formatter would actively mangle the hand-tuned reports).
9. **T04 — validated stream-ref population (code complete, green).**
   `record.NewStreamRefOrZero` (zero ref instead of malformed ref on empty
   entity ID — the producer-side counterpart of the v5 validating
   constructor); all three `AsRecord` bridges switched to it with updated
   doc comments; invariant tests added in record + event + command + query
   (populated refs must pass `Validate` and round-trip `Split`); table test
   for the constructor. Module tests green under `GOWORK=off` for all four
   modules (cache env redirected to /tmp). Sources committed by the daemon
   as `193a8c543`; the replace directives, api-stability golden (+1 export)
   and CHANGELOG entry landed via daemon commits `30405c219`/`b1a26cf70`.

## b) PARTIALLY DONE

1. **T04 final commit was not mine.** My completion commit (replaces +
   golden + CHANGELOG, message prepared) was **rejected by the pre-commit
   hook**: the hook's embedded api-stability check died on the corrupted
   `/mnt/buildcache` ("mkdir … no such device") and misreported the golden
   as stale. The auto-commit daemon then landed the same content in pieces
   (`30405c219`, `b1a26cf70`). Net effect on master is correct, but I never
   verified those daemon commits' completeness myself — next session must
   confirm the tree builds standalone before anything else.
2. **T15·a — verify-fast gate: one FAILED run, inconclusive.** It executed
   concurrently with my mid-flight T04 edits (violating the repo's own
   load-sensitivity rule) and under the corrupted cache. Needs a clean,
   exclusive re-run; until then no GREEN claim exists for this session's
   code changes.
3. **T16·a anchor check** was done ad hoc (python) rather than as the
   planned reusable script; **T16·b CSS template diff** not done at all.

## c) NOT STARTED

T05 (record.Cause), T06 (record.Stamp), T07 (record.Actor), T08
(Record.ID+Encoding), T09 (metadata capability interface — owner input was
ambiguous, see g1), T10 (ExecuteRef/LoadRef), T11 (TimerID branding), T12
(record.Type consolidation), T17–T21, T22 (naming decision), T23 (skill
fixes), T24 (post-landing sweep — only the golden-regen part is done), T25
(AGENTS.md memory update).

## d) TOTALLY FUCKED UP

1. **Malformed HTML shipped in a commit.** My Amendments-section edit
   injected literal `\n` sequences and orphan `n` line-starts (sloppy
   multiedit tool use), and — after sed cleanup — still left **3 unclosed
   `<strong>` tags**, which went out in `b8da8c491`. My own parse validator
   caught it, but only AFTER the commit, fixed in `904502e8d`.
2. **Validator and commit were chained with `&&`** — the validator printed
   MISMATCH warnings but exited 0, so the commit ran anyway and I read the
   output too late. Validation must gate the commit, not accompany it.
3. **Ran `#verify-fast` concurrently with active edits** — the exact
   anti-pattern AGENTS.md documents. The gate run is wasted and its failure
   tells us nothing.
4. **Foreseeable environment failure not pre-empted.** The hook's Go steps
   need the /tmp cache env redirect (documented in AGENTS.md for days); I
   exported it for my own commands but not for the commit → hook failure →
   daemon had to rescue the content.
5. **Misleading commit message** on `b8da8c491`: it claims TODO_LIST + plan
   changes were included, but the commit contains only the review HTML (the
   daemon had already taken the other files). Message/archive mismatch.
6. **Daemon race on multi-file work**: `193a8c543` shipped the T04 sources
   WITHOUT the sibling replace directives → master was briefly broken for
   `GOWORK=off` standalone builds of event/command/query until the daemon's
   `30405c219` caught up. Atomic multi-file changes must be committed as
   one unit, immediately.

## e) WHAT WE SHOULD IMPROVE (session-level lessons)

1. Export the full cache env (`GOCACHE/GOMODCACHE/GOPATH/GOLANGCI_LINT_CACHE=/tmp/…`,
   `GOTOOLCHAIN=auto`) at session start — the pre-commit hook inherits it.
2. Validate generated artifacts BEFORE the commit command, in a separate
   step; never chain validator+commit when the validator can exit 0 on
   warnings.
3. Never run gates while editing code; run them exclusively.
4. Commit atomic change sets (source + replaces + golden + CHANGELOG) the
   moment they're complete — the daemon will otherwise commit the pieces it
   can see, in whatever state it sees them.
5. After every daemon commit that touches go files, run a quick standalone
   build check (`GOWORK=off go build ./...` in the touched modules).
6. Treat free-text answers from the question tool as ambiguous; confirm the
   interpretation before building on them (see g1).

## f) Up to 50 things to do next (prioritized)

1. Verify master is standalone-green now: `GOWORK=off go build ./...` in
   event/command/query/record (daemon-committed state).
2. Decide what the current uncommitted ~58-module go.mod/go.sum replace
   sweep is (in progress at session end — NOT authored by me; investigate,
   then commit or roll back deliberately).
3. Clean exclusive `nix run .#verify-fast` run; capture result in plan (T15·a).
4. T05 PR: `record.Cause{Kind, ID}` + `CommonMetadata.Cause` + event bridge
   mapping; deprecate `CausationID`; tests + golden + CHANGELOG.
5. T06 PR: `record.Stamp{at, known}` replacing the three zero-time
   timestamps; deprecate old fields; bridges populate.
6. T07 PR: structural `record.Actor{Kind, Raw}` mirror; bridges stop
   stringifying the union at intake.
7. T08 PR: `Record.ID` + `Record.Encoding`; `event.AsRecord` fills both;
   mixed JSON+CBOR round-trip test; extend `metaengine/enginetest/record_stamp.go`.
8. T09: resolve g1, then implement the metadata capability interface +
   comparison-table memo; deprecate `query/audit.go:86,114` duck-typed
   interfaces.
9. T10 PR: `ExecuteRef`/`LoadRef` additive variants; deprecate pair forms;
   migrate scenario/ + examples.
10. T11 PR: `scheduling.TimerID = id.Of[TimerMarker]`, `Timer.Actor
    id.ActorID`; `#check-arch`; consumer-pin sweep.
11. T12 PR: `record.Type` consolidation + per-module aliases.
12. T24 sweep: api-stability meta-tests (`TestEvery*`), doc-check over skill
    refs, consumer-pin bump matrix under GOWORK=off.
13. Update `.agents/skills/go-cqrs-lite/references/*.md` for
    `NewStreamRefOrZero` (doc-check target; AGENTS.md procedure step 3).
14. Mark TODO_LIST harvest item (StreamRef bridges) done → move to CHANGELOG
    (docs-health rule: completed work leaves TODO_LIST).
15. T16·b: CSS diff of the review HTML vs the html-report-kit template.
16. T16·a: formalize the anchor-integrity check as a small script if wanted.
17. T17: `snapshot.NewSnapshot` constructor + codec stamp (ADR-0044 style).
18. T18: snapshot wire-tag migration audit across pebble/bbolt/sql/memory.
19. T19: deprecation census artifact → `docs/planning/v5-deprecation-sweep.md`.
20. T20: tombstone v5 deletion prep (migration doc verify + bridge coverage).
21. T21: extended data-model review (storage/system/stack/watermill/middleware).
22. T22: Stream/StreamRef/StreamID naming decision (now firmly unblocked by
    Option B).
23. T23: upstream skill fixes (reviews/brainstorming divergence; read-prior-
    reports + copy-template steps).
24. T25: AGENTS.md memory update — Option B outcome + new conventions
    (validated-population pattern, capability-interface rule).
25. Check `docs/DOMAIN_LANGUAGE.md` StreamRef entry for the OrZero
    constructor mention.

## g) Questions I CANNOT answer myself

1. **T09 direction (metadata access).** My checkpoint question asked whether
   external consumers hand-roll `Command`/`Query` implementations or embed
   `*BasicCommand`; your answer was "Table view comparison". I read that as:
   *show me a comparison table of the options in the T09 memo before
   deciding* — and since the capability interface is safe under every
   answer, I planned to proceed with it while the table rides the memo.
   Confirm that reading, or pick directly (capability interface / grow
   interface in v4.x / defer to v5).
2. **Commit granularity vs the auto-commit daemon.** The daemon split my
   atomic T04 set and briefly broke standalone builds (d6). For T05–T08
   (each touching record + 3 bridges + golden + CHANGELOG): keep the plan's
   one-PR-per-type granularity with immediate single commits (my plan), or
   batch several record types per commit to shrink the race window?
3. **Is local master auto-pushed anywhere?** The temporarily-broken
   standalone state existed on local master for ~30 minutes. If anything
   (daemon, CI hook) pushes automatically, future mid-edit states could
   leak to consumers; if push is manual-only (my assumption — I have not
   pushed), the risk is contained and g2's answer is just "commit faster".

---

*Point-in-time snapshot. Session paused mid-plan: decision tier + reference
tier DONE (T01–T03, T13, T14, T15·b/c), foundation tier started (T04 code
green, commit hygiene messy), surface + long-tail tiers untouched. WAITING
FOR INSTRUCTIONS.*
