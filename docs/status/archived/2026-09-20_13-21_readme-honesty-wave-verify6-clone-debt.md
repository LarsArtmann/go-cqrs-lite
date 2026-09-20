> **RESOLVED — docs-health 9th pass (2026-09-20):** Closed: S03 GREEN ended §b (16-39 report); taskmanager shipped as v0.2.1 — the planned v4.1.1 was moot because the module path is suffix-less and the v0 line is the served line (16-39 §a5); the bbolt "missing doc.go" claim was self-corrected in 16-39 (godoc lives in preset.go — no gap). §f done or tracked; owner items ride the W3 bundle.

# Status Report 2026-09-20 13:21 — README-Honesty Wave Landed; Verify 6 Red at Pre-Existing W2 Clone Debt

> Session window 11:00→13:21 (continuation of the SUPERB publish-and-prove
> execution after the owner lifted the pause). Honest self-assessment
> included. Point-in-time snapshot; next session verifies every claim.

## a) FULLY DONE (this session, verified)

1. **T36e — `scripts/check-readme-deprecated.sh` gate system.** The handed-off
   draft was broken (fixture READMEs named `README-dirty.md` never matched the
   scanner's `find -name README.md`; `grep -q`+pipefail SIGPIPE made leg 1
   fail on success). Rebuilt: **7-leg self-test** (detect / subtree-scoping /
   qualified / clean / banner / deprecation-framed / wrapped-framing), every
   leg **mutation-tested** (7 mutations, each caught). Engine: package-scoped
   (qualified `pkg.Sym` matches only its declaring package; bare `Sym` only
   within the README's subtree), framing-aware (deprecated-disclosed citations
   are intentional), discovery anchored to Go's `// Deprecated:` line-start
   convention (killed the `event/command/query.AsRecord` false-positive class
   from mid-line field notes). Real run: clean, **baseline 0 entries**.
2. **T36f wiring** — both README gates (`-links`, `-deprecated`) wired into
   `.github/workflows/nightly-gates.yml` self-test-then-gate pattern; YAML
   validated by parse; `check-readme-links.sh` re-verified (673 targets, 0
   broken).
3. **ADR-0123 README sweep.** 10 package-level deprecation banners (stack +
   all 8 presets + storage/view — each mirroring an existing code-level
   doc.go/preset deprecation), 13 living-package READMEs migrated/framed
   (decider, event, id→`ParseStreamID`, listing→modern `Stream*` surface,
   metadata, metaengine, projection, graph, storage, system, benchkit,
   stack/bench, root README already framed). Deprecated-citation baseline:
   **37 → 0** — debt eliminated, nothing grandfathered.
4. **TODO row 634 parts c/e/f struck with evidence** (b and d remain open by
   design).
5. **Verify-5 lint-debt repair.** 30 real findings fixed: 24× godoclint
   duplicate-package-docs across 12 metaengine files (blank-line demotion,
   doc.go stays the canonical godoc), scheduling/engine (err113 →
   `ErrEngineNotDueClaimer` sentinel; exhaustruct ×2: `mu`, `Limit`;
   prealloc), cmd/doc-check (usetesting ×2 → `t.Chdir`, t.Parallel dropped
   with comment). **Standalone full lint sweep over all 95 modules: GREEN.**
   Both modules build + test green.
6. **API/docs hygiene.** api-stability golden regenerated (7426 exports);
   CHANGELOG `[Unreleased]` Added entries (sentinel + gate system);
   `check-changelog-symbols` ✓; `nix fmt` ✓; `check-error-taxonomy` ✓ (525
   codes); `check-arch` ✓; doc-check full invocation ✓ (1195 refs, 49 docs).
7. **W3 owner bundle** — `docs/status/2026-09-20_11-36_owner-bundle-w3.md`
   (5 decisions requested).
8. **Memory** — 2 new gotchas recorded (grep-q/pipefail SIGPIPE; GNU sed 4.10
   `"${n}i\\"` silent no-op); AGENTS Quick Reference gained the README-gates
   row. Pushed through `54e8682a7`(+); tree clean.

## b) PARTIALLY DONE

1. **Composed verify (S03).** Attempt 6 (12:18–12:30): verify-docs, module
   coverage, build, vet, test, race, **lint (the repaired phase — green)**,
   check-arch, modsums, lint-config, docserver-css ALL GREEN, 0 FAIL lines —
   then **RED at `Check Duplication`: "2 new clone group(s) introduced
   (baseline had 60)"**. The two groups are **pre-existing W2-wave engine
   code** no composed verify ever reached (attempts 4 and 5 both died at
   lint, before duplication): (1) `tx_isolation*_test.go` clones across
   duckdb/mysql/sqlite engines, (2) `meta_graph_edges` INSERT/DELETE SQL
   strings across duckdb/mysql/pg `graph*.go`. Today's lint repair let verify
   get 3 phases further and exposed it. Fix queued: `//art-dupl:accept`
   annotations (the contract-19 intentional-clone class) or consolidation,
   iterated to "0 new clone groups", then verify 7.
