# Status Report — cqrs-lint DiscordSync Feedback Triage (2026-07-17 03:38)

> Session scope: triaged and acted on every item in
> `docs/feedback/2026-07-16_DiscordSync_cqrs-lint-feedback.md`. Four linter
> rules fixed, one doc updated, five regression tests added. No work outside
> cqrs-lint was performed this session.

---

## a) FULLY DONE

| #   | Item                                                                                                                                                                                                                                                                                                                                                  | Files                                                                           | Tests added                                                                                                                                       |
| --- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | **C001 closure-escape fix** — `txVarEscapesToArg` skips the dangerous false positive where the tx variable is passed to a callback that contractually owns the commit. The previous `TestC001_DetectsMissingCommit` encoded the false positive; replaced with a genuine missing-commit case.                                                          | `correctness/c001.go`, `correctness/tx_helpers.go`, `correctness/rules_test.go` | `TestC001_NoFindingWhenTxEscapesToCallback`, rewritten `TestC001_DetectsMissingCommit` + `TestC001_NoFindingForProperCommit`                      |
| 2   | **C008 money corroboration** — split money fields into strong (amount/price/cost/balance/fee — fire alone) and weak (value/total/charge/payment/salary — need money struct/package name). Added named-struct + anonymous-struct passes with a `handled` set to avoid double-counting.                                                                 | `correctness/c008.go`, `correctness/rules_test.go`                              | `TestC008_NoFindingForValueInObservabilityStruct`, `TestC008_FindingForValueInMonetaryStruct`                                                     |
| 3   | **D005 version-token regex** — replaced `HasPrefix("v") && len >= 3` with `^v\d+\.\d+`, which rejects prose words ("via", "version") and bare major versions ("v3") while keeping real semver references (v4.0.0, v4.0.x).                                                                                                                            | `consistency/d003_d005.go`, `consistency/new_rules_test.go`                     | `TestD005_NoFindingForProseWordVia`                                                                                                               |
| 4   | **A005 broadcast vs projection** — `classifyCallbackBody` inspects the SubscribeAll callback body and suppresses when it contains broadcast/notify calls (Notify/Broadcast/Send) and no persistence calls (Save/Set/Upsert/Insert/...). Conservative: empty callbacks, named-function subscribers, and bodies with any persistence signal still flag. | `api/a005.go`, `api/new_rules_test.go`                                          | `TestA005_NoFindingForBroadcastFanOut`, strengthened existing `TestA005_DetectsSubscribeAllWithoutProjectionHost` to use a real `store.Save` call |
| 5   | **Feedback doc updated** — added a resolution banner at the top and a full "Resolution Log (2026-07-17)" table at the end classifying all 7 suggested improvements as DONE / PARTIAL / already-fixed.                                                                                                                                                 | `docs/feedback/2026-07-16_DiscordSync_cqrs-lint-feedback.md`                    | —                                                                                                                                                 |

### Verification

- `go test -tags "goexperiment.jsonv2" ./...` (in `cmd/cqrs-lint`) — all packages pass.
- `go vet` on the three changed rule packages — clean.
- `nix fmt` — applied; only reformatted the two pre-existing dirty files (`doctor.go`, `feature_profile_test.go`); my rule files were already formatted.
- Meta-test `TestAllDetectorsInstantiate` + `TestCriticalDetectorsInstantiate` — pass (guards against the "doesn't compile" class of regression referenced in AGENTS.md commit `b3931503`).

---

## b) PARTIALLY DONE

| #   | Item                                     | What's done                                                                                                                                            | What's missing                                                                                                                                                                                                                                                                                                                                                                                       |
| --- | ---------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------ | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | **D002 external API contract detection** | Confirmed the rule exists and the false-positive classification is documented. D004 (the project-wide variant) was already removed in a prior session. | The rule still fires on snake_case fields that mirror an external API (Discord/Stripe/GitHub). Needs a config allowlist (e.g. `.cqrs-lint.json` → `external-api-packages` or struct-tag opt-out) so consumers can mark "these snake_case tags are intentional." This was the noisiest finding in the DiscordSync report (18 info deductions) and is the biggest remaining source of false positives. |
| 2   | **Feedback doc historical drift**        | Added a resolution banner pointing readers to current state.                                                                                           | The body of the doc still describes A016, E006, and D004 as open/false-positive even though A016 + E006 are already fixed via FeatureProfile/registry and D004 is deleted. A full rewrite of those sections (or strikethroughs) would be cleaner — deferred to avoid rewriting history the reader may still want.                                                                                    |

