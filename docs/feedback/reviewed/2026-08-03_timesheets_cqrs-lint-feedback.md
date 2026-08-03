# cqrs-lint — Consumer Feedback (Timesheets)

**Consumer:** [Timesheets](https://github.com/LarsArtmann/timesheets) — Go CLI tool for generating professional PDF/Excel/Image/Markdown/HTML/JSON timesheets from JSONL or CSV input, with a dual-mode architecture: stateless CLI + event-sourced CQRS with optional HTTP API server
**Version used:** go-cqrs-lite v4.2.0 (all modules aligned: event, command, query, decider, id, storage, storage/memory, middleware)
**lint version:** `cqrs-lint v0.2.2` (Nix-installed binary, commit `d6be91c`)
**Date:** 2026-08-03
**Codebase size:** 40 Go files analyzed

---

## Executive Summary

Ran cqrs-lint across two sessions on the timesheets codebase. The first session
used cqrs-lint to drive a major remediation effort (57 findings → 23 remaining,
health score 0/100 → 78/100). The second session was a follow-up gap-closure
pass focusing on lint remediation of the broader Go linter output
(`golangci-lint`) and verification of CQRS-layer concerns that cqrs-lint had
flagged.

The tool is **genuinely valuable** — it drove real fixes: wrapping `FoldTimesheet`
in `StrictApply`, adding `CommandRecovery` + `CommandLogging` middleware, adding
state cache, fixing discarded `RegisterTyped` errors, migrating to `event.New`,
adding `WithSchemaVersion`, and aligning module versions. These were all
legitimate architectural improvements that a standard Go linter would never
catch.

However, after remediation, the remaining 23 findings break down as **7 false
positives** (30%) and **16 legitimate declines** (out of scope for a single-process
CLI tool). The false-positive rate for the _residual_ findings is concerning
because it inflates the deduction and depresses the score below where it
rationally should be.

| Category                                | Count | Action taken                                                                                                   |
| --------------------------------------- | ----- | -------------------------------------------------------------------------------------------------------------- |
| **Valid findings (fixed in session 1)** | 34    | All fixed — A001, C003, C007, C023, C025, C028, C033, P001/B019, A029, B021/B005, B025, B023, D013, D005, V006 |
| **False positives (remaining)**         | 7     | Reported below with root-cause analysis — C036(×4), B022, E009, E016                                           |
| **Legitimate declines (remaining)**     | 16    | Out of scope for single-process CLI — documented below                                                         |
| **Linter bugs found**                   | 2     | `init` config generation broken; `B022` suggests nonexistent API                                               |

**Health score: 78/100 (Good).** The -22 deduction includes -12 from false
positives alone. With false positives excluded, the score would be ~90/100.

---

## Part 1: High-Value Findings — Fixed

These findings drove real architectural improvements. They demonstrate the
linter's core value proposition: catching CQRS/event-sourcing anti-patterns that
no standard Go linter detects.

### C003 — Fold silently ignores unknown events

**Verdict: Critical. Fixed.**

The fold function had a `default: return state, nil` that silently dropped
unknown event types. Wrapped in `decider.StrictApply(fold, knownEventTypes)`
which now returns an error on unknown types. This prevented a class of silent
data-loss bugs.

### B021/B005 — Fold not wrapped in `StrictApply`

**Verdict: Correct. Fixed.**

Related to C003 — the `StrictApply` wrapper was missing entirely.

### B023 — Command dispatcher has no middleware

**Verdict: Correct. Fixed.**

Added `middleware.CommandRecovery()` and `middleware.CommandLogging(slog.Default())`.

### B025 — Repository missing state cache

**Verdict: Correct. Fixed.**

Added `decider.WithStateCache[TimesheetState](decider.NewStateCache[TimesheetState](256))`.

### C028 — Discarded `RegisterTyped` errors (10 sites)

**Verdict: Correct. Fixed.**

All 10 `command.RegisterTyped` and `query.RegisterTyped` calls had their errors
discarded. Now checked and wrapped with context.

### C007 — `time.Now()` inside decider (3 sites)

**Verdict: Correct. Fixed.**

Decider functions must be deterministic for replay. Injected `now time.Time`
parameter into `DecideSubmitTimesheet`, `DecideApproveTimesheet`,
`DecideRejectTimesheet`. Handlers pass `time.Now()`; tests pass fixed time.

### P001/B019 — O(N²) `repo.Load` in `SubscribeAll` projection

**Verdict: Correct. Fixed.**

Read model gained `ApplyEvent(evt, initial)` for O(1) incremental folding.

### A029 — `UsePublish` stub returning nil

**Verdict: Correct. Fixed.**

`SyncBus.UsePublish` and `SyncBus.Use` were no-ops. Now store and apply
middleware correctly.

### D013 — Events missing `WithSchemaVersion`

**Verdict: Correct. Fixed.**

`createEvent` now passes `event.WithSchemaVersion(1)`.

### V006 — Mixed module versions

**Verdict: Correct. Fixed.**

`storage/v4` was at v4.4.0 while all other modules were at v4.2.0. Aligned to
v4.2.0 (identical source, only dependency version differences).

---

## Part 2: False Positives — Remaining

These findings are incorrect for this codebase. Each has a root-cause
explanation for why the linter's heuristic fails.

### C036 — "Custom-backed store" (4 instances, -6 deduction)

**Verdict: False positive. Largest score impact.**

**Files:** `internal/domain/persistence.go` — 4 call sites:

- `storage.OpenSQLiteInMemory()`
- `storage.SQLiteInitSchema(ctx, database)`
- `storage.OpenSQLite(dbPath)`
- `storage.SQLiteInitSchema(ctx, database)`

The linter flags these as "custom-backed store" and suggests "use a
custom-backed store to match the event store backend." But these calls **are
the library's own SQLite functions** — they come from
`github.com/larsartmann/go-cqrs-lite/storage/v4`. The `*sql.DB` is then passed
to `storage.NewSQLiteEventStore(db)`, which is also from the library.

**Root cause:** The linter likely pattern-matches on the `*sql.DB` type or the
function names (`OpenSQLite*`) and concludes this is a hand-rolled store. In
reality, the consumer is calling the library's documented SQLite API. The
linter doesn't recognize that `storage.OpenSQLite` IS the library's function,
not a user-defined one.

**Suggested fix:** The linter should check whether the call site resolves to a
function in the `go-cqrs-lite/storage` module. If it does, C036 should not fire.

### B022 — Suggests `decider.CommandCausalityEnricher` which does not exist (-2)

**Verdict: False positive. Linter suggests a nonexistent API.**

**File:** `internal/domain/cqrs.go:56`

The linter says:

> Replace the custom enricher with `decider.CommandCausalityEnricher`

The code uses `event.CommandCausalityEnricher` (from the `event` module). There
is no `decider.CommandCausalityEnricher` — it does not exist in any version of
the library. The linter is suggesting a function name that would cause a
compilation error if the consumer followed the suggestion.

**Root cause:** The linter's enricher detection logic likely looks for a
`WithEnricher` option call and assumes the enricher should come from the
`decider` package. But `CommandCausalityEnricher` lives in the `event` package
(which makes sense — enrichers stamp event metadata, which is an event-layer
concern).

**Suggested fix:** Update the suggestion text to reference
`event.CommandCausalityEnricher`, or detect that the code is already using the
correct enricher and suppress the finding.

### E009 — "No HTTP/gRPC transport layer" (-1)

**Verdict: False positive. HTTP layer exists but isn't detected.**

**File:** `internal/domain/commands.go:1` (package-level)

The linter says:

> Project has command and query dispatchers but no HTTP/gRPC transport layer

The project has a full HTTP API server at `internal/server/` using
`cqrs-htmx/v4` (REST endpoints, SSE, CSRF middleware, health checks). The
`serve` command exposes all CQRS commands and queries as REST endpoints.

**Root cause:** The linter likely looks for `transport/http` or `transport/grpc`
imports specifically. The `cqrs-htmx` package is a sibling library that provides
HTTP transport on top of go-cqrs-lite but is not the `transport/http` module.
The linter doesn't know about `cqrs-htmx`.

**Suggested fix:** Either (a) detect `cqrs-htmx` imports as satisfying the
transport requirement, or (b) add a config flag like
`--features transport=cqrs-htmx` so the consumer can tell the linter that
transport is handled externally.

### E016 — "No health check endpoint" (-2)

**Verdict: False positive. Health endpoint exists.**

**File:** `internal/domain/cqrs.go:117` → actually points at
`internal/server/server.go`

The linter says:

> Add `bundle.HealthCheck(ctx)` to your `/healthz` or `/readyz` endpoint

The server registers `GET /health` via `cfg.app.HealthHandler()` at
`internal/server/routes.go:24`. The health endpoint exists and is functional.

**Root cause:** The linter pattern-matches for `healthz` or `readyz` path
strings. This project uses `/health` (following the cqrs-htmx convention).
Additionally, the linter looks for `bundle.HealthCheck` specifically, but
cqrs-htmx provides its own `HealthHandler`.

**Suggested fix:** Broaden the health-endpoint detection to include `/health`,
or detect `cqrshtmx.HealthHandler` usage.

---

## Part 3: Legitimate Declines — Out of Scope

These findings are technically correct but intentionally declined because they
are out of scope for a single-process CLI tool with optional local HTTP server.

| Rule | Count | Deduction | Reason for decline                                                                                                                                                                                                                                         |
| ---- | ----- | --------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| B004 | 4     | -4        | Commands hand-written by design. `cqrs-gen` adds build complexity for 8 commands.                                                                                                                                                                          |
| A005 | 1     | -2        | Manual O(1) incremental projection is better than `projectionhost.Host` for single-process. No checkpoint journal needed — events are in-memory.                                                                                                           |
| A016 | 1     | -1        | No idempotency needed for synchronous single-process dispatch. No retries, no at-least-once delivery.                                                                                                                                                      |
| A020 | 1     | -2        | No exported memory bus in go-cqrs-lite v4.2.0. `SyncBus` is intentional and well-tested.                                                                                                                                                                   |
| B001 | 1     | -1        | `createEvent` wraps errors with `WrapCorruption` (data integrity class). `event.Single` uses `WrapRejection` (validation class). The project's error classification is correct — event creation failure is a corruption issue, not a validation rejection. |
| F003 | 1     | -1        | OpenTelemetry tracing out of scope for CLI tool.                                                                                                                                                                                                           |
| F004 | 1     | -1        | Prometheus metrics out of scope for CLI tool.                                                                                                                                                                                                              |
| F005 | 1     | -1        | No schema evolution yet. Single schema version (1). Upcasters premature.                                                                                                                                                                                   |
| F006 | 1     | -1        | Event payload encryption out of scope for local-only SQLite store. PII flag on `PersonEmail` field is a false alarm — the email is a business field, not a secret.                                                                                         |
| F007 | 1     | -1        | Same as A016. No at-least-once delivery in single-process synchronous mode.                                                                                                                                                                                |
| F013 | 1     | -1        | `transport/http` module not needed. HTTP exists via `cqrs-htmx` which provides SSE delivery and REST handlers.                                                                                                                                             |
| B014 | 1     | -1        | OTel bundle out of scope for CLI tool.                                                                                                                                                                                                                     |

**Total legitimate deduction: -17.** These are real architectural decisions,
not oversights. A config mechanism to mark them as "intentional" without
suppression comments would improve the developer experience.

---

## Part 4: Linter Bugs

### Bug 1: `cqrs-lint init` generates a config file that breaks all subsequent runs

**Severity: Showstopper for new users.**

**Reproduction:**

```bash
mkdir /tmp/test && cd /tmp/test
cqrs-lint init
# → "Created .cqrs-lint.json with default settings"

cat .cqrs-lint.json
# {
#   "min-severity": "info",
#   "min-confidence": "low",
#   "format": "text",
#   "exclude": [],
#   ...
# }

cqrs-lint .
# → Error: short="Domain-aware linter for go-cqrs-lite consumers",
#         initializing CLI "cqrs-lint": failed to load config file:
#         loading config file: failed to parse config file:
#         json: cannot unmarshal JSON array into Go string within "/exclude"
```

**Root cause:** `init` generates `"exclude": []` (JSON array), but the config
parser expects `"exclude"` to be a **string** (comma-separated rule list).

**Fix:** Either (a) change the parser to accept `[]string`, or (b) change `init`
to generate `"exclude": ""`. Option (a) is more Go-idiomatic.

**Impact:** Any new user who runs `cqrs-lint init` (the obvious first command)
will hit this error immediately and likely abandon the tool. This was also
reported in the Cyberdom feedback (2026-07-17); it appears unfixed as of v0.2.2.

### Bug 2: B022 suggests a function that does not exist

**Severity: Medium (misleading suggestion).**

The linter suggests replacing `event.CommandCausalityEnricher` with
`decider.CommandCausalityEnricher`. The latter does not exist in any version of
the library. Following the suggestion would cause a compilation error.

**Fix:** Update the suggestion text to reference the correct package
(`event.CommandCausalityEnricher`), or detect that the correct enricher is
already in use.

---

## Part 5: Feature Requests

### 1. Config-based suppression for intentional declines

Currently, the only suppression mechanism is `//cqrs-lint:ignore(RULE)` comments
in source code. For architectural decisions that span an entire project (e.g.,
"no OTel for this CLI tool"), a config-based exclusion is cleaner:

```json
{
	"exclude": "F003,F004,B014"
}
```

Or per-rule with reason:

```json
{
	"suppress": [
		{ "rule": "F003", "reason": "CLI tool — no distributed tracing needed" },
		{ "rule": "A020", "reason": "SyncBus is intentional — no memory bus in v4.2.0" }
	]
}
```

This would let the health score reflect genuine gaps rather than intentional
architectural decisions.

### 2. Third-party module awareness (`cqrs-htmx`)

The linter doesn't recognize `cqrs-htmx` as satisfying the transport (E009,
F013) and health-check (E016) requirements. Since `cqrs-htmx` is a first-party
sibling library that builds on go-cqrs-lite, the linter should either detect it
automatically or accept a config flag.

