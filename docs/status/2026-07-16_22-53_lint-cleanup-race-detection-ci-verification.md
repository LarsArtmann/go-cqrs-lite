# Status Report: Lint Cleanup, Race Detection & CI Verification

**Date:** 2026-07-16 22:53
**Session scope:** Post-buildflow lint fixes, race detection, canonical CI gate verification
**Commit range:** `bd849d1c` → `fe8f27e6` (external auto-commit) + 2 unstaged files
**Overall status:** All CI gates green. 2 unstaged lint fixes remain.

---

## a) FULLY DONE

### 1. Baseline Verification

- Full workspace build (`go build -tags "goexperiment.jsonv2" ./...`) — passes
- Full test suite (~80 packages) — passes
- Confirmed previous session's work was correct and committed by external buildflow process (commits `3a6f1f3a` + `fe8f27e6`)

### 2. Race Detector (`go test -race`)

- Ran across all 50+ modules in two batches (main + cmd/prometheus/grpc/kvstore/bench/turso/postgres)
- **Result: ZERO data races detected**
- This was the first time `-race` was run in either session — critical verification gap now closed

### 3. Lint Fixes (5 issues across 3 files)

| Rule               | File                                            | Fix                                                                                                                  |
| ------------------ | ----------------------------------------------- | -------------------------------------------------------------------------------------------------------------------- |
| `gochecknoglobals` | `cmd/cqrs-lint/pkg/rules/api/a011_a014_a017.go` | Removed global `eventPayloadSuffixes` var, restored `looksLikeEventPayload()` helper (matching pre-buildflow design) |
| `err113`           | `decider/strict_apply.go`                       | Exported `ErrStrictApplyUnknownType` sentinel error, wrapped with `%w` instead of dynamic `fmt.Errorf`               |
| `errname`          | `decider/strict_apply_test.go`                  | Renamed `errApplySentinel` → `errApplySentinelError`, added `errors.Is(err, ErrStrictApplyUnknownType)` assertion    |
| `exhaustruct`      | `projectionhost/host.go`                        | Explicit `resetConfig{purgeDLQ: false}` instead of empty struct literal                                              |
| `varnamelen`       | `projectionhost/worker_drain.go`                | Renamed `cp` → `checkpoint` (used across 15+ lines of scope)                                                         |

**Lint result:** `nix run .#lint` → **0 issues across all 47 modules** (EXIT_CODE=0)

### 4. Canonical CI Verification

- `nix run .#build` → EXIT_CODE=0
- `nix run .#vet` → EXIT_CODE=0
- `nix run .#test` → EXIT_CODE=0 (all 80+ packages pass)
- `nix run .#lint` → EXIT_CODE=0
- `transport/grpc` GOWORK=off test → passes
- `cmd/doc-check` → 880 references valid across 34 packages

### 5. `json.Marshal(map[...])` Audit

- Found 9 remaining calls across test files
- **9 are safe:** 6 in benchmark/scale tests (no golden comparison), 2 in `pg_bus_test.go` (round-trip JSON, key order irrelevant), 1 in benchmark helper
- No additional determinism fixes needed

### 6. `marshalSortedMap` Duplication Evaluation

- Duplicated in `otel/golden_test.go` and `watermill/golden_test.go` (identical 16-line implementations)
- **Decision: Do NOT extract to `testutil`** — would add 6+ transitive CQRS deps (command, event, id, dispatcher, metadata, codec) to two modules that currently have zero CQRS dependencies. The project's own rule says "extract at 3+ duplicates." Two copies is acceptable.

---

## b) PARTIALLY DONE

### Unstaged Lint Fixes (2 files)

- `projectionhost/host.go` — `exhaustruct` fix (`resetConfig{purgeDLQ: false}`)
- `projectionhost/worker_drain.go` — `varnamelen` fix (`cp` → `checkpoint`)
- These are the only 2 unstaged files. All other lint fixes were committed by the external buildflow auto-commit process during the session (`fe8f27e6`).
- **Not committed because user hasn't asked to commit.**

---

## c) NOT STARTED

- AGENTS.md update (no new patterns/gotchas discovered this session that belong there — `ErrStrictApplyUnknownType` is self-documenting API)
- `marshalSortedMap` extraction to testutil (deliberately skipped — see section a.6)
- `hierarchical-errors` adoption decision (1214 findings from buildflow — needs user input)
- Flaky test `TestSQLTimerStore_IntegrationWithScheduler` stabilization (pre-existing, not our change)

---

## d) TOTALLY FUCKED UP

### 1. Edit Tool Corruption of `a011_a014_a017.go`

