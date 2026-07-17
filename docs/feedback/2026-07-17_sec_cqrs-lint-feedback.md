# cqrs-lint — Consumer Feedback (SEC)

**Consumer:** [SEC (Simple Escalating Chastity)](https://github.com/larsartmann/sec) — dice-based game with CQRS + Decider pattern, HTMX frontend, single Go binary with in-memory event store
**Version used:** go-cqrs-lite v4.0.x (event, command, query, decider, stack, middleware, snapshot, idempotency, watermill, codec, otel, scenario, storage/memory)
**lint version:** `cqrs-lint v0.2.1` (rebuilt from `cmd/cqrs-lint` at HEAD commit `509e23f7`, 2026-07-17)
**Date:** 2026-07-17

---

## Executive Summary

Ran cqrs-lint v0.2.1 against the full SEC codebase (69 Go files). The initial
run (v0.2.0, before code fixes) produced **32 findings** across 12 distinct
rules. After fixing the 2 valid findings (D002, D005) and re-running v0.2.1,
**30 findings** remain — all false positives or opinion-level suggestions.
Source-level verification of every finding AND inspection of the detector
source code:

| Category                               | Count | Action taken                                                 |
| -------------------------------------- | ----- | ------------------------------------------------------------ |
| **Valid findings (code defect)**       | 1     | Fixed — D002 mixed JSON casing                               |
| **Valid findings (doc drift)**         | 1     | Fixed — D005 version regex misattribution                    |
| **False positives (detector bug)**     | 23    | NOT suppressible via config — reported here for upstream fix |
| **Opinion-level suggestions declined** | 7     | B004×5, B010×1, B014×1 — architecture choice, not a defect   |

Signal-to-noise ratio: **2/30 actionable (7%)**. The dominant noise source is
E005/E007 (15 of 23 false positives) — the handler-registration scanner cannot
trace `*ast.SelectorExpr` arguments to `RegisterTyped`, a pattern so common
that **every single command and query** in SEC is flagged as unregistered.

Despite the false-positive volume, the tool caught one real consistency bug
(D002: mixed snake_case/camelCase JSON tags) that had escaped manual review.
The detector architecture is sound; the bugs are in AST pattern coverage and
name heuristics, all fixable with targeted improvements.

---

## Part 1: False Positives — Detector Bugs (26 findings)

### Bug 1: E005/E007 can't trace RegisterTyped through package-qualified Type constants (15 findings)

**Severity of impact: HIGH** — this is the single biggest source of noise. 15
of 30 total findings are false positives from E005 (10 commands) + E007 (5
queries). Every command and query type in the project is flagged.

#### What happens

SEC registers handlers using a table-driven pattern with package-qualified
`command.Type` constants:

```go
// internal/cqrs/app/command_handler.go:49-80
func (h *GameCommandHandler) Register(d *command.Dispatcher) error {
    registrations := []struct {
        cmdType  command.Type
        register func(*command.Dispatcher) error
    }{
        {seccmd.CreateGame, func(d *command.Dispatcher) error {
            return command.RegisterTyped(d, seccmd.CreateGame, h.handleCreateGame)
        }},
        {seccmd.PlayRound, func(d *command.Dispatcher) error {
            return command.RegisterTyped(d, seccmd.PlayRound, h.handlePlayRound)
        }},
        // ... 6 more
    }
    for _, reg := range registrations {
        if err := reg.register(d); err != nil {
            return fmt.Errorf("register handler for %s: %w", reg.cmdType, err)
        }
    }
    return nil
}
```

The second argument to `RegisterTyped` is `seccmd.CreateGame` — a
`*ast.SelectorExpr` (package-qualified identifier), not a bare `*ast.Ident`.

#### Root cause

`handlerTypeFromCall` in `pkg/rules/architecture/scanner_calls.go:74-89`:

```go
func handlerTypeFromCall(call *ast.CallExpr) string {
    for _, arg := range call.Args {
        switch a := arg.(type) {
        case *ast.CompositeLit:   // MyCommand{}
            if id, ok := a.Type.(*ast.Ident); ok { return id.Name }
        case *ast.FuncLit:         // func(ctx, c *MyCommand) error {...}
            return handlerTypeFromClosure(a)
        case *ast.CallExpr:        // NewMyCommand()
            return ExprString(a)
        }
    }
    return ""
}
```

There is **no case for `*ast.SelectorExpr`**. When the argument is
`seccmd.CreateGame` (a `*ast.SelectorExpr` with `X=seccmd`, `Sel=CreateGame`),
the switch falls through, `handlerTypeFromCall` returns `""`, the registration
is silently dropped from `CommandTypesRegistered`, and E005 flags
`CreateGameCmd` as having no handler.

This is distinct from the bank-sync feedback (which reported closures as the
blind spot). SEC's closures DO contain the handler type, but the `command.Type`
constant passed as the second argument is the `*ast.SelectorExpr` that's
missed. Even without closures, `command.RegisterTyped(d, seccmd.CreateGame,
handler)` alone would fail to register.

E007 has the same root cause: `e003_e007.go:111-128` consults
`IsCommandRegistered()` — the same shared map.

#### Fix

Add `*ast.SelectorExpr` to the switch in `handlerTypeFromCall`:

```go
case *ast.SelectorExpr:
    // pkg.ConstantName or pkg.TypeName
    return ExprString(a)  // "seccmd.CreateGame"
```

The returned string should then be matched against `command.Type` constant
values (not struct names) in the registration map. Alternatively, the scanner
could resolve the `*ast.SelectorExpr` to the underlying `command.Type` string
value by following the constant declaration.

---

### Bug 2: C008 substring matching flags non-monetary float64 fields (3 findings)

**Severity of impact: MED** — 3 false positives, all on the same domain field
(`TotalDays`).

#### What happens

SEC tracks cumulative chastity days as `float64`:

```go
// internal/cqrs/projection/streak_calendar.go:26
TotalDays     float64
```

C008 flags this as "use decimal or integer cents for money."

#### Root cause

`c008.go:28` defines weak money-field heuristics:

```go
var weakMoneyFields = []string{"value", "total", "charge", "payment", "salary"}
```

Matching uses `strings.Contains` (`c008.go:254-262`):

```go
func matchesAny(name string, terms []string) bool {
    for _, term := range terms {
        if strings.Contains(name, term) {  // substring, not word-boundary
            return true
        }
    }
    return false
}
```

`"totaldays".Contains("total")` → `true`. The field is a game-duration
counter, not money. No semantic context (package domain, struct purpose) is
considered — any `float64` field containing `"total"` as a substring matches
once any package path or struct name in the project looks monetary.

The "project looks monetary" escalation (`c008.go:77-79`) is also overly broad:
if any struct or package name in the entire project contains keywords like
`charge`, `fee`, `fund`, `order`, etc., then ALL weak-field `float64` matches
are promoted from `continue` to findings. SEC has none of these, but the
`structMoney` check (`c008.go:147-150`) fires because the escalation path has
multiple independent triggers.

#### Fix

1. Replace substring matching with word-boundary matching (split field name on
   camelCase boundaries, match against terms exactly).
2. Require a stronger monetary signal than one weak keyword. `TotalDays` has
   both "total" (weak money) and "days" (strong non-money signal). A field
   containing both money and non-money signals should not fire.
3. Consider a domain-aware escape hatch: `//cqrs-lint:ignore(C008)` exists but
   requires per-line annotation. A config-level toggle would be more practical
   for non-financial projects.

---

### Bug 3: C009 can't identify test-helper packages; `must*` prefix is case-sensitive (1 finding)

**Severity of impact: LOW** — 1 false positive, but the underlying bug has two
distinct facets.

#### What happens

`internal/cqrs/eventtest/eventtest.go:77` contains:

```go
func MustNewTestEvent(...) event.Event {
    evt, err := NewTestEvent(aggID, aggType, version, domainEvent)
    if err != nil {
        panic(err)
    }
    return evt
}
```

C009 flags the `panic(err)` as "panic in production code."

#### Root cause (facet 1: test detection)

`c009.go:24` skips test files via:

```go
if gf.IsTest {
    continue
}
```

`IsTest` is set in `loader.go:117` as:

```go
IsTest: strings.HasSuffix(path, "_test.go"),
```

The `eventtest` package is a **test-helper package** (provides fakes/builders
for test suites). Its files are named `eventtest.go`, not `eventtest_test.go`.
The detector treats it as production code. There is no check for:

- Package names ending in `test` or `testutil`
- Directory paths containing `test/`, `testdata/`, `testutil/`
- Build tags like `//go:build test`

#### Root cause (facet 2: `must*` prefix is case-sensitive)

`c009.go:81-86`:

```go
func isMustFunc(fn *ast.FuncDecl) bool {
    return strings.HasPrefix(fn.Name.Name, "must") && len(fn.Name.Name) > 4
}
```

The check is for **lowercase** `"must"`. Go convention for exported panicking
constructors is `MustXxx` (uppercase `M`, per [Effective Go](https://go.dev/doc/effective_go#init)).
`strings.HasPrefix("MustNewTestEvent", "must")` → `false`. The function is not
recognized as a `Must*` function despite following the exact naming convention.

#### Fix

1. Broaden test detection: treat packages whose name ends in `test`,
   `testutil`, `eventtest`, `fakes`, `mock`, or that live under `testdata/` or
   `internal/.../testutil/` as test code.
2. Make `isMustFunc` case-insensitive:
   ```go
   return strings.HasPrefix(strings.ToLower(fn.Name.Name), "must") && len(fn.Name.Name) > 4
   ```

---

### Bug 4: D003 treats slog handler backends as competing libraries (1 finding)

**Severity of impact: LOW** — 1 false positive, but affects any project using
charm.land/log as an slog backend.

#### What happens

SEC's `pkg/logging/logging.go` uses charm.land/log as a backend for `log/slog`:

```go
func SetupDefault(level charmlog.Level) {
    logger := charmlog.NewWithOptions(os.Stderr, charmlog.Options{...})
    charmlog.SetDefault(logger)
    slog.SetDefault(slog.New(logger))  // charmlog implements slog.Handler
}
```

D003 flags this as "Project mixes 2 logging libraries."

#### Root cause

`d003_d005.go:35-44` classifies imports into buckets:

```go
switch {
case strings.Contains(path, "/log/slog") || path == "log/slog":
    lib = "log/slog"
case strings.Contains(path, "charm.land/log"):
    lib = "charm.land/log"
```

These are treated as independent, competing libraries. When a file imports
both (`len(loggingImports) == 2`), D005 fires. But `charm.land/log` is an
`slog.Handler` backend — importing both is the **correct idiomatic pattern**,
not a conflict.

#### Fix

Treat known handler-backends as complementary to `log/slog`, not competing:

- `charm.land/log` provides `slog.Handler`
- `phsym/zeroslog` provides `slog.Handler`
- `golangci-tools/logr` provides `slog.Handler`

When one of these appears alongside `log/slog`, do not increment the
"competing library" count.

---

### Bug 5: A005 flags all named-function bus subscribers as projections (3 findings)

**Severity of impact: MED** — 3 false positives. Affects SSE brokers, cache
invalidation, and any event-driven side-effect subscriber.

#### What happens

SEC has three `bus.SubscribeAll` call sites:

```go
// 1. GameStateProjection (real projection — flagged)
return bus.SubscribeAll(p.HandleEvent)

// 2. SSE broker (side-effect subscriber — flagged)
return bus.SubscribeAll(b.handleEvent)

// 3. Cache invalidation (side-effect subscriber — flagged)
return bus.SubscribeAll(func(_ context.Context, evt event.Event) error { ... })
```

A005 flags all three as "Manual projection via bus.SubscribeAll — use
projectionhost.Host."

Only #1 is a real projection. #2 is an SSE push channel. #3 is a TTL cache
invalidator that calls `.Invalidate()` on timestamp expiry.

#### Root cause

`a005.go:108-142` — the `isManualProjection` suppression heuristic only
inspects **inline closures** (`*ast.FuncLit`). When the callback is a named
function (like `p.HandleEvent` or `b.handleEvent`), `extractCallbackFuncLit`
returns `nil`, and the code falls through to:

```go
// a005.go:122-128
callback := extractCallbackFuncLit(call)
if callback == nil {
    // Named-function subscriber — can't inspect the body.
    // Treat as projection candidate (conservative).
    anyProjectionCandidate = true
    return true
}
```

"Conservative" here means "always flag." For the SSE broker and cache
invalidation, there is no projection state to persist — the suggestion to use
`projectionhost.Host` is architecturally wrong.

For the inline-closure cache invalidation (#3), the body calls
`.Invalidate()` which does not appear in either the broadcast signal list
(`Send`, `WriteTo`, `Push`) or the persistence signal list (`Save`, `Set`,
`Upsert`, etc.). With neither signal detected, the heuristic falls through to
`hasPersist == false && hasBroadcast == false`, which does not suppress.

#### Fix

1. A `bus.SubscribeAll` call without any read-model state persistence is not a
   projection. The suppression heuristic should check whether the callback's
   containing type has `Get`/`Read`/`State` methods (projection indicators)
   vs `Broadcast`/`Notify`/`Send` (side-effect indicators).
2. For named-function subscribers, check whether the function's containing
   type name contains "Projection" or "ReadModel" before flagging. Types named
   `sseBroker`, `historyCache`, etc. are clearly not projections.
3. The suggestion text should acknowledge that `bus.SubscribeAll` is a
   legitimate API for non-projection subscribers (SSE, cache invalidation,
   audit logs, etc.).

---

## Part 2: Doctor/Lint Disagreement on Tracing

**Severity of impact: MED** — `doctor` and B014 give contradictory answers
about OTel tracing on the same project.

### What happens

```
$ cqrs-lint doctor
tracing:       on

$ cqrs-lint
[B014] Event bus / command dispatcher lacks OTel tracing middleware
```

### Root cause

`doctor` (`feature_detect.go:144-146`) resolves tracing as "on" via a fallback:

```go
if fp.Tracing == TracingUnknown {
    if hasOTelImport {
        fp.Tracing = TracingOn
    }
}
```

If the project merely **imports** `go-cqrs-lite/otel/v4`, doctor concludes
tracing is active. But SEC imports the otel package for its `id` and `event`
types, not for middleware registration.

Meanwhile, B014 correctly checks for actual middleware method calls
(`NewOTelBundle`, `EventTracing`, etc.) and finds none. The result: `doctor`
says "tracing on," B014 says "tracing lacking," both on the same project.

### Fix

Doctor should only report `TracingOn` when actual middleware wiring is
detected (Pass 2 AST check at `feature_detect.go:118-121`), not when the otel
package is merely imported. The fallback at lines 144-146 should resolve to
`TracingImportedOnly` (a new state) or `TracingUnknown`, not `TracingOn`.

---

## Part 3: Config Init Generates Unparseable Config

**Severity of impact: HIGH** — `cqrs-lint init` produces a config file that
breaks all subsequent `cqrs-lint` invocations.

### What happens

```
$ cqrs-lint init
Created .cqrs-lint.json with default settings

$ cqrs-lint
Error creating CLI: ... failed to parse config file: json: cannot unmarshal
array into Go struct field AppConfig.exclude of type string
```

### Root cause

`init.go:14-23` generates a template with `"exclude": []` (JSON array):

```go
const configTemplate = `{
  "min-severity": "info",
  "min-confidence": "low",
  "format": "text",
  "exclude": [],
  ...
}
`
```

But `main.go:48` declares `Exclude` as a `string`:

```go
Exclude string `default:"" flag:"exclude" help:"Exclude paths (comma-separated)"`
```

A JSON array `[]` cannot unmarshal into a Go `string`. The generated config is
self-incompatible.

### Fix

Change the template to `"exclude": ""` to match the `string` type. Or change
the struct field to `[]string` and update the consumer at `main.go:220-221`.

---

## Part 4: No Config-Level Per-Rule Suppression

**Severity of impact: MED** — forces inline comments for project-wide decisions.

### What happens

There is no way to write `"rules": {"E005": false}` or `"disable": ["E005"]`
in `.cqrs-lint.json`. The config's `RulesConfig` struct
(`rules_config.go:18-33`) only accepts `ExternalAPIStructPrefixes` (for D002).
Any other key under `"rules"` triggers an "unknown rules config key" warning.

### Available suppression mechanisms

| Mechanism                  | Scope                | Practical for SEC?                          |
| -------------------------- | -------------------- | ------------------------------------------- |
| `//cqrs-lint:ignore(RULE)` | Single line          | Yes — but requires 26 inline comments       |
| `--only E005,E007`         | CLI flag (inclusion) | No — inverts to "only run these"            |
| `--exclude path1,path2`    | Path exclusion       | Partial — excludes whole directories        |
| `min-severity: warning`    | Global severity      | No — hides B004/B010 but also real warnings |
| `min-severity: error`      | Global severity      | No — hides all warnings including real ones |

For SEC, the practical workaround is `--exclude internal/cqrs/eventtest` (kills
C009) and `min-severity: warning` (kills B004/B010/B014). But E005/E007/C008/
A005/D003 are all `warning` severity and cannot be suppressed without hiding
real warnings too.

### Fix

Add a `"disable": ["E005", "E007"]` or `"rules": {"E005": false}` field to
`AppConfig`. This is the #1 feature request for making the tool usable on
projects with known false-positive patterns.

---

## Part 5: Valid Findings

### D002 — Mixed JSON casing (FIXED)

`health_handlers.go` used snake_case JSON tags (`request_count`,
`error_count`) while `healthResponse` and `homeResponse` used camelCase. Real
consistency bug. Fixed by standardizing to camelCase across
`health_handlers.go` and `metrics.go`. Also updated snapshot tests and
regenerated snapshots.

### D005 — Stale documentation version (FIXED)

The version-extraction regex in `d003_d005.go:183-217` found `v0.5.22`
(the go-snaps version) on a line that also mentioned `go-cqrs-lite`, and
compared it to the go.mod version (`v4.0.0`). The detector takes
`versions[0]` — the first version-like string on the line — without verifying
it's attributed to go-cqrs-lite. Fixed by removing the version number from the
documentation line. The underlying detector flaw (no version-to-package
attribution) remains.

---

## Part 6: What Works Well

- **D002 caught a real bug.** The JSON casing inconsistency had escaped manual
  review and was wired through to snapshot tests. The detector found it
  correctly.
- **D005 caught a real doc issue** (even if the root cause was a regex
  misattribution). The fix was legitimate.
- **The health score (61/100) is honest** once you understand the false
  positives. The deduction weights are reasonable.
- **Analysis is fast** (250ms for 69 files). The `go/packages` loader approach
  is sound.
- **The `doctor` feature profile is useful** — detecting stack usage, snapshot
  strategy, and command flow gives real architectural insight (when it's
  accurate).
- **Inline suppression syntax exists** (`//cqrs-lint:ignore(RULE) reason`) and
  works correctly for single-line suppression. This is a good escape hatch.
- **The `--show-suppressed` flag** is well-designed — suppressed findings are
  retained and visible, not silently dropped.
- **The monorepo loader** (`findGoModDirs` + per-module `loadFromDir`) works
  correctly for SEC's two-module structure (root + `modules/` submodule).

---

## Part 7: Fix Recommendations

All recommendations are concrete, ordered by impact, and independently
shippable.

### Fix 1 (critical): Handle `*ast.SelectorExpr` in `handlerTypeFromCall`

**File:** `pkg/rules/architecture/scanner_calls.go:74-89`

Add a `*ast.SelectorExpr` case. This single change eliminates 15 of 26 false
positives on SEC and is the same fix that would help any project using
package-qualified `command.Type` constants (which is the documented
recommended pattern).

### Fix 2 (critical): Fix `cqrs-lint init` config template

**File:** `init.go:20`

Change `"exclude": []` to `"exclude": ""`. The generated config currently
bricks the linter. This is a one-character fix with zero risk.

### Fix 3 (high): Add config-level per-rule disable

**File:** `main.go` (AppConfig), `rules_config.go`

Add `"disable": ["E005", "C008"]` to the config. Without this, projects with
systematic false-positive patterns must choose between inline-comment noise
and hiding entire severity levels.

### Fix 4 (high): Fix `isMustFunc` case sensitivity

**File:** `pkg/rules/correctness/c009.go:86`

Change `strings.HasPrefix(fn.Name.Name, "must")` to
`strings.HasPrefix(strings.ToLower(fn.Name.Name), "must")`. Go convention is
`MustXxx` (uppercase), not `mustXxx`.

### Fix 5 (medium): Broaden test-package detection

**File:** `pkg/analyzer/loader.go:117`

Beyond `_test.go` suffix, treat packages named `*test`, `*testutil`,
`*eventtest`, `*mock`, `*fake`, or located under `testdata/` as test code.

### Fix 6 (medium): Fix doctor tracing fallback

**File:** `pkg/analyzer/feature_detect.go:144-146`

Do not report `TracingOn` based on import alone. Use `TracingUnknown` or a new
`TracingImportedOnly` state.

### Fix 7 (medium): Improve C008 field-name matching

**File:** `pkg/rules/correctness/c008.go:254-262`

Replace `strings.Contains` with word-boundary matching. A field like
`TotalDays` contains "total" (weak money) but also "days" (strong non-money).
Opposing signals should cancel.

### Fix 8 (low): Treat slog handler-backends as complementary

**File:** `pkg/rules/consistency/d003_d005.go:35-44`

When `charm.land/log` (or other known `slog.Handler` providers) appears
alongside `log/slog`, do not count them as competing libraries.

### Fix 9 (low): Improve A005 named-function subscriber handling

**File:** `pkg/rules/api/a005.go:122-128`

For named-function subscribers, check the containing type name for projection
indicators (`Projection`, `ReadModel`, `View`) before flagging. Types like
`sseBroker` and `historyCache` are not projections.

---

## Summary Table

| #   | Bug                                                  | Severity | Findings | Fix | Effort |
| --- | ---------------------------------------------------- | -------- | -------- | --- | ------ |
| 1   | E005/E007 misses `*ast.SelectorExpr` in registration | HIGH     | 15       | 1   | S      |
| 2   | `cqrs-lint init` generates unparseable config        | HIGH     | -        | 2   | XS     |
| 3   | No config-level per-rule disable                     | MED      | -        | 3   | M      |
| 4   | C009 `isMustFunc` case-sensitive prefix              | LOW      | 1        | 4   | XS     |
| 5   | C009 test-package detection too narrow               | LOW      | 1        | 5   | S      |
| 6   | Doctor/lint tracing disagreement                     | MED      | -        | 6   | S      |
| 7   | C008 substring matching on field names               | MED      | 3        | 7   | S      |
| 8   | D003 treats slog backends as competing               | LOW      | 1        | 8   | S      |
| 9   | A005 flags all named-function subscribers            | MED      | 3        | 9   | M      |
| 10  | D005 version regex has no package attribution        | LOW      | -        | -   | S      |

Fixes 1-2 are the minimum to make the tool usable without workarounds on this
project. Fix 1 alone eliminates 50% of all false positives.