### 3. False-positive confidence scoring

The `--fp-suspects` flag correctly identifies 8 low-confidence findings, but
these still count fully against the health score. Consider either (a) weighting
low-confidence findings at 50% deduction, or (b) excluding them from the score
entirely unless `--strict` is passed.

### 4. Library function recognition

C036 fires on `storage.OpenSQLiteInMemory()` — a function FROM the library. The
linter should recognize calls that resolve to `github.com/larsartmann/go-cqrs-lite/`
as library-internal, not "custom."

### 5. Fix `init` before anything else

This is the lowest-effort, highest-impact fix. A broken `init` command is a
terrible first impression for new adopters.

---

## Part 6: What the Linter Got Right

Fairness demands acknowledging the genuinely valuable findings:

1. **C003/C025 (fold silently drops unknown events)** — caught a real data-loss
   risk. This alone justified running the linter.

2. **C007 (`time.Now()` in deciders)** — caught a replay-determinism bug that
   would have caused subtle issues during event replay.

3. **B023 (no middleware)** — drove the addition of `CommandRecovery` and
   `CommandLogging`, which now provide structured operational visibility.

4. **B025 (no state cache)** — simple fix, real performance improvement.

5. **C028 (discarded RegisterTyped errors)** — caught 10 unchecked error returns
   that could mask registration failures silently.