- My first `multiedit` on this file produced corrupted output: the `slices` import was moved to a wrong position, and the `slices.Contains()` call (zero args, broken from buildflow) was NOT fixed — my edit replaced the correct `slices.ContainsFunc` call instead.
- **Root cause:** The file on disk had been committed with the correct `slices.ContainsFunc` code (by the previous session's staged changes → external commit), but I was editing against the wrong baseline. The `multiedit` `old_string` matched a stale version.
- **Fix:** `git checkout HEAD -- <file>` to restore the committed version, then re-applied the lint fix cleanly with correct `old_string` matches.
- **Lesson:** Always verify `git status` and `git diff HEAD` before editing to know whether HEAD already has the fix. The file was committed by external process mid-session.

### 2. Failed to Notice External Auto-Commit

- During the session, an external process (buildflow) committed my changes as `3a6f1f3a` and `fe8f27e6`. I didn't notice this until the final verification phase when `git status` showed only 2 files instead of the expected 77.
- This caused the corrupted edit above (editing against stale working tree state).
- **Lesson:** In environments with auto-commit hooks/agents, check `git log` before every edit batch to detect external commits.

---

## e) WHAT WE SHOULD IMPROVE

### Code Quality

1. **`ErrStrictApplyUnknownType` is now exported** — consumers can `errors.Is()` it. But the doc comment on `StrictApply` still says "returns an error" without mentioning the sentinel. Update the doc comment to mention `ErrStrictApplyUnknownType`.
2. **`looksLikeEventPayload` allocates a slice on every call** — fine for a linter (not hot path), but if this were production code, the suffixes should be a package-level var. The lint rule `gochecknoglobals` conflicts with performance here. Acceptable tradeoff for a linter.
3. **`marshalSortedMap` duplication** — 2 copies of identical code in otel + watermill. If a 3rd module needs it, extract to a zero-dependency helper package (NOT testutil which pulls CQRS deps).
4. **Flaky test `TestSQLTimerStore_IntegrationWithScheduler`** — uses `t.Parallel()` + 20ms poll interval + 3s timeout. Fails intermittently under heavy parallel CPU saturation (50-module full suite). Should either increase timeout, reduce parallelism, or make the test serialization-safe.

### Process

5. **No `go test -race` in CI** — the race detector was clean this time, but the canonical `nix run .#test` doesn't include `-race`. There IS a `test-race` app in flake.nix but it's not in the CI pipeline. This is a gap.
6. **External auto-commit during sessions** — buildflow committed mid-session, causing edit corruption. Need to either disable auto-commit during active sessions or check `git log` more frequently.
7. **`goexperiment.jsonv2` build tag is systemic risk** — the entire build/test/lint cycle depends on a Go experiment flag. If Go 1.27 changes the behavior or removes the experiment, the entire build breaks. The project should track this closely.
8. **JSON v2 map ordering non-determinism** is documented in the previous status report but not in AGENTS.md. This belongs in AGENTS.md under "Testing" as a known gotcha.

---

## f) Up to 50 Things We Should Get Done Next

### High Priority (blocking/stability)

1. **Commit the 2 unstaged projectionhost lint fixes** (`host.go`, `worker_drain.go`)
2. **Update `StrictApply` doc comment** to mention `ErrStrictApplyUnknownType` and `errors.Is` usage
3. **Add `go test -race` to CI pipeline** (flake.nix `test-race` app exists but isn't in CI)
4. **Stabilize `TestSQLTimerStore_IntegrationWithScheduler`** — increase timeout to 10s or remove `t.Parallel()`
5. **Document JSON v2 map ordering gotcha in AGENTS.md** under Testing section

### Medium Priority (code quality)

6. **Replace remaining `json.Marshal(map[string]string{})` in `storage/benchmark_test.go`** with typed structs (4 calls) — not a determinism bug, but inconsistent with the pattern established this session
7. **Replace `json.Marshal(map[string]any{...})` in `pg_bus_test.go`** with typed structs (2 calls) — same reasoning
8. **Replace `json.Marshal(map[string]string{...})` in `integration/scale_bench_*.go`** with typed structs (3 calls)
9. **Create a zero-dependency `jsondt` package** for `marshalSortedMap` if a 3rd consumer appears — or add it to `codec/` which already has JSON dependencies
10. **Audit all `//nolint` directives** — verify they're still needed after lint config changes
11. **Check if `gochecknoglobals` is too strict** for linter/test tooling where package-level vars are idiomatic

### Architecture & Dependencies

12. **Evaluate `hierarchical-errors` (1214 findings)** — buildflow flagged this. Decide whether to adopt as project standard.
13. **File bug report for buildflow `go-auto-upgrade`** — it committed broken `slices.Contains()` (zero args) in commit `bd849d1c`. The tool needs a post-fix compilation gate.
14. **Verify `goexperiment.jsonv2` compatibility plan** — check Go 1.27 release notes for graduation timeline
15. **Run `nix run .#check-layers`** — dependency budget check not run this session
16. **Run `nix run .#check-arch`** — architecture check not run this session
17. **Run `nix run .#check-file-size`** — file size limit check
18. **Run `nix run .#coverage`** — coverage report not generated this session

### Testing

19. **Add property-based test for `StrictApply`** — verify `errors.Is(err, ErrStrictApplyUnknownType)` holds for arbitrary unknown event types
20. **Add golden test for `StrictApply` error message format** — prevent accidental message changes
21. **Run full test suite 5x in sequence** to verify determinism of all golden tests (not just the 3x from previous session)
22. **Add `-race` flag to example/taskmanager tests** — the example app has concurrent projection host logic
23. **Test with `-tags "goexperiment.jsonv2"` disabled** — verify the project still builds without the experiment flag (forward compatibility)
24. **Run `go test -shuffle=on`** to detect test-ordering dependencies

### Documentation

25. **Update AGENTS.md Testing section** with JSON v2 determinism gotcha + `marshalSortedMap` pattern
26. **Update `decider/README.md`** with `ErrStrictApplyUnknownType` and `errors.Is` usage example
27. **Write ADR for JSON v2 adoption** — why `goexperiment.jsonv2`, what changes, when to migrate
28. **Update `docs/adr/` index** if new ADRs are added
29. **Update `.agents/skills/go-cqrs-lite/SKILL.md`** if any consumer-facing API changed (ErrStrictApplyUnknownType is new exported API)
30. **Run `cmd/doc-check`** after any SKILL.md updates

### CI/CD

31. **Add `nix run .#test-race` to CI** as a separate job
32. **Add `nix run .#check-layers` to CI** if not already present
33. **Add `nix run .#check-arch` to CI** if not already present
34. **Add a "golden determinism" CI check** — run golden tests 3x, fail if any produce different output
35. **Add commit-msg hook** to prevent broken code from being committed (buildflow's `slices.Contains()` bug)
36. **Verify CI yaml** matches the flake.nix commands being run locally

### Cleanup

37. **Remove the `slices` import** from `a011_a014_a017.go` if it's still there (verify — should have been removed by the lint fix)
38. **Audit all `var eventPayloadSuffixes`-style globals** in cqrs-lint — the `gochecknoglobals` rule may flag more
39. **Check if `projectionhost.resetConfig` can avoid `exhaustruct`** by using functional options pattern more idiomatically
40. **Verify `checkpoint` rename in `worker_drain.go`** didn't miss any references (grep for `cp.` in the file)
41. **Clean up the `docs/status/` archive** — move old reports to archive/
42. **Remove duplicate `marshalSortedMap`** if a shared location is agreed upon

### Future Features

43. **Consider a `jsondt` test helper package** (zero deps) for deterministic JSON map marshaling in tests
44. **Add `StrictApply` variant that returns the event type** in the error for structured error handling
45. **Add a cqrs-lint rule** that detects `json.Marshal(map[...])` in golden tests and warns about non-determinism
46. **Add a cqrs-lint rule** that detects missing `errors.Is` checks after `StrictApply`
47. **Consider stabilizing `goexperiment.jsonv2`** by feature-detecting instead of using a build tag
48. **Benchmark the race detector overhead** — if <2x, enable `-race` for all CI test runs
49. **Add `go test -bench=. -count=5` to CI** to detect performance regressions
50. **Review all `t.Parallel()` tests** for timing sensitivity — any test with `time.After` + `t.Parallel()` is a flaky test waiting to happen

---

## g) Questions (cannot figure out myself)

### 1. Should the 2 unstaged lint fixes be committed?

The `projectionhost/host.go` (`exhaustruct`) and `projectionhost/worker_drain.go` (`varnamelen`) changes are the only unstaged work. Everything else was auto-committed by the external buildflow process. Should I commit these as a separate "fix: resolve remaining lint issues in projectionhost" commit?

### 2. Is `hierarchical-errors` (1214 findings) something to adopt?

Buildflow flagged 1214 findings from the `hierarchical-errors` tool. This is a large number and adopting it would be a significant codebase-wide change. Is this a tool/standard you want to adopt, or should it be dismissed like the other buildflow findings?

### 3. Is the `go-auto-upgrade` tool yours or third-party?

The buildflow commit `bd849d1c` contains broken code (`slices.Contains()` with zero arguments) introduced by `go-auto-upgrade:repair`. This tool urgently needs a post-fix compilation gate. If it's your tool, I can file a bug. If it's third-party, we should pin a working version or disable the `repair` mode.
