# Review: nsfw-classifier cqrs-lint Feedback

**Source feedback:** [`2026-10-03_nsfw-classifier-cqrs-lint-feedback.md`](2026-10-03_nsfw-classifier-cqrs-lint-feedback.md)
**Date reviewed:** 2026-10-03
**Outcome:** Every claim verified against source before triage. 8 items fixed,
2 declined with rationale, 3 already supported/upstream. All fixes land with
tests; the typed ones run against a new committed fixture module
(`cmd/cqrs-lint/testdata/scanfixture`) because `BuildContextFromSource`
provides no type information.

---

## False Positives

### 1. Helper indirection (soft-delete registry + A012) — FIXED

**Verified:** `detectSoftDeleteRegistry` reads only `EventTypesEmitted`
(feature_detect_helpers.go), and the existing const resolution
(`ResolveEmittedEventTypeConsts`) covers direct const args and alias chains —
but NOT a parameter passed through a constructor helper (`newRoomEvent(t
event.Type, ...)`), because at the `event.New` site `t` is a parameter, not a
constant. A012 additionally fired on ANY switch-fold in a soft-delete module —
it never inspected case-clause contents at all.

**Fixed:**
- `scanner_emit_helpers.go`: `scanEmitHelperFunc` registers helpers whose
  `event.New` type argument is one of their own parameters;
  `ResolveHelperEmitCalls` walks helper call sites post-scan and feeds the
  resolved constants into `EventTypesEmitted` exactly as direct emissions.
- `FoldInfo.SwitchCaseValues` records raw case values (literals and identifier
  names); `ResolveFoldTombstoneCases` resolves const identifiers via
  `TypeConstValues` and marks `HandlesTombstoneEvent`. A012 skips those folds
  (ADR-0114 event-type handling is present).
- `IsTombstoneLikeEventType` extracted — soft-delete detection and A012 now
  share one vocabulary.

**Tests:** `TestEmitHelperIndirectionResolvesSoftDelete`,
`TestResolveFoldTombstoneCases_{ConstIdentifierCase,LiteralCase,NonTombstoneStaysUnmarked}`,
`TestA012_SkipsFoldHandlingTombstoneViaConstCase`.

### 2. Importer-scope blindness (`server: false`, F028, F004) — PARTIAL, harm removed

**Verified:** the analyzer loads only go-cqrs-lite-importing packages; servers,
slog setup, and metrics registries in non-importer packages are invisible.
**Fixed (the misconfiguration vector):** `ToConfigFeatures` now emits
Server/SoftDelete pins only as POSITIVE evidence (true). A "false" is
absence-of-evidence inside the scan scope — pinning it silenced real rules for
the reporter. `doctor` additionally prints a NOTE beside the suggested config
explaining the scope limit. The findings themselves (F028/F004 firing on
scope-invisible evidence) remain a documented limitation — widening the load
scope is an architectural change tracked separately.

### 3. A011 single-word JSON keys counted as camelCase — FIXED

**Verified:** `countJSONKeyCasings` counted any underscore-free lowercase-start
key as camel. **Fixed:** camel requires a lowercase→uppercase hump
(`createdAt`); `id`/`kind`/`ID` carry no case convention and no longer
manufacture a mixture. **Tests:** `TestA011_NoFindingForSingleWordKeysBesideSnake`,
`TestA011_FiresWhenTrueCamelMixesWithSnake` (true camel + snake still fires).

### 4. F031 fires on `bufio.Scanner.Scan` — FIXED

**Verified:** the receiver match was purely syntactic (any `.Scan(`).
**Fixed:** receivers resolve through `packages.TypesInfo`; only metaengine
receivers are coached (type string contains `cqrs-lite/metaengine` — full
import-path form, so `bufio.Scanner`, `database/sql` rows, etc. are excluded).
Unresolvable receivers keep the legacy behavior. **Test:**
`TestF031_BufioScannerScanDoesNotFire` runs against the committed
`testdata/scanfixture` module (real types): the bufio loop and a TypedReader
Scan share one file; exactly one finding anchors on the reader scan.

