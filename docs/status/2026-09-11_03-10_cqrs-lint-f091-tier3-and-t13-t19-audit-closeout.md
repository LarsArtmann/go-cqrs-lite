# cqrs-lint: F091 Tier-3 completion + T13–T19 exhaustive rule-audit program

> **Status:** DONE 2026-09-11. Full `#verify` gate is the final step of this
> session (build + vet + test + race + lint + doc-check; cqrs-lint module
> suite green before it, API golden regenerated twice in-session and
> meta-tested green).

## What this session set out to do

Work the three living `TODO_LIST.md` cqrs-lint items to closure:

1. ApplyLayout rule (design done, implement)
2. 🔥 F091 Tiers 2–3 + F090(b) — remaining C035/C013 payload-shape confirmation
3. T13–T19 — exhaustive rule audit batches (explicitly low-yield)

## Outcome per item

### 1. ApplyLayout rule — was already done; TODO entry stale

P014 `applylayout-bypasses-plan-path` shipped 2026-09-09
(`pkg/rules/performance/p014.go`, typed fixture, pinned tests) and was already
documented in CHANGELOG `[Unreleased]`. The TODO entry predated the work.
Action: entry deleted (per TODO_LIST convention — completed work lives in
CHANGELOG), with a `[x]` pointer left in its place.

### 2. F091 Tier 3 — C035/C013 payload-shape confirmation — IMPLEMENTED

Design (`docs/planning/2026-09-06_cqrs-lint-t23-design-passes.md` §F091):
names select candidates, typed evidence confirms or rejects. Following the
C008 Tier-2 pattern exactly:

- Candidate classification split into **strong** (name suffix:
  EVENT/PAYLOAD/EVENTDATA; VIEW/READMODEL/READMODELSTATE/PROJECTION) and
  **weak** (file-location vibes: events.go, views.go, handler.go, …).
- Under `--typed-info on|auto`, weak candidates need structural evidence:
  - C013 payload branch: scanner registry (`event.New`/`NewEvent` payload
    flow) or a payload-conventional `Type() string` method.
  - C013 view branch: any `json` tag (serialization reflection point).
  - C035: a live selector reference to the map field in the analyzed files.
- Strong candidates and `--typed-info=off` / syntax-only loads keep the
  historical heuristic. Evidence walks live in
  `pkg/rules/correctness/typed_confirm.go`; the name/file vocabularies moved
  to shared primitives (`lintutil/name_heuristics.go`: `HasEventPayloadNameSuffix`,
  `IsPayloadFileName`, `HasReadModelNameSuffix`, `IsReadModelFileName`) —
  `lintutil.LooksLikeEventPayload` now composes them (F-series unchanged).
- **Scanner fix found on the way:** `capturePayloadType` recorded
  `event.New(..., T{...})` but missed the dominant `&T{...}` form, so the
  evidence registry under-recorded real emissions. Fixed + scanner test.
- Tests: 4 C013 typed-gate tests (Type-method confirm, silent-without-
  evidence, typed-off fallback, event.New-flow confirm), 2 C013 view-branch
  tests, 5 C035 typed tests (weak silent, map-use confirms, typed-off fires,
  strong skips gate, file candidate gated).
- taskmanager golden unaffected (its C013 finding is a strong-suffix payload).

### 3. T13–T19 — exhaustive rule audits — DONE, with real defects fixed

Per-rule checklist (impl + tests + catalog vs builder + RULES.md + comment
drift + negative coverage) across V(7), T(8), E(18), D(18), B(31), A(32, 2
waves), F(30), and the S001 line-by-line remainder. **Defects fixed:**

| Defect | Fix |
| --- | --- |
| V006 lexicographic semver sort (v4.10.0 < v4.9.0) | `semverCompare` numeric ordering + multi-digit regression test |
| V007 discarded `Build()` errors ×3, literal tool name | `lintutil.AppendBuild` guards, `toolName` const |
| v007.go 399 lines (>350) | path/lookup helpers split into `v007_paths.go` |
| E017 `.Stop()`/`.Shutdown(` suppression could never match (ExprString renders no parens) → real FPs | selector-name matching + 2 negative tests (taskmanager's true `Shutdown` no longer flagged) |
| D001 finding anchor randomized by map iteration | deterministic (file, line) sort |
| D005 parsed this repo's own `module go-cqrs-lite` as a version | `module`/non-`v` line skip |
| D011, A002, A003, A022, A024, A027, A030 hardcoded package-qualifier names (A014 alias-blindness class) | new `lintutil.QualifierTargetsModule` (type checker → import table → segment fallback) |
| A013 missed `*command.BasicCommand` (qualified pointer embed) | StarExpr→SelectorExpr accepted; taskmanager reports the 10 real embeds |
| S001 missed package-level var/const, composite-literal fields, map-key assignment | three new coverage paths + `s001LHSName`; FP-gate negative test added |
| B021 missing method-fold `StrictApply` suppression parity with B005 | `lastSegmentOfFoldName` fallback + injected-registry tests (fold scanner currently skips methods — noted) |
| T004 dead `go-snaps` disjunct; 9 comment drifts | removed/corrected |

Golden updates (both intended): `testdata/taskmanager_golden.txt` regenerated;
`taskmanagerGoldenProfile` gains `"A013": 10`, drops `"E017": 1`.

### Deferred (recorded as a TODO_LIST follow-up, not forgotten)

Loose heuristic gates (import-scope substrings, B018 containsBus casing,
A015 name-collision at error severity, A016/A017 project-wide suppressions,
A019 vendor heuristic, F006 payload-class wiring, V002/V003/V006
root-go.mod scope) — each needs individual FP analysis and golden churn;
confidence levels mitigate today. Over-length files `b022_b025.go` (495) and
`a020_a021_a022_a023.go` (~357) ride the pending file-size-gate policy
decision.

## Verification trail

- `cmd/cqrs-lint` full module suite: green (exit 0) after each wave.
- API golden regenerated (6781 → 6782 exports; new:
  `lintutil.QualifierTargetsModule`) + `TestEvery` meta-tests green.
- `scripts/check-changelog-symbols.sh`: green after rewording a prose
  backtick that the gate read as a symbol citation.
- Full `nix run .#verify`: final step of this session.

## Incidents

- First `#verify` run failed in `cmd/api-stability` (stale golden — the +4
  lintutil exports from the Tier-3 work had been added after the last regen).
  Regenerated + meta-tested; also freed a full tmpfs (/tmp 100%) that broke a
  test build mid-regen by trashing stale tool caches
  (`/tmp/go-build*`, `/tmp/gomod-verify`).
- A concurrent session touched `docs/DOMAIN_LANGUAGE.md` and status files;
  no conflicts with this work.
