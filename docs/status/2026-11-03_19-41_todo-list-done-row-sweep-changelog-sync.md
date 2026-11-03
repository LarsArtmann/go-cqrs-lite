# TODO_LIST done-row sweep + CHANGELOG sync — session status report

**When:** 2026-11-03 19:41 CET
**Session product:** TODO_LIST.md 3370 → 2524 lines (114 done rows deleted, 5 open
tails re-rowed, 2 deletion-damage repairs, 1 pre-existing hazard fixed);
CHANGELOG.md +1 entry (E023, the only genuinely unrecorded shipped work).

## Scope — what this report VERIFIED vs did NOT verify

**Verified this session (command/artifact named):**
- All 114 `- [x]` blocks inventoried with a block-scanner script (start line +
  continuation lines; 970-line dump read in full before any deletion).
- CHANGELOG coverage for every done row: grep matrix over `CHANGELOG.md`
  (~40 marker patterns) + targeted reads of the wave sections (14-tag wave
  96–183, trust wave II 244–298, reentrancy wave 279–298, 42-tag train
  323–1168, `[Unreleased]` 28–86).
- E023 release status: `git log --diff-filter=A` (commit `dcb786f0c`, 10-24
  18:06) vs `git for-each-ref` tag date (v4.18.0, 10-24 06:44) + `git tag
  --contains` (empty) — the rule is in NO tag.
- Post-sweep state: `rg '\[x\]'` → 0 matches in any form (incl. blockquote);
  dangling-pointer sweep (`receipt above|noted in the row above|see above`,
  etc.) → clean; setext-hazard scan (`---` glued to a paragraph) → clean;
  section-index headers all present; `bash scripts/check-changelog-symbols.sh`
  GREEN (22 citations verified against the API golden).
- The parallel session's `docs/status/2026-11-03_19-09_rawquery-*` file:
  now COMMITTED by the daemon (`df2d6e2b0`) — it was untracked at session
  start.

**NOT verified (and why):**
- **Formatter compliance of my markdown edits** — never ran `nix fmt
  --fail-on-change` or a scoped dprint check on TODO_LIST.md/CHANGELOG.md
  (a whole-tree formatter pass in a multi-session tree risks colliding with
  in-flight work; no treefmt markdown ownership was confirmed either). CI
  will tell.
- **Per-hunk review of all 858 deleted lines** — verified structurally (0
  checkboxes, 0 dangling pointers, sections intact, blank-run collapse
  produced 0 setext hazards) but not eye-line-by-line.
- **The ~90 surviving open rows** were NOT re-verified against code (out of
  scope: this session was a done-row sweep). At least one stale premise was
  NOTICED and is reported below (item b-3).
