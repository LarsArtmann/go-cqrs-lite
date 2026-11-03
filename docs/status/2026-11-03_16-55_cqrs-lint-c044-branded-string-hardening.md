# Status Report: cqrs-lint C044 branded-`.String()` hardening + first real catch

**Date:** 2026-11-03 16:55 CET
**Session type:** TODO_LIST execution (picked the 🔥 row "cqrs-lint branded-`.String()`
rule, warn-first (plan v2 A10)" from the disk-truth residue section)
**Working tree at start:** clean, master @ `a5ebc34e3` (auto-commit daemon active; several
`chore: auto-commit` entries absorbed this session's mid-flight state — expected).

---

## a) FULLY DONE (verified, receipts below)

1. **TODO row triaged and closed — it was STALE.** The row (re-rowed 2026-10-23) asked for
   a rule a parallel session had already shipped on 2026-10-24 as **C044
   `branded-id-display-string`** (commit b78476219, rides cmd/cqrs-lint v4.18.0). This
   session initially built a duplicate (see section d), caught it via the taskmanager
   integration golden, deleted it, and **strengthened C044 in place** instead. Row closed
   `[x]` with receipt; record in CHANGELOG `[Unreleased]` → Changed.

2. **C044 strengthened** (`cmd/cqrs-lint/pkg/rules/correctness/c044.go` +
   new `c044_display.go` after the file-size gate forced a split):
   - **Nested-depth display whitelisting**: arguments of display calls are whitelisted at
     ANY nesting depth (span-based), not just direct arguments — `slog.Info("x", "k",
     strings.ToUpper(sid.String()))` no longer false-positives.
   - **`Parse*ID` round-trip whitelisted** — the documented law
     (`ParseStreamID(id.String()) == id`) was previously flagged.
   - **Log-attribute packages whitelisted** — otel `attribute` (last path segment).
   - **Typed qualifier resolution** (`ResolveQualifierTyped` with the
     `PathContainsFragment` version tolerance): versioned module paths
     (`…/id/v4` binds as `id`) and aliased imports now classify correctly; the old
     syntactic import-table match could not see them at all.
   - **Benign-receiver carve-outs (FP control)**: cbid.ID instances whose marker lacks
     `Name()` render the BARE value in `String()` (id.EventID + the ULID family) — no
     longer flagged; id-package passthroughs (`StreamType.String`, `ActorKind.String`) —
     no longer flagged.
   - **Lossy overrides get boundary-specific guidance**: `ActorID.String()` drops the
     actor kind → message points at `PrefixedString()`/`ParseActorID`;
     `StreamRef.String()` is the display composite → message points at `StreamKey()`
     (the 2026-10-17 disk-truth fix).
   - **Confidence medium → high** (builder + catalog + severity-meta lockstep).
   - Marker check reads the CALL-SITE receiver instance's type args (method declarations
     carry uninstantiated receivers); `types.Unalias` handles the alias blindspot
     (`id.StreamID` arrives as a `*types.Alias`, not `*types.Named`).

3. **The rule's first real catch — 3 live bugs healed in `example/taskmanager`:**
   - `decider.go:452` + `decider.go:486`: `TaskBlockedByPayload`/`TaskUnblockedPayload`
     serialized `StreamMarker:`-prefixed display forms into EVENT PAYLOADS (the tolerant
     `ParseStreamID` fold masked the prefix on read-back — exactly the fragile pattern).
   - `http.go:98`: HTTP JSON response body returned the prefixed display form while
     `http.go:240` in the SAME file already used `.Get()` — the canonical bare form.
   - All three now use `.Get()`; taskmanager module suite green.

4. **Test contracts updated and green:**
   - `testdata/typedfixture/c044fixture.go` extended: 5 bad (SQL arg, cursor key, map
     key, ActorID, StreamRef) / 6 good (slog, log.Printf, fmt.Sprintf, bare EventID,
     nested display arg, ParseStreamID round-trip); `TestC044_Fixtures` pins 5/0,
     `TestC044_ActorAndRefMessages` pins the tailored messages, `TestC044_WarnSeverityFirst`
     unchanged-green.
   - `pkg/rules/integration_test.go` golden: C044 entry REMOVED with a dated receipt
     comment (zero-fire entries cannot live there — the LOST check reads absence from
     `byRule` as regression). Root `testdata/taskmanager_golden.txt` regenerated via
     `CQRS_LINT_UPDATE_GOLDEN=1` (37 → 31 findings).
   - Full cqrs-lint suite **green ×2 (all 19 packages)** after every stage.

5. **Pins kept in lockstep** (the AGENTS "Add a Lint Rule" ritual, applied in reverse for
   the duplicate removal): RULES.md regenerated; README counts stay 215/architecture 23
   (E024 added then removed — net zero); `meta_test.go` count back to 215;
   **api-stability golden delta = exactly one line (the duplicate's `NewE024Detector`
   removal)** — verified via git diff.

6. **Gates run and green:** file-size ratchet (after splitting 382-line c044.go →
   243 + 150), changelog-symbols (21 citations verified), treefmt `--fail-on-change`
   (also incidentally formatted the parallel session's unformatted
   `catalog/cmd/catalog-compat/main.go`), `explain C044` renders the strengthened
   contract (acceptance criterion).

7. **On-sight fix:** README's stale "Detect runs all 208 rules" line (7 behind the gated
   215 headline, ungated) corrected to 215.

