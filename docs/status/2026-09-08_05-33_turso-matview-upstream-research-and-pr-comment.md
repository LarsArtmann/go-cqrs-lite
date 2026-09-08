# Status Report: Turso Materialized Views — Upstream Research, PR #8257 Comment, Silent-Divergence Discovery

- **Date**: 2026-09-08 05:33 CEST
- **Session part**: continuation of the 2026-09-07 matview session (feature
  shipped + benched there; see
  `docs/status/2026-09-07_19-25_turso-materialized-views-operator-option.md`)
- **This session covered**: upstream deep research on tursodatabase/turso,
  issue-draft + PR-comment drafting (concise, disclosed, SHA-pinned),
  discovering and characterizing the **grouped-view silent-divergence bug**,
  posting the PR #8257 comment in the user's name, and hardening our own
  feature against the divergence (docs + Doctor warning + TODO handoffs)

---

## a) FULLY DONE

1. **Upstream deep research (tursodatabase/turso)** — issues AND PRs
   searched via `gh`: no duplicate of our COMMIT-abort bug; **PR #8257
   identified as the exact mechanism** (COMMIT's view-delta merge faults the
   persisted B-tree from disk → I/O yield while `commit_state == Ready` →
   re-entry misclassified); VDBE matview rewrite (#7942) closed unmerged;
   open `dbsp` correctness bugs catalogued (#6771, #4089, #8531, #8639,
   #8640, #8641). PR #8257 full body read; `llms.txt` read — corrected our
   terminology (Turso = ground-up SQLite rewrite; libSQL = legacy fork;
   tursogo = official Go SDK for the embedded database, purego).
2. **Driver release currency checked**: v0.7.2 → **v0.8.0-pre.8 is latest**;
   our repro re-run against it — **still broken** (12/12 at chunk 27000).
3. **The repro upgraded from "flaky" to deterministic**: 50k rows × 1
   grouped SUM view × 316 groups → **24/24 rounds across 2 fresh processes,
   always at exactly chunk 27000** (v0.7.2), 12/12 (v0.8.0-pre.8).
4. **NEW BUG DISCOVERED, characterized, verified — silent grouped-view
   divergence** (defects A+B):
   - A: grouped SUM views match the base table within ONE transaction;
     **from the second transaction on, groups spanning transactions lose
     part of their delta** (at 2k rows: 3 of 316 groups — c81/c184/c287 —
     each ~half wrong; constant 430.50 delta from ~1.3k rows). Onset
     bisected: between 1000 (1 transaction, exact) and 1100 rows.
   - B: at 27k rows the view **collapses** (496,034 vs true 1,308,429;
     reads succeed silently) — reproduced twice, identical numbers,
     independent of any commit failure.
   - **Scalar SUM views exact in every test** (1k and 27k).
   - Post-failure file state: base table correct (no partial persistence),
     but the file then rejects even single-row view-maintaining COMMITs.
5. **False claim caught in my own drafts before filing**: "data is never
   corrupted (clean rollback)" — replaced by verified statements after the
   rollback/divergence experiments; also fixed a 12/12-vs-3-rounds summary
   mismatch and a dead permalink (SHA not yet pushed).
6. **PR #8257 comment POSTED in the user's name**:
   https://github.com/tursodatabase/turso/pull/8257#issuecomment-5576078646
   — AI disclosure first line (GLM-5.3 via charmbracelet/crush, operated
   and reviewed by @LarsArtmann), scope question (creating-connection merge
   goes disk-backed at volume — does `commit_in_flight()` cover it?),
   verified data incl. the divergence aside, compact self-contained repro
   in a `<details>` block, SHA-pinned permalink.
7. **Posted bytes re-verified from GitHub**: comment body re-fetched via
   API, code fence extracted from those exact bytes, compiled, gofmt-clean,
   re-run → 3/3 at chunk 27000 on v0.8.0-pre.8. (CRLF from the API broke
   my first extraction — extraction bug, not the posted code.)
8. **In-repo hardening against the divergence** (consumer safety):
   - AGENTS.md: new top gotcha (grouped views silently wrong; scalar safe)
   - ADR-0135: consequences rewritten (grouped unsafe > one transaction's
     rows; scalar = recommended shape today)
   - Bench doc: CORRECTNESS WARNING section (grouped numbers measure
     speed, not currently-correct results >1k rows)
   - recipes §2.29: upstream correctness-limit notice
   - **`Store.Doctor` now emits a WARN line for grouped specs**
     (`materialized_view_doctor.go`) — metaengine full suite green, 0 lint
     issues
   - TODO_LIST.md: new "Turso materialized views — upstream handoffs"
     section (6 items incl. the BLOCKED standalone issue)
9. **Issue draft rewritten** as the standalone-issue body for defects A+B
   (`docs/research/2026-09-07_turso-go-ivm-commit-failure-issue-draft.md`):
   three-defect structure, evidence tables, per-group diff, expected
   behavior, honest pre-filing checklist (dup-search ✅, latest-release ✅).
10. **doc-check green** after all doc edits (1016 refs across 45 packages).

## b) PARTIALLY DONE

1. **Standalone upstream issue for defects A+B** — draft 100% ready and
   verified; **not filed** (awaiting user approval; external action).
2. **Comment permalink points at the older pushed SHA** (`1c9f3bf`) — the
   divergence-containing draft revision lives in an unpushed commit
   (`18b2c495c`); 11 local commits unpushed. Plan: after push, edit the
   comment link (GitHub comments are editable). Recorded in TODO_LIST.
3. **Doctor WARN for grouped specs ships untested** — added and full suite
   green, but no dedicated test pins the warning line (TODO_LIST item,
   Effort XS).
4. **Divergence characterization is two data points deep, not exhaustive** —
   onset bisected (1k–1.1k), per-group diff at 2k, collapse at 27k; did NOT
   test: DELETE/UPDATE-driven divergence (only INSERT OR REPLACE), AVG/
   COUNT/MIN/MAX grouped variants, group-count dependence of defect A's
   magnitude, whether defect A exists on the CLI (non-Go) builds.

## c) NOT STARTED

1. Remote Turso Cloud benchmark (libsql:// DSN) — untouched, needs
   credentials.
2. Mechanical code guard for grouped specs (validation refusal vs config
   flag vs status field) — decision deferred until upstream timeline known
   (TODO_LIST item).
3. turso-go release tracking loop (re-run repro per release; un-warn when
   fixed) — process defined in TODO_LIST, no automation yet.
4. Regression test pinning the Doctor WARN; regression gate extension for
   the matview bench; `check-coverage` run for the new code — all still
   open from the previous report.
5. Tag wave (pin bumps + replace strips for the four modules) — open.
6. In-repo `ivmrepro` test (build-tagged) — the repro lives in markdown
   drafts only; no executable copy under test tags in the repo.

## d) TOTALLY FUCKED UP

1. **My drafts contained a fabricated-adjacent claim**: "data is never
   corrupted (clean rollback)" — written from inference, never tested. The
   verification pass (triggered by the user's "EVERYTHING 100% verified?")
   not only found it false-adjacent but **uncovered the silent-divergence
   bug** — the most important finding of the session, discovered only
   because the user pushed on verification. Lesson re-earned: every claim
   in a public artifact must trace to an executed experiment.
2. **Posted with a stale permalink** by necessity: the pinned SHA
   (`1c9f3bf`) was the last PUSHED commit; the content the comment
   references lives unpushed (`18b2c495c`). Mitigated by making the comment
   self-contained and documenting the post-push re-link — but ideally the
   push would have preceded the post.
3. **Two extraction bugs of my own in one verification**: `sed`-extracted
   the wrong variant first (0 lines), then the CRLF-blind awk silently
   swallowed the closing fence and produced a "syntax error" that briefly
   looked like corruption of the posted code. Both mine; both caught within
   the same step. Slow, sloppy tooling around the one artifact that
   mattered most.
4. **Earlier in the session: the "happy to verify against your branch"
   offer** — an unfulfillable promise drafted into a public comment
   (branch testing requires building their Rust packaging pipeline against
   a hash-pinned embedded-lib loader). Caught by the user's "do it or leave
   it out" — removed.
5. **Wrong Crush repo URL** (`crusheio/crush` — doesn't exist) survived
   several draft rounds until the link-rot question surfaced it. A dead
   link under the user's name in a public repo, missed by me twice.

## e) WHAT WE SHOULD IMPROVE

1. **Verification-first writing for external artifacts**: draft claims as
   `TODO(verify)` until the experiment exists; the "clean rollback" line
   should never have been typed.
2. **Push-before-post policy**: external comments linking repo evidence
   should link a PUSHED SHA; coordinate the push (or inline everything)
   before posting.
3. **Byte-exact extraction discipline**: when verifying posted/quoted code,
   strip CRLF and anchor fences explicitly — build the extraction once,
   correctly, instead of improvising twice.
4. **Repro-as-code**: keep an executable, build-tagged copy of any repro we
   publish (e.g. `//go:build ivmrepro`) so "run what we posted" is one
   command, not a markdown-extraction exercise.
5. **Extend the divergence characterization before filing issue A+B**:
   UPDATE/DELETE-driven divergence and other aggregate fns strengthen (or
   correctly scope) the report — 30 minutes of probing now saves a
   maintainer round-trip later.
6. **Doctor warnings need pinning tests** the moment they're added — an
   unpinned operator-facing warning can silently disappear in a refactor.

## f) NEXT — up to 50 actionable items (priority order)

**Upstream (this feature's blockers)**
1. Get user approval and file the standalone issue for defects A+B (draft
   ready; everything below the first `---`).
2. Push the 11 local commits (user action), then edit the PR #8257 comment
   permalink to `18b2c495c`.
3. Probe UPDATE/DELETE-driven divergence (defect A scope) before filing.
4. Probe grouped COUNT/AVG/MIN/MAX divergence (is it SUM-specific?).
5. Probe defect A's group-count dependence (31 vs 99 vs 316 groups).
6. Check whether defect A reproduces via the `turso` CLI (non-Go path) —
   would rule the Go driver in/out for A (C is engine-side per #8257).
7. Watch #8257 for maintainer response; answer the scope question if they
   engage.
8. Track tursogo releases; re-run the repro per release (TODO_LIST item).
9. When fixed: remove grouped-view warnings (Doctor, recipes, AGENTS,
   bench doc) + un-skip ≥10k bench cases.
10. Consider a follow-up comment on #8257 linking the standalone issue once
    filed (cross-reference both directions).

**Our feature (safety + quality)**
11. Pin the Doctor WARN line with a dedicated test (XS).
12. Decide + implement the mechanical grouped-spec guard (validate/refuse
    vs `AllowGroupedViews` flag vs status-only) once upstream timeline is
    known.
13. Add `//go:build ivmrepro` executable repro under
    `metaengine/tursoengine/` mirroring the posted code.
14. Run `nix run .#verify` end-to-end (still never run this feature; only
    per-module gates) — close the stale-GREEN gap flagged in the prior
    report.
15. Run `nix run .#check-coverage`; record numbers for the new files.
16. Golden test (go-snaps) for `matViewDDL` output.
17. Property test: matview-served aggregate == base-table aggregate (would
    have caught nothing at ≤200 rows — extend with a second-tx case that
    WOULD catch defect A once upstream fixes it; guards the fix).
18. Add a regression test asserting the 2-tx divergence so the day upstream
    ships a fix, our test flips and tells us.
19. Bench-regression gate extension for the matview serving path.
20. COUNT_VIA_GROUPED bench case (derivation coverage).
21. Re-run the write bench on an idle machine; replace the mixed-load table
    in the bench doc with one clean run.
22. `MaterializedViewsReporter` → `GetEngineStats` programmatic surface.
23. Matview registrations in `system.Introspection()`.
24. Spec `String()`/`LogValue` for debug ergonomics.
25. Restart-with-removed-spec (orphan view lifecycle) test.
26. Two-engines-one-file-DSN sequential spec-set test.
27. Fuzz `Validate()`/`ViewName()` with rapid.
28. Concurrent-writer IVM soak under `-race`.
29. Filtered-view spec variant (v2 feature).
30. Planned-table matviews (v2; ordered with ApplyLayout + backfill).
31. Multi-aggregate/DISTINCT serving from views (v2).
32. Routing integration: cost model learns matview-covered shapes are
    O(1)/O(groups) so cross-engine routing prefers Turso for covered
    aggregates.
33. IVM write-amplification otel counter per view.
34. cqrs-lint rules: matview-on-unsupported-driver; matview-plus-planned-
    table staleness trap.
35. Example project for the YAML operator option end-to-end.

**Repo hygiene**
36. Tag wave: bump pins, strip replaces (sqliteengine/tursoengine/system).
37. cqrs-lint taskmanager golden refresh in the same wave (V006 coupling).
38. TODO_LIST: add the items from the 2026-09-07 status report's §f that
    are still missing there (this report supersedes that list — reconcile).
39. Archive the 2026-09-07 status report per docs-health conventions when
    the next docs pass runs.
40. docs-site page for the operator option + the correctness warning.
41. FAQ entry: "why is my grouped matview aggregate wrong?" → Doctor WARN +
    upstream issue link once filed.
42. DOMAIN_LANGUAGE.md entries: IVM, view-maintained write, materialized
    view acceleration.
43. Sweep the `.art-dupl-baseline.json` re-pin into a titled commit message
    if the daemon's heuristic commit bothers anyone (it's documented).
44. Foreign session: `cmd/cqrs-upgrade` LAYER/DEP_BUDGET entries still
    missing (check-arch still red on their files) + their committed 10.7 MB
    binary — flag to that session's owner.
45. Confirm `#verify-ci` includes the new test files (it should; one check).
46. Consider CI leg running the ivmrepro-tagged test against new tursogo
    pre-releases (early-warning for the fix landing).
47. Load-sweep before next `#verify` if timing paths get touched by
    follow-ups.
48. Re-pin the duplication baseline only via titled commits going forward
    (per the gotcha's spirit).
49. Update SKILL.md read-model matrix row with the grouped-view caveat
    (currently only recipes carry it).
50. Reconcile this report's §f with TODO_LIST so exactly one list is
    canonical (TODO_LIST wins; this report is point-in-time).

## g) QUESTIONS (cannot answer from the repo myself)

1. **File the standalone upstream issue for defects A+B now, or extend the
   characterization first** (UPDATE/DELETE + non-SUM aggregates, ~30–60 min
   of probing) so the report covers more ground in one shot?
2. **Push the 11 local commits yourself, or should anything in them be
   reviewed first?** (They include foreign-session work: `cmd/cqrs-upgrade`
   files and an otel change.) Once pushed, I'll edit the PR comment's
   permalink to the full-draft SHA.
3. **Doctor's grouped-view WARN is advisory-only.** Do you want the
   stronger guard (construction fails on `GroupBy` specs until upstream
   fixes — safest, breaks nothing that currently returns correct data), or
   keep it advisory with the docs doing the rest?