---

## c) NOT STARTED (from the feedback doc / observed this session)

| #   | Item                                                        | Why not                                                                                                                                                                                                                                                                                                                                                                                                                                                                                |
| --- | ----------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | **B007 per-route middleware awareness**                     | DiscordSync flagged 8/7 consecutive `mux.HandleFunc` with different middleware wrapping as a style preference. B007 currently counts consecutive Register/Handle/Subscribe calls. It does NOT fire on `HandleFunc` (it only matches `Register`/`RegisterTyped`/`Handle`/`Subscribe`), so DiscordSync's B007 finding was already a non-issue in current code — **no action needed unless the rule is widened to include `HandleFunc`**, which would re-introduce the noise. Left as-is. |
| 2   | **A009 stack/ preset guidance for shared-DB architectures** | DiscordSync intentionally shares one `*sql.DB` across CQRS + relational reads, incompatible with stack/ presets. A009 still fires. Could be suppressed when a `storage/` (non-stack) import + relational reads are detected, but this is a legitimate Info-level nudge, not a false positive. No change made.                                                                                                                                                                          |
| 3   | **D003 (logging library) review**                           | Not mentioned in DiscordSync feedback; noticed while reading `d003_d005.go`. The rule looks correct but has no test for the single-library happy path beyond the "no finding" implicit case. Low priority.                                                                                                                                                                                                                                                                             |

---

## d) TOTALLY FUCKED UP (honest self-critique)

1. **D002 is the real unfinished business.** I called it "PARTIAL" and moved on, but it's the single biggest source of false-positive noise in the entire DiscordSync report (18 of 34 findings). "Needs a config allowlist" is a deferral, not a fix. If DiscordSync re-runs the linter today, D002 still spams them with 18 deductions (-18 points off the health score). This is the gap I should have closed or at least prototyped.
2. **First C008 rewrite used non-existent helpers** (`posFromFilePosition`, `ctxPos`) — I wrote a convoluted two-pass structure with a fake `ctxPos` shim before throwing it away and rewriting cleanly. Wasted a tool call. Should have written the clean version first.
3. **D005 over-tightened on first attempt** — bare `v3`/`v4` regex `^v\d+` broke the existing `TestD005_NoFindingForMigrationArrow` test. Had to broaden to `^v\d+\.\d+` (require major.minor). The first attempt was lazy thinking — I didn't consider that "v3" appears in real migration prose.
4. **No end-to-end test against the actual DiscordSync repro.** All verification was unit-level. I did not clone DiscordSync or build a synthetic project that reproduces the original 34 findings to prove the health score moved. The unit tests prove each rule behaves correctly in isolation, but I haven't demonstrated that the **sum** of fixes reduces false positives on a real consumer.
5. **A005 `isBroadcastSignal` list is hand-curated and narrow.** It only matches exact method names (`Notify`, `Broadcast`, `Send`, ...). DiscordSync's actual code uses `notifier.Notify()` and `broker.Broadcast()` — both covered. But any consumer using `Publish`, `Emit`, `Forward`, `Dispatch`, or a custom fan-out method will still get flagged. The heuristic is better than before but still pattern-matches names rather than semantics.

---

## e) WHAT WE SHOULD IMPROVE (quality observations this session)

### Linter architecture

1. **No integration/fixture test for false-positive suppression across rules.** Each rule has unit tests, but there's no golden "run the linter on a synthetic DiscordSync-like project and assert the health score" test. The `meta_test.go` only checks detectors instantiate. A fixture-based regression suite would catch interaction bugs and give confidence that reported false-positive rates actually drop.
2. **FeatureProfile is good but under-used.** A016 and A012 already consult it; C008, D002, and A005 could too. For example: a project with `FeatureProfile.Store == StoreCustom` and no monetary domain signals could suppress C008 entirely. The profile is centralized context that most rules don't read yet.
3. **Health score is opaque and easy to game.** 18 D002 info findings = -18 points, same as 2 critical bugs. The scoring weights (`-10` critical, `-2` warning, `-1` info) make info-level false positives disproportionately punishing. A weighted-by-confidence or false-positive-discounted score would be more honest.
4. **No suppression feedback loop.** DiscordSync chose NOT to add `//cqrs-lint:ignore` comments because "we prefer the linter improve." That's the right instinct, but the linter has no way to learn from suppressed findings — no telemetry, no "commonly suppressed rules" report, no "this rule has a 60% suppression rate, maybe reconsider the heuristic" dashboard.
5. **Rules don't share AST helpers as much as they could.** `txVarEscapesToArg`, `classifyCallbackBody`, and `isMoneyStructName` are all pattern-matchers that could be reusable utilities. Each rule file re-implements `ast.Inspect` + `SelectorFromExpr` boilerplate.