6. **D013 (missing schema version)** — good forward-looking practice.

7. **V006 (module version misalignment)** — caught a real version drift that
   would have caused subtle compatibility issues.

The **health-score** concept is excellent. Having a single number to track over
time is motivating and provides a clear "are we getting better?" signal. The
78/100 score feels honest — it's not gaming-proof (false positives inflate the
deduction), but the directionality is correct.

---

## Summary Assessment

| Dimension                          | Rating | Notes                                                                         |
| ---------------------------------- | ------ | ----------------------------------------------------------------------------- |
| **Value delivered**                | ★★★★☆  | Drove 34 real fixes across the codebase                                       |
| **False-positive rate (residual)** | ★★☆☆☆  | 7 of 23 remaining findings are false positives (30%)                          |
| **False-positive rate (initial)**  | ★★★★☆  | Most of the original 57 findings were valid                                   |
| **Suggestion accuracy**            | ★★☆☆☆  | B022 suggests a nonexistent function; C036 misidentifies library functions    |
| **Config ergonomics**              | ★☆☆☆☆  | `init` is broken; exclusion is comment-only                                   |
| **Health score**                   | ★★★★☆  | Excellent concept, slightly inflated by false positives                       |
| **Documentation**                  | ★★★☆☆  | Suggestions are actionable when correct, but no inline docs for config format |

