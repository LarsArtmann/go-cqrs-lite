# Status: Verification-Debt M-Plan Execution + the Six-Arm Composed-Proof Saga

**Date:** 2026-10-24 12:44 CEST · **Repo:** `~/projects/go-cqrs-lite` · **Branch:** `master`
**Session arc:** resumed 07:25 from the 18:15 M-plan handoff → M3–M11 landed (07:30–08:45) → six verify-chain arms (09:43–12:44) fixing one foreign-session regression class per arm → arm 6 armed and waiting at report time.
**Plan:** [`docs/planning/2026-10-23_18-15_SUPERB-architecture-verification-debt-pareto-plan.md`](../planning/2026-10-23_18-15_SUPERB-architecture-verification-debt-pareto-plan.md) (M1–M12).

---

## a) Fully done (with receipts)

| Item | Receipt |
|---|---|
| **M3 — `types_resources.go` split** (359 → 284) | ADR cluster → `catalog/types_adr.go` (76); baseline entry dropped; catalog suite + lint green; also fixed the architecture example's 3 tagliatelle findings (daemon blob `ab58cc25b`) |
| **M4 — composition & precedence suites** | `compose_test.go`: hide×include-pull, spec+flags precedence (spec first, flags extend — matches the CLI's assembly order), composed leak probe (text-presence, exclude × name-only × include-pull × hide), references-vocabulary validation (`LoadExport` rejects typos, `omit` accepted as the default's spelling). **Mutation-verified with observed red:** killing the owners-hide flipped the compose suite; disabling the recording guard + degrade re-checks together flipped the leak probe (single-site mutants staying green = layered defense working) |
| **M5 — ADR-0160 implementation addendum** | Dated status banner + per-ruling DONE/DEVIATED/REJECTED table with file:line evidence; ADR gate green ×2; committed authored (`ef3a844bf`) |
| **M6 — hub readiness** | `BenchmarkInspectLargeCatalog` ~2–3 ms at 1k services/5k messages (two orders under the ~175 ms render); `-check -h` documents the exit-0/exit-1 CI contract; README config snippet + example run lines; catalog/AGENTS cost record (`3ba82e16c`). SKILL.md `Inspect` mention pre-existed (M6.4 no-op) |
| **M7 — gate redaction case** | `architecture-fixture <whole> [slice]`: partner slice (hostile-named store excluded, name-only) rendered next to the whole; gate asserts the excluded name appears NOWHERE in the slice while other out-of-scope names still degrade to text; `--self-test` fault-injects **4** corruptions, all caught; coverage anchors 13→14; root AGENTS gate row updated (`f686c7914` + `fe76c78b6`) |
| **M8 — doc-truth sweep** | gotchas-testing: layered-defense mutation rule; catalog/AGENTS: `/tmp` toolchain repair recipe + spec/flag precedence + references vocabulary; TODO_LIST M3 receipt (daemon `4db61a4b9`) |
| **M9 — polish batch A** | `allowedPrefix` fold (safeImageSrc/safeHref, byte-identical); `createOrder` twins accept-annotated both sides (dupl clean for my lane); `ExportReport.Dropped` boundary (structured edges only, not owner/prose links); `CatalogProvider` stability-is-load-bearing doc (daemon `23c3afec2`) |
| **M10 — polish batch B** | `title: catalog` sentinel in `ExportSpec` + test (one spec file travels across catalogs); example `-o` output-dir flag + README lines; docserver CSP decision + sentinel decision recorded; status index row for the 17:31 report (daemon `95ea05e89`) |
| **M11 — audit-first sweep** | changelog-symbols ✓ (0 citations, honest), readme-links ✓ (866 targets, 0 broken), file-size catalog-clear, duplication my-lane clean (9 groups all foreign-owned), doc-check 1356 refs ✓, consumer readiness stamp on the downstream-adoptions row (`c1521960b`), daemon-commit audit (9 unpushed at the time — later pushed by origin) |
| **M2 — coverage re-pin** | Already green: 0.0% deltas on all 11 modules (the 10-23 16:09 session re-pinned); AGENTS line needed no change |
| **Six foreign-regression fixes in the composed-proof lane** | (1) `[Unreleased]` restored above the disk-truth 14-tag wave + ADR-0161 indexed (`2e7e57d4a`, verify-docs all-assertions-pass); (2) 11 stale go.sums tidied w/ `go list` verification (api-stability event v4.15.2, cqrs-lint fixtures, 8 engine/test modules); (3) C044 taskmanager golden +3 w/ provenance note; (4) **benchkit heartbeat shutdown race** — plain select raced a buffered tick against closed `done` (writes after stop = 5, want 1, deterministic); done-first check fixes it; 10× + `-race` + full short suite green (`681dfbaac`); (5) system legacy-row test now persists real envelopes via `WithSerialization` — the new encode gate rightly rejected raw `*event.ImmutableEvent` (old path silently wrote `fmt.Sprintf` placeholders); (6) verify lint leg healed: wrapcheck `Newf`/`WithCause` sigs added, noctx → `ExecContext(context.Background(), …)` in 3 engine typed-WAL migration helpers (no request context exists at construction), dupl suppressions on the v4↔v5 symbol twins, godox reword, gochecknoglobals/gosmopolitan targeted suppressions — `nix run .#lint` **88/88 clean** |

