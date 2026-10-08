# Status Report — Turso Bump + Dedup Cleared, Stack Seam v4.5.0 Shipped, Bridge Tagged (Unpushed)

- **Date:** 2026-10-08 18:30 (Thursday)
- **Session span:** ~17:45 → 18:30 (resumed from the 16:35 halt per handoff; full-execution mode continues)
- **Driving plan:** [`2026-10-08_14-59_SUPERB-v5-goal-pareto-plan.html`](../planning/2026-10-08_14-59_SUPERB-v5-goal-pareto-plan.html) (T01–T27 / f001–f150)
- **Authoritative state:** master; `stack/v4.5.0` PUSHED + proxy-smoked; `stack/metaengine/v4.0.0` tagged LOCALLY — **NOT pushed yet** (halt point); all four W1 preflight blockers from the halt are now repaired.

---

## a) FULLY DONE (this session)

### T03 final blocker — Turso IVM verified range → v0.8.2 ✅
1. **Fresh ivmrepro receipt:** `-tags ivmrepro` re-ran on the current tree (tursogo v0.8.2 pin) — GREEN in 35.9s (green = defects A/B/C still present, identical 430.50 delta signature → range extension is safe).
2. `metaengine.TursoGoIVMVerifiedThrough` v0.8.1 → **v0.8.2**, `TursoGoIVMLastVerified` → 2026-10-08 (`metaengine/materialized_view_versions.go`).
3. **All 7 gated live citations moved** (check-turso-version's LIVE_FILES): FEATURES.md, gotchas-tooling-build.md (2 phrases), ADR-0135, benchmarks/2026-09-07_turso-materialized-views.md, readmodels.md, recipes.md (2 sites) + the non-gated repro comment (`ivm_repro_test.go`).
4. CHANGELOG `[Unreleased]` Changed entry citing the constant (runbook requirement).
5. Gates: `#check-turso-version` **EXIT 0** ("All live turso-go IVM citations match v0.8.2"); metaengine Doctor/materialized-view tests green.

### Dedup gate — was dirty-tree-blocked at halt, now CLEAN ✅
`#check-duplication` initially reported **4 new clone groups** (post-wave residue). Judged each:
- **Extracted (real consolidation):** `metaengine/adttest.requireVectorBackend` (shared vector-ADT prelude, concurrency.go + vector_dimension.go); `system.CachedEventStore.advanceGeneration` (identical 3-line critical section in beginWrite/endWrite — call-site ordering contract stays visible).
- **Accept-annotated (intentional):** catalog↔metaengine typed-slice→string idiom (cross-module, 5-line idiom ≠ dep edge); watermill command/event bus publish-tail twins (independent payload types keep the buses decoupled).
- Result: **art-dupl 0 new clone groups** (baseline 186); stack/system/catalog/watermill standalone builds green; system + metaengine `-short` suites green.

### Stray 9 MiB tracked blob — found + killed ✅ (found BY the release tooling)
The first `stack/v4.5.0` cut was **refused by the pre-push zip-content guard**: `cmd/cqrs-lint/testdata/busfixture/busfixture` (9,096,063 bytes) — a compiled fixture BINARY the auto-commit daemon absorbed on Oct 7 (`5cd59bd97`); tests load the fixture's SOURCE via `packages.Load`, never the binary. Fixed: `git rm` + `.gitignore` entry beside its sibling fixture binaries (typedfixture/scanfixture were already listed — busfixture was simply missing). Tree-wide rescan: **zero blobs > 4 MiB remain**.

### T06 tag dance — 1 of 3 tags fully shipped ✅
- **`stack/v4.5.0` CUT + PUSHED + PROXY-SMOKED** (attempt 1; strip-commit 68537be19). Carries the `MetaEngineStore` seam (WithMetaEngine/Bundle.MetaEngine → `Close() error` interface), root stack metaengine-free.
- **Handoff tag-order error CORRECTED before any damage:** the handoff said bridge-first; verified `stack/v4.4.3`'s `Bundle.MetaEngine()` returns `*metaengine.Store` (concrete) — the bridge's type assertion `b.MetaEngine().(*metaengine.Store)` is ILLEGAL against non-interface return ⇒ bridge could never compile against v4.4.3. Correct order: seam first, then bridge. (tag-release.sh would have caught it at the bridge's standalone verify — caught earlier by reading `git show stack/v4.4.3:stack/bundle.go`.)
- **`stack/metaengine` pinned to stack v4.5.0** (`go get` also rode watermill v4.6.5 + flightrecorder v0.2.1), **standalone build + `-short` test GREEN** (resolving stack v4.5.0 from the proxy — proving dual-compatibility of the bridge).
- **`stack/metaengine/v4.0.0` TAGGED locally** (8b416a7f7) — push/smoke pending.

---

## b) PARTIALLY DONE

### T06 — stack/metaengine module (#36): publish ~70% complete
Remaining: ① push `stack/metaengine/v4.0.0` + `--smoke`; ② stack/sqlite: `go mod edit -dropreplace=github.com/larsartmann/go-cqrs-lite/stack/metaengine/v4` → pin bridge v4.0.0 (+ stack rides to v4.5.0 via MVS) → tidy → `GOWORK=off` test → tag `stack/sqlite/v4.3.5` → push → smoke; ③ CHANGELOG dated mini-wave section ([Unreleased] stays first); ④ `check-versions-manifest.sh --update` + `--check --remote` (script already wrote 111 trains locally; README.md/versions.json currently dirty — daemon absorbing); ⑤ `gh issue close 36` with receipt; ⑥ TODO_LIST row receipt.

### T03 — composed `#verify` re-record: all 4 preflight blockers repaired, run NOT yet launched
lint-config ✅ (prior session, verified committed) · templ ✅ · duplication ✅ (this session) · turso-version ✅ (this session). The full chain (`preflight-composed.sh` → `can-run-composed-gate --wait-loop` → `#verify`) has not been started; load1 was 5.5 at session start vs the <5 ceiling — quiet window still pending.

---

## c) NOT STARTED
- **T04** MySQL/MariaDB legs (f047–f052), **T05** calibration (f053–f057) — quiet-window gated.
- **W2 remainder:** T07 cursors (f062–f064), T08 consumer DX (f065–f071), T10 goldens (f074–f076), T11 engine surfaces (f077–f082).
- **W3–W6** (T12–T27): v5 branch + deletion cascade, universal fold/scan flip/encryption/FilterOp, docs + THE CUT, closure.

---

## d) TOTALLY FUCKED UP (incidents + my mistakes this session)
1. **Daemon raced TWO authored commits (again):** (a) the turso batch — my commit landed with only CHANGELOG.md; the other 8 files rode daemon chore `2b1d0516a`; (b) the .gitignore/blob-removal commit **died entirely** at ref-lock ("cannot lock ref 'HEAD' … expected aab1f715f") after the 5-minute BuildFlow precommit opened the window — content survived via daemon `592b711e4`. Same class as the prior session's incident #1; my countermeasures (small+fast) were still too slow against a 5-minute hook.
2. **Started the tag dance on the handoff's (wrong) order** — caught it before cutting a broken bridge tag only because I read the bridge source + v4.4.3 API mid-flight. Should have verified pin-vs-API compatibility BEFORE beginning the dance.
3. **Did not pre-scan the tree for zip-guard violations** before the first cut — the guard caught the 9 MiB blob, burning a cut cycle (and leaving a harmless orphan prep commit `cf4bd8a96` in history). One `git ls-tree -r -l` would have caught it upfront.
4. **Edit-tool round-trips wasted:** attempted edits on 6 files after only bash-grepping them ("must view first" rejections). Discipline slip, no damage.
5. Noticed (NOT mine, ambient): BuildFlow preflight FAIL `workspace/pseudo-version-hygiene` (metaengine/go.mod sqliteengine v4.5.1 drifted off zero-pseudo) + 3 modules flagged need-tidy (busfixture, integration, storage); govulncheck "failed 9/11 times (82%)" warning. All pre-existing — listed for the record, untouched.

---

## e) WHAT WE SHOULD IMPROVE
1. **Pre-tag pin-vs-API compat check:** before any dependency-ordered wave, diff each module's go.mod pins against the API its new code uses (the v4.4.3 concrete-return catch generalizes: "would the tagged go.mod compile the code?"). Candidate: a `check-pin-api-compat` leg in preflight.
2. **Pre-tag tree scan for guard violations:** `git ls-tree -r -l | awk '>4MiB'` + control-char paths belongs in `preflight-composed.sh` (or tag-release's preflight), so blobs die before a cut cycle, not during.
3. **Fixture-binary hygiene gate:** every new testdata fixture dir must add its compiled name to `.gitignore` in the same change (the daemon absorbs any binary otherwise). Candidate: warn on newly-tracked blobs >1 MiB in pre-commit.
4. **Commit-vs-daemon policy decision needed:** authored commits keep losing to the 5-minute BuildFlow precommit. Options: (a) accept daemon attribution for everything (current de-facto), (b) tiny+fast commits only (insufficient — proven twice), (c) sanctioned `git commit --no-verify` for docs/chore-only batches, (d) shorten the precommit for non-code diffs (it already skips Doc-only; the .gitignore batch was misclassified as code because `git rm` staged a .go-adjacent artifact). Needs an owner call — see questions.
5. Carried (still open, from 16:35 report): restore-depguard post-splice nesting assertion; use batch tool's printed push line verbatim (done this session ✓); config-war canary in `#verify-fast`.

---

## f) NEXT — up to 50, in execution order
**Close T06 (minutes):**
1. `git push origin stack/metaengine/v4.0.0` → `scripts/tag-release.sh --smoke stack/metaengine v4.0.0`.
2. stack/sqlite: `go mod edit -dropreplace=github.com/larsartmann/go-cqrs-lite/stack/metaengine/v4` + `go get …/stack/metaengine/v4@v4.0.0` + tidy + `GOWORK=off go test -short -count=1 ./...`.
3. Wait-clean-tree → tag `stack/sqlite v4.3.5` ("metaengine preset via typed bridge; drops temp replace") → push → smoke.
4. CHANGELOG: cut dated section for the 3-tag mini-wave; [Unreleased] restored first; verify-docs + `TestTagContentMatchesChangelog`.
5. `check-versions-manifest.sh --update` then `--check --remote`.
6. `gh issue close 36` with receipt comment (seam + bridge + tags + smoke evidence).
7. TODO_LIST T06/W2 row dated receipt.
**Close T03 (W1):**
8. Check `cat /proc/loadavg`; when load1<5: `bash scripts/preflight-composed.sh && nix run .#can-run-composed-gate -- --wait-loop && nix run .#verify`.
9. f046 receipts on dedup-(a)/W1-sibling/Layer-1 rows.
**T04 (quiet window):**
10. `#integration-mysql-vm` hardened run + F52 AGENTS rows. 11. snapshot-migration MySQL live. 12. shuffle-seed replay. 13. G-T13 ADTSet mysql-VM leg. 14. nspawn leg (needs root). 15. claiming metrics suite.
**T05:**
16. calibration-gate loop. 17. SearchQuery count=5 (+supersede if >5%). 18. dgraph constants re-anchor. 19. baseline titled re-pin. 20. supersede-note on the 09-19 capture.
**W2 remainder:**
21. T07 f062 engines fill `ScanResult.NextCursor` (3 slices). 22. f063 ScanPage prefers NextCursor + ParseCursor normalizes. 23. f064 wire goldens + readmodels row.
24. T08 f065–f066 AsyncAPI response schemas. 25. f067 bindings/securitySchemes. 26. f068 pushdown cookbook recipe. 27. f069 F024/F025 utilization variants. 28. f070 load-scope widening/confidence tier. 29. f071 B005 hardening test.
30. T10 f074 TestEvery* re-record. 31. f075 v007 drift ×2. 32. f076 E-items golden + record/v4 consumer pin sweep.
33. T11 f077 Tier-0 flock helper. 34. f078 EngineConfig.SingleWriter wiring (ADR-0150). 35. f079 AggregateOn QueryOption. 36. f080 MatViewSpecReporter + O(1) pricing. 37. f081 Doctor INFO uncovered. 38. f082 WithContentionRetry export.
**W3:** 39. T12 v5 branch + 98×go.mod path flip + reverse registration sweep. 40. T13 L4 smalls (BuildWhereClause, ADR-0126 shells, tombstone metadata API). 41. T14 L5 projection cascade. 42. T15 stack/ total deletion. 43. T16 transports deletion. 44. T17 NewStreamRef validation.
**W4:** 45. T18 universal fold + ApplyBatch atomicity. 46. T19 scan-default flip (Option C). 47. T20 encryption-at-rest. 48. T21 FilterContains/Prefix + queue.WithClock.
**W5:** 49. T22 V5-MIGRATION-GUIDE (+SQL-column section first-class), CHANGELOG v5.0.0, SKILL.md.
50. T23 THE CUT (full verify + vulncheck + integrations, tag v5.0.0, proxy check, v6 markers).
(W6: T24 goal gates + FEATURES flip, T25 filings pack, T26 tooling polish incl. 3 gate-red catalog file splits, T27 watchlist register.)

---

## g) QUESTIONS ONLY YOU CAN ANSWER
1. **Daemon-commit policy (new, blocks nothing but wastes every authored attempt):** may release/doc batches use `git commit --no-verify` to escape the 5-minute BuildFlow precommit window the daemon always wins? (Alternatives: keep losing attribution to daemon chores, or shorten the hook for non-code diffs — the .gitignore+`git rm` batch ran the FULL Go pipeline because the removal touched a compiled artifact.)
2. **GitHub Actions billing (R20, carried):** every paid CI job has failed in 3–7s since ~2026-07-17; all remote-CI evidence (and the ruled F040 branch protection) gates on this. Fix when, or stay local-`#verify`-only indefinitely?
3. **F153 — pkg.go.dev license (carried):** repo is deliberately PROPRIETARY so pkg.go.dev hides all module docs BY DESIGN. Relicense OSS (unblocks public godoc) or accept local-godoc-only? This shapes T22's migration-guide wording (how much it can lean on pkg.go.dev links) — T22 is close now.

*(ERRAUDIT_PAT / R21 also still user-gated: the error-audit CI job arms the moment the secret exists; nothing currently blocks on it.)*