### 5. F026 coaches WithPrefetch for Get-only readers — FIXED

**Verified against metaengine source:** `TypedReader.Get` never touches the
prefetch cache (it lives on the Scan cursor path); the rule's message even
claimed Get benefits. **Fixed:** F026 gates on an actual `.Scan(` call site;
message/suggestion now scope the claim to Scan page fetches.

### 6. F010 linear-scan coaching — DECLINED

Heuristic coaching by design; the documented suppression path
(reason + `--fail-on-stale-suppressions`) is the intended answer for
deliberate small-slice scans.

### 7. A009 coaches stack/ presets at system.New users — FIXED

**Verified:** A009 recognized only `stack/` imports and the storage-facade
escape. **Fixed:** importing `go-cqrs-lite/system/` (the ADR-0123 composition
root; presets are removed at v5) counts as adoption. Match is
trailing-slash-precise so `systemtest/` does NOT suppress. **Tests:**
`TestA009_NoFindingForSystemCompositionRoot`,
`TestA009_StillFiresForSystemtestOnlyImport`.

---

## Design Suggestions

### 1. Feature pins as explicit declines (F003/tracing) — FIXED

**Root cause found deeper than reported:** detection collapsed
absence-of-OTel-evidence to `TracingOff`, so a pin was indistinguishable from
"no evidence". **Fixed:** detection now leaves `TracingUnknown` on absence;
`TracingOff` is exclusively a config pin, and F003 honors it as a decline —
the same contract `features.monetary` already uses for C008. Doctor
consequently no longer suggests pinning `tracing: "off"` for un-traced
projects (it cannot distinguish scope-blind absence). **Test:**
`TestF003_TracingOffPinIsADecline`.

### 2. F005 stable anchor — DECLINED (for now)

The alphabetically-first-package anchor is real, but moving it (e.g. to
go.mod, the A009 precedent) churns every existing suppressor —
`--fail-on-stale-suppressions` is the safety net that already catches drift
loudly. Revisit only if more consumers hit it.

### 3. Stale-suppression warnings naming the anchor — FIXED

`StaleSuppression.FiresAt` records where the rule actually fires (first
finding, deterministically sorted); the warning renders
`rule X does not fire here (fires at file.go:LINE); safe to remove or move`.
**Test:** `TestDetectStaleSuppressions_NamesActualFireAnchor`.

### 4. `--config` flag — ALREADY SUPPORTED (upstream)

cmdguard v4.0.2 ships a native `-c/--config` flag that redirects the config
load before flag parsing (verified live: present in `--help`, accepted on
every subcommand, flags still beat config values). The reporter's build
(commit 3756eb4) predated it. No cqrs-lint change needed.

### 5. DomainKind discoverability — FIXED

The profile renders `domain: unknown (pinnable: internal, security,
financial)` when unknown — `internal`/`security` were previously learnable
only by reading analyzer source (`detectDomainRegistry` returns only
financial/unknown by design; the hint says so).

### 6. B005 same-name folds — NOTED

`StrictApplyFolds` stays name-matched (last-identifier segment). Cross-package
collisions remain a documented limitation; a disambiguation pass is a future
test-suite item, not a rule change.

---

## Scorecard

| Feedback area                          | Verdict                                                        |
| -------------------------------------- | -------------------------------------------------------------- |
| Helper indirection + A012 const cases  | Fixed (registry machinery + case-value resolution + tests)     |
| Scope blindness                        | Suggestion-harm removed; load-scope widening deferred          |
| A011 / F031 / F026 / A009              | Fixed, each with a regression test                             |
| F003 decline semantics                 | Fixed at the detection layer (absence ≠ off)                   |
| Stale anchor naming                    | Fixed                                                          |
| `--config`                             | Already supported via cmdguard v4.0.2                          |
| F010 / F005 anchor / B005              | Declined or noted, with rationale                              |

The "one root cause" thesis held: indirection + scope blindness accounted for
the bulk of the false positives, and both now have structural fixes.