## b) PARTIALLY DONE

- **Workspace lint gate (`nix run .#lint`)**: launched, still RUNNING at report time
  (job 002). All lint-relevant local checks (build, vet, gofmt/treefmt, full tests)
  already green; the composed gate result is the one open box.
- **Plan v2 B36 "waiver example"**: the catalog description + RULES.md entry now document
  BOTH dismissal paths (`//cqrs-lint:ignore(C044) <reason>` and the
  `.cqrs-lint.json` scorecard waiver), but no dedicated runnable waiver example file was
  added (deemed covered by the generated docs; noted below as an optional polish).

## c) NOT STARTED (unchanged TODO_LIST rows this session ignored on purpose)

- The keycodec v2 A9 engine-adoption wave, both OWNER QUESTIONs (deployed-DB census, A9
  scheduling), the 190-group `check-duplication` fork-campaign decision, journal
  dump⇄load, the test-isolation strategy doc, T10/T13–T15/T17–T20/T22/T24-tail, and the
  rest of the standing backlog — untouched, still rowed.

## d) TOTALLY FUCKED UP (honest ledger)

1. **Built a duplicate rule.** I researched the E-rule lineage, the cqrs-lint skeleton,
   and the TODO row — but never grepped the FULL rules tree for the concept ("branded",
   "display") before implementing. E024 was ~90% complete (detector, 9 tests, all pins)
   before the taskmanager integration golden exposed that C044 had existed since
   2026-10-24. Cost: one full build/test cycle + count churn (215→216→215) + one
   api-golden regen round-trip. The safety net (golden + count pins) worked exactly as
   designed — the mistake was skipping the duplicate-check, not the recovery.
2. **Fixture compile errors ×3** (unused import; a trimmed stub missing `NewID`; a
   nonexistent `slog.InfoArgs` API in a fixture) — each cost a debug round-trip. Fixture
   authoring discipline was sloppy.
3. **One multiedit joined a doc comment and a `func` declaration onto one line** (old/new
   strings differing only by a trailing newline) — caught immediately by inspection,
   fixed in one edit.
4. **Edited the integration golden to `"C044": 0` without first reading the assertion
   loop** — the LOST check treats a zero-fire golden entry as regression, so the edit
   could never pass. Read the consumer before the edit; I did it in the wrong order.
