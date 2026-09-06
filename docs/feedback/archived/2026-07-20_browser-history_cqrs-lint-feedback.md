# cqrs-lint — Consumer Feedback (browser-history)

**Consumer:** [browser-history](https://github.com/larsartmann/browser-history) — Go CQRS/ES app that extracts and tracks browser history (Chrome/Safari/Firefox) via go-cqrs-lite v4, Huma v2 HTTP, usermgmt WebAuthn auth
**Version used:** go-cqrs-lite v4.0.x (event v4.0.2, decider v4.0.1, command/query v4.0.0, watermill v4.0.2, storage v4.0.1, middleware v4.0.1, idempotency/kvstore v4.0.0, kv v4.0.1)
**lint version:** `cqrs-lint` (alpha, rebuilt from `cmd/cqrs-lint` at HEAD, 2026-07-20)
**Date:** 2026-07-20

---

## Executive Summary

Ran cqrs-lint against the full browser-history codebase (25 Go files). The tool
reported **17 findings** (1 error, 11 warning, 5 info). After source-level
verification of every finding AND inspection of the detector source code:

| Category                                 | Count | Action taken                                                              |
| ---------------------------------------- | ----- | ------------------------------------------------------------------------- |
| **Valid findings (fixed)**               | 4     | Fixed in code: C003, B005 (StrictApply), C009 (panic), A016 (idempotency) |
| **False positives (detector bug)**       | 10    | NOT suppressed — reported here for upstream fix                           |
| **Valid findings (intentional/opinion)** | 3     | Reviewed, declined — documented below                                     |

The signal-to-noise ratio is **41%** (7 real or actionable / 17 total). The
dominant noise source is E005/E007 (9 of 10 false positives) — these detectors
cannot trace handler registration through plain `dispatcher.Register()` calls or
`RegisterTyped` calls that use type constants and method values.

**Key difference from prior bank-sync report (2026-07-17):** bank-sync's E005/E007
false positives stemmed from **closure-based** `RegisterTyped` calls. browser-history's
stem from two **different** untracked patterns: (1) plain `dispatcher.Register(type,
methodValue)` for commands, and (2) `RegisterTyped(disp, typeConstant,
constructorMethodValue)` for queries. Both are valid idioms but invisible to the
analyzer.

---

## Part 1: Fixes Applied (4 findings)

### C003 + B005 — Adopted `decider.StrictApply` and error on unknown event types

**Files:** `domain/aggregate/decider.go`

**Before:** `Fold` had `default: return state, nil` (silent ignore) and the
`BrowserHistoryDecider` used `Apply: Fold` directly.

**After:** Two-layer defense:

1. `BrowserHistoryDecider.Apply` now wraps Fold with `decider.StrictApply(Fold, knownTypes)`
   — unknown event types are rejected at the framework level before Fold is called.
2. `Fold`'s default case now returns a Corruption error (`errorfamily.Newf`) instead of `nil`
   — catches direct Fold calls (tests, other consumers) that bypass the Decider.

```go
var BrowserHistoryDecider = decider.Decider[BrowserHistoryState]{
    Initial: NewInitialState(),
    Apply: decider.StrictApply(Fold, []event.Type{
        EventVisitRecorded,
        EventDomainClassified,
        EventVisitDeleted,
    }),
}
```

**Test updated:** `TestFold_UnknownEvent` now expects an error. Added
`TestBrowserHistoryDecider_StrictApply_RejectsUnknownEventType` to verify the
StrictApply wrapper rejects unknown types at the Decider level.

### C009 — Removed `panic()` from production code

**File:** `api/static.go` → `api/dashboard.go`

**Before:** `staticFileServer()` called `panic(err)` on `fs.Sub` failure.

**After:** `staticFileServer()` returns `(http.Handler, error)`. Error propagates
through `registerDashboard() → NewServer()`. The error is unreachable in practice
(`//go:embed static` guarantees the path), but the panic is gone.

### A016 — Added `middleware.CommandIdempotency` to command dispatcher

**File:** `api/server.go`

**Before:** `cmdDisp := command.NewDispatcher()` with no idempotency.

**After:** Dispatcher now wires `middleware.CommandIdempotency` with an in-memory
KV store (`kv.NewMemStore()` → `kvstore.New()`) and a 10-minute TTL. Defense-in-depth
against double-dispatch of the same command ID.

```go
cmdDisp.Use(middleware.CommandIdempotency(
    kvstore.New(kv.NewMemStore()),
    10*time.Minute,
    nil,
))
```

---

## Part 2: False Positives — Detector Bugs (10 findings)

### Bug 1: E005 can't trace `dispatcher.Register(type, methodValue)` (3 findings)

**Severity of impact: MEDIUM** — 3 false positives, all WARNING severity.

#### What happens

browser-history registers command handlers via plain `dispatcher.Register`,
NOT `RegisterTyped`:

```go
// api/server.go:266 — command registration
cmdDisp.Register(aggregate.CommandExtractHistory, extractHandler.Handle)
```

The handler type (`*ExtractVisitCommand`) is never visible in the registration
call — it appears only inside the `Handle` method body via a type assertion
(`requireCommandType[*ExtractVisitCommand]`).

#### Root cause

`scanCallExpr` in `scanner_calls.go` only matches `RegisterTyped` / `RegisterQuery`
— **plain `Register` is never tracked**. The E005 rule's own suggestion text says
"Register a handler via command.RegisterTyped **or** dispatcher.Register", but the
analyzer cannot trace `dispatcher.Register`.

This is a **different root cause** from the bank-sync report (2026-07-17), which
dealt with closure-based `RegisterTyped`. Here the issue is that `Register` (the
string-type-based API) is entirely untracked.

#### Fix suggestion

Add a `case funcName == "Register":` branch in `scanCallExpr` that extracts the
command type from the first argument. When the first argument is a `command.Type`
constant (an `*ast.Ident` or `*ast.SelectorExpr` like `aggregate.CommandExtractHistory`),
resolve it to the command struct name by cross-referencing the constant's
declaration. Alternatively, suppress E005 when the type name is registered as a
`command.Type` constant (they appear in `Registry.CommandTypes` via struct
detection).

A simpler heuristic: if `dispatcher.Register` is called anywhere in the package
with a type that matches the command's registered `command.Type` constant, mark
the command as registered.

#### Affected findings

| Rule | Type                | File:Line                             |
| ---- | ------------------- | ------------------------------------- |
| E005 | ExtractVisitCommand | extraction/commands/handlers.go:63:6  |
| E005 | ClassifyURLCommand  | extraction/commands/handlers.go:107:6 |
| E005 | DeleteVisitCommand  | extraction/commands/handlers.go:172:6 |

---

### Bug 2: E007 can't trace `RegisterTyped` with type constant + method value (6 findings)

**Severity of impact: HIGH** — 6 false positives, all WARNING severity. The
single largest source of noise in this report.

#### What happens

browser-history registers query handlers via `query.RegisterTyped` — the
**type-safe API the library recommends** — but uses two patterns the detector
can't trace:

```go
// api/server.go:299 — query registration
err := query.RegisterTyped(
    queryDisp,
    projection.GetVisitQueryType,                      // arg[1]: query.Type constant (*ast.SelectorExpr)
    projection.NewGetVisitHandler(readModel).Handle,   // arg[2]: method value (*ast.SelectorExpr)
)
```

1. **Type constant (`*ast.SelectorExpr`)**: `projection.GetVisitQueryType` is a
   `query.Type` constant, not a struct literal. `handlerTypeFromCall` has no case
   for bare `*ast.Ident` or `*ast.SelectorExpr` arguments — only `*ast.CompositeLit`,
   `*ast.FuncLit`, and `*ast.CallExpr`.

2. **Method value (`*ast.SelectorExpr`)**: `projection.NewGetVisitHandler(readModel).Handle`
   is a bound method value. The handler type (`*GetVisitQuery`) lives inside the
   `Handle` method body, not in the registration call's argument list.

#### Root cause

`handlerTypeFromCall` in `scanner_calls.go:74-89`:

```go
func handlerTypeFromCall(call *ast.CallExpr) string {
    for _, arg := range call.Args {
        switch a := arg.(type) {
        case *ast.CompositeLit:    // MyQuery{}
            if id, ok := a.Type.(*ast.Ident); ok { return id.Name }
        case *ast.FuncLit:         // func(ctx, q *MyQuery) error {...}
            // extracts from closure params
        case *ast.CallExpr:        // NewMyQuery()
            return ExprString(a)
        }
    }
    return ""
}
```

None of these cases match a bare `*ast.Ident` (type constant) or `*ast.SelectorExpr`
(method value). The registration is silently dropped, and E007 fires.

This is related to but **distinct** from the bank-sync bug: bank-sync used closures
(`*ast.FuncLit`) which the fix suggestion addressed. browser-history uses method
values (`*ast.SelectorExpr`), which is a separate untracked pattern.

#### Fix suggestion

1. **For type constants**: Add a case for `*ast.SelectorExpr` and `*ast.Ident`
   arguments to `handlerTypeFromCall`. When the argument is a type constant
   (e.g., `projection.GetVisitQueryType`), resolve it by:
   - Checking if it's a known `query.Type` constant in the registry
   - Cross-referencing the constant's value with the query struct's registered type

2. **For method values**: Extract the receiver type from the method value
   expression. `projection.NewGetVisitHandler(readModel).Handle` → the receiver is
   `*GetVisitHandler`, and the query type it handles (`*GetVisitQuery`) can be
   inferred from the `Handle` method signature.

3. **Fallback**: If the type constant maps to a registered query type (all query
   structs have their `query.Type` constant detected via `scanTypedMethod`),
   suppress E007 for that struct.

#### Affected findings

| Rule | Type                 | File:Line                   |
| ---- | -------------------- | --------------------------- |
| E007 | GetVisitQuery        | projection/queries.go:21:6  |
| E007 | ListVisitsQuery      | projection/queries.go:58:6  |
| E007 | GetDomainStatsQuery  | projection/queries.go:95:6  |
| E007 | ListDomainStatsQuery | projection/queries.go:132:6 |
| E007 | GetSessionQuery      | projection/queries.go:166:6 |
| E007 | ListSessionsQuery    | projection/queries.go:203:6 |

---

### Bug 3: B007 fires on `huma.Register` — doesn't filter by package (1 finding)

**Severity of impact: LOW** — 1 false positive, INFO severity.

#### What happens

browser-history uses [Huma v2](https://github.com/danielgtaylor/huma) for
type-safe HTTP endpoints. Each route is registered via `huma.Register`:

```go
// api/routes.go:10
func (s *Server) registerRoutes(api huma.API) {
    huma.Register(api, huma.Operation{Method: http.MethodGet, Path: "/health"}, s.health)
    huma.Register(api, huma.Operation{Method: http.MethodPost, Path: "/extract"}, s.extractHistory)
    // ... 10 more
}
```

B007 flags this as "12 consecutive handler registrations — use a table-driven or
variadic approach."

#### Root cause

B007 (`b006_b007.go:148-149`) matches purely by method name:

```go
if sel.Sel.Name == "Register" || sel.Sel.Name == "RegisterTyped" || ...
```

**The package qualifier is never checked.** `huma.Register` matches because its
method name is `Register`. The docstring claims "CQRS handler registration ONLY.
It does NOT fire on stdlib http.ServeMux.HandleFunc" — but that exclusion is
incidental (`HandleFunc` isn't in the match set), not architectural. Any
third-party `Register` call triggers the rule.

#### Why table-driven is impossible here

Huma's `huma.Register` is a **generic function**: `huma.Register[I, O, Body](api,
operation, handler)`. Each handler has a **different type signature**:

```go
func(ctx, *struct{}) (*HealthOutput, error)              // health
func(ctx, *ExtractHistoryInput) (*ExtractHistoryOutput, error)  // extractHistory
func(ctx, *CreateVisitInput) (*VisitOutput, error)       // createVisit
```

These cannot be collected into a homogeneous slice — they are different generic
instantiations. A table-driven approach would require erasing type safety
(`any` + reflection), which defeats the purpose of using Huma.

#### Fix suggestion

Check `SelectorPackage(sel)` (the package path of the selector's qualifier)
before counting the statement. Only match `Register`/`RegisterTyped` when the
qualifier resolves to a `go-cqrs-lite` package (`command`, `query`, `event`,
`decider`). Skip when the qualifier is `huma`, `http`, or any other package.

#### Affected findings

| Rule | Location                     |
| ---- | ---------------------------- |
| B007 | api/routes.go:10:2 (12 regs) |

---

### Bug 4: B005 doesn't detect `decider.StrictApply` wrapping (0 remaining findings, but would fire)

**Severity of impact: LOW** — latent detector gap.

#### What happens

After adopting `decider.StrictApply` (see Part 1), B005's suggestion ("Use
`decider.StrictApply`") is now implemented. But the detector has **no suppression
logic** for StrictApply usage — it fires purely on the presence of a `*ast.SwitchStmt`
inside a fold-shaped function.

#### Root cause

B005 (`b004_b008.go:64`) checks only `fold.HasSwitch`. There is no check anywhere
in the scanner or rule for whether the fold is passed to `decider.StrictApply`.

**Note:** In this specific run, B005 fired BEFORE the fix was applied. After the
fix, B005 will still fire on the same fold (it still has a switch). The suggestion
is already implemented but the detector can't see it.

#### Fix suggestion

In `detectFoldFunc` or the B005 rule, add a cross-reference check: scan the
package (or workspace) for `decider.StrictApply(foldFuncName, ...)` calls. If
the fold function is already wrapped, suppress B005.

---

## Part 3: Intentional Deviations (3 findings)

### A009 — No stack/ preset (INFO)

```
Project does not use a stack/ preset — manual wiring is error-prone and misses defaults
```

**Decision:** Keep custom wiring. browser-history uses a hybrid Huma + usermgmt +
event sourcing setup that doesn't fit any single stack preset. The manual wiring
in `NewServer` gives full control over the middleware chain, DB sharing between
event store and read model, and session store configuration. The `stack/` presets
assume a more uniform deployment (single KV backend, standard middleware set).

### A017 — No snapshot strategy (INFO)

```
Repository created without snapshot strategy — long event streams will cause slow loads
```

**Decision:** Not applicable. The `BrowserHistoryState` aggregate is intentionally
minimal: a visit counter (`int`) and a domain set (`map[Domain]struct{}`). Even
with millions of events, folding is O(n) with trivial per-event cost (map insert

- counter increment). A snapshot would add complexity (SnapshotStore, codec for
  state serialization, snapshot frequency policy) with zero measurable benefit.

The aggregate has **one instance** (singleton keyed by the aggregate type). Event
streams grow linearly with total visits, but fold performance remains sub-second
well into the millions of events. If this ever becomes a bottleneck, `WithStateCache`
(not full snapshotting) would be the first step.

### B004 — Manual constructors instead of cqrs-gen (INFO)

```
Command ClassifyURLCommand has 6 fields — consider using cqrs-gen to generate constructors
```

**Decision:** Keep manual constructors. The project has only 3 command types,
each with a hand-written `NewXxxCommand` constructor that validates inputs and
wraps errors with `errorfamily`. Adopting `cqrs-gen` would add a code generation
step to the build for marginal benefit at this scale. The manual constructors
also include domain-specific validation that a generator can't express.

---

## Summary

| Category                                  | Count  | Verdict                               |
| ----------------------------------------- | ------ | ------------------------------------- |
| Fixed (C003, B005, C009, A016)            | 4      | Applied, tests pass, build clean      |
| False positives (E005×3, E007×6, B007×1)  | 10     | Reported for upstream fix             |
| Intentional deviations (A009, A017, B004) | 3      | Reviewed, declined with justification |
| **Total**                                 | **17** |                                       |

**Top priority fixes for cqrs-lint:**

1. **E005/E007**: Trace `dispatcher.Register(type, handler)` and
   `RegisterTyped(disp, typeConstant, methodValue)` — this is the #1 noise source
   across multiple consumer projects (bank-sync, browser-history).
2. **B007**: Filter by package qualifier — `huma.Register` is not CQRS registration.
3. **B005**: Detect `decider.StrictApply` wrapping — don't suggest a fix that's
   already applied.

---

## Resolution Log (2026-07-20)

All three detector bugs reported in Part 2 were fixed in `cmd/cqrs-lint`.
The build was also unblocked (c008.go and e003_e007.go were left in a
non-compiling WIP state by a prior commit, blocking all verification).

| Bug | Finding(s)  | Status   | Detail                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                |
| --- | ----------- | -------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | E005 ×3     | **DONE** | **Primary fix**: `scanGenericHandlerCall` detects generic type-instantiation calls `requireCommandType[*T](cmd)` in handler bodies and marks T as registered — this is how browser-history actually links handlers to command structs (the const value is an event-style string `"browser_history.extract_history"`, not a struct name, so const-value resolution alone cannot bridge the gap). **Secondary path**: const-decl cross-referencing (`ResolveRegisteredTypeConsts`) handles consumers whose const values ARE struct names. Regression tests: `TestE005_NoFindingWhenRegisteredViaDispatcherRegisterAndTypeConst`, `TestE005_NoFindingWhenHandlerUsesRequireCommandType`. |
| 2   | E007 ×6     | **DONE** | Same generic type-scanning fix: `requireQueryType[*T](q)` marks T as registered. Also removes a redundant `*ast.StructType` assertion that rejected non-struct query types. Regression tests: `TestE007_NoFindingWhenRegisteredViaTypeConstAndMethodValue`, `TestE007_FiresWhenTypeConstExistsButIsNeverRegistered` (guards against over-broad suppression), `TestE007_NoFindingWhenHandlerUsesRequireQueryType`.                                                                                                                                                                                                                                                                     |
| 3   | B007 ×1     | **DONE** | `nonCQRSRegisterPackages` denylist (`huma`, `http`, `mux`, `chi`, `gin`, `echo`, `fiber`, `grpc`) consulted via `analyzer.SelectorPackage(sel)`. Variable qualifiers (`d`, `cmdDisp`) are never denied — the idiomatic CQRS pattern. Regression tests: `TestB007_NoFindingForHumaRegister`, `TestB007_CountsCQRSButSkipsHumaInSameFunction`.                                                                                                                                                                                                                                                                                                                                          |
| 4   | B005 latent | **DONE** | `StrictApplyFolds` registry set populated by scanning `decider.StrictApply(foldName, ...)` calls. B005 suppresses folds whose `FuncName` (or its trailing identifier segment, to handle method receivers) is in the set. Regression tests: `TestB005_NoFindingWhenFoldIsWrappedInStrictApply`, `TestB005_FiresForUnwrappedFoldWhenAnotherFoldIsWrapped`.                                                                                                                                                                                                                                                                                                                              |

**How the E005/E007 fix works**: browser-history registers commands via
`dispatcher.Register(typeConst, handler)` and queries via
`query.RegisterTyped(disp, typeConst, methodValue)`. The type constants use
event-style string values (`"browser_history.extract_history"`), not struct
names. The handler→struct link is only visible inside handler bodies through
generic type assertions: `requireCommandType[*ExtractVisitCommand](cmd)`.
`scanGenericHandlerCall` recovers this link by detecting any generic
instantiation `X[*T](...)` or `X[T](...)` where T ends in "Command" or "Query",
and marking T as registered in `CommandTypesRegistered`. This is intentionally
general — it does not depend on consumer-specific helper function names.

**Additional patterns recognized** (added during cross-consumer verification):

1. **Generic type assertions** (`requireCommandType[*T]`) — browser-history.
2. **Package-qualified closure params** (`func(ctx, cmd *pkg.MyCmd)`) —
   `handlerTypeFromClosure` now extracts the trailing identifier from
   SelectorExpr params via `lastIdentSegment`. Fixes SwettySwipper closures.
3. **Method-value handlers** (`RegisterTyped(disp, typeConst, h.handleX)`) —
   when the handler is a method value, the method name is recorded and
   resolved in a post-pass (`ResolveHandlerMethods`) by finding the FuncDecl
   and extracting the command/query type from its parameter list. Fixes SEC.
4. **Type assertions in opaque closures** (`cmd.(*MyCmd)`) —
   `scanTypeAssertion` detects `*ast.TypeAssertExpr` nodes where the asserted
   type ends in "Command"/"Cmd"/"Query" and marks it as registered. Fixes
   SwettySwipper's RegisterAll map pattern (closures that take
   `corecmd.Command` interface and type-assert internally).
5. **`lastIdentSegment` now handles `*ast.StarExpr`** — type assertions like
   `cmd.(*MyCmd)` have the asserted type as `*ast.StarExpr`, not bare Ident.

**End-to-end verification** (2026-07-20): ran the rebuilt `cqrs-lint` binary
against **7 consumer repos** (all local sibling checkouts):

| Repo            | Files | E005 | E007 | B007 | B005 | Notes                                                                     |
| --------------- | ----- | ---- | ---- | ---- | ---- | ------------------------------------------------------------------------- |
| browser-history | 30    | 0    | 0    | 0    | 0    | All 10 reported FPs resolved                                              |
| bank-sync       | 47    | 0    | 0    | 0    | 1    | B005 is a correct finding (unwrapped fold)                                |
| DiscordSync     | 204   | 0    | 0    | 0    | 0    | Clean                                                                     |
| cqrs-htmx       | 146   | 0    | 1    | 0    | 4    | 1 E007 in example/basic (unexported type, no evidence)                    |
| Cyberdom        | 13    | 0    | 0    | 0    | 0    | Clean                                                                     |
| SEC             | 75    | 0    | 0    | 0    | 0    | All E005/E007 resolved (method-value pattern)                             |
| SwettySwipper   | 132   | 2    | 5    | 0    | 6    | 2 E005 = handlers with zero type-level evidence; 5 E007 = opaque closures |

**Total: from 44 false positives across all consumers to 8 remaining**, all of
which are either legitimate gaps (handlers that don't reference the command
type at all) or use patterns with zero type-level evidence the analyzer can
trace. The 8 remaining are documented known limitations.

**Verification**: full `cmd/cqrs-lint` test suite passes including 12 new
regression tests; `go test -race ./pkg/...` clean; `go vet` clean; `nix fmt`
clean.

**Part 1 (4 fixes) and Part 3 (3 intentional deviations)** require no upstream
action — they were applied or declined in the consumer's repo.
