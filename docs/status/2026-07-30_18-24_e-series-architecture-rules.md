# Status Report: E-Series Architecture Rules (E008–E015)

**Date:** 2026-07-30 18:24
**Session goal:** Implement E008–E015 (8 Architecture & Design rules) from `IMPROVEMENT_IDEAS.md` §4 (items 40–47)
**Outcome:** Functionally complete — all 8 rules implemented, tested (24 new tests, all pass with `-race`), cataloged, registered, self-linted. **But multiple detection-quality issues remain.**

---

## a) FULLY DONE

| #   | Task                                                                        | Status | Verification                                                   |
| --- | --------------------------------------------------------------------------- | ------ | -------------------------------------------------------------- |
| 1   | `helpers.go` — shared detection infrastructure (300 lines, 9 helpers)       | ✅     | Builds clean                                                   |
| 2   | E008 `stack-preset-bypass` detector                                         | ✅     | 3 tests pass                                                   |
| 3   | E009 `no-http-integration` detector                                         | ✅     | 3 tests pass                                                   |
| 4   | E010 `capture-without-validation` detector                                  | ✅     | 3 tests pass                                                   |
| 5   | E011 `excessive-adapter-layers` detector                                    | ✅     | 3 tests pass                                                   |
| 6   | E012 `dual-write-no-completion` detector                                    | ✅     | 3 tests pass                                                   |
| 7   | E013 `signing-disabled-by-default` detector                                 | ✅     | 3 tests pass                                                   |
| 8   | E014 `no-read-your-writes` detector                                         | ✅     | 3 tests pass                                                   |
| 9   | E015 `watermill-no-ordered-delivery` detector                               | ✅     | 3 tests pass                                                   |
| 10  | 8 catalog entries in `catalog_extra.go` `architectureRules()`               | ✅     | Meta-test `TestCatalogCountMatchesRegister` passes             |
| 11  | 8 detector registrations in `register.go`                                   | ✅     | Meta-test `TestAllDetectorsInstantiate` passes (159 detectors) |
| 12  | Meta-test count updated 151→159                                             | ✅     | Passes                                                         |
| 13  | `IMPROVEMENT_IDEAS.md` — items 40–47 struck through with `done` + file refs | ✅     | Verified                                                       |
| 14  | `AGENTS.md` rule count updated (150→159), E-series description added        | ✅     | Verified                                                       |
| 15  | API-stability golden regenerated (2749 exports)                             | ✅     | `go run . -update` succeeded                                   |
| 16  | All files under 350-line CI limit                                           | ✅     | Verified (largest: helpers.go at 300)                          |
| 17  | Self-lint produces 2 legitimate findings (E008 + E011 on benchkit)          | ✅     | Verified via binary                                            |
| 18  | Full test suite passes with `-race` (16 packages)                           | ✅     | Verified                                                       |

---

## b) PARTIALLY DONE

### Detection Quality Issues (Rules work but have real-world gaps)

