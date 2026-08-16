# Status Report — Docs-Health Audit Wave E+F Annotation Sprint: 7 Files / ~330 Verdicts Landed (2026-08-16 17:38)

Session resuming `docs/planning/2026-08-16_13-40_SUPERB-DOCS-HEALTH-ANNOTATE-ARCHIVE-EXECUTION.md` (Waves E–G remainder). Prior session state: Waves A/B/C/D committed, 01-33 + 02-11 annotated, three open §g questions parked with standing defaults. This session executed Wave E fully and Wave F ~80%.

---

## a) FULLY DONE (verified this session)

1. **01-33 closed out (items 27 + 30 dispositioned)** — item 27 was a one-line code fix, not a
   harvest: added the why-comment above the genproto split-pin in `integration/go.mod` AND
   restored `replace metadata/v4 => ../metadata` (root-caused: `metadata/ids.go`
   `BrandedString` is unpublished — event's own replace doesn't cascade; integration standalone
   build went from RED to GREEN, `go mod tidy` + `go build` verified, comment survives tidy).
   Item 30 confirmed covered by TODO_LIST:690. Item 28 Won't-implement (YAGNI).
2. **02-16 annotated (13 verdicts)** — §f items 1–11 (the whole executed 22-tag chain: done
   verdicts citing tag names; 10/11 = partial with precise remaining-replace inventory), plus
   2 inline corrections: TL;DR "chain is blocked" strike-through → "executed 04:12–04:24",
   §b.1 "prepared, not executed" struck. f.12–f.25 left open by design (T2–T17 TODO refs).
3. **03-10 annotated (70 verdicts — largest file)** — §b 6, §c 14, §f all 50. Every verdict
   evidence-backed via fresh grep (batch_size.go byte-cap guard, WithBatchCommit,
   WithCopyAppend 1.41x/1.49x, pebble knobs, durability de-nesting, idempotencyTracker
   capacity, adopt API, BENCHMARKS.md pad decisions, etc.). Open items honestly marked
   (ScanSlice cap-64, relational one-tx-per-event, DuckDB view validation, planner COPY
   modeling).