2. **S03 record** — blocked on verify green.

## c) NOT STARTED (queued behind the verify-green quiet window)

taskmanager v4.1.1 cut + `--smoke-all`; T13 load-sweep; T14 bench baseline
with provenance header; T15 `#verify-ci`; bbolt doc.go deprecation marker
(the 1 of 8 presets missing it); `docs/status/README.md` ledger entry;
push of the ~2 unpushed daemon commits.

## d) TOTALLY FUCKED UP (honesty ledger)

1. **Verify 6 was a 12-minute burn a 5-second pre-flight would have saved.**
   I pre-flighted error-taxonomy and check-arch but NOT the duplication gate
   — exactly the failure class the `can-run-composed-gate.sh` header warns
   about ("a 5-second check would have caught it"). The 46-minute quiet-gate
   wait made the burn feel worse.
2. **GNU sed quoting no-op** — 12-file sweep silently edited nothing while
   mtimes moved (looked done); caught only by re-verifying line content.
   Now a gotcha; never trust sed -i rc alone.
3. **The handed-off T36e self-test had never been run** (prior session
   shipped it "written"); first execution failed. Gate code is not done
   until its self-test runs green AND a mutation fails it.
4. **`grep -q` + pipefail SIGPIPE** turned a passing leg into a failure
   report — the kind of harness bug that quietly rots a gate's credibility.
5. **/tmp supervisor duplicated the repo's own canonical gate scripts**
   (`wait-for-quiet.sh`, `can-run-composed-gate.sh`) instead of calling them.
   Behavior-equivalent, but reinventing repo tooling is the clone-class sin
   this repo lints against.
6. **Single-file doc-check invocation** produced a false broken-anchor
   failure (`§2.11`) because the § fallback needs the full doc pool; full
   invocation green. Tool semantics learned the slow way.
7. **One sloppy edit hack** (a placeholder never-match old_string) instead
   of writing the real replacement — worked, but it's how mistakes hide.

## e) WHAT WE SHOULD IMPROVE

- **Pre-flight every cheap composed-gate phase before burning a verify run**
  (duplication, error-taxonomy, arch, modsums, lint-config are all <5 min).
- **Always call the repo's canonical gate scripts**; /tmp re-implementations
  drift.
- **Self-test + mutation-test before calling any gate shipped** (house rule
  that the handoff violated and this session paid for).
- **Verify phases after lint have now been dark since before W2** — every
  wave should end with a composed verify, not "next session will".

## f) NEXT (prioritized, ~30 of the possible 50)

1. Annotate/consolidate the 2 W2 clone groups → 0 new clone groups.
2. Verify 7 (canonical gate scripts) → record S03 (date, commit, durations).
3. taskmanager v4.1.1: `tag-release.sh` + `batch-release.sh --smoke-all`.
4. T13 `nix run .#load-sweep`.
5. T14 bench baseline + provenance header.
6. T15 `nix run .#verify-ci`.
7. bbolt doc.go deprecation marker (+ gate re-run).
8. Ledger entry + closing report; push.
9. Owner bundle decisions (5) from `2026-09-20_11-36_owner-bundle-w3.md`.
10. TODO 634b: READMEs into the doc-check gate (flake app/CI).
11. TODO 634d: quick-start drift-guard tests (5 modules).
12. Docs censuses (module-map 73/95 rowed → 95; FEATURES maturity matrix).
13. flake.nix go-version loud gate (owner question 4 dependency).
14. Verify-window advisory flock (owner question 5 dependency).
15. `readme_claims_test.go` ownership resolution (owner question 1).
16. iroh P99 follow-up probe (owner question 2).
    17–30. Standing TODO_LIST items (composed-verify row, W4+ plan slices,
    system/v4 follow-ups, metaengine live-latency doc sync, etc.).

## g) QUESTIONS ONLY THE OWNER CAN ANSWER

1. **Ratify the 10:31 repair ruling?** (restored `go 1.27.1` contract +
   kept the formatter-consistent markdown reformats that rode along with
   the unidentified concurrent editor's corruption).
2. **flake.nix go pin:** document `GOTOOLCHAIN=auto` as the contract plus a
   loud host-version gate (recommended), or explicitly pin/fetch go 1.27.1
   now, or wait for nixpkgs?
3. **`readme_claims_test.go`** (foreign, green, unowned): keep as a tracked
   gate under whose ownership, or remove?

---

_Evidence: `/tmp/verify-attempt6.log` (duplication findings),
`/tmp/lint-sweep.log` (95-module green), `scripts/readme-deprecated-baseline.txt`
(0 entries), TODO_LIST.md row 634 (strikes)._
