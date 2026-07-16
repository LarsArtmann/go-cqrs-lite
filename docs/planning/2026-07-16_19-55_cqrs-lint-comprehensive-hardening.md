# cqrs-lint: Comprehensive Hardening Plan

<!-- historical-artifact-banner -->

> **Historical session artifact.** This is a point-in-time snapshot from a past
> session. Many items marked TODO / Open / Not Started / Broken have since been
> resolved. See [CHANGELOG.md](../../CHANGELOG.md) and
> [TODO_LIST.md](../../TODO_LIST.md) for current state.
> Last documentation health audit: 2026-07-16.

**Date:** 2026-07-16 19:55
**Goal:** Fix everything from the status report + what we forgot, sorted by impact/effort

## What We Forgot (Honest Reflection)

1. **Pipeline Metrics are ALREADY BUILT but never wired** — go-finding pipeline has `RecordDetector`, `DetectorTime`, `StageDuration`, `Snapshot()` — we set `Metrics: nil` implicitly. `--verbose` should print these.
2. **cmdguard has `TimingMiddleware`, `ValidateConfig`, custom types** — none used. Config is unvalidated, timing is manual.
3. **Dead code**: `internal/ast/` is empty, `TypeResolver` struct is never instantiated, `Packages` field on AnalysisContext is never read by rules.
4. **Duplicate helpers**: `selectorPackage` in 3 files, `baseTypeName` in 2, `extractJSONTag` in 2, `isCQRSModule`/`isCQRSModulePath` in 2.
5. **`finding.Builder` has `WithTags`, `WithRange`, `WithRelated`** — none used by any detector. We're leaving API value on the table.
6. **Exclude filter is `strings.Contains`** — no glob support. Should use `filepath.Match` or `doublestar`.
7. **6 files over 250 lines** — correctness/helpers.go (333), b009 (315), boilerplate/rules.go (278), a015_a019 (267), a011_a014_a017 (264), d003_d005 (255).
8. **Catalog is 3 manually-maintained files** — decoupled from detectors, drift-prone.

## Execution Plan (sorted by impact/effort)

| #   | Task                                                                      | Impact | Effort | Deps |
| --- | ------------------------------------------------------------------------- | ------ | ------ | ---- |
| 1   | Delete empty `internal/ast/` dir                                          | Low    | 1 min  | —    |
| 2   | Delete dead `TypeResolver` from types.go                                  | Low    | 2 min  | —    |
| 3   | Consolidate `selectorPackage` → use `analyzer.SelectorPackage` everywhere | Med    | 8 min  | —    |
| 4   | Consolidate `baseTypeName` → use `analyzer` version                       | Med    | 5 min  | 3    |
| 5   | Consolidate `extractJSONTag` → shared helper                              | Med    | 5 min  | —    |
| 6   | Consolidate `isCQRSModule`/`isCQRSModulePath` → one in analyzer           | Med    | 5 min  | —    |
| 7   | Split `correctness/helpers.go` (333→2 files)                              | Med    | 8 min  | —    |
| 8   | Split `boilerplate/b009_b010_b012_b015.go` (315→4 files)                  | Med    | 10 min | —    |
| 9   | Wire pipeline `Metrics` into result                                       | High   | 8 min  | —    |
| 10  | Print per-detector timing in `--verbose` mode                             | High   | 8 min  | 9    |
| 11  | Add pipeline `OnFinding` callback for streaming output                    | Med    | 5 min  | 9    |
| 12  | Fix `capturePayloadType` for variable references                          | High   | 10 min | —    |
| 13  | Fix `EventTypesEmitted` to track multiple sites                           | Med    | 8 min  | —    |
| 14  | Improve `--exclude` to use `filepath.Match`                               | Med    | 5 min  | —    |
| 15  | Add `--verbose` integration test                                          | Med    | 8 min  | 10   |
| 16  | Add `--color` mode unit tests                                             | Low    | 5 min  | —    |
| 17  | Add `--exclude` filter unit tests                                         | Low    | 5 min  | 14   |
| 18  | Add SARIF golden file test                                                | Med    | 8 min  | —    |
| 19  | Add snippet presence assertions to existing rule tests                    | Med    | 10 min | —    |
| 20  | Improve E004/E006 emission location (already done, verify)                | Low    | 2 min  | —    |
| 21  | Improve E001/E002 to point at import statements                           | High   | 10 min | —    |
| 22  | Improve E003 to point at first CQRS construct                             | Med    | 8 min  | —    |
| 23  | Improve A016 to point at dispatcher construction                          | Med    | 8 min  | —    |
| 24  | Improve B013 to point at repository construction                          | Med    | 8 min  | —    |
| 25  | Improve B014 to point at bus/dispatcher construction                      | Med    | 8 min  | —    |
| 26  | Add cmdguard config validation                                            | Med    | 8 min  | —    |
| 27  | Use cmdguard `TimingMiddleware`                                           | Low    | 5 min  | —    |
| 28  | Add exit code documentation to `--help`                                   | Low    | 3 min  | —    |
| 29  | Add `--debug` flag to dump scanner registry                               | Med    | 8 min  | —    |
| 30  | Consolidate 3 catalog files into 1                                        | Med    | 10 min | —    |
| 31  | Final nix fmt + lint + commit + push                                      | —      | 5 min  | all  |