4. **03-44 annotated (17 verdicts)** — §b 3, §c 4, §f 10. §g deliberately untouched (already
   carries `[RESOLVED]` markers from the session's own §h). Verified against §h truth
   (`5d66308c3` fixture fix, SOAK_SKIP_BOLT wiring, 1145s soak measurement).
5. **04-00 annotated (59 verdicts)** — §b 5 bullets, §c 4 bullets, §f all 50. Mapped the
   F-item ladder to ship hashes: `9541df676` (F55–F59), `921147a01` (turso DSN),
   `ca64b3517` (DecorateJournal), `8961bb6c3` (pin-drift), `30711eb79b` (conformance),
   `a1334d8c5` (seq-carrying), `342699d00` (pads). Items 14–18 honestly BLOCKED (tag-batch
   dependent), 48–50 honestly open (AGENTS MySQL-VM gotchas never added).
6. **04-24 annotated (28 verdicts + title correction)** — §b 8 (stranded-commit state verified
   against master via `merge-base --is-ancestor`), §e.4 20 resolved-only items annotated, 30
   left as-is (unknowns that stay unknown). §a.2 header corrected inline: "20 tags" → note
   "day total: 22 tags + 3 aux submodule tags" with timestamps from `for-each-ref`.
7. **07-12 annotated (56 verdicts)** — §b 3, §c 3, §f all 50. Verified kv.Cache shared-`*T`
   still open, `metaJSON, _ :=` residues in adapter_command/query_serial.go (2 sites), worktree
   `/tmp/cqrs-tagwt` STILL registered, v5 endgame plan exists only in stash.
8. **09-13 annotated (27 verdicts + 1 honesty correction)** — §b 2, §c 3, §f 22. **Inline
   correction**: the "ran green on memory/sqlite/pebble/bbolt/badger/iroh/pg/duckdb" claim
   struck with "CORRECTION (2026-08-16): iroh was later found RED by the conformance suite
   (11-33) and fixed (12-39) — 09:13's 'iroh green' was module-suite green, not conformance
   green."
9. **All annotations committed + pushed** — daemon commits swept them (verified present in
   HEAD; local == origin/master as of last check). Explicit-path discipline held: zero foreign
   cqrs-lint/metaengine WIP staged by me.

## b) PARTIALLY DONE

1. **10-51 annotation script written but NOT executed** — 50 §f verdicts + 5 §c verdicts + Q1/Q2
   resolution markers coded; script hit an assert on a line-wrapped anchor twice
   ("…it may hit the / same toolchain issue." spans lines), anchor fixed, re-run pending.
2. **11-00 read but not annotated** — §b 2, §c 4, §f 21 items pending verdicts.
3. **11-33, 12-39, 13-15** — not yet started (plan calls for minimal annotation: 11-33 §b/§f,
   12-39 §f 24 items, 13-15 references 12-39's list).

## c) NOT STARTED

1. **Wave G**: classify + archive (standing default: strict → nothing archivable), doc-check
   gate run, cross-file consistency sweep.
2. **Health report** (INLINE, Accuracy + Fitness with visible math) — the audit's closing
   deliverable.
3. **Final explicit-path commit + push** of anything the daemon hasn't swept by then.

## d) TOTALLY FUCKED UP (honest ledger — 3 script-corruption incidents, all caught)

1. **02-16 annotation CORRUPTED the file on first run** — the item-regex matched §a's numbered
   items (1–6) instead of §f's (no section scoping; `pattern.search(text)` searched the whole
   document). §a got strikethroughs citing the WRONG verdicts ("T8 COMPLETE" ← "tagged v4.5.0").
   Caught by diff inspection, reverted via `git restore` (my own broken edit), fixed with
   `text.index("## f) NEXT")` section slicing. The prior session's summary even warned about
   script failure modes — mine was a NEW one (section leakage, not derived-string replace).
2. **04-00 script failed 3× before success**: (i) `.*` under `re.S` swallowed the entire
   section inside a lazy group; (ii) sequential strike-then-rematch shifted bullet indices
   (struck bullet 0, re-found matches at wrong offsets → "MISSING bullet 2") — fixed by
   matching ALL bullets up-front, splicing reverse; (iii) §c bullets lack the `**` prefix —
   added `require_bold=False` mode. Total: ~4 wasted cycles on one file.
3. **10-51 anchor assert failed twice on line-wrap** — I wrote anchor strings from visual
   reading instead of grepping the exact wrapped text; both failures were pre-write (file
   never corrupted) but cost 2 cycles. Fix applied, run still owed.
4. **Minor**: daemon races twice invalidated Edit-tool reads mid-flight ("file has been
   modified since last read") — re-read + re-apply each time; also initially misjudged 01-33
   as under-annotated because I grepped for `done at` while the file uses `done —` (em-dash
   style) — one wasted verification cycle.

## e) WHAT WE SHOULD IMPROVE (repo-level, from this run)

1. **A shared annotation helper is overdue** — I now have 5 near-identical
   `annotate_items`/`strike_bullets` implementations across /tmp scripts, each debugged
   independently. The docs-health skill references `resolving-items.md` patterns but ships no
   tested script. One hardened `scripts/annotate-report.py` (section-scoped, wrap-safe
   anchors, all-matches-upfront) would delete this entire failure class.
2. **Section-scoping must be the default** in any numbered-item annotation — whole-document
   regex will always find SOME earlier list. My 02-16 corruption is the canonical example.
3. **Anchor strings for prose edits must be grepped, not typed** — line-wrapping defeats
   visually-copied anchors every time.
4. **Daemon-commit races are now the dominant interruption source** (5+ this session). The
   working protocol that emerged: explicit-path commits only, verify presence in HEAD after
   every daemon commit, never trust Edit-tool state across a race.
5. **`done —` vs `done at` marker styles diverge across already-annotated files** (01-33 uses
   both). Harmless for humans, but any future machine-sweep of annotation markers needs a
   union regex.

## f) Up to 50 things we should get done next (prioritized)

**Finish Wave F (immediate):**
1. Re-run the fixed `/tmp/annotate_1051.py` (10-51: 55 verdicts coded).
2. Annotate 11-00 (§b 2, §c 4, §f 21 — most verdicts already derivable from this session's
   evidence: hook fixed, golden regen, full verify GREEN, AGENTS replace-hygiene gotcha added
   by Wave B).
3. Annotate 11-33 §b (iroh-RED investigation outcome is in 12-39) + §f.
4. Annotate 12-39 §f (24 items — most closed by 13-15's GREEN gates).
5. Annotate 13-15 minimally (§e references 12-39's list; §d open decisions re-surface the
   tag-batch authorization).
6. Verify daemon swept 10-51–13-15 edits; explicit-path commit if not.

**Wave G:**
7. Classify all 19 dated 2026-08-16 files: ANNOTATE/ARCHIVE/SKIP/LEAVE ALONE per skill.
8. Archive decision per user's Q3 answer (strict default → likely zero archives).
9. Run doc-check gate: `cd cmd/doc-check && GOWORK=off go run -tags "goexperiment.jsonv2" .
   ../../SKILL.md ../../.agents/skills/go-cqrs-lite/references/*.md ../../AGENTS.md`.
10. Cross-file consistency sweep (TODO_LIST ↔ FEATURES ↔ CHANGELOG ↔ annotations; no
    completed item double-listed; marker-style union regex).
11. Emit the INLINE health report (Accuracy + Fitness, visible math, per-doc table).
12. Final commit + push (authorized by the 13-40 plan).

**Code debts this session verified STILL OPEN (for a code session, not docs):**
13. pebble/bbolt standalone RED until event re-tag ships adopt API (TODO_LIST 🔥).
14. `092b5e8a8`/`4907b6afc` still off master (retract directives not on master command/query).
15. projectionhost checkpoint-options tag + master pin bumps (command/query→metadata v4.5.x,
    stack/mysql→storage v4.7.x).
16. `/tmp/cqrs-tagwt` worktree still registered — `git worktree remove` cleanup.
17. `metaJSON, _ :=` silent-discard residues: `system/adapter_command_serial.go:26`,
    `adapter_query_serial.go:24` (same ADR-0126 class as the fixed event one).
18. ScanSlice cap-64 → RowCount() pre-size.
19. relational/projection.go one-tx-per-event batching.
20. kv.Cache shared `*T` → copy-on-write.
21. DuckDB view BatchSet validation never run.
22. Planner costs don't model COPY append path.
23. bbolt deserializeEvent benchmark (last extrapolated claim standing).
24. AGENTS.md MySQL-VM gotchas ×3 (port 33070 check, GOWORK=off-tag default path, shared-DB
    isolation) — from 04-00 f.48–50.
25. AGENTS.md retraction-incident note + CONTRIBUTING retract-and-republish pattern.
26. GitHub Releases for the 22-tag chain (needs gh auth).
27. v5 endgame plan sits in `stash@{0}` (detached-HEAD WIP) — never landed on master.
28. SESSION_MILESTONES revival-or-retire decision (TODO_LIST:690).
29. api-stability parse-skip loud-fail + BuildFlow gofmt gate (TODO_LIST standing).
30. SKILL FAQ retract recipe.

**Process (next docs session):**
31. Promote the annotation helper into `scripts/` with tests (e.1).
32. Normalize marker style going forward: `done at <hash>` for shippables, `done — <fact>` for
    verifications, keep both in the sweep regex.

## g) QUESTIONS (cannot resolve from the repo alone)

1. **Archive threshold (blocks Wave G step 8).** Strict skill default ("EVERY item resolved")
   yields ZERO archivable files — every 2026-08-16 report retains an open or foreign-tracked
   item. Pragmatic alternative: archive files where every item has a VERDICT (done/partial/
   open-with-annotation), since the annotation IS the resolution signal. Which do you want?
2. **Wave-4 tag batch authorization (blocks the 🔥 pebble/bbolt item).** TODO_LIST carries the
   exact batch (event re-tag w/ adopt API, metadata v4.5.1+ BrandedString, schema,
   projectionhost options, storage v4.7.2, metaengine + irohengine, + land the two stranded
   commits). Tags are user-authorized only: may I cut them in a follow-up session, or do you
   want the CHANGELOG entries reviewed first?
3. **Scope confirmation for the audit's finish line.** The 13-40 plan named 16 dated files;
   the directory now holds 19 (14-06/14-07/14-09 from concurrent sessions + 14-24 mine).
   Annotate the 3 concurrent-session reports too (their lanes' verdicts are largely derivable
   from HEAD), or leave concurrent-authored files to their own sessions and close the audit
   with the planned 16?

---

_Session artifacts: 8 annotation scripts under /tmp (`annotate_0216/0310/0344/0400/0424/0712/0913/1051.py`),
all verdicts evidence-backed by fresh grep/git queries at annotation time. Daemon swept all
completed work; nothing of mine is uncommitted at report time._