- **"Adequate coverage" judgments** for four minor clusters (M12/M14 polish,
  benchkit LICENSE fix, ephemeral-script passthrough, vm-mysql port fix)
  were judgment calls ("internal tooling polish, receipts live in archived
  plans"), not verified changelog rows.
- **Root cause of the M4–M7 duplicate block** (pre-existing) — not
  investigated.

---

## a) FULLY DONE

1. **114 done rows deleted from TODO_LIST.md** (3370 → 2524 lines). Block
   boundaries computed mechanically (assertion: exactly 114), each block read
   in full BEFORE deletion; zero checkboxes of any form remain
   (`- [x]`, `> - [x]`, blockquoted).
2. **CHANGELOG coverage audit, all 114 rows.** One genuinely missing entry
   found: **E023 `satellite-owned-sql`** — added to `[Unreleased]` Added
   (CHANGELOG.md:62) with its untagged status stated. Everything else was
   already recorded in wave sections (14-tag wave, trust wave II, reentrancy
   wave, 42-tag train, 10-05 waves) or `[Unreleased]`.
3. **Five open tails re-rowed** (they lived inside done blocks and would
   have been destroyed):
   - cqrs-htmx test battery vs system/v4.15.4 to close C03 (TODO_LIST:137)
   - nightly-lint timer install (owner, XS) (TODO_LIST:469)
   - ghost-module legs: go-retry/go-flightrecorder replaces + crm-exec-stage
     merge markers (TODO_LIST:1737)
   - hub T15.2 + T18 responses (TODO_LIST:424)
   - owner external actions F153/evals/billing, ERRAUDIT_PAT marked resolved
     (TODO_LIST:354)
4. **Deletion-damage repairs** (both found by post-hoc sweeps, both fixed):
   - turso flip row's "see the receipts above" → now points at the CHANGELOG
     `TURSO_TEST_SYSTEM_LIB` entry.
   - E023 release row's "scorecard-wiring follow-up already noted in the row
     above" — the referenced row was deleted; the idea is now inlined
     (TODO_LIST:468).
5. **Pre-existing setext hazard fixed**: `---` glued to `_(Effort: M)_`
   rendered as a setext h2 (present in HEAD, same line number).
6. **Header records the 14th docs-health pass** (2026-11-03): 114 done rows
   deleted, five tails re-rowed.
7. **One blockquoted done sub-item struck** per the file's own legend,
   keeping the standing cqrs-lint rebuild recipe.
8. **Gates**: `check-changelog-symbols.sh` green after the new entry (its
   test-symbol citation `TestShouldExitWithError_InfoAdvisoryNeverGates`
   verified present in `cmd/cqrs-lint/run_severity_test.go`; E023 verified
   in `RULES.md` + `register.go` — no ghost citations).
9. **The E023 release blocker is now coherent end-to-end**: CHANGELOG entry
   (shipped, untagged) ↔ TODO_LIST:468 (cut v4.19.0 + bank-sync pin bump)
   ↔ RULES.md/register.go (rule exists at HEAD).

## b) PARTIALLY DONE

1. **The sweep itself is complete, its verification loop is not closed**:
   per docs-health, this report's section (f) is HARVEST input for
   TODO_LIST. Not yet harvested (you said report-then-wait; say the word).
2. **Markdown formatting**: edits are structurally clean but formatter
   compliance is UNVERIFIED (see Scope). One `nix fmt --fail-on-change`
   run would close it.
3. **Stale premise in a surviving row, noticed but not touched** (out of a
   done-row sweep's mandate): TODO_LIST "Post-tag adoption tail" still says
   "blocked ONLY on the next schema/decider/snapshot tag wave (owner Q2)" —
   those tags shipped 2026-10-12 (schema v4.7.0, snapshot v4.7.0, decider
   v4.8.0). The bank-sync patch / DiscordSync workaround-drop /
   cqrs-htmx UpcasterRegistry-collapse legs are unblocked NOW.
4. **The C04/F41 micro-tail** ("doc gates to re-run at wave closeout",
   from the deleted A16 row) has NO host row in TODO_LIST — unknown whether
   it was executed at the 10-24 wave closeout or silently dropped. One
   cheap gate re-run (`check-md-go` + doc gates) clears it either way.
5. **CHANGELOG ↔ TODO_LIST untagged-work pairing** is prose-linked ("rowed
   in TODO_LIST"), not anchored — will drift again the next time one side
   moves. Acceptable for now; see (e).

## c) NOT STARTED

Everything below was discovered by this session but intentionally not
executed (each is rowed or listed in (f)):

1. E023 tag cut (`cmd/cqrs-lint` ≥ v4.19.0) + bank-sync pin bump.
2. cqrs-htmx battery run (C03 close).
3. Nightly-lint timer enable (owner).
4. Hub T15.2/T18 responses (blocked on hub filing its issues).
5. F153 / evals CLI / billing confirmations (owner, unverified since 10-08).
6. Ghost-module legs (retry/flightrecorder re-pins, crm-exec-stage markers).
7. TODO_LIST hygiene gate (see (e) — does not exist yet).
8. HARVEST of this report into TODO_LIST.
9. Scorecard wiring for E023 (idea preserved today; never started).

## d) TOTALLY FUCKED UP

Nothing catastrophic — no data loss (all deleted rows live in git history
and their shipped work is CHANGELOG-recorded). Honest near-misses, all
caught within the session:

1. **My initial inventory missed the blockquote checkbox form** (`> - [x]`).
   The scanner regex was `^\s*- \[x\]`; the one blockquote instance survived
   until the final residue grep caught it. If that grep had been skipped,
   a done marker would silently remain. Lesson recorded: checkbox scans
   must include blockquote prefixes.
2. **My deletion broke two cross-references** in surviving rows ("receipts
   above", "noted in the row above"). One was caught in the planned
   post-sweep check; the second only surfaced while writing this report's
   self-review. Both fixed, but the class is real: **bulk deletion needs a
   built-in dangling-reference assertion**, not a manual grep afterward.
3. **False-zero greps**: the first coverage batch used `\|` escaping
   (literal pipe in rg), silently reporting 0 hits for patterns like
   `StateVersion\|...` — several rows would have been misjudged as
   unrecorded. Caught by cross-checking "trust wave II" section content;
   re-run with proper alternation.
4. **Two self-inflicted typos** (lost trailing newline; stray `)` in the
   header note) — both caught and fixed by re-reads.
5. **Pre-existing, not mine, but ugly**: the M4–M7 block existed TWICE,
   byte-identical (L218–237 + L238–252). Some earlier session/daemon pass
   duplicated a whole block and three docs-health passes read past it.
   Root cause uninvestigated.

## e) WHAT WE SHOULD IMPROVE

1. **A `check-todo-list-hygiene` gate** (the repo's own pattern: what
   matters gets a gate). It should fail on: any `- [x]`/`> - [x]` row;
   byte-identical duplicate blocks (hash normalized block text); and —
   cheapest of all — any `above`/`below` receipt-pointer in TODO_LIST whose
   target phrase cannot be found within N lines. This session produced the
   exact fault inventory the gate would encode (114-row debt, 1 duplicate,
   2 broken pointers, 1 blockquote survivor).
2. **Deletion scripts must carry their own safety assertions**: after
   removing blocks, assert (a) zero checkbox matches in ALL forms, (b)
   dangling-pointer scan clean, (c) no setext hazards introduced, (d)
   trailing newline preserved. I did all four manually, sequentially, and
   still missed one until the report forced a re-sweep.
3. **Untagged-work pairing**: when CHANGELOG `[Unreleased]` notes point at
   TODO_LIST rows, link by stable anchor/row-name, not "the row above".
   The E023 pair is the current instance; make it the pattern.
4. **Drain `[Unreleased]`**: it now carries ~29 Added + 8 Changed entries
   across multiple feature trains (testing wave, architecture redaction,
   compat policy, E023…). It is functioning as a second backlog; the
   pending tag waves should cut it down.
5. **Stale-premise sweep as a docs-health VERIFY leg**: "blocked on tag X"
   rows can be checked mechanically against `versions.json`/tags. One
   instance found today by eye; the pattern generalizes.
6. **Multi-session collision hygiene**: a parallel session's report file
   appeared mid-session (untracked → daemon-committed while I worked).
   The claims-ledger convention (rowed in ROADMAP ideas) is still not
   practiced in the two files I touched; a one-line "claimed <file> <time>"
   note would have de-tangled it.
7. **The docs-health "done = delete" rule needs the receipt question
   answered per row**: most deleted rows' receipts were already in
   CHANGELOG, but four minor clusters now live ONLY in git history +
   archived plans. Either accept that (my current judgment) or add a
   distilled-archive step to the deletion procedure.

## f) Next — up to 50, impact-sorted (BRAINSTORM, not commitments; most are ROADMAP fuel; provenance marked)

**Direct follow-ups from this session [S]:**
1. [S] Cut `cmd/cqrs-lint` v4.19.0 carrying E023 (rel-tests +
   `check-example-standalone` first, per the row).
2. [S] Bump bank-sync's cqrs-lint pin to that tag + re-verify its
   `cqrs-lint-disable-audit` marker list in the same change (consumer repo).
3. [S] Run the cqrs-htmx test battery against system/v4.15.4 pins → close
   C03 (TODO_LIST:137).
4. [S] Run `nix fmt --fail-on-change` (or scoped dprint) over my two touched
   files → close the formatter-verification gap.
5. [S] Re-run the docs gates once (`check-md-go`, doc-check) → resolves the
   C04/F41 micro-tail either way.
6. [S] Owner: `systemctl --user enable --now go-cqrs-nightly-lint.timer`.
7. [S] Refresh the stale "Post-tag adoption tail" row (blockers shipped
   10-12): bank-sync snapshot patch, DiscordSync workaround drop, cqrs-htmx
   UpcasterRegistry collapse.
8. [S] Build the `check-todo-list-hygiene` gate (spec in (e)-1).
9. [S] HARVEST this report's (f) list into TODO_LIST (docs-health rules).
10. [S] Re-arm the composed verify chain in a quiet window; record the dated
    receipt on the surviving 🔥 row.

**E023/satellite-SQL thread [S]:**
11. Decide whether E023 should grade consumers (scorecard wiring) — the
    idea now lives in TODO_LIST:468.
12. Re-run the in-repo self-lint after the tag: the 3 deliberate
    middleware/projectionhost findings should stay visible-by-design.
13. Consider a consumer-facing note in the error-taxonomy or migration docs
    about the E023 advisory for heavy-SQL adopters (bank-sync shape).

**Docs/truth [S+M]:**
14. Annotate/harvest the parallel session's committed rawquery report
    (`docs/status/2026-11-03_19-09_*`) per docs-health (next-items →
    TODO_LIST before session end).
15. Full stale-premise sweep over surviving open rows ("blocked on tag X"
    vs versions.json — mechanizable).
16. Verify the CHANGELOG ↔ TODO_LIST E023 pairing survives the v4.19.0 cut
    (entry must move to the wave section at tag time).
17. Root-cause the M4–M7 duplicate block (which pass wrote it twice).

**Inherited high-value rows seen during the sweep [I] (already rowed —
listing here so this report is a self-contained map; do NOT re-row):**
18. [I] A9 keycodec-v2 engine migration wave (pebble reference + badger +
    bbolt + legacy stores; owner-confirmed as own wave).
19. [I] Owner question: deployed-DB census (sticky-key fallback vs re-key).
20. [I] `check-duplication` RED on HEAD: ~190 novel groups, ~180 ADR-0152
    mirror twins (one-campaign decision: directives vs end-of-campaign
    re-pin).
21. [I] MarkDemo untagged-API incident: cut the next catalog train;
    heal `example/mesh-demo` GOWORK=off build.
22. [I] Gate it: build `examplePaths` with GOWORK=off against the pinned
    graph (would have caught MarkDemo at commit time).
23. [I] The composed-proof re-arm (same as 10 — keep one row, dedupe on
    harvest).
24. [I] Testing-plan tail: T10 mutation gate, T13 synctest wave, T14 planner
    model-based test, T15 differential engine harness, T16 behaviour-test
    half, T17 executable-examples leg (= 22), T18 tail-latency matrix, T19
    heap/goroutine/disk budgets, T20 crash-consistency, T22 broker fault
    matrix, T24 owner family decision (signing Corruption vs Rejection +
    canceled-context writes), T25 companion-suite remainder.
25. [I] V5 schema-evolution follow-ups: compat-gate subject decision;
    `kv` durable-backing owner decision (engine Map adapter vs cache-only).
26. [I] Consumer-audit asks: catalog `FactoryFromSystem`-analog from
    DomainConfig; `scenario.WhenVersioned` — wait, that SHIPPED (verified in
    trust wave II); the remaining asks are catalog-Registry-derivation,
    cqrs-lint 2.0→7.4s regression investigation, fleet-census case studies.
27. [I] Wave-C delta audit tails (middleware/command/query/otel/id/
    scheduling content deltas → minor/patch sweep).
28. [I] #58 owner verdict: pre-system journal → system.New adoption path
    (DeploymentConfig store override vs migration helper).
29. [I] Fleet census: invite the five 2026-10-24 auditors to contribute case
    studies.
30. [I] EventCatalog: flow-id collision rename next mesh-demo touch;
    T15.2/T18 hub responses (= c-4).

**Process/tooling ideas surfaced by this session [P]:**
31. [P] Stale-premise detector as a script (see (e)-5, 15).
32. [P] Deletion-procedure checklist doc in docs-health terms (the four
    assertions from (e)-2).
33. [P] Consider `done`-row distillation policy (see (e)-7).
34. [P] CHANGELOG anchor-linking convention for TODO_LIST cross-refs.
35. [P] Claims-ledger one-liner for parallel sessions touching shared docs.
36. [P] Add blockquote-form checkboxes to any checkbox scan (the scanner
    lesson from (d)-1).
37. [P] A "receipts-only-in-git-history" disclosure line when deleting done
    rows whose receipts never made CHANGELOG (four clusters today).
38. [P] TODO_LIST size: 2524 lines / ~90 open rows — consider a section
    quota or auto-archive threshold in the hygiene gate (brainstorm).
39. [P] The `#verify` exclusivity rule vs the daemon: the composed chain
    keeps losing quiet windows to daemon/census churn (10-23/10-24
    receipts) — worth a dedicated quiet-window scheduling fix.
40. [P] When a done row cites an archived plan as its only receipt, link
    the plan path in the CHANGELOG entry at cut time (preserves
    provenance through wave-section churn).

**Bigger arcs (context only, not this session's scope) [B]:**
41. [B] v5 deletion cascade slices + the cut-order checklist (v5-cut-
    readiness remains the authority).
42. [B] E023 + C044 + rawquery-warning converge into one "modernization
    teeth" story for the next cqrs-lint minor.
43. [B] Turso platform-libs flip (open 🔥 row; blocked upstream).
44. [B] Dgraph integration story for example/graph-native.
45. [B] metaengine package-split (ADR-0162) execution when the review
    ratchet bites.
46. [B] Journal dump⇄load round-trip (testfixtures-inspired 🔥 row).
47. [B] Fleet v5 migration order: cqrs-htmx + go-appkit first (ADR-0152).
48. [B] `systemscenario` absorbs scenario/v4 at v5 (ADR-0153 note).
49. [B] Journal-tailed deriver host (ADR-0154 direction).
50. [B] The Declined section stays a do-not-re-litigate guard — nothing
    this session changes it.

## g) Questions I cannot answer myself

1. **E023 release timing** — cut `cmd/cqrs-lint` v4.19.0 NOW (it is a
   warn-first info advisory, zero gate risk, and bank-sync is blind until
   it lands), or hold it for the next broader cqrs-lint wave so the tag
   also drains the `[Unreleased]` pool? I can prepare either; only you
   know whether an isolated release now is worth the train overhead.
2. **Receipt preservation policy** — when deleting done rows, is
   "git history + CHANGELOG + archived plans" an acceptable receipt home,
   or do you want a distilled archive (one file, one line per deleted row)
   appended to `docs/status/archived/`-style storage? Today I chose the
   former for 110 rows and it cannot be undone cheaply once the daemon
   commits.
3. **The hygiene gate's teeth** — should `check-todo-list-hygiene` be
   warn-first (flake app, `#verify` leg, like the other doc gates) or
   hard-CI from day one? The 114-row debt says hard, but a hard gate on a
   file two concurrent sessions edit can red-leg CI on a formatting
   race; you own that tradeoff.

---

*Report by the 2026-11-03 TODO_LIST done-row sweep session. Point-in-time
snapshot; stale the moment TODO_LIST moves. Section (f) is HARVEST input.*
