# cqrs-lint T20/T21 — Subsystem Line-by-Line Reviews

**Date:** 2026-09-07 · **Scope:** T20 (scanner*.go, feature_detect*.go,
loader.go, registry.go, module_catalog*.go, upcaster.go) + T21
(doctor*.go, health.go, scorecard*.go, output*.go, explain.go).
**Verdict:** no correctness bugs found in the reviewed surface. One
F089-completeness gap in doctor (fixed same-session), two
determinism bugs fixed same-session, one real detection gap and several
accepted heuristics recorded below for the next hardening wave.

## Fixed during this review

| ID | File | Finding |
| --- | --- | --- |
| T21-1 | doctor.go, doctor_json.go | doctor preset/effective panels and the JSON report did not render `rules.severity-overrides` (F089 completeness). Added `formatSeverityOverrides` + `severityOverrides` JSON field. |
| T20-2 | loader.go | `primaryModuleProfile` broke equal-depth module ties on map iteration order — nondeterministic primary profile. Added name tie-break. |
| T21-3 | doctor.go | `renderDoctorPerModuleProfiles` sorted module dirs by length only — nondeterministic for equal lengths. Added name tie-break. |
| T21-2 | doctor.go | `renderDoctorFeatureProfile.hasOverrides` ignored the `Monetary` feature pin. |

## Recorded findings (fix in a future wave)

### T20-1 — store detection misses non-preset engines (gap, M)

`feature_detect.go` `detectImports` maps store backends only from the
v5-removed `stack/*` presets plus metaengine engines
`sqlite/pebble/duckdb/postgres`. A consumer on `metaengine/mysqlengine`,
`badgerengine`, `dgraphengine`, `tursoengine`, `bbolt`, or `irohengine`
resolves to `StoreNone`/`StoreUnknown`, so store-conditional rules and the
scorecard see the wrong tier. Fix direction: extend
`metaengineEngineFromImport` + add an engine→StoreKind entry for every
shipped engine (the full list lives in `metaengine/*engine`). Needs a small
table test per engine.

### T20-3 — Pass-1 import scan is order-sensitive (nondeterminism, S)

`detectFeatureSignals` Pass 1 iterates `pkg.Imports` (a Go map). The
`stack/*` branches overwrite `fp.Store` unconditionally (only the
`storage/` branch has a `StoreUnknown` guard), so a package importing two
different presets gets a nondeterministic Store across runs. Rare shape
(a package importing two presets), but the fix is the same first-wins
guard the storage branch already uses.

### T20-4 — `handlerTypeFromCall` stores call text as registry keys (smell)

Constructor-call handlers record `ExprString(call)` (e.g.
`NewMyCommand(bus)`) as a `CommandTypesRegistered` key that can never match
a struct name. Lookups are unaffected; doctor/audit dumps and future
iteration-based rules see garbage keys. Should be a distinct
"unresolved constructor" record.

### T20-5 — `CommandInfo.Fields` mixes field names and embed exprs (smell)

`scanStructFields` appends embedded-field expression text
(`BasicCommand`) into the same `Fields` slice as real member names.
Consumers must know which entries are names. Split into `Embeds`.

### T20-7 — `IsInsideUpcasterClosure` is O(file) per query (perf, S)

Full `ast.Inspect` of the file for every A014/C005 candidate call. Cache a
per-file list of upcaster-closure ranges once (scan in the analyzer pass).

### T20-8 — remaining alias-blind helpers (tracked by F091 Tiers 2–3)

`capturePayloadTypeFromVar`, `looksLikeEventType`, and
`IsInsideUpcasterClosure` string-match the unaliased `event`/`schema`
qualifiers. `ResolveQualifierTyped` (F091 Tier 1, shipped 2026-09-07)
provides the exact resolution these need; adopting it there is the Tier-2
entry point.

### Accepted heuristics (documented, no action)

- `looksLikeEventType`, `isOOAggregate`, generic `X[*T Command]` handler
  evidence, method-name server/dispatch detection, health-route literal
  scan — each conservative in a direction that yields missed detections or
  extra-enabled rules, never wrong severity or crashes. Verified in each
  file's comments.
- `BuildContext` skips packages with load errors entirely (partial
  analysis); surfaced via LoadErrors + `--strict-load`.

### Clean

`module_catalog.go` (immutable catalog, single-source priorities),
`scanner_adapters.go`, `scanner_calls_helpers.go`, `registry.go`,
`output.go` (color-layering contract documented in-file),
`output_rulesmd.go`, `scorecard.go` structure, `explain.go`.
