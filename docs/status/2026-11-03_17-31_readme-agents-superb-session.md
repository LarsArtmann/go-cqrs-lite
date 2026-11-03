# README + AGENTS.md Quality Wave — Session Status

**Date:** 2026-11-03 17:31 CET · **Session:** two commissioned waves ("Make the README superb", "Make the AGENTS.md superb") · **Basis:** repo at master under auto-commit daemon; parallel sessions were LIVE during the whole window (a 16:55 status report and a 16:22 report predate this session's end; one of them touched AGENTS.md line 90 mid-flight).

## Scope line

**Verified here (fresh commands, this session):**

- `bash scripts/check-readme-links.sh` — 932 targets / 110 files, 0 broken (re-run after final edit).
- `bash scripts/check-readme-deprecated.sh` — clean (0 grandfathered, 0 new); was 8 NEW failures in `scheduling/sqlstore/README.md` before my banner fix.
- `bash scripts/check-doc-links.sh` — 1257 targets / 611 files, 0 broken (covers AGENTS.md + README.md).
- `cmd/doc-check` (`GOWORK=off go run . ../../SKILL.md ../../.agents/skills/go-cqrs-lite/references/*.md ../../AGENTS.md`) — "All 1372 references valid across 57 packages", run AFTER the AGENTS.md restructure (no references lost in the Ground-Rules move).
- `bash scripts/check-canonical-facts.sh` — the legs my edits touch are green: go.mod count (103 cited un-backticked in README = 103 derived), recipes 106/106. Two PRE-EXISTING failures remain (see d.4).
- `bash scripts/check-versions-manifest.sh --update` then `--check` — fresh (112 module trains; system v4.16.0 was tagged but unrefreshed).
- `nix run .#check-md-go` — "All 1529 code blocks valid (84 skipped)", 105 baselined; run at 17:31 (the gate I had skipped during the work — see d.5).
- Quick-start compile + behavior proof: assembled the README's exact snippets into a temp `_test.go` inside `example/getting-started`, then `GOWORK=off go vet .` (clean) and `GOWORK=off go test -count=1 ./...` (ok, incl. the SQLite engine-swap test); temp file removed after.
- Every count I changed was re-derived, not remembered: toolchain via `nix develop -c go version` (go1.27.3); edges via `check-cycles --json` (510); coverage/runtime packages via baseline row counts (170/170); symbols via `docs/api_surface.txt` line count (6,422); sleeps via awk sum (182 sites / 85 files); frozen rows via `grep -vc '^#'` (1,088 / 540 / 116 / 21 / 61 / 8 / 89).

**NOT verified here:**

- `nix run .#verify` / `#verify-fast` — never run; the change set is docs + one generated manifest, and I ran every gate that police exactly those surfaces instead. Composed verify not attempted.
- Competitor-table claims (README "How it compares") — carried over with their August 2026 verification date and links; not re-checked against eventhorizon/watermill today.
- AGENTS.md "540 of **546**" errtax total and "**6,422**" symbol total — derived by proxy (row counts / file lines); the gates' own printed totals were not run. The parallel session's 14-47 report cites 6,421 symbols — a 1-off against my 6,422 that I did not reconcile.
- The two parallel-session status reports (16-22, 16-55) — not read; intentionally left unindexed (ownership question, g.3).

---

## a) FULLY DONE

1. **README.md rewritten** (363 → ~370 lines, same skeleton rebuilt). Changes: Quick Start now teaches the mandated `system.New` + `metaengine` fold path (ADR-0123) instead of the deprecated manual-decider + `stack/sqlite` path; alignment-verified two-column pipeline diagram; new "Engines" section (11 registry drivers + iroh, sourced from the generated `docs/engine-capabilities.md`); examples table (all 8; was a 4-example text wall); Key-modules table gains system/metaengine/systemscenario/scheduling/watermill; legacy v4 wiring consolidated under "Prefer explicit wiring?" with the v5 heads-up as ONE callout; comparison table's dead "one-call presets" row replaced by "Deployment-time engine choice" + "Cost-based query planner". Evidence: all gates above green; quick start compile- and behavior-verified (scope line).
2. **Stale-fact purge in README**: 98→103 go.mod (now written UN-backticked so `check-canonical-facts` enforces it), "3 third-party deps"→"only third-party production dependency is CBOR" (verified against `event/go.mod`: 10 first-party + 1 third-party production), coverage numbers re-derived (86–96%: event 90 / decider 96 / id 86 / dispatcher 88).
3. **Versions manifest regenerated** — `versions.json` + README fold-out refreshed (system v4.16.0 had drifted); `--check` green.
4. **`scheduling/sqlstore/README.md` deprecation banner added** — closed 8 NEW `check-readme-deprecated.sh` CI failures (the 2026-10-24 deprecation wave forgot the module README; gate skipped once bannered).
5. **AGENTS.md repaired + restructured**: the truncated "Versions gate" Quick-Reference row completed (was chopped mid-`[--check`, no description); TL;DR rules moved from line ~204 to the top as "## Ground Rules (read first)" (verbatim, zero concept loss); empty "Canonical Design Docs" heading folded into "### v2 Architecture — read before working on metaengine"; contract numbering normalized to continuous 1–30; 8 stale facts fixed (toolchain go1.27.1→go1.27.3 ×2, edges 506→510, coverage packages 168→170, runtime rows 168→170, symbols 6,334→6,422, "99-module workspace"→98, "99 rowed as of 2026-10-09"→98 non-testdata census-derived); AGENTS.md go.mod counts 101/102→103.
6. **Verified-accurate and deliberately untouched**: 89 coverage modules, 116 fuzz targets, 21 bench ceilings, 61 file-size + 8 package-surface baselined offenders, 1,088 frozen untested exports, 540 frozen errtax rows, 182 sleeps / 85 files, 106/106 recipes (gate-enforced).

## b) PARTIALLY DONE

1. **AGENTS.md numeric claims** — everything actionable fixed, but two totals rest on proxy derivations (errtax "546"; symbols "6,422" vs the parallel session's "6,421"). Effort S: run `nix run .#check-error-taxonomy-tests` and `#check-untested-exports`, read their printed totals, correct if needed.
2. **Canonical-facts gate** — my legs green; gate overall still RED on 2 pre-existing parallel-session items (unrowed `scratch/counterkey`; docs/status index missing 2 rows). Diagnosed, deliberately not touched (scope discipline + live parallel sessions).
3. **README quick-start compile pin** — verified transiently (temp test file, removed). No permanent pin exists yet: nothing will catch a future API change breaking the README fences. Effort S/M (see f.2).

## c) NOT STARTED

1. **docs-health HARVEST of this report's §f** into TODO_LIST/ROADMAP — deliberately deferred: user instructed report-then-wait.
2. **`system/` experimental-status framing** — FEATURES.md marks it 🧪 EXPERIMENTAL; my README presents it as THE path per ADR-0123/SKILL.md. The honesty tension was smoothed (heads-up box) rather than surfaced as an explicit caveat. Needs an owner call (g.1).
3. **module-map.md banner staleness** — "all 98 repo `go.mod` files are rowed" (backticked, gate-exempt) predates today's 103/98+scratch reality. Noticed, not fixed.
4. **Indexing the two parallel-session status reports** — gate stays red until someone does (g.3).
5. **Repo-wide toolchain-version sweep** — other docs (e.g. `docs/agents/gowork-modes.md`) may still cite go1.27.1; every nixpkgs bump re-drifts this class.

## d) TOTALLY FUCKED UP

1. **Zero authored commits for a commissioned two-file wave.** Ground Rule #4 says commit at phase boundaries if authored history is wanted; I never did, so the entire README + AGENTS.md wave is daemon-absorbed `chore:` history (interleaved with a parallel session's commits, making attribution archaeology worse). Severity: history/process, not code. Mitigation: this report + the diffs are the record.
2. **First write-collision retry was blind.** The `write` tool failed twice on "modified since read"; my first instinct was retry-verbatim instead of diff-first. The diff check before re-applying is what protected the parallel session's line-90 backtick edit from being clobbered — discipline on the second try, luck on the first.
3. **Temp verification file leaked into git history.** `example/getting-started/readme_verify_test.go` was daemon-committed before I trashed it — a chore: add + later delete in history. Should have anticipated the daemon (it had already raced me once).
4. **Shipped a misaligned diagram, then fixed it.** The first README pipeline diagram had inconsistent border columns (85/86/87); my own later check caught it. Correct order — generate programmatically + verify geometry BEFORE inserting — was learned mid-task, not applied upfront.
5. **`check-md-go` skipped during the work.** I reasoned the walk excludes root README/AGENTS (baseline has 0 README entries; old README had non-parsing fences unbaselined) but never ran the gate until this report forced it (green, 17:31). Probably-right reasoning carried as unverified for ~40 minutes — the exact "remembered vs fresh" failure the scope-line rule exists for.

## e) WHAT WE SHOULD IMPROVE

1. **Permanent compile pin for README fences** — the repo already has the pattern (`example/getting-started/docs_compile_test.go` pins `docs/getting-started.md`). The README's new quick start deserves the same; otherwise the compile-truth I established decays to a claim.
2. **Close the canonical-facts regex hole** — backticked `N go.mod` citations dodge the gate (AGENTS.md documents this by hand-fix instruction). A gate that strips backticks before matching kills the hole; documented holes re-drift.
3. **Derive-or-gate version claims** — "toolchain go1.27.3" will drift on the next nixpkgs bump like go1.27.1 did. Either pin the claim to a flake-derived check or stop citing patch versions in prose.
4. **Status-report index discipline** — three consecutive reports now unindexed at write time (2 parallel + this one until I indexed it). The writing session should index in the same edit; the gate failing afterward is the tell.
5. **Count framing split-brain I worsened** — README now says "90+ modules", "(103 go.mod files)", manifest says "112 module trains", AGENTS says "98 non-testdata modules". All true, all different bases; a reader must reconcile four framings. One canonical "how we count modules" sentence in AGENTS.md would fix it.
6. **Collision protocol for parallel sessions** — re-read + `git diff` before ANY retry after a "file modified" failure; never retry blind. Should be a Ground Rule, not a lesson.

## f) Next tasks

| # | Task | Impact | Effort | Category |
|---|------|--------|--------|----------|
| 1 | Run `#check-error-taxonomy-tests` + `#check-untested-exports`; reconcile 546-total and 6,421-vs-6,422 in AGENTS.md | Medium | S | Documentation |
| 2 | Add `readme_compile_test.go` in example/getting-started pinning the README Quick-Start fences (docs_compile_test.go pattern) | High | S/M | Quality |
| 3 | Owner decision: experimental caveat on README's system-first Quick Start (g.1) | High | S | Documentation |
| 4 | Index the two parallel-session status reports (after g.3 answer) | Medium | S | Documentation |
| 5 | Row or exclude `scratch/counterkey` in module-map census (canonical-facts green) | Medium | S | Cleanup |
| 6 | Fix module-map.md banner count ("all 98 repo go.mod files are rowed") | Medium | S | Documentation |
| 7 | Teach check-canonical-facts to catch backticked `N go.mod` citations | Medium | M | Quality |
| 8 | Sweep repo docs for toolchain patch-version citations (gowork-modes.md et al.); replace with minor-form or derived | Low | S | Documentation |
| 9 | Add "how this repo counts modules (go.mod / non-testdata / trains)" definition line to AGENTS.md | Medium | S | Documentation |
| 10 | Add collision protocol (diff-before-retry) to Ground Rules | Low | S | Process |
| 11 | CHANGELOG [Unreleased] entry for the README restructure? (docs-only; owner call whether user-visible) | Low | S | Documentation |
| 12 | Re-verify README comparison-table cells against eventhorizon/watermill current releases (Aug 2026 data) | Low | M | Documentation |
| 13 | Banner/pointer pass on example/readme-quickstart README → link the system-path getting-started | Low | S | Documentation |
| 14 | Add v5 pointer banner to docs/getting-started.md (manual-wiring tutorial) mirroring the stack-presets heads-up | Medium | S | Documentation |
| 15 | Update docs/status/README.md index header date "(2026-10-26)" | Low | S | Cleanup |
| 16 | Consider README "90+ modules" → single canonical framing after f.9 lands | Low | S | Documentation |
| 17 | Run composed `nix run .#verify` in a quiet window as the wave's final receipt | Medium | M | Quality |
| 18 | docs-health HARVEST of this report's §f into TODO_LIST (canonical next step for this file) | High | S | Process |

## g) Questions I cannot answer myself

1. **README positioning of `system/`:** FEATURES.md marks `system` 🧪 EXPERIMENTAL while ADR-0123 + SKILL.md hard-mandate it for all new apps; my README now teaches it as THE quick start with only the v5 heads-up (no experimental caveat). Which is the official public story — mandate (current README) or caveat?
   > **RESOLVED 2026-11-03 (same day, follow-up session):** owner answered "system/ is the future and I think not EXPERIMENTAL any more" — evidence audit agreed (fleet census 38 direct consumers; deriver-deadlock fix shipped 2026-10-10/ADR-0154; suite green incl. -race). Promoted: FEATURES.md header + 29 rows + matrix row → FULLY_FUNCTIONAL/Production, ADR-0123 Proposed → Accepted, SKILL.md/modules.md labels updated, CHANGELOG row added. The README's mandate framing now matches the labels.
2. **Permanent compile pin for the README quick start:** wanted? If yes, which harness — a `readme_compile_test.go` beside the existing docs-compile pattern (cheap), or extending the doc-check/recipes temp-module harness to README (stronger, ~100 s cold CI cost)?
3. **Ownership of the two unindexed parallel-session status reports** (16-22, 16-55; canonical-facts stays red until indexed): may I read + summarize + index them, or must their owning sessions do it?

---

*Point-in-time snapshot. §f feeds docs-health HARVEST; annotate, never rewrite, when superseded.*
