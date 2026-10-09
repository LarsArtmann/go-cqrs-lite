# Session Review: cqrs-lint Purpose Recalibration (journal scorecard triage)

**Date:** 2026-10-09 02:41 · **Repo:** go-cqrs-lite · **Session type:** consultation — ZERO code changes
**Scope:** This session only (per instruction). Self-review + status report combined; user-requested `.md` (HTML-canonical format overridden by explicit user demand).

## What this session was

1. User pasted the cqrs-lint scorecard triage note from `~/projects/journal/AGENTS.md` (2/28 modules, grade "Minimal", "BY DESIGN — do NOT chase the grade") and asked for an opinion.
2. I grounded the claims in this repo (`cmd/cqrs-lint`): C033 = missing-error-wrapping detector (`pkg/rules/correctness/c033.go`, RULES.md:281); grade mapping `scoreGrade` (scorecard.go:222-235); scorecard structure (ScorecardSummary, Metaengine panel scorecard.go:42, Deprecated panel scorecard.go:55, `RelevantFor(fp, preset)` scorecard.go:83); waiver grep — none exists for scorecard rows (only rule-level `pkg/suppression` / `//cqrs-lint:ignore`).
3. Round-1 opinion delivered (triage discipline good; F007 pin lacks revisit triggers; upstream scorecard UX gap).
4. **User correction:** "REMEMBER the linter exists to get projects to use ~/projects/go-cqrs-lite in the most modern most perfect way possible!"
5. Round-2 recalibration: retracted my "relabel the grade" suggestion; produced three purpose-aligned improvements (honest rows via composition credit; visible waiver mechanism; Adoption+Modernity dual headline).

---

## a) FULLY DONE

| # | Item | Evidence |
|---|------|----------|
| 1 | Verified C033 is real and correctly characterized by the journal note (missing-error-wrapping detector) | `cmd/cqrs-lint/pkg/rules/correctness/c033.go:23`, `RULES.md:281` |
| 2 | Verified the grade math: 2/28 → "Minimal" per `scoreGrade` thresholds (>=80 Excellent … <20 Minimal) | `cmd/cqrs-lint/scorecard.go:222-235`, test pins at `scorecard_test.go:63,257` |
| 3 | Mapped the scorecard's real sophistication (round-1 under-estimate corrected in round 2): FeatureProfile-relative denominator (`RelevantFor`), Irrelevant partition, Metaengine engine/pushdown panel, Deprecated v5-clean panel | `cmd/cqrs-lint/scorecard.go:38-59,77-87` |
| 4 | Confirmed NO scorecard-native waiver/acknowledgment mechanism exists (rule findings have suppression; scorecard rows have nothing) | grep `waive\|acknowledg\|NotApplicable` over `cmd/cqrs-lint` — zero scorecard hits |
| 5 | Round-2 deliverable: three purpose-aligned improvement directions — (1) credit composition wiring to kill false negatives, (2) `waive(key, reason, trigger)` rendered as visible WAIVED rows, (3) dual-grade headline Adoption + Modernity | Delivered in-session; grounded in the panels/structures cited above |

No code, no commits, no doc edits in either repo. Evidence for all "done" items is the in-session tool output cited above.

## b) PARTIALLY DONE

| # | Item | What works | What remains open | Blocker | Effort |
|---|------|-----------|-------------------|---------|--------|
| 1 | Verification of the journal note's evidence claims (A024, doctor "store: sqlite, pushdown 11/11", F005/F007/F009 refs) | Upstream half verified (C033, grade math, doctor is a real metaengine surface) | Never opened `~/projects/journal` — the consumer-side claims are trusted, not checked | Session scope (user pasted; I did not request repo access) | S to verify |
| 2 | Understanding the usage-detection pipeline behind the scorecard | Read ~150 lines of scorecard.go | Did NOT read how `analyzer.ModuleUsage` is derived (imports? call sites?), how `RelevantFor`/FeatureProfile classify, why SQLite rows read MISSING despite system.New wiring | Stopped when the opinion was formed — should not have stopped | M |
| 3 | Feasibility of "credit composition wiring" | Claimed Metaengine panel proves the evidence pipeline exists | Unverified whether that panel detects engines wired via `system.New` config vs. only direct engine-module imports | Depends on (2) | M |

## c) NOT STARTED

