# Status: Skill-Docs Backlog Execution — Self-Review Round 2

> **STATUS (docs-health pass 2026-09-16):** §f4/§f5/§f8/§f9/§f10 SHIPPED 2026-09-16 (struck inline) — the anchor/§/arity validation this session prototyped as throwaway scripts is now a permanent `cmd/doc-check` gate. §f1/§f2 (master CI diagnosis) were root-caused 2026-09-13 (magic-nix-cache throttling; see TODO_LIST CI triage row). §f13's checklist suggestion landed as the 2026-09-16 doc-check gates.

**Date:** 2026-09-13 09:17 CEST
**Session scope (this round):** Execute the backlog from
[`2026-09-13_08-47_skill-docs-audit-metaengine-goal-readiness.md`](2026-09-13_08-47_skill-docs-audit-metaengine-goal-readiness.md):
close the gate chain, finish the partial reads, harden the skill docs, harvest
TODO_LIST. Report covers this round's run only.
**Format:** `.md` per explicit user instruction (overrides HTML default; flagged).

**One-line verdict:** All 8 backlog items executed and verified; 9 more defects
found and fixed (including 2 caused by MY round-1 fixes). The round also proved
round-1's audit had a systematic blind spot — same defect classes existed in
files I had marked "audited".

---

## a) FULLY DONE (this round)

1. **evals/ verified safe** — § refs are §2.0–2.5 (all stable under renumbering);
   `trigger-eval-set.json` holds trigger queries with `should_trigger` flags and
   does NOT pin description text → my SKILL.md trigger edit is structurally safe.
2. **doc-assertions gate located and run** (`scripts/verify-docs.sh`): ALL PASS
   (build, CHANGELOG, module counts, license, ADR index, error families). It
   caught **my own 137→136 ADR off-by-one** (fixed; the gate's count wins).
3. **AGENTS.md § refs programmatically validated** — all valid post-renumbering.
4. **Exhaustive link + anchor crawl** (GitHub-exact slug rules): 2 REAL
   pre-existing broken TOC anchors found + fixed (advanced §6.8, faq eventtest).
5. **390-file fmt drift root-caused**: ADR-0128 go-codec sweep committed
   unformatted via auto-commit daemon; `nix fmt` normalization absorbed by the
   daemon; tree format-clean confirmed (`nix fmt` → 0 changed).
6. **100% read of all 7 skill docs** (advanced 747, faq 376, recipes 2,173 lines
   — all previously partial). Content quality holds end-to-end.
7. **Structure fixes**: advanced.md TOC +§6.15–6.19; orphan "failure modes" block
   → §6.16b; faq.md TOC +Turso-encryption; 11 unnumbered metaengine `###` →
   `####` subsections of §2.21b; Metadata-Serialization note → §2.21a.
8. **Round-1 misdiagnosis corrected**: "§6.19 ghost pointer" was WRONG — the
   Operator-Driven Layout Planning recipe EXISTS (case-sensitive grep false
   negative). advanced.md re-fixed properly; round-1 report annotated honestly.
9. **Skill quality**: SKILL.md triggers now name `metaengine`/`system`/
   `projectionhost` + goal phrases; core.md §0 carries the explicit north-star
   vision; axes-table deprecation markers added.
10. **All 3 example binaries run** (exit 0; metaengine-quickstart 4/4 demos
    incl. operator `cqrs.yaml` boot; outputs match docs).
11. **TODO_LIST harvested**: 5 novel items (CI-portable anchor/§ validation,
    arity spot-check, v5-story consolidation, example discoverability, `Infer`
    end-state), deduped against the parallel session's same-day batch and
    existing entries (turso/coeffects/compile-harness already covered).
12. **Gates at close, all green**: doc-check 1,055 refs / 47 pkgs ·
    doc-assertions ALL PASS · links/anchors/§refs valid · `nix fmt` clean ·
    tree fully absorbed (git status empty).
13. `scripts/check-doc-links.sh` confirmed file-links-only (anchor fragments
    stripped) → the anchor-validation TODO is NOT a dupe.

## b) PARTIALLY DONE

1. **CI validation:** master CI red since ≥06:30 — **pre-dates my session**
   (~08:30). The 07:00 failure log shows Magic Nix Cache throttling (HTTP 418)
   plus passing tests in the visible portion; root cause NOT diagnosed (out of
   session scope). My session's ~400-file commits have **no CI run yet**
   (daemon push timing unknown). "Is my tree CI-green?" is unanswered.
2. **Slugger validator:** took 3 iterations (underscores kept; inline-code
   content kept). The two anchor fixes shipped after iteration-1 rules —
   correct in hindsight, but I trusted tool negatives before validating the
   tool against known-good anchors.
3. **Crawl output polish:** final script still prints 2 known false positives
   (core.md inline-code generics) + 6 legit "§2.3 moved" pointers — dismissed
   in prose, not filtered in the tool. The TODO'd CI port must include the
   filtering (the existing check-doc-links.sh already models it).

## c) NOT STARTED

1. `nix run .#verify-fast` — deferred again (docs-only rationale); still the
   one canonical gate never run this session.
2. Master CI failure diagnosis (pre-existing, see b)1).
3. The productization TODOs (anchor gate, arity check) — filed, not built.