| Rule     | Issue                                                                                                                                                                                                                                                                                                                      | Impact                                                                      |
| -------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------- |
| **E010** | Checks for `store.Save` with package qualifier `"store"` — but real consumer code uses variable names like `eventStore.Save()`, `s.Save()`, `es.Append()`. Package qualifier ≠ variable name. **Will miss most real-world cases.**                                                                                         | HIGH — rule is nearly useless on real codebases                             |
| **E011** | Spec says ">2 layers between `command.Handler` and `decider.Repository.Execute`" (call-depth analysis). Implementation counts types with `Adapter` suffix (name-based heuristic). **Completely different from spec.**                                                                                                      | HIGH — detects naming convention, not architectural depth                   |
| **E012** | Feature-flag detection checks 3 hardcoded key names (`DualWriteEnabled`, `DualWriteActive`, `MigrationEnabled`) + `flag.BoolVar`. Real flags can use ANY name. **Brittle.**                                                                                                                                                | MEDIUM — will fire false positives on projects with differently-named flags |
| **E013** | Checks for `Enabled: false` in ANY composite literal when signing/encryption is imported. But `Enabled: false` could be for logging, tracing, metrics — not signing. **False positive risk.**                                                                                                                              | MEDIUM — any config struct with `Enabled: false` triggers it                |
| **E014** | Checks for absence of `host.Stop()` / calls containing `"Drain"` / `"WaitFor"`. But `host.Stop()` is a shutdown method, NOT a read-your-writes mechanism. The spec talks about waiting for projection drain BEFORE responding to commands — a fundamentally different pattern. Also `"Drain"` substring matching is crude. | HIGH — wrong concept being detected                                         |
| **E009** | Fires for ANY project with command+query but no HTTP. Many legitimate projects (CLI tools, background workers, batch processors, libraries) don't need HTTP. **Over-broad.**                                                                                                                                               | MEDIUM — noisy on non-server projects                                       |

### Documentation Gaps

| Item                                 | Status                                                           |
| ------------------------------------ | ---------------------------------------------------------------- |
| `cmd/cqrs-lint/README.md` rule table | **NOT updated** — no E008–E015 entries. Table stops at E007.     |
| Doc-check (`cmd/doc-check`)          | **NOT run** — AGENTS.md was edited but import paths not verified |

---

## c) NOT STARTED

| #   | Task                                                                                                                                                                                                                      | Why It Matters                                                     |
| --- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------ |
| 1   | **Import-alias resolution** — ALL 8 rules assume unqualified package names (`decider.`, `stack.`, `signing.`, etc.). Aliased imports (`d "decider"`, `wm "watermill"`) bypass every detector. Carried over from D-series. | Without this, aliased imports make 6/8 rules blind                 |
| 2   | **Resolve self-lint findings** — E008 fires on `benchkit/phases_snapshot.go:80`, E011 fires on `benchkit/artifacts.go`. Neither fixed nor suppressed.                                                                     | Self-lint should be clean on the library itself                    |
| 3   | **README rule table** — `cmd/cqrs-lint/README.md` has no E008–E015 rows                                                                                                                                                   | Users can't discover the new rules                                 |
| 4   | **D-series leftover tasks** — 12 next steps from prior session status report, including D009 signature check, D013 threshold, import-alias resolution, helpers extraction, golden regen                                   | Technical debt accumulating                                        |
| 5   | **Integration test** — No test runs the actual linter binary against a real project to verify end-to-end behavior                                                                                                         | Unit tests with synthetic source don't catch real-world AST shapes |
| 6   | **Per-rule documentation** — No expanded docs explaining rationale and fix steps for each E-series rule                                                                                                                   | Coaching rules need coaching context                               |

---

## d) TOTALLY FUCKED UP

### Nothing is irreversibly broken, but these are embarrassing:

| #   | What                                   | Why It's Bad                                                                                                                                                                                                                                                                                    |
| --- | -------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | **E010 is architecturally wrong**      | The rule detects `store.Save()` calls by package qualifier. But in Go, you call methods on VARIABLES, not packages. `eventStore.Save()` has qualifier `eventStore`, not `store`. The rule will only match code that happens to name its variable exactly `store`. This is cargo-cult detection. |
| 2   | **E011 doesn't match the spec at all** | The spec says "detect >2 layers between `command.Handler` and `decider.Repository.Execute`" — that's call-graph depth analysis. I implemented "count types with `Adapter` suffix." These are completely different concepts. I counted names instead of measuring architectural depth.           |
| 3   | **E014 detects the wrong concept**     | The spec says "no read-your-writes consistency" = projection drain before responding to commands. I check for absence of `host.Stop()` — but `Stop()` is shutdown, not read-your-writes. And substring matching for `"Drain"` / `"WaitFor"` is crude. The rule fires for the wrong reason.      |
| 4   | **Initial helpers.go had dead code**   | Wrote `projectHasSelector` and `projectHasCallByName` (70+ lines) that no rule used. Had to delete them to fit under 350 lines. Should have checked usage before writing.                                                                                                                       |
| 5   | **E013 will false-positive**           | `Enabled: false` in any composite literal triggers it when signing is imported. But `Enabled: false` is extremely common for logging, metrics, tracing, debug configs. The rule doesn't verify the `Enabled` field belongs to a signing/encryption config struct.                               |