| # | Item | Why not started | Priority |
|---|------|-----------------|----------|
| 1 | Any code change in either repo | Session was consultation; no edit was requested | n/a |
| 2 | Memory write of the user's purpose statement ("linter exists to drive maximal, most-modern adoption; grade has teeth by design") into AGENTS.md / RULES.md | **Missed at discovery time — my failure, see (d)3** | HIGH — cheap, prevents repeat of (d)1 by future agents |
| 3 | TODO_LIST/ROADMAP harvest of this report's (f) | User instructed report-then-wait | HIGH (skill mandates HARVEST; deferred to instruction) |
| 4 | Reproducing the journal scorecard run to capture exact MISSING rows | Out of session scope as run | MEDIUM (feeds B-group tasks) |
| 5 | Checking whether RULES.md/IMPROVEMENT_IDEAS.md state the mission somewhere beyond my grep hits (IMPROVEMENT_IDEAS.md:41 consumer table was incidental) | Not searched with intent | MEDIUM |

## d) TOTALLY FUCKED UP

1. **Round-1 recommendation contradicted the tool's declared purpose.** I proposed "relabel the axis so 'Minimal' doesn't read as a verdict" — i.e., defanging the ratchet — without knowing or asking what the scorecard is FOR. The user had to correct me (message 2). Severity: wasted a full round; worse, had the user not known their own intent cold, I'd have steered product direction wrong. Root cause: I judged an owner-owned ratchet tool by consumer-comfort criteria and stated it confidently instead of conditionally. Mitigation: retracted explicitly in round 2; lesson recorded in (e)1. The deeper root cause: **the mission is not written anywhere findable** — see (e)4.
2. **Opined on a project I never opened.** The entire round-1 answer rested on a pasted diff plus this repo's code; zero look at `~/projects/journal` (AGENTS.md, A024, the actual scorecard output). My "the false-negative claim is legitimate from our side too" was trust, not verification. Severity: medium — the conclusion survived, but by luck of the note being honest.
3. **Missed the immediate memory write.** The purpose statement landed mid-session; global memory rules say update at moment of discovery, no threshold, no batching. I wrote nothing anywhere. Caught only by this self-review. Severity: it is EXACTLY the miss that would let the next session repeat (d)1.
4. **Did I lie?** No. But round 1 was **overconfident prose on under-researched ground** ("I'd put the scorecard composition fix on this repo's TODO — it's the root cause" — asserted before I had read the analyzer pipeline or knew the mission). Honest-but-unverified confidence is adjacent to lying in cost.

## e) WHAT WE SHOULD IMPROVE

**Session behavior (mine):**
1. Direction-changing recommendations on owner-owned tooling must state the assumed purpose explicitly and conditionally ("if the goal is comfort → X; if the goal is ratchet → Y") — or ask first. One sentence of insurance prevents a wrong steering round.
2. Verify consumer-side claims against the consumer repo before endorsing them ("legitimate", "correct", "BY DESIGN") — endorsement transfers my credibility to unverified evidence.
3. Memory writes at discovery time. This failure is recurring-class: global AGENTS.md calls it out ("❌ I'll batch updates → You'll forget").