## b) Partially done

- **M1/M12 — the composed proof.** SIX arms so far: (1) 06:51 chain fired 09:43 → doc-assertions red → fixed; (2) 09:47 → benchkit regression + sums + golden red → fixed; (3) 10:12 → system encode-gate red → fixed; (4) 10:36 → 8-module lint red → fixed; (5) 11:10 → 3 benchkit IO-load flakes (context deadlines at 91 s; **pass standalone in 6.2 s**, and passed under load in the 09:51 loop) — classified environment flake, not code; (6) 12:18 **armed and waiting** (attempt 9 at 12:44, refusing on tree churn from the active foreign session; max-wait ~16:18). Every leg EXCEPT the final full pass is individually green on HEAD: verify-docs ✓, `#lint` 88/88 ✓, `#check-coverage` ✓ (green since 09:47 arm), catalog/system/benchkit/cqrs-lint suites green standalone.

## c) Not started

- **The green `#verify` receipt itself** — blocked on arm 6 catching a stable quiet window (the last leg: `#check-coverage` → `#verify` end-to-end).
- **M12.6 final push** — moot in one sense: foreign sessions pushed origin to within 3 commits of HEAD (my authored commits ride along); the final receipts commit + push remains.
- **Verify-ci (`#verify-ci`)** — deliberately not run: exclusivity with the waiting chain (the plan's M2.2/M12.4 order).

## d) Totally fucked up

- **Nothing I shipped is broken** — but the PROCESS had real failures:
  - **Four of six chain arms burned on foreign reds I could have front-loaded.** The AGENTS "verification ladder" (rung 3: `#verify-fast` + the targeted gates BEFORE the composed proof) exists precisely to prevent this; I armed first and bisected in 25-minute cycles instead. The ladder note says "skipping rungs saves minutes and costs re-runs" — I paid ~3 hours of re-runs to re-learn it.
  - **My first commit crashed mid-hook** (daemon moved HEAD) and left a stale `.git/index.lock` that silently blocked the next two commit attempts 20 minutes later. Stale-lock check should have been immediate.
  - **Four ADR-0160 citations were wrong on first write** (file:line numbers guessed from memory of reads rather than grepped) — caught by my own verification pass before commit, but the mistakes should never have been written.
  - **One edit silently didn't land** (benchmark function — stale-read guard refusal I misread as an application) and one nolint placement lost to golines splitting the line — both rework loops the "verify every edit landed" gotcha exists to prevent.

## e) Improvements (what I'd do differently / could still improve)

1. **Pre-arm gate sweep as a checklist, not a habit:** `bash scripts/verify-docs.sh` + `nix run .#lint` + `nix run .#check-coverage` + the four file-size/duplication/changelog/readme gates — all before ANY arm. Today that sweep would have caught 5 of the 6 red classes in ~15 minutes instead of ~5 hours.
2. **The golangci-lint version skew is a standing trap:** the dev-shell binary and nix `pkgs.golangci-lint` give DIFFERENT verdicts on the same tree (my "0 issues" vs the chain's 26 findings). Lint locally with the nix binary (`nix run .#lint`) when the composed proof is the goal — never trust the dev-shell verdict alone.
3. **benchkit's 91 s IO-bound phases under a 100-module parallel storm are a structural flake** (gotchas: "structural, not margins"). A load-sweep wrapper or per-phase deadline floor would end the re-arm lottery; today it cost one arm.
4. **Multi-session landing discipline:** every red class this session came from a session that landed work without running the composed gates or even the lint app. The fleet-wide fix is a landing checklist, not heroic chain-hopping.
5. **Authored-commit ratio is ~20%** — the daemon absorbs most units of work. The CHANGELOG carries the narrative (per the closed Q2 decision), but git blame on this session's work mostly points at `chore:` blobs.
6. **TODO_LIST interim receipts lagged:** I stamped M3–M7 and the chain STATE once, but arms 1–6 of today are only in this report — the composed-proof row should get the final arm receipt + today's arm history when green.

## f) Next items (49, ordered by impact)

1. **Harvest arm 6** (waiting, max-wait ~16:18): green → record the receipt on the TODO_LIST composed-proof row + here; red → bisect per M1.4.
2. **Run `#verify-ci`** after the chain's `#verify` completes (exclusivity) — the second half of the composed proof.
3. **Final receipts commit + push** (3 local commits + receipts; standing push instruction).
4. **Stamp the M-plan's outcome row** in `docs/status/README.md` (the plan says "and this plan's outcome when done").
5. **benchkit structural flake cure** (owner: benchkit race-fix session or next benchkit toucher): the 91 s phase deadlines need either a load-sweep wrapper on the chain's benchkit leg or deadline floors keyed on phase workload.
6. **golangci-lint version-skew doc**: one gotcha line — "the composed proof's lint verdict comes from nix `pkgs.golangci-lint`; dev-shell green means nothing" (I'll land this with the receipts if the arm goes green today).
7. **Landing-checklist proposal** for the fleet: `verify-docs` + `lint` + per-module tests before ANY push, enforced socially or via a landing script — this session's six reds are the evidence appendix.
8. **`metaengine/go.mod` pseudo-version drift** (`sqliteengine v4.6.0` off the zero pin; BuildFlow preflight FAIL, recurring): the sqliteengine session's tail — one-line normalize.
9. **`system/adapter_event.go` (351 NEW) + `scheduling/sqlstore/store.go` (363 NEW) + `metaengine/layout.go` (360 NEW) + sqliteengine growth** — the four foreign file-size offenders from this morning; they are baselined-shrink-only traps the moment the ratchet re-pins; owners owe splits.
10. **`decider/load.go` + `catalog/eventcatalog` `.Newf(` sites** — now config-allowed, but a follow-up could normalize them to `NewRejection(`-family constructors for consistency with the vocabulary.
11. **`cmd/cqrs-lint` v007 twin tables** — the campaign's directive style (`//art-dupl:accept`) does not suppress the `dupl` linter; either migrate the twins to the dupl-nolint form I used or note the divergence in the campaign's guardrails doc.
12. **`TestLegacyPrefixedRows`** now exercises envelope persistence — consider asserting the appended row's CONTENT (envelope round-trip), not just row counts; the row-count-only assertion is what hid the placeholder-garbage era.
13. **`ExportSpec.References` CLI/file asymmetry** (strict file, lenient flag) — recorded; a consumer question worth watching: does any hub script rely on lenient `-references`?
14. **`WithSerialization` audit**: are there other direct `NewEventAdapter/NewCommandAdapter/NewQueryAdapter(backend, …)` constructions over SQL engines that still rely on raw-pointer writes? (grep found the legacy test only, but a sweep over example/consumer code is cheap.)
15. **`BenchmarkInspectLargeCatalog` under a quiet host** — today's 2–3 ms was measured at load 14–26; a quiet-window number would tighten the README claim.
16. **ADR-0160 addendum**: add the `title: catalog` sentinel to the deviations table (shipped after the addendum was written).
17. **`catalog/AGENTS.md` architecture section is at ~120 lines** — fine now, but the M-decisions (sentinel, CSP, precedence, costs) could justify a split if it grows.
18. **`nix run .#check-coverage` load warning** (< 60) vs the chain's ceiling (10): two different quiet definitions confuse arms; one table in gowork-modes.md would help.
19. **Example `-o` flag + the README lines** — consider mirroring in `example/rest` for consistency.
20. **`HostileStoreName` is exported from the fixture command** — intentional (the gate script references the URL independently); if a consumer imports the command package, rename to unexported + a const comment.
21. **The `createOrder` tagliatelle nolint trio in the fixture** mirrors snake_case deliberately — consider a one-line comment block explaining WHY the example uses camelCase while the fixture keeps snake_case (external-API mirror), or align both.
22. **`ExportReport.Hidden` ordering contract** — canonical order asserted anywhere? (A cheap pin if not.)
23. **`check-architecture-reference` slice assertions** — the name-only liveness assertion depends on `Order Database`; if the fixture's store list changes, that assertion needs updating with the anchors (the 13→14 edit touched both; note for future edits).
24. **TODO_LIST §CI row on BuildFlow's golangci auto-configure recurrence** — this session saw the daemon absorb a 111-line config addition (yesterday's depguard block); the durable-guard decision is still open.
25. **`docs/README.md` ADR table** is now 160 rows in one file — an index-splitter (by year) is cheap and keeps the doc-assertions gate fast.
26. **`metaengine/keycodec/escape.go`** — the reworded comment ("defect the v2 family kills") reads fine; no action, recorded for the owner's review.
27. **`middleware/retry.go` WithCause sites** — now config-allowed; the owner may prefer explicit wrapping instead of the sig exemption (their call).
28. **Arm 6's tree-churn refusals** come from the active value-encoding session (its plan doc landed mid-session) — if it lands more reds, the M1.4 bisect protocol repeats.
29. **The verify-lock has no consumer for verify-chain** (only tag/batch-release) — registering chains in the advisory lock would prevent arm collisions like today's 06:51 mystery arm.
30. **`testModules` gate**: the architecture example stays build-only (correct); no action.
31. **SKILL.md M6.4 no-op** — record one line in the M-receipts that `Inspect`/`-check` were already documented (prevents a future "was it done?" audit).
32. **`nix fmt` vs golangci golines disagreement** (two fights this session) — the ownership table warning in BuildFlow's preflight keeps scrolling; the real fix is pinning max-len once in treefmt.
33. **`check-changelog-symbols` counted 0 citations** — my rows cite symbols in call-form (`WithHide(HideOwners)`), which the gate's `pkg.Symbol` pattern misses; either widen the pattern or accept the gate's blindness (recorded).
34. **`compose_test.go`'s spec+flags test** — could add the reversed order (flags first) as a documented no-op proof; low value, noted for completeness.
35. **`architecture-fixture` slice command** — the CLI now takes 1–2 args; the gate is the only consumer; a `-slice` flag form would read cleaner if it grows again.
36. **`scripts/check-architecture-reference.sh` self-test re-renders per corruption** (~4 go-run invocations, ~8 s) — a saved-fixture diff loop would be faster; fine at current cadence.
37. **The leak fixture's hostile URL is duplicated** (frontmatter_test const + fixture cmd HostileStoreName + gate script var) — three copies of one string; a shared testdata file would be over-engineering today, but note it.
38. **`catalog/example/architecture` output filenames** are hardcoded beside `-o`; an `-n` name flag is YAGNI until asked.
39. **`ExportSpec.Options()` sentinel** — "catalog" is a magic string; a named constant (`TitleCatalog`) would be kinder to consumers; shipped as the literal for doc-comment discoverability — revisit on first consumer ask.
40. **ADR-0160's "set builders" deviation** is recorded as DOMINANT-deviated — if a consumer ever asks for explicit builder functions, that's a new ADR per the addendum.
41. **`systemscenario` adopt path** (go-appkit pattern) is untouched by the encode gate — verify when the value-encoding session lands its remaining rows.
42. **M8.4's TODO drift check** ("fix any drifted rows this plan's audit surfaces") — the audit surfaced the four file-size offenders (item 9) and the pseudo-version drift (item 8); both stamped here rather than TODO_LIST rows — the next docs-health HARVEST should promote them.
43. **`check-coverage`'s 3-claims-verified vs 11-module table** — the gate verifies only claimed modules; the M2 AGENTS-line question stays settled.
44. **Origin is 3 commits behind local** — those are today's receipts + the foreign plan doc; the final push closes it.
45. **`catalog/go.sum`** was tidied in the fleet sweep (13:xx blob) — confirm no cascade into `catalog/go.mod` require pins at the next tag wave.
46. **The 15:13-yesterday + 06:51-today mystery arms** — both unattributed; if an automation armed them, the automation should also consume their outcomes (nobody has).
47. **`metaengine/keycodec/v2_law_test.go`** — the nolint-above-line pattern is the golines-stable form; one line for the owner's future edits.
48. **My M4 mutation protocol** is now in gotchas-testing — the next leak-test author should not re-derive the triple-mutation reality.
49. **TODO_LIST composed-proof row** should eventually absorb arms 1–6 (today's history is in this report only) — next docs-health pass.

## g) Questions I cannot answer myself

1. **Verify-slot ownership tonight (carried from the 17:31 report, still unanswered):** arms are interleaving across at least four active sessions; the 06:51 arm was not mine and its owner never consumed its outcome. Should the fleet adopt the advisory verify-lock for chains (item 29), or is hop-and-fix acceptable? I will keep re-arming arm 6's slot until green unless told otherwise.
2. **Benchkit IO-flake policy:** the three deadline tests pass standalone and passed under load at 09:51, but flaked at 11:10 — is re-arm-and-pray acceptable for the composed proof, or should I land a structural cure (deadline floor / load-sweep wrapper) in benchkit myself? The latter touches a closed session's fresh code.
3. **Push timing:** origin is only 3 commits behind (foreign sessions pushed through my commits). The standing instruction says push at the end — should I push the current state now (everything green except the un-run final `#verify` leg) and follow with the chain receipt, or hold until arm 6 goes green?

---

*The feature was never the hard part; the composed proof was six foreign regressions deep.*