### Process

6. **The feedback doc drifted from reality.** A016, E006, D004 were already fixed before this session, but the doc still described them as open. Consumer feedback needs a "re-verify date" on each row, not just a top-level banner.
7. **No CHANGELOG entry yet.** I fixed 4 rules but haven't written `CHANGELOG.md` entries. Consumers won't know the false positives are fixed unless we publish it.
8. **No re-tag of cqrs-lint.** The fixes aren't in a release. DiscordSync is pinned to `v4.0.0` (`c34dd604`); they'd need to upgrade to a new tag to benefit.

---

## f) Up to 50 things we should get done next

Ranked roughly by impact × effort.

### High impact — close the feedback loop

1. **Write CHANGELOG entries** for C001/C008/D005/A005 fixes.
2. **Tag cqrs-lint v0.2.0** (or next patch) so DiscordSync can upgrade.
3. **D002 config allowlist** — add `.cqrs-lint.json` → `external-api-packages: ["discord"]` that suppresses D002 for structs whose package matches.
4. **D002 struct-tag opt-out** — honor `//cqrs-lint:ignore(D002)` on the struct or a `//cqrs-lint:external-api` line comment.
5. **Integration fixture test** — build a synthetic `testdata/discordsync-like/` project and assert the health score + finding count drops after the fixes.
6. **Re-verify against real DiscordSync** — clone the repo, run the new linter, confirm the 4 fixed rules no longer fire.

### Medium impact — linter correctness & coverage