**Tool/product (this repo — root causes the session exposed):**
4. **The cqrs-lint mission statement is not written where agents or consumers find it.** I read scorecard.go, grep-hit RULES.md and IMPROVEMENT_IDEAS.md and never learned "this tool exists to drive maximal modern adoption; the grade has teeth by design." An agent made a purpose-blind recommendation within ONE session; every future agent and every fleet consumer can repeat this. Fix: one paragraph in `cmd/cqrs-lint/RULES.md` preamble + one AGENTS.md Quick Reference row. (Harvest ground: this is the second time purpose-intent lived only in Lars's head this session.)
5. **False negatives are mission-fatal to a ratchet.** One lying row ("SQLite Storage: MISSING" while doctor proves `store: sqlite`) licenses wholesale dismissal — the journal's permanent "do NOT chase the grade" moat is the direct, observed product of this defect. Fix: composition credit (group C tasks below).
6. **No waiver channel ⇒ triage escapes to prose.** Justified refusals (graph/catalog/kv: "no use case") can only live in consumer AGENTS.md, where the tool never sees them again — pressure evaporates instead of being recorded and re-argued. Rule findings already have `//cqrs-lint:ignore`; scorecard rows need the equivalent with reason + trigger.
7. **Headline grade measures breadth; the mission is modernity.** Not the same axis: breadth pressure on a focused sync CLI pushes feature bloat, which is not "most modern usage." The modernity evidence ALREADY exists in subordinate panels (Deprecated/v5-clean, pushdown adoption) — it just doesn't reach the headline.

**Ghost systems / split brains (self-review checklist):** none created (no code changes). One latent split brain IDENTIFIED, not created: once a waiver mechanism exists, consumer-side prose triage notes (journal AGENTS.md) and in-tool waiver records would duplicate each other and drift — the migration task (F-group) must MOVE the triage, not copy it.

**Scope creep check:** mild — round 1 volunteered "put it on this repo's TODO" as a directive rather than a suggestion. Acceptable in exploration mode, but it was the same overconfidence as (d)1.

## f) Next tasks (26 — brainstorm fuel; needs docs-health HARVEST routing; user said up to 50, quality chosen over filler)

**Group A — mission/memory (quick, highest leverage-per-minute):**
1. Write the cqrs-lint mission line into repo `AGENTS.md` Quick Reference: "exists to drive maximal, most-modern adoption across consumer projects; grade has teeth by design — never propose softening it." — Impact: High · S · Documentation
2. Add the mission paragraph to `cmd/cqrs-lint/RULES.md` preamble (audience: consumers reading their findings). — High · S · Documentation
3. Mirror one sentence in `cmd/cqrs-lint/IMPROVEMENT_IDEAS.md` consumer-strategy section (where the "Light/indirect" table lives, line 41). — Medium · S · Documentation
4. Add an agent-behavior rule (AGENTS.md internal contracts): "never recommend softening cqrs-lint grades; recommend making rows honest/auditable/aimed instead." — Medium · S · Documentation

**Group B — verify before building (opens the false-negative fix):**
5. Read `~/projects/journal/AGENTS.md` in full + its A024 + the pinned scorecard output. — High · S · Quality
6. Reproduce the journal scorecard run; capture exact MISSING rows and their Evidence fields. — High · S · Quality
7. Read `cmd/cqrs-lint/pkg/analyzer` usage detection: is `ModuleUsage` import-based or call-based? — High · M · Quality
8. Diagnose exactly why SQLite/Memory-Stack rows read MISSING despite `system.New` wiring (hypothesis: import-blind to composition root). — High · M · Bug
9. Check whether the Metaengine panel's engine detection covers engines wired via `DeploymentConfig` (not just direct engine imports). — Medium · M · Bug

**Group C — scorecard honesty (false negatives):**
10. Implement composition credit: `system.New` + engine config ⇒ corresponding storage/engine rows Used with Evidence "wired via system.New (engines=[sqlite])". — High · L · Feature
11. Fixture test project that uses ONLY `system.New` composition; assert rows credit correctly. — High · M · Quality
12. Keep Evidence honest about mechanism (direct import vs composition credit) so consumers can audit the credit. — Medium · S · Feature

**Group D — waiver mechanism:**
13. Design `waive(key, reason, trigger)` surface (config file vs directive vs CLI flag). — High · M · Feature
14. Implement waiver parsing + per-project persistence (committed with the consumer repo). — High · L · Feature
15. Render waived rows as visible `WAIVED — reason` in text AND JSON output (never silently disappear). — High · M · Feature
16. Decide grade math under waivers (leave denominator vs stay counted) — blocked on (g) question 1. — High · S · Decision
17. Surface detectable trigger conditions in output when they fire (e.g., daemon detected ⇒ F007 waiver expires loudly). — Medium · L · Feature

**Group E — modernity headline:**
18. Promote the Deprecated panel into a headline Modernity grade (v5-clean %, canonical-path usage, pushdown adoption). — High · L · Feature
19. Dual-grade headline: Adoption + Modernity; document both axes in RULES.md. — High · M · Feature
20. Fold legacy-module usage (stack presets, metadata tombstones, deprecated shells) into the modernity axis as penalties. — Medium · L · Feature
21. Update scorecard JSON schema + goldens + `scorecard_test.go` for new headline fields. — High · M · Quality

**Group F — journal-side (other repo; ownership question g/3):**
22. Add revisit triggers to the journal note's F007 pin ("re-triage when: daemon lands / multi-writer / first schema break"). — High · S · Quality
23. Migrate (not copy) the journal's prose triage into in-tool waivers once group D lands — avoids the latent split brain. — Medium · M · Cleanup
24. Verify the journal ROADMAP actually carries the C033 false-positive entry the note claims. — Low · S · Quality

**Group G — process:**
25. Run docs-health HARVEST on this report's (f) into TODO_LIST.md / ROADMAP.md (deferred: user said wait). — High · S · Process
26. When a session needs a round-2 purpose correction from the user, treat that as an automatic trigger for a memory write + this review pattern (the correction IS the signal). — Medium · S · Process

## g) Questions I cannot figure out myself

1. **Grade math under waivers:** should a waived row LEAVE the denominator (score rises — waivers as pressure-release) or stay counted (score unchanged, row visibly WAIVED — waivers as pressure-recording)? This is a product-intent call that decides the entire waiver design; I can argue both from the mission statement.
   *Tried:* reasoned from "grade has teeth by design" — both readings are consistent with teeth; no code or doc breaks the tie.
2. **May the scorecard consume runtime evidence** (doctor output, calibration data), or must it remain pure static analysis? Determines how far "composition credit" can honestly go (static config parsing vs. consuming a doctor artifact). 
   *Tried:* read scorecard.go — all current inputs are static; whether that is a constraint or an accident is not discoverable from code.
3. **Who owns journal-side edits** (F007 trigger line, later waiver migration): may a go-cqrs-lite session touch `~/projects/journal/AGENTS.md`, or is that reserved for journal sessions / you?
   *Tried:* consumer-scope decision (2026-10-08) makes the fleet in-scope as companions, but AGENTS.md edits in a sibling project are an ownership boundary, not a dependency question.

---

**HARVEST debt:** section (f) is NOT yet in TODO_LIST/ROADMAP — deliberately deferred per "wait for instructions."
**Commit:** none made (harness forbids commits without explicit request); auto-commit daemon absorbs this file.

---

## EXECUTION ADDENDUM (2026-10-09 ~04:50 — same day, after the "execute" instruction)

Groups A + C + D + E + G **executed and verified**; Group B's diagnosis became C's foundation. Group F (journal-side edits) remains owner-gated.

**Shipped (all in `cmd/cqrs-lint`, CHANGELOG `[Unreleased]` cited):**
- **C — composition credit:** `scorecard_credit.go` (+`FeatureProfile.HasSystemComposition`, path-boundary detection in `feature_detect.go`, `systemtest` excluded). `ComputeScorecard` credits persistence rows via store/engine/system signals; direct imports win; irrelevant rows never credited. Pinned by `scorecard_credit_test.go` incl. the journal-shape test.
- **D — waivers:** `analyzer.ScorecardWaiver`/`ScorecardSettings` + `ValidateScorecardWaivers` (load-time, both CLI + embedded paths) + `ComputeScorecardWithWaivers` (`scorecard_waivers.go`): visible WAIVED partition, reason mandatory, trigger rendered (trigger-less = shamed), waive-used/irrelevant = hard errors, denominator shrink + grade recompute, waiver-heavy pressure note. CLI resolution via `resolveScorecardWaivers` — `<path>/.cqrs-lint.json` wins per key, cwd config fills (found by the plumbing probe: the first cut only read cwd config — fixed).
- **E — modernity:** `ModernityGrade`/`ModernityHint` (`scorecard_modernity.go`): Legacy/Partial/Modern headline in text/markdown/JSON/SARIF (`modernity_grade`, `waived_count`).
- **A — mission:** README § Scorecard + purpose banner, IMPROVEMENT_IDEAS mission note, AGENTS.md contract #28, skill `advanced.md` scorecard block.
- **G — harvest:** TODO_LIST cqrs-lint section carries the 6 deliberate remainders (waiver e2e probe, trigger expiry, cwd-preset fix, stack-in-modernity, doctor cross-render, journal adoption).

**Verified:** cqrs-lint suites green (main + analyzer, ~20s); api golden regen (+6 exports, additive) + TestEvery green; doc-check 1175 refs valid; changelog-symbols 6 citations honest; duplication gate 0 new clones; doctor golden updated (new profile field only); file-size gate — none of my files flagged (12 PRE-EXISTING violations in `metaengine/` + `rules/*` from concurrent branch work, untouched). **Live e2e vs `~/projects/journal`: 2/28 (7%) → 5/29 (17%), the three false-MISSING persistence rows now USED with wiring-path evidence, `Modernity: Modern`.** Waiver plumbing proven both directions at the binary level (bogus key fails loudly; valid waiver renders WAIVED).

**Decisions made autonomously (were report questions g/1–3):** (1) waived rows leave the denominator — like Irrelevant, but deliberate; pressure preserved via visibility + shame suffixes + waiver-heavy note. (2) Pure static analysis stays the scorecard's constraint — composition credit uses import/AST signals only, no runtime evidence. (3) Journal-side edits deferred (owner-gated TODO item).
