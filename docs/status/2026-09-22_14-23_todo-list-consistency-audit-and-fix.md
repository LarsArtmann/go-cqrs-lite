# TODO_LIST.md Consistency Audit + Fix — Full Status Report

> **Session:** 2026-09-22 ~11:30–14:25 · **Scope:** the user's two asks — (1) "Is
> TODO_LIST.md consistent with itself?", (2) "Fix?" — plus this mandated status
> report. Everything below derives from THIS session's run and what it
> directly surfaced. No unrelated repo research was performed.
> **Format note:** written as `.md` per the owner's explicit instruction
> (status-report skill default is HTML; override flagged per skill contract).

**Inputs:** TODO_LIST.md (1406 lines pre-fix → 1231 post-fix), CHANGELOG.md,
`docs/planning/2026-09-21_direction-ruling-evidence-and-decision-memo.md`,
`docs/planning/2026-09-22_01-25_SUPERB-unblock-prove-deliver-pareto-plan.md`,
`docs/status/2026-09-22_01-20_docs-health-eleventh-pass-full-audit.md`,
`docs/benchmarks/2026-09-20-21_t18b-record.md`, flake.nix, ci.yml, git history.

**Context:** a CONCURRENT session was live-editing TODO_LIST.md and code during
this whole fix (systemtest split, v5 E-items, DeferClose note). Its edits and
mine interleaved; the auto-commit daemon absorbed both into `chore:` commits.

---

## a) FULLY DONE

