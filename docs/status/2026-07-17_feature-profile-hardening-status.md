# cqrs-lint: Feature-Profile Hardening — Final Status

**Date:** 2026-07-17
**Scope:** Follow-up to the feature-profile implementation session — bug fix, test hardening, cleanup.
**Baseline:** 184 tests → **211 tests** (0 failing). Build + vet clean. Doctor emits valid JSON.

---

## What changed

### Bug fixed

| Bug                                                                                         | Root cause                                        | Fix                                                                                                                                                                                                                     |
| ------------------------------------------------------------------------------------------- | ------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `cqrs-lint doctor` emitted invalid JSON (trailing comma) when tracing/snapshot were unknown | Hand-formatted `fmt.Printf` with hardcoded commas | Replaced with `FeatureProfile.ToConfigFeatures()` + `encoding/json.MarshalIndent`, which is structurally incapable of producing trailing commas. Added a zero-value guard (empty `StoreKind` is `""`, not `"unknown"`). |

### Tests added (14 top-level, 9 subtests)

Each suppression test reuses a fixture that **would** fire, then proves the FeatureProfile gate suppresses it — pinning the rewiring against silent regression.

| Test                                       | Rule      | What it proves                                                         |
| ------------------------------------------ | --------- | ---------------------------------------------------------------------- |
| `TestS002_DowngradedForLocalCLI`           | S002      | HasServer toggle: Error → Info (downgrade, not full suppress)          |
| `TestS003_SuppressedForNoServer`           | S003      | HasServer=false → 0 findings                                           |
| `TestA015_SuppressedForNoServer`           | A015      | HasServer=false → 0 findings                                           |
| `TestB014_SuppressedForNoServer`           | B014      | HasServer=false → 0 findings                                           |
| `TestA016_SuppressedForReadOnlyFlow`       | A016      | CommandFlow=ReadOnly → 0 findings                                      |
| `TestA012_SuppressedForNoSoftDelete`       | A012      | HasSoftDelete=false → 0 findings                                       |
| `TestA009_AdaptiveSuggestion`              | A009      | Store backend → matching `stack/` suggestion (sqlite/postgres/pebble)  |
| `TestToConfigFeatures_OmitsUnknownFields`  | doctor    | All-unknown profile omits everything except server/soft-delete         |
| `TestToConfigFeatures_IncludesKnownFields` | doctor    | Fully-detected profile includes all 6 flags                            |
| `TestToConfigFeatures_JSONAlwaysValid`     | doctor    | Marshal+unmarshal roundtrip across 6 profile shapes, no trailing comma |
| `TestDetectFeatures_HTTPServer`            | detection | `http.ListenAndServe` → HasServer=true                                 |
| `TestDetectFeatures_GRPCServer`            | detection | `grpc.NewServer` → HasServer=true                                      |
| `TestDetectFeatures_Tracing`               | detection | otel import + `EventTracing` → Tracing=on                              |
| `TestDetectFeatures_Snapshot`              | detection | `WithSnapshotStore` → Snapshot=on                                      |

### Cleanup

- Removed dead `ptr[T any]` helper + its `//go:fix inline` directive from `feature_profile.go` (zero usages; cleared ~13 gopls `newexpr` hints).
- Removed two stale "has been replaced by" comment blocks in `s002_s003.go` and `a009_a013.go`.
- Added per-preset doc comments to `PresetLocalCLI`/`PresetProduction`/`PresetLibrary`/`PresetReadOnly` explaining **why** each pins its flags.

---

## Corrected finding: `new(false)` is valid Go

The prior session's status report listed `new(false)` as a code smell (item d.4, f.35) and a "next step" to fix it. **This was wrong.** Empirically verified: `new(false)`, `new(true)`, `new(SomeConst)` are valid Go 1.26 — `new(expr)` returns a pointer initialized to the expression's value (`*new(false) == false`). The presets are correct and idiomatic. The actual issue was the **dead `ptr` wrapper** that duplicated this capability; that has been removed.

---

## Verification

```
BUILD: OK   (go build -tags "goexperiment.jsonv2" ./...)
VET:   OK   (go vet -tags "goexperiment.jsonv2" ./...)
TESTS: 211  (0 failing)
DOCTOR: valid JSON (json.Unmarshal roundtrip passes)
```

---

## Deferred (separate epics, out of scope for this hardening pass)

SDK one-liners (`WithEncryptionFromEnv`, `WithObservability`, `WithSigningFromEnv`), SARIF profile enrichment, `--explain`/`--profile` CLI flags, CI self-lint job, A017 FeatureProfile consultation (per-call check is more precise), integration test against `example/taskmanager`, rename `globalCandidate.origName` → `Name`.
