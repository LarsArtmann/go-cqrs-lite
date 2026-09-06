# Status Report — catalog/encryption TODO wave: 8 items executed, 3 latent bugs exposed

**Timestamp:** 2026-09-06 08:26 CEST
**Branch:** master, 5 commits ahead of origin (unpushed — owner decision still pending from prior session)
**Session mission:** execute the pasted TODO block (Catalog + Encryption/consumer asks, 8 items) — READ/UNDERSTAND/RESEARCH first, then execute and verify one item at a time.
**Concurrent activity:** a second session live throughout (doc-check work appeared mid-session as untracked files; iroh/system/graph workstream touched `example/taskmanager/README.md` and AGENTS.md); the auto-commit daemon swept my work into ~12 anonymous "chore: auto-commit N file(s)" commits.

> **Honesty up front:** every item in the paste was genuinely open (unlike the prior session's stale paste — I verified against code before executing). Verification this session is **module-scoped GREEN, full `#verify` NOT run** (parallel session live; the AGENTS rule forbids heavy concurrent gates). The paste was executed 100%, but two "quick verification" items I promised myself (cqrs-lint self-lint delta, go-retry consumer-pin check) were skipped — see §c.

---

## a) FULLY DONE

1. **go-retry `DoWithValue[T]` release verified and shipped.** The commit was pushed but contained in NO tag (8 commits past v0.4.0). Promoted CHANGELOG `[Unreleased]` → `[0.5.0]`, race+vet green first, annotated tag, pushed master+tag, then verified resolvable **through proxy.golang.org** (not just locally).
2. **api-stability golden sub-package coverage — fixed the class, not the symptom.** The item said "include `catalog/v4/docserver`"; the real bug was that `collectExports` read only each module's ROOT directory, making ~42 sub-packages invisible repo-wide. `collectModuleExports` now walks non-internal/non-testdata sub-dirs, prefixes symbols with the package path, skips `main` sub-packages, and fails loudly on unparseable files. Golden: 4299 → **6671 exports** (regen run three times as new symbols landed). Meta-tests (`TestEvery*`, `TestAPISurface*`, idempotency) green.
3. **Encryption key-management helpers (bank-sync ask).** `encryption/keys.go`: `GenerateKey`, `GenerateKeyBase64`, `EncodeKeyBase64`, `ValidateKey`, `DecodeKeyBase64`, `LoadKeyFromEnv`, `LoadKeyFromFile`, sentinel `ErrKeyNotSet`. Table-driven tests; openssl-style trailing newlines tolerated; malformed keys wrap `ErrInvalidKey` with observed-vs-required byte counts. API names follow the consumer's own proposal doc.
4. **Eventcatalog embedded-flattening golden test.** `TestGolden_EventCatalog_FlattenedSchema`: multi-level embedded fixture (`payment` embeds `paymentMethod`, payload embeds both + a plain field) through `SchemaFromType` → exporter; golden pins the flattened `schema.json` AND asserts no embedded type name leaks into properties. `cmd/cqrs-gen` fallout check done: zero catalog/schema coupling (pure struct-name codegen) — documented in TODO_LIST.
5. **cqrs-lint C040 follow-ups (#207 + #208) + doctor flags.** (a) The ~90%-identical fold-case walkers (analyzer `CollectFoldCaseStrings` vs c040-local `collectFoldCasesWithPos`) are now ONE position-aware collector `analyzer.CollectFoldCasesWithPos` + `FoldCaseInfo`; C038/C040/E006 share it. (b) Const-identifier fold cases (`case userCreatedType:` / `case events.X:`) resolve through `TypeConstValues` — scanner widened to also record `event.Type` consts (verified: only consumers are the fold resolver + `ResolveRegisteredTypeConsts`). New tests pin both directions (resolved const cited in finding; unknown identifiers ignored). (c) `doctor --format json` (`doctor_json.go`, 194 lines) and `--fix --dry-run` (`suppression.PlanStaleInlineSuppressions` — a classify-only twin of the fixer, test-asserted to NOT mutate files). All smoke-tested against `example/taskmanager` with a built binary; 18/18 module packages + lint green.
6. **`check-eventcatalog` flake app — verified green end-to-end.** `catalog/cmd/ec-fixture` (stable demo catalog) + `scripts/check-eventcatalog.sh` (generate → npm install → `npx eventcatalog build` → FAIL on non-zero exit or "invalid content reference"; pipe-exit-code gotcha avoided) + flake app. Real run: 57 pages built, zero warnings, EXIT=0.
7. **CSP browser validation — `check-csp` app.** `catalog/docserver/csp_browser_test.go` (skip-gated on `CQRS_BROWSER`): headless Chromium loads index/Scalar-OpenAPI/AsyncAPI-React/D2 pages with CSP ENABLED; asserts zero "Refused to"/CSP-violation console messages, DOM markers rendered, and 200-fetches of the embedded `scalar.js`/`asyncapi-react.js` via a status-recording wrapper. Verified green with nixpkgs Chromium. This closes "never browser-validated" with a repeatable app, not a one-off manual check.
8. **Snapshot-encryption PG/SQL test + rotation write-back.** `storage/pg_integration_snapshot_encryption_test.go` (integration-tagged, ephemeral-PG): asserts the JSONB column holds ciphertext (plaintext marker absent, envelope `kid` present), then rotates keys and asserts the FIRST load re-encrypts under the active key IN THE DATABASE and the retired kid is gone. Unit tests for the rewrite-on-read semantics (stale→active swap, partial-config rejection, second-load idempotence). Full ephemeral-pg battery: ✅ "Integration tests passed".
9. **Three latent production bugs found BY the new tests and fixed at root** (details in §d-adjacent evidence, all green now): encrypted-snapshot JSONB save failure (v2 envelope), `[]byte`→bytea binding failure (text bind), storage module not compiling (`containsString`).
10. **Per-task gates actually run per task** (not batched): module suites + golangci per touched module; api golden regen three times (4299→6660→6669→6671); `check-changelog-symbols` (162 citations verified honest); `check-workspace-sync`; `check-arch` (incl. "storage passed"); doc-check (961 references); full workspace + per-module builds.

## b) PARTIALLY DONE

1. **Skill reference updates.** Added the key-lifecycle snippet to `recipes.md` §2.7 (doc-check green afterwards). NOT yet documented in skill refs: envelope v2 wire format, rotation write-back recipe, `doctor --format json`, the two check apps. CHANGELOG + code docs carry the truth; the skill surface lags.
2. ~~**AGENTS.md gotchas.** Nothing added for: envelope v2 (wire-format change in a minor), the pgx `[]byte`→bytea/JSONB trap, the "release an external sibling repo" checklist, exhaustruct_v5 nolint-placement quirks.~~ partially done — pgx bytea/JSONB + envelope v2 recorded by the 2026-09-06 evening docs-health pass; the rest routed to TODO_LIST (Docs section).
3. **Verification scope.** Module-scoped GREEN everywhere; the full `nix run .#verify` (build+vet+test+race+lint+doc-check battery) was NOT run — parallel session live, box contended. Stated as such rather than claimed.
4. **C040 follow-up #206 skipped.** Projection-handler dead-case detection (the third documented follow-up) not started — the idea itself flags higher false-positive risk; I did the two safe ones (#207 dedup, #208 const resolution) and left #206 with its design note.
5. **cqrs-lint self-lint delta not measured.** The collector unification changed C038/C040 semantics (IsTest skip now shared; const resolution added). Module tests green, but I did not re-run the linter against the repo itself to measure the finding-delta on real code.
6. **go-retry consumer-pin check skipped.** Released v0.5.0 and verified proxy resolution, but did not grep whether any go-cqrs-lite module (or sibling project) pins go-retry and needs a bump. Release is consumable regardless.
7. **doctor JSON multi-module depth.** Smoke-tested single-module (taskmanager); the multi-module `Modules` array ships but the text↔json parity for per-module profiles was not golden-tested.
8. **go.sum drift fixed only where it failed.** `idempotency/sqlstore`'s standalone go.sum was missing `klauspost/compress` (surfaced by the ephemeral-pg run); tidied and green — but I did not sweep OTHER modules for the same standalone-build drift.
9. **Encryption goldens and the v2 wire format.** The module suite passed WITHOUT `UPDATE_SNAPS`, so nothing pins the v1-base64 → v2-raw-JSON output change in a golden — the compat tests I wrote are the only format lock. A dedicated wire-format golden is missing.

## c) NOT STARTED (residue this session created or confirmed)

1. **CI wiring** for `check-csp` (cheap; chromium from nix) and `check-eventcatalog` (needs npm network — nightly candidate). Both are flake apps only.
2. **README / AGENTS quick-reference rows** for `check-csp` + `check-eventcatalog`.
3. **Tag wave for this session's unpublished symbols** — encryption (keys + v2 envelope), snapshot (`NewRewritingTransformedStore`), cmd/cqrs-lint (analyzer/suppression/doctor), catalog. **`storage/go.mod` currently carries `replace => ../encryption` and `=> ../snapshot`** (the documented unpublished-sibling pattern; tag-release strips at cut) — a pin sweep + strip is REQUIRED at the next wave.
4. ~~**Push** — 5 commits ahead of origin; prior session's "owner word required" still stands.~~ resolved — master pushed and in sync with origin (2026-09-06 evening).
5. **Full `#verify` / vulncheck / coverage / duplication battery.**
6. **Envelope v2 consumer note** (V5-MIGRATION-GUIDE / FAQ): "old readers stay compatible; no action needed" is true but undocumented outside CHANGELOG.
7. **C040 #206** (projection-handler dead-case) — design note exists in IMPROVEMENT_IDEAS, untouched.
8. **Brutal-self-review HTML variant** — the skill's default output is a styled HTML report; the user's explicit instruction (markdown status file here) won. Noted, not done.

## d) TOTALLY FUCKED UP

1. **I shipped broken code to disk twice, deliberately.** (a) `doctor_json.go` was written with a literal placeholder function named `report-disabled-list` (hyphens — not even legal Go) because I hadn't resolved a design detail; (b) the first `csp_browser_test.go` draft contained leftover scaffolding (`logged` mux, invented `catalogRegistryForCSP`/`catalogServiceID` helpers that referenced types I never defined). Both caught immediately by build, both cost a full rewrite cycle. Writing "fix it below" placeholders is a habit from throwaway REPLs, not production files.
2. **I guessed API signatures instead of reading them.** `MarshalEnvelope` returns `(string, error)` — I wrote a one-value return TWICE (once in reencryptState, again after "fixing" the wrong thing). `snapshot.NewSnapshot` returns a value, not a pointer — same indirect-the-pointer mistake in two test files. `containsString` vs `slices.Contains` sat in the same file I later patched. `lsp_definition` exists for exactly this and I used it zero times this session.
3. **`t.Setenv` inside `t.Parallel()`** — panicked the test binary. This is a first-week-of-Go-testing constraint; the module's own tests demonstrate the correct pattern. Sloppy.
4. **I asserted error-message wording I invented** ("got 24 bytes") without reading `ValidateKey`'s actual message ("key is 24 bytes, need exactly 32"), then "fixed" it by matching reality to my guess. Should read-then-assert, never assert-then-adjust. Same class: guessed the d2-view DOM marker (`order-svc`), failed, guessed again (`Architecture Diagram`) — one look at `render.go` would have gotten it first try.
5. **A mechanical refactor failed to compile.** The `collectDirExports` three-value conversion updated one error-return but not the `ReadDir` one — the compiler caught it, but it signals I was pattern-replacing without tracking every exit path of the function I restructured.
6. **The indentation-destroying multiedit.** In `snapshot_state.go` I replaced a nested block with wrongly-indented replacement text (and earlier hit the identical-text-in-two-functions trap in the same file — `decrypter := active` exists in both `rotatingRestore` and `reencryptState`). Two avoidable collisions in one file; the second fix (matching on a larger unique block) should have been the FIRST approach.
7. **I "fixed" a test by changing its expectation without ceremony.** `TestSQLSnapshotStore_Save` pinned the bytea binding; I flipped it to pin the text binding in a scripted sed. The change is correct — but the honest sequence is: recognize the pin as the BUG (it enshrined broken behavior), say so in the commit/CHANGELOG (done in CHANGELOG), and add the regression coverage (the PG test) in the same breath. The PG test came after; ordering was luck.
8. **The daemon destroyed this session's git archaeology.** My ~10 distinct tasks (release, golden regen ×3, three bug fixes, two new apps, doctor flags) are buried under ~12 "chore: auto-commit N file(s)" commits. I did nothing to prevent it (no interim manual commits with real messages, no daemon pause). Same complaint as the prior session's report; I repeated the outcome anyway.
9. **Edit-tool mod-time collision thrash.** The daemon touched files mid-edit at least twice (`main.go`, `envelope_test.go`, `csp_browser_test.go`, `snapshot_state.go`), and I burned 5+ round trips re-failing `edit` calls before switching to single python patches with in-script assertions. The adaptation came late.
10. **I found the storage compile break LATE.** I was 6 tasks in before the first storage build revealed master wasn't compiling (`containsString`). The AGENTS lesson "build immediately after deletes, before editing dependents" generalizes: run a repo-wide build BEFORE starting work, not as a side effect. A concurrent session presumably broke it and my early full build would have caught + attributed it cleanly.

## e) WHAT WE SHOULD IMPROVE

1. **Read-then-write discipline for signatures**: `lsp_definition`/`grep` the exact return signature before first use — would have prevented 4 of the compile failures this session.
2. **Never write placeholder/broken code to disk**: resolve the design detail in the message-to-self first; the file goes down complete or not at all.
3. **Lint AS you go, not at task end**: exhaustruct_v5/wsl_v5/tagalign placement quirks cost three lint→fix→re-lint cycles that a per-file lint would have folded into one.
4. **Run the workspace build at session START** — catches inherited breakage (containsString) before it contaminates attribution of your own work.
5. **Commit manually with real messages at task boundaries** — the daemon will sweep regardless, but interleaved real commits preserve archaeology. This is the second session bleeding on the same blade.
6. **Test-message assertions should quote the implementation's message** (copy from the error, don't invent), or assert on structural facts (errors.Is + length substring) only.
7. **For dual-shape wire formats, add the golden FIRST** (v2 output pin) so the format change is a reviewed diff, not just passing tests.
8. **Browser-gated tests pattern works** — worth extracting to a testutil (env-var-gated binary + dump-dom + console capture) instead of docserver-local.
9. **Check-then-fix external repos with the same rigor as local ones** — the go-retry flow was clean, but the consumer-pin check I skipped is part of that checklist.

## f) NEXT 50 (roughly impact-ordered; most need owner word or a quiet box)

**This workstream, immediate**

1. Push master (5 ahead) so CI validates the wave — owner word.
2. Run full `nix run .#verify` on a quiet box (module-scoped greens above are not gate-scoped).
3. Re-run cqrs-lint **self-lint** on the repo post-collector-change; triage the finding delta (const resolution + shared IsTest skip can shift C038/C040/E005/E007 counts).
4. GOWORK=off standalone build matrix over ALL modules — hunt for more `sqlstore`-style go.sum drift from the sync wave.
5. Tag wave to publish this session's symbols: encryption v4.4.0, snapshot v4.5.0, cmd/cqrs-lint, cmd/api-stability, catalog; **strip `storage/go.mod`'s `=> ../encryption` / `=> ../snapshot` replaces** + pin sweep in the same wave.
6. Post-wave: api golden refresh + standalone build matrix (per AGENTS multi-module tag-wave mechanics).
7. Wire `check-csp` into CI (nix chromium, no network needed beyond git).
8. Wire `check-eventcatalog` into CI as nightly (npm network; commit a `package-lock.json` from the exporter to make installs reproducible first).
9. AGENTS.md gotchas: envelope v2 wire format; pgx []byte→bytea vs JSONB binding; "release external sibling repo" checklist (go-retry v0.5.0 as the worked example).
10. README + AGENTS quick-reference rows for `check-csp` / `check-eventcatalog`.

**Encryption / snapshot follow-through**
11. Golden-pin the envelope v2 wire format (encrypt→Marshal output as a reviewed artifact).
12. Property test: v1-base64 ↔ v2-JSON decode symmetry over random ciphertexts.
13. encryption/README.md + doc.go: document key helpers and the v2 format (module docs untouched this session).
14. KeyProvider tier (env/file composite provider) from the bank-sync proposal — deliberately deferred; ROADMAP it.
15. Fuzz seeds for `DecodeKeyBase64`/`LoadKeyFromFile` (input parsers; existing fuzz harness extended).
16. MySQL-flavored twin of the PG encrypted-snapshot test (JSON column vs JSONB nuances).
17. Rewriting store: document upsert-idempotency under racing loads (two loads of the same stale snapshot double-write — safe via ON CONFLICT, worth a doc line + race test).
18. Rewriting store: expose rewrite counts (counter/hook) so operators can see migration convergence.
19. Dialect-level unit test: state binds as string for Postgres/SQLite/DuckDB/MySQL dialects (lock the bytea fix dialect-neutrally).
20. example/taskmanager (or getting-started): dogfood `LoadKeyFromEnv` where encryption is wired.

**cqrs-lint**
21. Golden/schema test for `doctor --format json` output shape (drift guard for CI consumers).
22. `--fix --dry-run --format json` combined smoke test in the module suite (currently manual smoke only).
23. Exit-code contract for `doctor --audit-suppressions` (non-zero when stale/unknown > 0) for CI gating.
24. C040 follow-up #206: projection-handler dead-case — needs the false-positive design first.
25. `-shuffle=on` in the cqrs-lint CI leg (proven cheap locally, carried over).
26. Encode the wsl_v5/exhaustruct_v5 nolint-placement quirks in `.golangci.yml` comments or AGENTS (they cost three fix cycles).
27. Suppression parser fuzz (carried over; input-handling code).

**Catalog / docs**
28. Recipes/skill refs: add envelope v2 + rotation write-back recipe + doctor JSON (close the §b-1 gap).
29. ~~docs-health pass: harvest this report + prior reports into TODO_LIST/FEATURES; FEATURES.md lacks the new capabilities (key helpers, write-back, check apps).~~ done (docs-health pass 2026-09-06 evening).
30. EventCatalog exporter: emit `package-lock.json` (reproducible npm install for check-eventcatalog).
31. ec-fixture: widen coverage (domains, channels, custom docs) toward real-consumer render paths; dedupe with cattest builders.
32. check-csp: also pin the CSP-OFF pages as byte-identical (protect the opt-in contract against drift).
33. Consider firefox as a second browser engine in the browser test (chromium-only blind spot).
34. docs/V5-MIGRATION-GUIDE (or FAQ): envelope v2 consumer note.
35. Extract browser-gated test harness to testutil if a second consumer appears (YAGNI guard: not before).

**Infra / hygiene (carried over + new)**
36. Daemon Q2 resolution (BuildFlow formatter/commit-message behavior) — this session added 12 more garbage-sweep commits of evidence.
37. Nightly "all CI jobs green or annotated" sentinel (carried over).
38. Single-source the file-size gate script (carried over).
39. Local-replace hygiene sweep: scripted pre-tag check now has two more directives to catch (storage).
40. Confirm no go-cqrs-lite module pins go-retry (skipped check; likely zero, but zero-evidence claims need the grep).
41. api-stability: unit tests for `collectModuleExports` (main-skip, internal-skip, rel-prefix) — meta-tests only today.
42. check-arch: consider a budget line asserting test-only sibling imports don't become production deps (storage/encryption pattern is legit but unpoliced).
43. Q3 decision memo (severity-in-minor) — now concretized by envelope v2, see question 3.
44. doc-check sqlstore-alias resolution (carried over; the concurrent session's untracked resolve.go/scan.go suggests it's in flight — coordinate, don't collide).
45. Annotate AGENTS contract #1 (350-line gate) interim reality (carried over; still red repo-wide).
46. Verify no `*_templ.go` exceeds 350 lines post ci.yml exclusion fix (carried over).
47. Consider npm/node version pin for check-eventcatalog in flake (survive channel bumps).
48. Add `check-csp` to the pre-push personal checklist (fast, high-signal for docserver changes).
49. Session-start repo-wide build (turn §e-4 into muscle memory; possibly a `verify-fast` leg).
50. Post-push: annotate the CI run for the infra-failure history so the sentinel (item 37) starts from a clean baseline.

## g) QUESTIONS (cannot answer from the repo myself)

1. **Tag-wave timing:** this session left `storage/go.mod` with `replace => ../encryption` and `=> ../snapshot` (the documented unpublished-sibling pattern, stripped at cut). Do you want the go-cqrs-lite tag wave (encryption v4.4.0, snapshot v4.5.0, cmd/cqrs-lint, cmd/api-stability, catalog, storage) cut **now** to republish properly, or held for the coordinated wave you mentioned earlier?
2. **CI policy for the new check apps:** `check-csp` needs nix chromium per CI run; `check-eventcatalog` needs npm network. Per-PR or nightly? And is a nightly "network tier" acceptable at all in this repo's CI budget?
3. **Q3 concretized — wire-format-in-minor:** the v2 envelope is a write-format change shipped in a v4.x minor (v1 reads stay compatible). Does your pending severity-in-minor policy treat this as "documented in CHANGELOG is enough" (option A), or do you want a compatibility knob (e.g. an escape hatch forcing v1-format writes for auditors) before it ships in a tag?
