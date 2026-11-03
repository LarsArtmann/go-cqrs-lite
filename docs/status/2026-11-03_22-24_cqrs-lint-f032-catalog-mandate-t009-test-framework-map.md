# Session Status: F032 Catalog Mandate + T009 Test-Framework Coverage Map

**Date:** 2026-11-03 22:24 CET
**Scope:** This session only — two turns: (1) "linter should error if a project uses go-cqrs-lite but not go-cqrs-lite/catalog" → F032 shipped + released; (2) "what about the testing/benchmarking frameworks?" → T009 shipped + coverage map made deliberate. A parallel session was concurrently active in the same repo (cqrs-lint v4.19.0 release, catalog/compat, scratch/counterkey, core/v5) — several findings below are about the seam between the two.
**Verdict up front:** Both rules shipped with the full gate ritual (detector, fixtures, registration, catalog, count pins, RULES.md, README, api golden, CHANGELOG, symbols gate) and the full cqrs-lint suite is green. F032 even rode the parallel session's `cmd/cqrs-lint/v4.19.0` tag — verified consistent inside the tag, surface by surface. T009 is complete but **UNRELEASED** (landed after that tag) and I forgot to row its release. The session also produced four self-inflicted wounds (all self-caught, none destructive) and left three repo-wide gates red from the PARALLEL session's in-flight work.

---

## d) TOTALLY FUCKED UP (lead with failures, per form)

1. **Empty-RULES.md committed by the daemon through my redirect (the big one).** I ran `go run . rules --markdown > RULES.md`; the shell truncates the file at redirect-open while `go run` compiles for seconds — the auto-commit daemon snapshotted the 0-line file (commit `966d62242`, 19:28:28). The working tree was later rewritten correctly (1783 lines) so the FINAL tree is right, but history carries a busted commit and anyone reading that commit sees a wiped RULES.md. Fix applied for T009 (generate to `/tmp` + `cp`), but only AFTER the incident — the generate-to-tracked-file pattern should have been daemon-safe from the start.
2. **Misread the golden-update contract three times (wasted roundtrips).** `CQRS_LINT_UPDATE_GOLDEN=1` on `TestIntegration_TaskmanagerExpectedFindings` only LOGS a paste block — I assumed it wrote the file, saw "ok", moved on; rg then showed F032 missing; my next TWO grep attempts to capture the log lines also failed (tab-prefixed t.Logf lines) before I dumped output to a file and read it. Meanwhile the sibling test (`TestLintExampleTaskmanager`) DOES self-write its golden — two different contracts in one file. Should have read the test source before running anything.
3. **`head -30` masked two real test failures (pipeline masking).** First full-suite run: `go test ./... | grep -Ev "^ok|..." | head -30` printed test-fixture noise and looked green; a re-run with a FAIL-only grep revealed `TestLintExampleTaskmanager` + `TestCatalogEveryGoWorkModuleCovered`. I declared victory on truncated output once — exactly the failure class the linter-building skill warns about.
4. **Two hook blocks on the same trap (`rg -rn`).** The host hook blocks `rg -rn` (reads like replace); I hit it twice in this session. The block message even says "third strike of this exact trap on this host."

Smaller self-inflicted wounds, self-caught: a `var _ = strings.TrimSpace` unused-import guard hack written then immediately removed from t009.go (should never have been typed); a `cd ../../..` that left the repo and produced a wall of git-usage noise; a wrong prediction that T009's literal builder call would be statically verified by the severity meta-test (the scanner skips the `testrules/` dir entirely — the suite corrected me, cost one roundtrip).

## a) FULLY DONE (verified green unless noted)