## d) TOTALLY FUCKED UP (own goals, honestly scored)

1. **Round-1 audit was not systematic across files.** I fixed recipes.md's TOC
   and duplicate section numbers, then STOPPED — advanced.md's TOC (missing
   5 sections + an orphan block), faq.md's TOC (missing 1), and 12 unnumbered
   recipes.md sections survived my "audit" of those same files. The defect
   classes were identical; I checked them in one file and declared victory.
   Round 2 found 4+ more defects in "done" files.
2. **My round-1 fix WAS a new defect:** "43 ADRs" → "137 ADRs" was still wrong
   (137 = `ls` count including README.md; canonical gate says 136). I
   introduced an off-by-one while fixing a staleness bug — caught only because
   round 2 ran the gate I should have run first.
3. **Round-1 misdiagnosis shipped as a fix** (§6.19): verified a negative
   ("recipe deleted") with a case-sensitive grep and rewrote a pointer that was
   pointing at real content. Root cause class: unexamined tool semantics on
   negative results.

## e) WHAT WE SHOULD IMPROVE

1. **Sweep the class, not the file**: on finding a defect class (dup numbers,
   TOC gaps, orphan sections), immediately run the same check on ALL sibling
   files. Audits are checklists per class, not per file.
2. **Negatives need the strictest tool**: case-insensitive search + canonical
   gates before declaring anything missing/deleted.
3. **Run the canonical gate before trusting hand counts** (136 vs 137).
4. **Validate validators** against a known-good corpus before trusting their
   negatives (the slugger iterations).
5. **Watch CI when mutating the tree** — `gh run list` was one command away all
   session; I only ran it during this self-review (and found pre-existing red).
6. **Productize throwaway verification scripts** — this round's crawl found 2
   real defects no existing gate catches; that logic dies with the session
   unless ported (now TODO'd).

## f) Next things (session-fallout, impact-sorted; real items only)

1. Diagnose the pre-existing master CI failure (runs 06:30/06:47/07:00) —
   cache-throttle flake vs real test failure; re-run if infra.
2. Confirm CI green on this session's commits once the daemon pushes (fmt gate
   - docs + ~400 formatted files all ride together).
3. Run `nix run .#verify-fast` before the next release-adjacent merge.
4. ~~Port anchor + § cross-ref validation into `scripts/check-doc-links.sh`
   (file-links-only today; slug rules + false-positive filtering documented in
   the TODO). _(Effort: S)_~~ done 2026-09-16 — ported into `cmd/doc-check` instead (Go, unit-tested, zero-warning-gated; runs in CI via the doc-check leg)
5. ~~doc-check arity spot-check for fenced-Go call shapes (both round-1 criticals
   passed symbol-level checks). _(Effort: M)_~~ done 2026-09-16 — shipped in cmd/doc-check (go/ast arity comparison + precision filters)
6. Generated TOCs: consider a script that emits the recipes TOC from headers
   (my session regen proves it's mechanical) — generated TOCs cannot drift.
7. Re-run the skill trigger evals after the frontmatter change (evals exist;
   runner not yet located) — verify pass-rate kept/improved.
8. ~~Consolidate the v5-deprecation story (6+ tellings → 1 canonical + pointers).~~ done 2026-09-16 — canonical in faq.md; others point at it
9. ~~Link `example/metaengine-quickstart` from README + metaengine module README.~~ done 2026-09-16
10. ~~Decide `metaengine.Infer` end-state (deprecate at v5 vs promote with story).~~ done 2026-09-16 — deprecated, removal at v5
11. Coordinate with the parallel session's TODO batch (compile-harness overlap)
    — one owner, not two.
12. CHANGELOG: decide whether consumer-visible doc/skill changes (trigger
    description) warrant `[Unreleased]` entries (likely no — not library API —
    but decide explicitly).
13. Add "audit meta-sections first" + "sweep the class across files" to the
    docs-health skill's VERIFY checklist (skill-meta improvement, your call).
14. If the daemon pushes rarely: consider pushing session-critical commits
    sooner so CI actually exercises them.

## g) Questions I cannot answer myself (max 3)

1. **Master CI is red since before this session** (06:30+, Magic Nix Cache
   throttle visible in the log). Diagnose-and-fix as the next task, or is this
   already known/owned (e.g., by the parallel session I saw harvesting
   TODO_LIST at 08:52)?
2. **Re-run the skill trigger evals** after my frontmatter trigger change, or
   leave as-is? I could not locate the eval runner this session (evals exist;
   no command found in flake/scripts).
3. **Generated vs validated TOCs**: make recipes TOC generation a checked-in
   script + CI gate (drift-proof), or keep hand-maintained TOCs with
   validation-only (anchor gate from f)4)?

---

**Gates at close:** doc-check ✓ (1,055 refs / 47 pkgs) · doc-assertions ✓ ALL
PASS · links+anchors+§refs ✓ · `nix fmt` ✓ (0 changed) · tree clean (daemon
absorbed) · `#verify-fast` ✗ deferred · CI on my commits ✗ no run yet (master
red pre-session).