1. **Audit (ask #1):** 18 internal-consistency findings in TODO_LIST.md,
   classified: 6 direct contradictions, 1 policy-vs-content violation, 8
   corrupted/fused text sites, 3 structural (index/section-order). Delivered
   as an in-chat report before any edit.
2. **ADR-0146 double-booking resolved.** Evidence: the SingleWriter-lease
   one-pager claimed 0146 first (commit `61ba5cb71`, 2026-09-21 15:05 wave;
   every archived report corroborates); the direction-ruling memo's later
   same-day claim was the outlier. Fix: lease KEEPS ADR-0146; direction
   ruling → **ADR-0147** (actual next free slot; `docs/adr/` stops at 0145).
   Applied in 4 files, 6 sites: TODO_LIST G-T01 row (+ `~~ADR-0141~~ →
   ADR-0147` lead strike), memo routing note + §c + matrix row (with dated
   re-slot rationale), 09-22 plan T25, 11th-pass report item 46 (dated inline
   correction, non-destructive).
3. **19 completed rows/fragments deleted**, every one with a receipt verified
   BEFORE deletion: Reconciliation-wave block + 3 orphan done-paragraphs +
   retract-check / tag-audit / buildinfo fragments, smoke-probes row,
   "92 releases created" fragment, ADR-0128 repin fragment, pin-sweep
   fragment, M16 §2.11 paragraph, Scan doc-lie orphan tail, `[x]` rows T16 /
   T14 / T03 / T15 / md-go self-test, go.work drift row (root-caused + gate
   shipped, CHANGELOG receipt), pre-commit env-hygiene row (go-env.sh shipped
   - force-sourced by the hook), go-env.sh helper row (shipped), Go-1.27
     "anchor" row, "More extended-review follow-ups — DONE" row (concurrent
     session's), Turso "Code guard follow-up" row (shipped as T16:
     `ErrGroupedViewBugRefused` + `WithKnownGroupedViewBug`), CV "Tag `system`"
     row (coeffect gate shipped in system/v4.8.0, 2026-09-19 train — verified in
     CHANGELOG §2788+).
4. **10 stale/partial rows restructured** to lead with the open remainder:
   Dead-path (only (b) eventtest-dead-tags remains; un-blocked), Post-wave
   hygiene (V006 golden only), T18b (chain landed green 2026-09-21 18:14 +
   retired → hardening MOOT; two owner questions routed to T25), Scan-default
   (RULED Option C first, v5-branch flip remains), md-go Gate wiring tail
   (nightly-gates.yml only), W0 (j) struck resolved (BuildFlow root cause),
   M20 (d) struck ruled, Push decision → Push-cadence ruling (premise cleared:
   **0 unpushed commits** verified via `git rev-list`), CV lease floating
   paragraph → proper `[ ] [BLOCKED]` row cross-linked to M20 (a), Feedback #4
   `[x]` → systemtest tag-wave tail (strip sibling replaces), check-release-
   scripts wiring row scoped to the still-missing self-tests (md-go /
   calibration-gate / check-go-version / check-golangci-hash already wired —
   verified against the flake list).
5. **Corrupted text repaired:** dangling "Original: one" completed; the fused
   "Stale-reference sweep" bullet lead restored; header blockquote broken
   sentence (stray `- drift gate…` fragment); orphan fragment mis-attached to
   the "Tag `system`" row removed; calibration row's receipt pointer
   re-anchored to the T18b canonical record.
6. **Structure fixed:** cqrs-htmx section moved above Declined (header's
   "Declined at the bottom" true again) + added to the section index; empty
   Vector-search section deleted (+ index entry); Legend documents the
   `~~strike~~` convention; `[BLOCKED]` style normalized to `- [ ]
   [BLOCKED]` (was 2 checkbox-less stragglers, now 30/30 uniform).
7. **Cross-references repointed:** "scoped-gates row above" (nonexistent —
   resolved by deleting its row), "metaengine wave (row above)" → correct
   section pointer, AggregateOn now jointly tracked between the Turso routing
   row and M20 (b)/(c), "billing-gated (row above)" + GATE STATUS "row above"
   verified directionally correct.
8. **CHANGELOG receipts added FIRST** (delete-second discipline): md-go
   `--self-test` entry (Added) and `metaengine.DeferClose` Deprecated note
   (Changed) — so the deleted TODO rows keep their receipts in the canonical
   place.
9. **Verification (green):** 27/27 relative links resolve; all 24 index
   anchors resolve (the 8 "bad" hits were my checker's em-dash collapsing —
   GitHub keeps double hyphens; index was already correct); zero `[x]`, zero
   orphan done-fragments, zero checkbox-less `[BLOCKED]`; ADR-0146 = lease
   only, coherent across all 4 files; `nix run .#check-md-go` GREEN (1461
   blocks valid, 104 baselined, no new errors); `scripts/check-changelog-symbols.sh`
   GREEN (25 pkg.Symbol citations honest against the API golden).

## b) PARTIALLY DONE

- **Verification breadth:** md-go gate + changelog-symbols + static
  link/anchor checks are green, but no composed `#verify` was run (scoped
  decision: markdown-only changes). The open "composed #verify re-record" row
  would cover this when it runs.
- **Concurrent-session merge:** handled, but reactively (see §d-2). Two
  `[x]` rows the sibling session landed mid-fix (systemtest, DeferClose) were
  caught only in the FINAL sweep, not at the first mtime warning.
- **Header passes-enumeration:** the doc's date list (~10 passes) vs
  "8th/11th pass" labels is countable-but-murky; flagged in the audit,
  deliberately not rewritten (needs the archived-pass lineage to recount
  honestly).
- **§f of this report** is NOT yet harvested into TODO_LIST/ROADMAP (owner
  said: report, then wait — HARVEST is the documented next step).

## c) NOT STARTED (deliberate, this session)

- Mechanical TODO_LIST consistency gate (checkbox states, index↔headings,
  done-fragment detector, ADR-slot collision check) — nothing protects the
  file today; every finding fixed here was human-only.
- ADR slot registry (a "next free / reserved" note in `docs/adr/README.md`).
- AGENTS.md Quick Reference census split-brain (says **96** go.mod files;
  module-map.md + the systemtest landing say **97**). Noticed during the
  session; not mine to land silently while a sibling session owns the
  systemtest work.
- Harvest of this report; docs-health "12th pass" lineage bookkeeping.

## d) TOTALLY FUCKED UP!

Nothing destroyed, nothing irreversible, no receipt-less deletion, no wrong
fact introduced (all four gates green). Honest near-misses, all recovered:

1. **Three edit round-trips failed on whitespace assumptions** (missing blank
   line before `FilterContains`; single-line 1.2k-char calibration row I
   wrapped in old_string; F153 context line that wasn't adjacent). Zero
   damage — each was retried from an exact view — but all three were
   avoidable by viewing the exact region first instead of trusting my earlier
   read's line wrapping.
2. **The mtime warning was under-reacted to.** The edit tool told me the file
   changed mid-session; I re-read 180 lines, saw identical content, and
   proceeded — the sibling session's edits were FURTHER DOWN. Correct move
   was an immediate full `git diff` re-baseline. Consequence: two done-rows
   slipped past the first deletion pass and were only caught in the final
   scan. The fix outcome is still correct; the process was lucky, not tight.
3. **Policy called unilaterally (flag for owner):** (i) enforced strict-DELETE
   of done rows per the header's own law, against the recent sessions'
   mark-`[x]` habit; (ii) un-blocked the Dead-path row (`[BLOCKED]` → `[ ]`)
   since its remainder is pure documentation; (iii) chose first-claimant
   chronology for the ADR re-slot. All three defensible, none owner-ratified
   (→ §g).

## e) WHAT WE SHOULD IMPROVE!

1. **Gate the TODO_LIST.** A `check-todo-list.sh --self-test` (checkbox
   grammar, index↔headings equality, done-fragment `[x]`/`done YYYY-MM-DD`
   detector, `[BLOCKED]` style, ADR-number collision grep repo-wide, "row
   above/below" direction sanity) wired into `#check-release-scripts` would
   have caught ~all 18 findings before they landed. Today only go-fence
   validity is gated (`#check-md-go`).
2. **ADR slots need a registry.** 0146 was double-booked within one day
   because "next free slot" claims live free-floating in 5+ docs. One
   "Reserved: …" table in `docs/adr/README.md`, claimed at one-pager time,
   ends the class.
3. **Concurrent-edit protocol:** mtime mismatch ⇒ FULL re-diff before the
   next edit batch, always. (This session did it reactively; make it
   reflexive — it also protects against daemon reformat waves.)
4. **Pick ONE done-row convention and encode it** (header delete-law vs
   strike-with-receipt habit). The drift between them is exactly what
   produced half the findings; put the winner in CONTRIBUTING + the gate.
5. **Receipt-first deletion SOP** (verify the CHANGELOG/archived receipt
   exists BEFORE deleting the row; add the entry if missing) — this session
   did it; the docs-health skill should say it explicitly so the next pass
   doesn't delete receipt-less.
6. **Authored commits for docs surgery:** the daemon absorbed a 4-file
   coordinated fix into `chore: auto-commit` blobs, so no reviewable unit
   exists at history level. AGENTS rule 4 says commit at phase boundaries for
   plan-driven work; the harness says never commit unprompted. Resolve the
   tension once (→ §g-3).
7. **Read-then-batch:** my 3 failed edits all came from batching on stale
   wrapping. The file had 1.2k-char single-line rows — view the exact target
   region first when old_string spans lines.

## f) Things to get done next (session-derived, impact-ordered)

1. Fix AGENTS.md Quick Reference census 96→97 + verify with
   `find . -name go.mod -not -path './vendor/*' | wc -l`.
2. Make the go.mod count script-derived in `check-canonical-facts.sh`
   (F69/M13 overlap — kills the last hand count).
3. Build `check-todo-list.sh` + `--self-test`; wire into `#check-release-scripts`.
4. ADR slot registry table in `docs/adr/README.md` (0146 reserved: lease;
   0147 reserved: direction ruling).
5. HARVEST this report's §f via docs-health (route items; drop resolved).
6. md-go wiring tail remainder: `check-md-go` into nightly-gates.yml (XS).
7. Dead-path (b): write the `event/v4/eventtest` dead-tags note into
   modules.md + pin-sweep note (XS, now unblocked).
8. T25 bundle: decide T18b deadline-lapse moot-or-rule (flagged "possibly
   moot" in the restructured row).
9. G-T02 ruling session: ratify ADR-0147 (draft exists — memo §6 scripts
   every outcome).
10. Encode the winning done-row convention in CONTRIBUTING + the new gate.
11. Header passes-enumeration recount in TODO_LIST (make 8th/11th countable
    from the listed dates).
12. md-go section header cleanup: "The harvested open tail: — source:" colon
    collision.
13. Normalize the exhaustruct repro-prep fragment (92-tag §) into its bullet
    body — it still trails the `_Effort_` line like the fragments I removed.
14. Convert remaining "row above/below" prose pointers in TODO_LIST to
    section-name pointers (survives future section moves like mine).
15. Cross-check the sibling session's v5 E-item landings regenerated the api
    golden in the same edit (contract 5; E1/E6/E7/E8/E11/E15 all touch
    exported surface).
16. Index-vs-section-count check into `check-canonical-facts.sh` (the 9th/
    10th-pass "index rot" hygiene row (a) — TODO_LIST is now index-complete,
    keep it that way mechanically).
17. Resolve the dangling "[x] rollout item" citation in the Green MySQL-VM
    row (which doc/row was it?).
18. Quiet-window composed `#verify` re-record (existing row) — also re-proves
    the docs chain end-to-end including this session's edits.
19. Weekly docs-health cadence owner decision (open row; this session is
    evidence FOR it: two sessions drifted the same file in one day).
20. Push-cadence ruling (owner) — answerable now that unpushed = 0.
21. Consider a wrap/line-length normalization pass for TODO_LIST megalines
    (the calibration row is one ~1.2k-char line; broke my first edit, will
    break others).
22. Sweep live (non-archived) docs/status reports >48h old → annotate/archive
    (11th-pass lineage; this report becomes the 12th data point).
23. Add "(ADR-0147)" to the lease one-pager's G-T01 adjacency note for
    cross-file clarity (it currently references the ruling number-free).
24. Verify no sibling consumer repos cite TODO row positions (they cite
    report §f numbers — quick grep, closes the move-risk I took).
25. Feed the "status-report default-format divergence" (skill says HTML,
    owner asked .md) back into the crush-config skill repo as a spec note.

## g) Questions I can NOT figure out myself

1. **ADR numbering:** ratify first-claimant (lease = ADR-0146, direction
   ruling = ADR-0147 — what I shipped), or do you prefer ONE bundled
   owner-decisions ADR covering the T25 rulings (the 11th-pass report's item
   46 read that way), renumbering again?
2. **Done-row convention:** strict DELETE (the header's written law — what I
   enforced) or strike-with-receipt (the recent sessions' habit)? One must
   win; the other needs its rows migrated.
3. **Authored commits:** should multi-file docs surgery like this session
   land as one authored commit at the phase boundary (readable history), or
   are daemon `chore:` blobs acceptable? If authored: say "commit" and I'll
   adopt the pattern going forward.

---

**Provenance:** files touched this session — TODO_LIST.md (25+ edits, net
−175 lines), CHANGELOG.md (+2 entries), direction memo (3 sites), 09-22 plan
(T25), 11th-pass report (1 annotation). Gates at close: `#check-md-go` GREEN,
`check-changelog-symbols.sh` GREEN (25 citations), link/anchor/checkbox scans
clean. Daemon commits `b8bc8c278`, `f2a3ba728`, … absorbed the work.

**Status: REPORT WRITTEN — WAITING FOR INSTRUCTIONS.**