**Net recommendation:** Keep using cqrs-lint — the value outweighs the noise.
But fix `init`, fix B022, and add config-based suppression. Those three changes
would move this from "useful but frustrating" to "indispensable."

---

## Review Summary (2026-08-03)

**Overall assessment:** High-value feedback. 34 valid findings drove real architectural improvements. The tool is genuinely valuable for CQRS/event-sourcing anti-patterns.

### False Positives — Resolution Status

| Item                                                           | Status        | Evidence                                                                                                                         |
| -------------------------------------------------------------- | ------------- | -------------------------------------------------------------------------------------------------------------------------------- |
| C036 ×4 (library's own storage functions flagged as "custom")  | **Mitigated** | `collectEventStoreBackends` improved in round-2/3 reviews; shared `*sql.DB` detection added. May still trigger on some patterns. |
| B022 (suggests nonexistent `decider.CommandCausalityEnricher`) | **FIXED**     | Suggestion now references `event.CommandCausalityEnricher` (`b022_b025.go:13-16`)                                                |
| E009 (no transport — doesn't detect cqrs-htmx)                 | **FIXED**     | `feature_detect.go:78` detects `cqrs-htmx` as transport layer                                                                    |
| E016 (no health endpoint — misses `/health`)                   | **FIXED**     | `e016.go:141` now detects `/health`, `/healthz`, `/readyz`, `/livez`                                                             |

### Linter Bugs — Resolution Status

| Bug                                                                                | Status         | Notes                                                                                                                                                                                                                               |
| ---------------------------------------------------------------------------------- | -------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `cqrs-lint init` generates `"exclude": []` (JSON array) vs parser expecting string | **STILL OPEN** | SHOWSTOPPER for new users. The default preset template in `init.go:30` generates `"exclude": []` but the `Exclude` CLI field is `string`. Fix: change template to `"exclude": ""` or change the field to `[]string`. See TODO_LIST. |
| B022 suggests nonexistent API                                                      | **FIXED**      | See above                                                                                                                                                                                                                           |

### Feature Requests — Resolution Status

| Request                                           | Status                                                                 |
| ------------------------------------------------- | ---------------------------------------------------------------------- |
| Config-based suppression for intentional declines | **DONE** — `c008-ignore-fields` + `c008-ignore-structs` config shipped |
| Third-party module awareness (cqrs-htmx)          | **DONE** — `feature_detect.go` detects cqrs-htmx                       |

### Routed to TODO_LIST

- `cqrs-lint init` broken config — SHOWSTOPPER, code fix needed
- C036 library recognition — may need `IsLibrarySelfLint()` expansion