5. **The LSP diagnostics noise** (`golangci_lint_ls` context-deadline on the 101-module
   workspace root) polluted every tool result this session — pre-existing environment
  issue, not mine, but I should have suppressed/scoped it early instead of re-reading
  "errors" that were noise.

## e) WHAT WE SHOULD IMPROVE (systemic observations from this run)

1. **Stale-row prevention:** a parallel session completing a row does not close the row.
   The TODO_LIST header policy exists, but no mechanical guard. Cheap fix candidate: a
   docs-health VERIFY pass that greps each row's named deliverable (`pkg.Symbol`, rule
   ID) against the repo before dispatching work.
2. **Golden semantics are implicit:** "zero-fire entries must be deleted, not zeroed"
   lives only in the assertion loop's behavior. A one-line comment on
   `taskmanagerGoldenProfile` would have prevented mistake #4.
3. **Duplicate-concept check for rules:** a tiny meta-test asserting no two detectors
   share a normalized concept (name/description token overlap) would have flagged
   E024-vs-C044 at registration time. (Cheap, heuristic, warn-only.)
4. **Pre-existing red meta-tests at HEAD poison trust:** `TestEveryModuleGoSumIsTidy`
   (benchkit) and `TestEveryGoModDirIsInModulesList` (scratch/counterkey) are red on a
   clean checkout — from the parallel counter-key session's committed residue. Anything
   running TestEvery (e.g. api-stability work) trips over them and burns triage time.

## f) NEXT — up to 50 things (roughly impact-ordered; ⏳ = in flight right now)

**This session's tail:**
1. ⏳ Read job 002 (`nix run .#lint`) result; fix anything it names.
2. Run `nix run .#check-duplication` — the c044 split added a file; the no-new-clones
   gate has NOT been run this session (my miss; run before wave close).
3. Run `nix run .#check-test-runtime` — correctness package gained typed-fixture tests;
   budget drift unknown.
4. Run `nix run .#check-coverage` — correctness coverage moved; baseline tolerance check.
5. Run `nix run .#check-package-surface` — correctness package file count grew by one.
6. Run `cmd/doc-check` + `nix run .#check-md-go` — RULES.md/README/CHANGELOG changed
   (generated/fenced content — expected clean, but verify).
7. Optional polish: runnable `.cqrs-lint.json` waiver example for C044 (B36 remainder).
8. Optional: pin a POINTER-receiver fixture case for C044 (covered by code path, unpinned
   by fixture).

**Pre-existing red (parallel-session residue at HEAD):**
9. benchkit `go.sum` not tidy (`TestEveryModuleGoSumIsTidy` red) — `GOWORK=off go mod
   tidy` in benchkit + commit both files.
10. `scratch/counterkey` has go.mod but is not in api-stability modules slice
    (`TestEveryGoModDirIsInModulesList` red) — register it or relocate the scratch.
11. Same class: verify `TestEveryGoModDirIsInTestModules` heals with #9/#10.

**High-value standing rows (unchanged):**
12. 🔴 BLOCKING row: stale-read TOCTOU in `system.CachedEventStore` (TODO testing
    section, found 2026-10-27, sketch exists) — the one red-row left in that plan.
13. keycodec v2 A9 engine adoption wave (pebble → badger/bbolt twins → legacy stores +
    README migration notes; patterns to copy from the typed-WAL wave).
14. OWNER QUESTION: deployed-DB census (sticky-key fallback vs re-key migration).
15. OWNER QUESTION: A9 scheduling confirmation.
16. `check-duplication` RED (190 novel groups): fork-campaign directive sweep or ONE
    deliberate end-of-campaign re-pin (never mid-campaign).
17. Composed `#verify` in an exclusive quiet window for the changed set.
18. catalog: cut the next train containing `Registry.MarkDemo` (untagged; mesh-demo
    broken against the proxy) + verify mesh-demo `GOWORK=off` after.