7. **C001: also detect `tx.Exec`/`tx.QueryRow` as tx-use signals** so a function that uses tx but never commits and never passes it to a callback still flags (currently `writeNoCommit` is caught only because it has `return nil`).
8. **C008: consult `FeatureProfile`** — if no monetary domain signal anywhere in the project, downscale C008 to Info or suppress.
9. **A005: widen broadcast signal list** — add `Publish`, `Emit`, `Forward`, `Dispatch`, `WriteTo`, `Flush` after auditing what real consumers call.
10. **A005: inspect named-function subscribers** — when `bus.SubscribeAll(myFunc)` passes a bare identifier, resolve the function decl and classify its body (currently conservatively flagged).
11. **D005: also check `go.sum` / lockfile references**, not just prose docs.
12. **D003: add a positive test** for the single-library (no-finding) case.
13. **B007: decide policy on `mux.HandleFunc` chains** — either widen to include them (and accept DiscordSync-style noise) or document that B007 only covers CQRS handler registration.
14. **A009: suppress when `storage/` (non-stack) + relational reads detected** — shared-DB architecture is a valid reason to skip stack/ presets.
15. **Health score: discount Info findings** or cap their total contribution (e.g. Info deductions can't exceed 20% of the score).
16. **Health score: weight by confidence** — a `ConfidenceLow` finding shouldn't deduct the same as `ConfidenceHigh`.
17. **Add a "false-positive reporter" mode** — `cqrs-lint --fp-suspects` that flags low-confidence findings for human triage.
18. **CQRSRegistry: track `command.Publisher` / `event.Bus` vs `command.Dispatcher`** as distinct types so A016 can verify the dispatcher type, not just method name.
19. **E006: add a test** that a SQL row struct named `*Candidate` is NOT registered as an emitted event (regression for the DiscordSync case).
20. **Rules share an `asthelpers` package** — extract `txVarEscapesToArg`, `classifyCallbackBody`, `exprReferencesIdent` into `pkg/analyzer/asthelpers.go`.
21. **C008: handle embedded structs** — currently only scans direct fields; an embedded `MoneyMixin{Value float64}` would be missed.
22. **C001: detect `pgx.BeginTx` / `sqlx.Beginx`** in addition to `database/sql` (selector-name based, already works, but add tests).
23. **D002: detect `json:"-"` omitempty edge cases** and skip those fields.
24. **Add property-based tests (rapid)** for C008 and D005 — generate random field names / doc lines and assert no panics + monotone behavior.
25. **Doctor command: print per-rule suppression counts** so consumers see which rules they're ignoring.

### Documentation & DX

26. **Rewrite the DiscordSync feedback body sections** for A016/E006/D004 to note they're already fixed (strikethrough or "[RESOLVED]" tags).
27. **Add a "Consumer Integration Testing" doc** — how to run cqrs-lint against your project in CI.
28. **Document the FeatureProfile flags** in the cqrs-lint README (currently only in `feature_profile.go`).
29. **Add examples of `.cqrs-lint.json` config** for each preset (local-cli, production, library, read-only).
30. **Document the suppression comment syntax** (`//cqrs-lint:ignore(RULE_ID)`) in the README — DiscordSync discovered it by reading source.
31. **Add a `cqrs-lint rules` subcommand** that prints all rule IDs + descriptions (currently you have to read `register.go`).
32. **SARIF output: include `suppress` suggestions** for each finding so GitHub code-scanning can show one-click suppressions.

### Code quality & maintainability

33. **`scanMoneyFields` allocates a slice per struct** — could append into a caller-owned slice to reduce GC pressure on large codebases.
34. **`extractCallbackFuncLit` in A005 duplicates logic** in `scanner_calls.go` — consolidate.
35. **`versionTokenRe` is package-level** — fine, but `extractCQRSVersion` re-scans every doc file; cache the compiled regex is already done, but cache the parsed go.mod version across calls.
36. **C008 anonymous-struct pass re-walks the whole AST** — could fold into pass 1 with a parent map.
37. **`isMoneyStructName` and `packageLooksMonetary` share keyword lists** — unify into one `moneyKeywords` set with a `contextKind` enum.
38. **Add benchmarks** for the four changed detectors to catch performance regressions on large consumer repos.
39. **`tx_helpers.go` now has 6 helpers** — consider splitting into `tx_begin.go` / `tx_commit.go` / `tx_escape.go` if it grows further.
40. **Lint the linter** — run `cqrs-lint` on `cmd/cqrs-lint/` itself (dogfooding).

### Strategic / bigger bets

41. **Type-information-aware rules** — load `go/types` and resolve receiver types so A016 can verify `command.Dispatcher` vs `event.Bus` definitively instead of name-matching.
42. **Cross-rule correlation** — if C001 fires and C009 fires on the same function, escalate severity.
43. **Auto-fix mode** — `cqrs-lint --fix` for safe fixes (C009 panic→error, D005 version bump in docs).
44. **Rule severity calibration via telemetry** — collect (opt-in) finding/suppression rates to recalibrate confidence thresholds.
45. **Projection-coverage report** — E006 as a gate: "these 3 event types have no projection; is that intentional?"
46. **Catalog integration** — cross-reference `catalog.Registry` with emitted events to detect undeclared event types.
47. **Migration assistant** — detect pre-v4 API usage (`event.Event` interface, old metadata API) and suggest migration.
48. **Stack-preset migration** — A009 auto-generates the diff to adopt a stack/ preset when feasible.
49. **Multi-module workspace support** — run cqrs-lint across a `go.work` monorepo and de-duplicate findings per module.
50. **Versioned rule sets** — pin rule behavior to a cqrs-lint version so consumers don't get surprise new findings on upgrade.

---

## g) Questions I CANNOT figure out myself

1. **Should D002 be suppressed via config (`.cqrs-lint.json` allowlist of external package names) or via an in-source marker (`//cqrs-lint:external-api` on the struct/field)?** The config approach is cleaner for "Discord's whole API is snake_case"; the in-source marker is more precise but adds noise to consumer code. DiscordSync explicitly said they dislike inline suppressions, which hints at config — but I want your call before building it.
2. **Do you want a new cqrs-lint tagged release (v0.2.0) now, or batch these fixes with the next chunk of work (e.g. D002) into one release?** Tagging now lets DiscordSync upgrade immediately; batching reduces release churn but delays the benefit.
3. **Is there a second consumer besides DiscordSync whose feedback I should pull in before over-fitting the heuristics to one project?** C008's keyword lists and A005's broadcast-signal list are now shaped by a single consumer; a second datapoint would calibrate them. I can't tell from inside this repo whether you have other known consumers.