1. **F032 `no-catalog-import` — error severity, owner policy 2026-11-03** (`cmd/cqrs-lint/pkg/rules/adoption/f032.go`): fires once per project when any non-test file imports a go-cqrs-lite module but no file imports the catalog family (`catalog` any package, or `systemcatalog` which derives a catalog from declared truth), anchored at the first library import. Error severity gates the exit code (pinned contract: info/warning never gate, error does). Test-only usage exempt (no production domain → no documentation duty). 7 fixtures in `f032_test.go` (fire + severity/anchor pins; silenced by catalog / `catalog/v4/simple` / systemcatalog / catalog-only / no-usage / test-only). Self-lint exempt via consumer-coaching registration.
2. **F032 released in `cmd/cqrs-lint/v4.19.0`** — cut by the PARALLEL session at 20:40 after my work was daemon-absorbed. Verified inside the tag: detector, registration, meta_test 216, README 216, both taskmanager goldens, RULES.md, CHANGELOG bullet (moved to the wave section by their release flow). No re-release needed.
3. **T009 `hand-booted-system-in-tests`** (`cmd/cqrs-lint/pkg/rules/testrules/t009.go`): fires when a test file calls `system.New` and nothing imports `systemscenario` (`System`/`Adopt` both satisfy), anchored at the hand-boot call; consumer-side twin of the in-repo `check-systemscenario-adoption.sh` gate (script + reasoned allowlist owns self-lint; rule registered consumer-coaching to avoid double-reporting the 7 allowlisted modules). 6 fixtures incl. an Adopt-wrapper silencer and a qualifier-discrimination negative. Full suite green.
4. **Test-framework coverage map made deliberate** (CHANGELOG bullet): scenario = T001/T002, eventtest = T003, systemscenario = T009, benchkit/testutil = deliberately uncoached (EXPERIMENTAL/CLI-at-arm's-length; internal utility — both already excluded from the module catalog with reasons).
5. **Full gate ritual for both rules**: rule catalogs, `register_consumer.go`, meta_test 217, severity_meta T-loop 1→9, README counts + curated table rows, RULES.md regenerated (216→217; adoption 32, testing 9), api-stability golden (+1 mine, +1 parallel session's `AssertDeterministicMapEncoding`), `check-changelog-symbols` green (18 citations), gofmt clean on all touched Go files.
6. **Wiring pins updated on both taskmanager goldens** (F032: 1 — honest detector behavior; comment explains the TODO row). T009 silent there (taskmanager imports systemscenario) — no golden churn.
7. **TODO_LIST rows**: taskmanager systemcatalog wiring (S) with consumer guidance for fleets hitting F032.
8. **Stale-doc fix on sight**: `DefaultCatalog` comment claimed "6 core / 32 scored"; reality 7/36 (count test pins it) — corrected.

## b) PARTIALLY DONE

- **T009 release**: rule complete and green but UNRELEASED — v4.19.0 was cut before T009 landed, so no tag carries it (the exact E023-unreleased pattern from this morning). **I did NOT row this in TODO_LIST — that's the gap of the session** (see f).
- **`nix fmt` / treefmt verification**: I ran `gofmt` on my Go files only. My markdown edits (README, RULES.md, CHANGELOG) were never checked against treefmt's md formatting; CI's `nix fmt --fail-on-change` could still flag drift. Likewise `cmd/doc-check` was not run this session (AGENTS symbol-change ritual step 4) — I reasoned no skill reference lists rule counts, but did not verify.
- **Shared build cache**: corrupted mid-session (concurrent go invocations; stdlib cache entries vanished). I worked around it with a private `GOCACHE=/tmp/f032-gocache` — the shared cache root cause was neither diagnosed nor repaired, and the private cache (~GBs) was left in /tmp.

## c) NOT STARTED (noticed this session, deliberately not touched)

- **Three repo-wide gates are RED right now from the PARALLEL session's in-flight work** (verified, not mine, untouched per no-foreign-fix-forward): `check-package-surface` (correctness package grew 47→48 files), `check-untested-exports` (`catalog/compat.ErrBaselineFormatVersion` named by no test), and — earlier in the session — the module-catalog coverage test for `scratch/counterkey` (that one was green again by session end, they fixed it).
- **F032 fleet rollout**: bank-sync's pin-bump row (parallel session's) implicitly covers F032 since it shipped in v4.19.0 — but no explicit fleet-wide sweep exists (other consumers on older pins cannot see the error rule).
- **Scorecard wiring for F032**: the scorecard's `catalog` module row is breadth-coverage; nothing yet makes a MISSING catalog row carry grade consequences the way the owner might now want given F032 is a hard error. (Parallel session rowed the same idea for E023.)

## e) WHAT WE SHOULD IMPROVE (session-derived)

1. **Daemon-safe file generation**: every `cmd > tracked-file` step (RULES.md, api_surface.txt, goldens) is a truncation race with the auto-commit daemon. This session produced one busted commit. Improvement: a repo convention (or script wrapper) that generates to a temp file and `cp`s atomically — worth a row in `docs/agents/gotchas-tooling-build.md`.
2. **Golden-update UX**: two golden mechanisms in one test file (log-and-paste vs self-write) invited my misread. Improvement: make the integration golden self-write like its sibling, or make the failure message say "PASTE MANUALLY".
3. **Never conclude green from piped/truncated test output**: my `head -30` moment. Improvement: always end test pipes with a FAIL-only grep (this session's own correction) — or just read the exit code.
4. **F032 anchor determinism**: `firstLibraryImport` walks `ctx.GoFiles` in loader order; real loads are deterministic, but the property is untested for multi-file projects. A one-line sort by (Path, line) would make it contract instead of luck.
5. **T009 alias blindness**: `import syss ".../system/v4"; syss.New(...)` is a false negative (pkgIdent match, same limitation as T001). Type-info resolution (like F031's receiver check) is the known upgrade path.
6. **Rule-count commentary**: the README "N rules" headline, meta_test pin, RULES.md header, and CHANGELOG wording all re-pinned by hand each time — a tiny generator or single-source test would kill the four-way drift risk (the meta-test already catches README; the rest is manual).

## f) NEXT — up to 50, ordered by leverage (session-scoped)

**Release & rollout (do first):**
1. Row + cut `cmd/cqrs-lint ≥ v4.20.0` carrying T009 (tag-release.sh; rel-tests + example-standalone first; same discipline as the v4.19.0 row).
2. After tagging: bank-sync pin-bump row already exists — extend its text to note F032+T009 both ride ≥ v4.19.0/≥ v4.20.0 respectively.
3. Fleet census sweep: list consumers pinning cqrs-lint < v4.19.0 and row their bumps (they cannot see the catalog mandate at all).
4. Decide F032 rollout story for wrapper libs (see question 1 below) and, if carved, add the preset/config row.

**Example debt (makes the mandate credible):**
5. Wire `systemcatalog` into `example/taskmanager` (TODO row exists, S) — flagship example should satisfy its own error rule; regenerates both goldens to drop `F032: 1`.
6. Check `example/getting-started` and other examples for F032 hits; either wire catalog docs or banner why not.

**Rule quality (small, sharp):**
7. F032: sort anchor candidates by (path, position) — determinism contract.
8. T009: resolve the `system` qualifier through TypesInfo when available (F031 precedent) to kill alias false negatives.
9. F032: consider counting `systemcatalog` usage WITHOUT direct import as still-satisfied via call detection (`FromDomainConfig`/`DeriveInto`) — import check subsumes it today; verify no edge where the import is blank/aliased away.
10. Suppression integration coverage: add F032/T009 rows to `suppression_integration_test.go` (generic mechanism, but untested for these two).

**Session-left debt (mine):**
11. Run `nix fmt` and verify zero treefmt drift on the session's markdown edits (README/RULES.md/CHANGELOG/TODO_LIST).
12. Run `cmd/doc-check` over SKILL.md + references (confirm no rule-count claims broke).
13. Clean `/tmp/f032-gocache`; diagnose the shared-cache corruption window (two concurrent `go` runs — go claims cache concurrency safety; if reproducible it's worth a root-cause note).
14. Record the daemon-truncation lesson in `docs/agents/gotchas-tooling-build.md` (with commit `966d62242` as evidence).

**Parallel-session seam (flag, don't fix without coordination):**
15. `check-package-surface` red (correctness 47→48) — their re-pin or split decision.
16. `check-untested-exports` red (`ErrBaselineFormatVersion`) — their test or baseline row.
17. api_surface.txt now carries their `AssertDeterministicMapEncoding` — confirm their CHANGELOG cites it before the next release cut.

**Strategic (owner decisions, not tasks):**
18. Scorecard teeth for F032: should a MISSING catalog row cost ModernityGrade points now that absence is an error?
19. F002 (info, 3+ events, NewBuilder-call signal) — with F032 error at import level, consider upgrading F002 to warning (imported-but-unwired is sloppier than absent).
20. benchkit: if it ever leaves EXPERIMENTAL, revisit the deliberate non-coaching (module-catalog entry + maybe a T-rule).

## g) QUESTIONS (cannot be figured out from the repo)

1. **F032 for wrapper libraries:** the `library`/`library-framework` presets disable F002 (catalog coaching) for library-shaped consumers — F032 has NO preset carve-out, so a library that wraps go-cqrs-lite (cqrs-htmx class) now gets a CI-blocking error for not importing catalog. Intentional (catalog docs are everyone's duty) or should `library` presets disable/downgrade F032?
2. **T009 release timing:** cut `cmd/cqrs-lint/v4.20.0` now to get the systemscenario coaching into fleet hands, or hold and bundle with the next content wave (fewer tags, later teeth)?
3. **Taskmanager now or later:** wire `systemcatalog` into the flagship example this evening (S effort, goldens regenerate, mandate becomes self-consistent), or leave it rowed for a dedicated example-polish pass?