19. Gate the class: `GOWORK=off` build of `examplePaths` against the pinned graph.
20. Journal dump⇄load round-trip (typed, envelope-stamped; L effort).
21. Consumer-facing test-isolation strategy doc (4 strategies + reset-by-replay; S).
22. C044 dogfood sweep: run cqrs-lint over the OTHER examples (getting-started,
    goal-shaped-app, graph-native, scheduler-otel-status) — cheap real-world FP-rate
    validation of today's hardening.
23. cqrs-lint analysis-time regression investigation (2.0s → 7.4s; InboxClean ask).
24. catalog `FactoryFromSystem` (derive Registry from DomainConfig; design with owner).
25. scenario: version-carrying decide funcs in `Given` (deletes consumer adaptDecide
    wrappers; S).
26. Fleet census: invite the five 2026-10-24 auditors to contribute case studies (XS each).
27. Wave III seed: inspect ≥5-file content deltas on taggable trains (middleware 28,
    command 14, query 13, …).
28. Issue #58 owner verdict: pre-system journal → `system.New` adoption path.
29. T10 mutation-verify gate (tooling evaluation + PR-scoped timeboxing).
30. T13 `synctest` migration wave 1 (scheduling ticker/deadline tests).
31. T14 planner model-based property test (`rapid`, hysteresis + quarantine).
32. T15 differential engine harness (one script, all engines, divergence allowlist).
33. T16 remainder: per-code error-family behaviour tests + deprecation pins (540 frozen).
34. T17 executable examples / tag fixtures / promoted recipes.
35. T18 tail-latency concurrency matrix (P99 ceilings, nightly quiet window).
36. T19 heap/goroutine/disk budgets (replay/SSE heap, post-Close goroutines).
37. T20 crash-consistency tests (`kill -9` mid-write on sqlite/pebble/bbolt).
38. T22 broker partition/redelivery/DLQ/outbox crash matrix.
39. T24 owner decisions: `signing.invalid_signature` family (Corruption vs Rejection)
    + whether writes should abort on canceled context.
40. Turso matview caveats/upstream handoffs tail (TODO section).
41. EventCatalog v5 licence shift row.
42. Upstream asks (EventCatalog hub + cqrs-htmx) sections.
43. md-go-validator CI tail.
44. benchkit statistical-rigor tail.
45. Add a C044-style duplicate-concept meta-test (see e-3).
46. Document golden zero-entry semantics on `taskmanagerGoldenProfile` (see e-2).
47. LSP: scope or raise the golangci-lint timeout for this workspace (session-long noise).
48. Declined-section hygiene: nothing to do — guard intact (verified by read).

## g) Questions I cannot answer myself (3 max)

1. **Pre-existing red meta-tests** (benchkit go.sum untidy; scratch/counterkey
   unregistered) are the parallel counter-key session's committed residue. Fix on sight
   next session, or leave strictly to that session's owner?
2. **C044's `fmt.Sprintf` display whitelist** (shipped 2026-10-24, pinned by its
   fixture) is broader than plan v2 B34's "slog/attribute/log" wording — meaning the
   StreamKey key-via-Sprintf class is whitelisted today. Keep the shipped decision
   (my recommendation: warn-first + waiver covers it), or align to the plan and take the
   Sprintf noise?
3. **Composed `#verify`** for this changed set: arm it now (exclusive window, may queue
   behind load) or let tonight's nightly/CI carry the proof?

---

**Receipts (commands + observed):**
`GOWORK=off go test ./... -count=1` in cmd/cqrs-lint → green ×2 (19 packages, final run
139.8s root). `bash scripts/check-file-size.sh` → green after split (61 baselined
offenders unchanged). `bash scripts/check-changelog-symbols.sh` → 21 citations verified.
`nix fmt -- --fail-on-change` → 0 changed (second run). `cqrs-lint explain C044` →
renders warning/high + new description. `git diff docs/api_surface.txt` → exactly the
`NewE024Detector` line removal. Lint gate `nix run .#lint` → running at report time.