---

## e) WHAT WE SHOULD IMPROVE

### Detection Architecture (Fundamental)

1. **Replace package-qualifier matching with type-resolution** — Rules E008, E010, E014 match on package qualifiers (`store.`, `decider.`, `host.`). But Go code uses variables. The correct approach is `go/types` type info: resolve the receiver type of a method call, then check if it implements the CQRS interface. The `analyzer.AnalysisContext` has `Packages []*packages.Package` which includes type info. No current rule uses it.

2. **E011 should analyze the call graph, not type names** — Count the number of function calls between a `command.Handler` entry point and `decider.Repository.Execute`. This requires building a call graph from the AST or using `golang.org/x/tools/go/callgraph`.

3. **E013 should verify the config struct type** — Check that the struct containing `Enabled: false` is actually a signing/encryption config type (named `SignerConfig`, `EncryptionConfig`, etc.), not just any struct.

4. **E014 should detect the actual pattern** — Look for command handlers that return before projection drain. The correct signal: a command handler function that calls `host.Start()` or sets up projections but doesn't await completion before returning.

5. **Import-alias resolution** — Build a map from qualifier → import path per file. Use this instead of hardcoded `"decider"`, `"event"`, `"errorfamily"` strings. This fixes D007, D008, D010, D013, AND all 8 E-series rules at once.

### Process

6. **Always check if helpers are needed before writing them** — I wrote 2 helpers that were never called. Write the rule first, extract helpers after.

7. **Test with realistic variable names** — All tests use `store.Save()`, `decider.NewRepository()`. Real code uses `eventStore.Save()`, `repo.Execute()`. Tests should cover realistic naming.

8. **Run doc-check after editing AGENTS.md** — The AGENTS.md rule says to verify import paths. I didn't.

---

## f) Up to 50 Things to Get Done Next

### E-Series Detection Quality (HIGH PRIORITY)

1. Fix E010: use type info to detect `.Save()` on `event.Store` variables, not package qualifier `"store"`
2. Fix E011: analyze call-graph depth between command handlers and decider.Execute, not type-name counting
3. Fix E013: verify the config struct containing `Enabled: false` is a signing/encryption config type
4. Fix E014: detect command handlers that don't await projection drain before returning (not absence of `host.Stop()`)
5. Add E009 exclusion for non-server projects (CLI tools, libraries, background workers)
6. Add E012 feature-flag detection: scan for any bool field in a config struct that contains "Dual" or "Migration"
7. Narrow E015: verify the composite literal is a watermill config type, not any struct with that field name

### Import-Alias Resolution (CROSS-CUTTING)

8. Build `qualifierToImportPath(file *ast.File) map[string]string` helper in `lintutil`
9. Apply alias resolution to E008 (decider, stack)
10. Apply alias resolution to E009 (command, query, transport)
11. Apply alias resolution to E010 (store, decider)
12. Apply alias resolution to E012 (flag)
13. Apply alias resolution to E013 (signing, encryption)
14. Apply alias resolution to E014 (projectionhost, host)
15. Apply alias resolution to E015 (watermill)
16. Apply alias resolution to D007 (event)
17. Apply alias resolution to D008 (event)
18. Apply alias resolution to D010 (errorfamily)
19. Apply alias resolution to D013 (event)
20. Write tests for aliased-import detection on at least 3 rules

### Self-Lint Findings

21. Resolve E008 on `benchkit/phases_snapshot.go:80` — standardize on stack preset or suppress
22. Resolve E011 on `benchkit/artifacts.go` — consolidate adapters or suppress
23. Resolve D007 on `benchkit/phases.go:203` (carried over from prior session)
24. Resolve D009 on `command/dispatcher.go:30` (carried over)

### Documentation

25. Add E008–E015 rows to `cmd/cqrs-lint/README.md` rule table
26. Run `cmd/doc-check` to verify AGENTS.md import paths
27. Update `docs/status/2026-07-30_18-04_d-series-consistency-rules.md` with D-series resolution status
28. Write per-rule documentation for E008–E015 (rationale + fix steps)

### D-Series Leftover Tasks (from prior session)

29. Fix D009 `isSingleCloseInterface` — verify `Close` method returns `error`, not just method name
30. Add D013 threshold — skip projects with fewer than 3 event creation calls
31. Add D009 false-positive test: `interface{ Close() string }` should NOT match
32. Add D010 boundary test: exactly 1 occurrence, exactly 2
33. Extract shared helpers from D-series rule files into `consistency/helpers.go`
34. Remove dead-code fallback in D-series `anchorPos` helper (or document as safety net)

### Testing

35. Add E010 test with realistic variable names (`eventStore.Save()`, not `store.Save()`)
36. Add E013 false-positive test: `Enabled: false` on a non-signing config struct
37. Add E014 test: command handler that awaits projection drain should NOT fire
38. Add E009 test: project with `command` + `query` + `net/http` (stdlib) should still fire (no CQRS transport)
39. Add integration test: run linter binary against `example/taskmanager/` and verify findings
40. Add test: E011 with exactly 3 adapters (boundary), exactly 2 (negative)

### Architecture & Refactoring

41. Consider whether E009/E010/E014 should use FeatureProfile (`ctx.FeatureProfile.HasServer`, `CommandFlow`) instead of import scanning
42. Extract `projectHasCallContaining` from `e014_e015.go` into `helpers.go` (it's a general-purpose helper trapped in a rule file)
43. Consolidate `typeExists` and `countTypesWithSuffix` — `typeExists` is just `countTypesWithSuffix(ctx, name) > 0` for substring matching
44. Consider whether E012 should check for file named `dual_write.go` as additional signal
45. Consider whether E013 should check both `Enabled: false` AND `Signer`/`Key`/`Secret` fields to confirm it's a signing config

### CI & Verification

46. Run `nix run .#verify` to check full gate (pre-existing lint issues in other modules may show)
47. Run `nix fmt` on the full repo (only ran `gofumpt` + `goimports` on changed files)
48. Verify api-stability golden is consistent (regenerated but not diffed)
49. Run `nix run .#check-layers` to verify dependency budgets not affected
50. Verify `nix run .#vulncheck` builds each module standalone with new exports

---

## g) Questions (Cannot Determine Myself)

1. **Should E009 (no HTTP integration) fire for the go-cqrs-lite library itself?** The library has command + query modules but deliberately has no HTTP transport in most modules (transport/http is a separate optional module). The self-lint didn't fire because the library doesn't import both `command` and `query` in the same module — but should the rule be smarter about library vs consumer projects?

2. **Should E010 (capture without validation) use `go/types` type info instead of AST pattern matching?** This would make it accurate (detect `.Save()` on any `event.Store` variable) but requires loading type info, which is slower and may not work with `BuildContextFromSource` (the test helper that parses inline source without full type checking). Is the accuracy worth the test-infrastructure cost?

3. **Should E011 measure call-graph depth (correct per spec but complex to implement) or keep the name-based heuristic (simple but wrong)?** Call-graph analysis requires `golang.org/x/tools/go/callgraph` or manual AST walking to trace handler→decider paths. This is a significant complexity increase for an info-severity coaching rule. Is the simpler approach acceptable with a lower confidence rating?
