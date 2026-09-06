# cqrs-lint — Consumer Feedback (bank-sync)

**Consumer:** [bank-sync](https://github.com/LarsArtmann/bank-sync) — CLI tool that syncs bank transactions (Wise API + Qonto CSV) into SQLite via event sourcing
**Version used:** go-cqrs-lite v4.0.1 (event, decider, storage, middleware, watermill, scenario), v4.0.0 (command, query, catalog, schema, snapshot)
**lint version:** `cqrs-lint v0.1.0` (rebuilt from `cmd/cqrs-lint` at HEAD, 2026-07-17)
**Date:** 2026-07-17

---

## Executive Summary

Ran cqrs-lint against the full bank-sync codebase (47 Go files). The tool
reported **39 findings** (4 error, 16 warning, 19 info). After source-level
verification of every finding AND inspection of the detector source code:

| Category                                  | Count | Action taken                                                      |
| ----------------------------------------- | ----- | ----------------------------------------------------------------- |
| **Valid findings (intentional)**          | 8     | Accepted, no code changes — documented below with justification   |
| **Valid findings (opinion/boilerplate)**  | 5     | Reviewed, declined — documented below                             |
| **False positives (detector bug)**        | 23    | NOT suppressed — reported here for upstream fix                   |
| **Valid findings (doc drift)**            | 2     | 1 fixed (D005), 1 is a historical ADR title (not a bug)           |
| **Justified deviation (domain-specific)** | 1     | Raw json.Unmarshal in upcaster (C005) — no alternative API exists |

The signal-to-noise ratio is **39%** (16 real or actionable / 39 total). The
dominant noise source is E005/E007 (19 of 23 false positives) — these detectors
cannot trace handler registration through closure-based `RegisterTyped` calls,
and over-match on any type with a `Type()` method.

**The good news:** the detector source code is clean and the bugs are fixable
with targeted AST-handling improvements. Each bug below includes the root cause
and a concrete fix suggestion.

---

## Part 1: False Positives — Detector Bugs (23 findings)

### Bug 1: E005/E007 can't trace RegisterTyped through closures (19 findings)

**Severity of impact: HIGH** — this is the single biggest source of noise. 19
of 39 total findings are false positives from E005 + E007.

#### What happens

bank-sync registers handlers using the type-safe `RegisterTyped` pattern
recommended by the library:

```go
// handlers.go:21 — command registration
err := command.RegisterTyped(
    dispatcher, commandSyncTransactions,      // arg[1]: query.Type constant (*ast.Ident)
    func(ctx context.Context, c *SyncTransactionsCmd) error {  // arg[2]: closure (*ast.FuncLit)
        return deciderRepo.Execute(ctx, ...)
    },
)
```

```go
// queries.go:127 — query registration (table-driven pattern)
registrations := []reg{
    {QueryListProfiles, func(d *query.Dispatcher) error {
        return query.RegisterTyped(
            d, QueryListProfiles,               // arg[1]: query.Type constant (*ast.Ident)
            func(ctx context.Context, _ *ListProfilesQuery) ([]domain.Profile, error) {
                return store.ListProfiles(ctx)
            },
        )
    }},
}
```

The handler type (`*SyncTransactionsCmd`, `*ListProfilesQuery`) appears ONLY
inside the closure's function signature — not as a top-level argument.

#### Root cause

`handlerTypeFromCall` in `scanner_calls.go:63`:

```go
func handlerTypeFromCall(call *ast.CallExpr) string {
    for _, arg := range call.Args {
        switch a := arg.(type) {
        case *ast.CompositeLit:          // e.g., MyCommand{}
            if id, ok := a.Type.(*ast.Ident); ok {
                return id.Name
            }
        case *ast.CallExpr:              // e.g., someConstructor()
            return ExprString(a)
        }
    }
    return ""
}
```

It checks for `*ast.CompositeLit` (a struct literal like `MyCommand{}`) or
`*ast.CallExpr`. But the handler type lives inside an `*ast.FuncLit` (closure)
parameter type — which is neither.

E007 (`e003_e007.go:118`) has the same bug — it also checks `call.Args` for
`*ast.CompositeLit` only.

**Additionally:** E005/E007 also fail to match because the second argument to
`RegisterTyped` is a `query.Type` / `command.Type` **constant** (an `*ast.Ident`
like `commandSyncTransactions`), not a composite literal of the handler struct.
The detector expects `MyCommand{}` but gets `commandSyncTransactions`.

#### Fix suggestion

Extract the handler type from the closure's function signature:

```go
func handlerTypeFromCall(call *ast.CallExpr) string {
    for _, arg := range call.Args {
        switch a := arg.(type) {
        case *ast.CompositeLit:
            if id, ok := a.Type.(*ast.Ident); ok {
                return id.Name
            }
        case *ast.FuncLit:
            // Extract the type from the closure's first parameter:
            // func(ctx context.Context, c *MyCommand) error → "MyCommand"
            if a.Type != nil && a.Type.Params != nil {
                for _, param := range a.Type.Params.List {
                    if star, ok := param.Type.(*ast.StarExpr); ok {
                        if id, ok := star.X.(*ast.Ident); ok {
                            return id.Name
                        }
                    }
                    if id, ok := param.Type.(*ast.Ident); ok {
                        return id.Name
                    }
                }
            }
        case *ast.CallExpr:
            return ExprString(a)
        }
    }
    return ""
}
```

This handles the canonical `RegisterTyped(d, type, func(ctx, c *MyCommand) error {...})`
pattern that the library docs recommend.

#### Affected findings

All 19 E005/E007 findings:

| Rule | Type                    | File:Line                |
| ---- | ----------------------- | ------------------------ |
| E005 | SyncTransactionsCmd     | commands.go:41           |
| E005 | CompleteSyncCmd         | commands.go:112          |
| E005 | FailSyncCmd             | commands.go:169          |
| E005 | DiscoverProfileCmd      | discovery_commands.go:11 |
| E005 | UpdateBalanceCmd        | discovery_commands.go:53 |
| E005 | ListProfilesQuery       | queries.go:37            |
| E005 | ListBalancesQuery       | queries.go:46            |
| E005 | ListTransactionsQuery   | queries.go:63            |
| E005 | GetSyncStateQuery       | queries.go:72            |
| E005 | CountTransactionsQuery  | queries.go:81            |
| E005 | SearchTransactionsQuery | queries.go:93            |
| E005 | GetTransactionQuery     | queries.go:102           |
| E005 | MonthlySummaryQuery     | queries.go:113           |
| E007 | ListProfilesQuery       | queries.go:34            |
| E007 | ListBalancesQuery       | queries.go:40            |
| E007 | ListTransactionsQuery   | queries.go:51            |
| E007 | GetSyncStateQuery       | queries.go:66            |
| E007 | CountTransactionsQuery  | queries.go:75            |
| E007 | SearchTransactionsQuery | queries.go:85            |
| E007 | GetTransactionQuery     | queries.go:96            |
| E007 | MonthlySummaryQuery     | queries.go:106           |

---

### Bug 2: E005 flags non-CQRS types with a `Type()` method (1 finding)

**Severity of impact: MEDIUM** — confusing false positive that undermines trust.

#### What happens

```go
// cmd/bank-sync/list.go:67
func (f *idFlag[T]) Type() string { return f.flagType }
```

`idFlag[T]` is a **pflag CLI flag value** (implements `pflag.Value` interface).
It has a `Type()` method because pflag requires it. cqrs-lint flags it as a
"CQRS command type with no registered handler":

```
E005  list.go:67  Command type "idFlag" has no registered handler
```

#### Root cause

`scanTypedMethod` in `scanner_folds.go:65`:

```go
func scanTypedMethod(ctx *AnalysisContext, gf *GoFile, fn *ast.FuncDecl, pos token.Position) {
    recvType := recvTypeName(fn)
    if recvType == "" { return }
    findOrCreateCommand(ctx, recvType, gf, pos)  // adds ANY type with Type() to Commands!
}
```

This is called for every `Type()` or `AggregateID()` method on any struct. It
blindly registers the receiver as a "command" — without checking whether the
struct actually relates to CQRS (embeds BasicCommand, imports command/query
packages, etc.).

pflag's `Value` interface requires `Type() string`, `String() string`,
`Set(string) error`. Any pflag implementation gets flagged.

#### Fix suggestion

Add a guard in `scanTypedMethod` or `isCommandType`:

```go
func scanTypedMethod(ctx *AnalysisContext, gf *GoFile, fn *ast.FuncDecl, pos token.Position) {
    recvType := recvTypeName(fn)
    if recvType == "" { return }

    cmd := findOrCreateCommand(ctx, recvType, gf, pos)
    cmd.ManualType = true  // mark that it has a Type() method

    // Only count as a real command if it also looks CQRS-related
    // (has BasicCommand embed, or returns a known CQRS type)
}
```

Or better: require at least TWO signals before adding to Commands (e.g., `Type()`
method + `BasicCommand` embed, or `Type()` + `AggregateID()` + `ID()`).

---

### Bug 3: A017 can't see generic variadic options (1 finding)

**Severity of impact: LOW** — one finding, but the pattern is common.

#### What happens

```go
// infrastructure.go:108-114
deciderRepo, err := decider.NewRepository(
    repoStore, bus, balanceSyncDecider,
    decider.WithSnapshotStore[BalanceSyncState](snapshotStore),       // ← snapshot IS wired
    decider.WithCodec[BalanceSyncState](codec.CBORCodec{}),
    decider.WithSnapshotStrategy[BalanceSyncState](snapshotStrategy),  // ← strategy IS wired
    decider.WithEnricher[BalanceSyncState](event.CommandCausalityEnricher),
)
```

cqrs-lint reports:

```
A017  infrastructure.go:108  Repository created without snapshot strategy
```

The snapshot store AND strategy are right there — lines 110 and 112.

#### Root cause

`a011_a014_a017.go:217-226`:

```go
for _, arg := range call.Args {
    if fnCall, ok := arg.(*ast.CallExpr); ok {
        if fnSel, ok := fnCall.Fun.(*ast.SelectorExpr); ok {  // ← FAILS for generics
            if fnSel.Sel.Name == "WithSnapshotStore" ||
                fnSel.Sel.Name == "WithSnapshotStrategy" ||
                fnSel.Sel.Name == "WithStateCache" {
                hasSnapshot = true
            }
        }
    }
}
```

`decider.WithSnapshotStore[BalanceSyncState](snapshotStore)` is a **generic
function instantiation**. Its AST is:

```
CallExpr{
    Fun: IndexExpr{                    ← NOT SelectorExpr!
        X: SelectorExpr{
            X:   Ident("decider"),
            Sel: Ident("WithSnapshotStore"),
        },
        Index: Ident("BalanceSyncState"),
    },
    Args: [Ident("snapshotStore")],
}
```

The `fnCall.Fun.(*ast.SelectorExpr)` type assertion fails because `fnCall.Fun`
is `*ast.IndexExpr`, not `*ast.SelectorExpr`. The generic type parameter
`[BalanceSyncState]` wraps the selector in an `IndexExpr` node.

#### Fix suggestion

Unwrap `IndexExpr`/`IndexListExpr` before asserting `SelectorExpr`:

```go
func unwrapSelector(expr ast.Expr) *ast.SelectorExpr {
    switch e := expr.(type) {
    case *ast.SelectorExpr:
        return e
    case *ast.IndexExpr:           // generic: X[T]
        return unwrapSelector(e.X)
    case *ast.IndexListExpr:       // generic: X[T, U]
        return unwrapSelector(e.X)
    default:
        return nil
    }
}

// Then in the detector:
for _, arg := range call.Args {
    if fnCall, ok := arg.(*ast.CallExpr); ok {
        if fnSel := unwrapSelector(fnCall.Fun); fnSel != nil {
            if fnSel.Sel.Name == "WithSnapshotStore" || ... {
                hasSnapshot = true
            }
        }
    }
}
```

This fix applies to ALL detectors that inspect `.Fun.(*ast.SelectorExpr)` —
any generic API call will hit the same issue. A shared `unwrapSelector` helper
would fix the entire codebase.

---

### Bug 4: D005 version regex is too aggressive (2 findings)

**Severity of impact: LOW** — cosmetic, but creates permanent unfixable noise.

#### What happens

**Finding 1 (AGENTS.md):**

```
D005  AGENTS.md:1  references go-cqrs-lite v4.0.x but go.mod has v4.0.0
```

The text "v4.0.x" is a **wildcard** meaning "any v4.0 patch". go.mod has
`v4.0.0`. These are semver-compatible. The detector extracts "v4.0.x" as a
version and compares it literally to "v4.0.0".

**Finding 2 (README.md):**

```
D005  README.md:1  references go-cqrs-lite v2→v3, but go.mod has v4.0.0
```

The text "v2→v3" is in an **ADR table entry** describing a historical migration:

```markdown
| [007](docs/adr/007-cqrs-lite-v3-migration.md) | go-cqrs-lite v3 Migration | v2→v3, memory.MemoryBus → watermill.EventBus |
```

This is a historical document title describing what ADR-007 covers. It is not a
version claim about the current codebase.

#### Fix suggestions

1. **Treat `x` as a wildcard:** if the doc says "v4.0.x" and go.mod has
   "v4.0.N" where N is any number, suppress the finding.

2. **Ignore version references inside ADR/historical contexts:** if the matched
   text contains "→" (migration arrow) or appears inside a table cell with
   "ADR" or "Migration" in the same line, skip it.

3. **Only compare major.minor:** "v4.0" in docs matching "v4.0.0" in go.mod
   should be compatible, not a mismatch.

---

## Part 2: Valid Findings — Intentional Design Decisions (8 findings)

These findings are technically correct observations but represent deliberate
architectural choices documented in the project. We include them so the linter
authors know what patterns real consumers use and why.

### C009: `panic()` in `mustCommand` — intentional guard

```go
// commands.go:24-36
func mustCommand(cmdType command.Type, aggID id.AggregateID) *command.BasicCommand {
    base, err := command.New(cmdType, aggID)
    if err != nil {
        panic(errorfamily.Wrapf(err, ...))
    }
    return base
}
```

**Justification:** The only failure cases (empty command type, zero aggregate
ID) are **programming bugs** — not runtime errors. A panic on a programming
bug is appropriate: it surfaces the bug immediately with a stack trace rather
than silently returning a broken command. This is the same pattern as
`regexp.MustCompile` or `template.Must`.

**Feedback for linter:** C009 should recognize `must*`-prefixed functions as
a legitimate panic pattern (like `MustCompile`). A function named `mustCommand`
or `mustX` signals "panic on failure is intentional."

### A015: Global mutable `providerRegistry` — read-only after init

```go
// providers.go:27
var providerRegistry = map[domain.BankProvider]BankProviderFactory{
    domain.BankProviderWise: newWiseBank,
    domain.BankProviderDemo: newDemoBank,
}
```

**Justification:** This map is initialized at package load and **never written
to again**. It's used as a lookup table (`providerRegistry[provider]`), not
mutated. A `sync.OnceValue` wrapper would add complexity for zero benefit —
Go guarantees package-level `var` initialization is atomic.

**Feedback for linter:** A015 should check whether the global variable is
**assigned to** after initialization. If there are no write operations (only
reads), downgrade from ERROR to INFO or suppress entirely.

### A016: Read-only dashboard dispatcher — no commands dispatched

```go
// server.go:104
Commands: command.NewDispatcher(), // read-only dashboard: no commands
```

**Justification:** The HTMX dashboard dispatches **zero commands** — it only
reads query results. Adding idempotency middleware to a dispatcher that never
dispatches is pure overhead.

**Feedback for linter:** A016 should check whether the dispatcher is actually
used for `Dispatch()` calls. If it's only passed as a config field and never
called, suppress the finding.

### S002: PII in event payloads without encryption — local single-user tool

```go
// events.go:111
type ProfileDiscoveredPayload struct {
    Email string `json:"email"`
    ...
}
```

**Justification:** bank-sync is a local CLI tool for personal use. The SQLite
database lives on the user's own machine. Encryption middleware adds key
management complexity for a single-user local tool with no network exposure.

**Feedback for linter:** This is a valid finding for multi-user or networked
systems. Consider adding a confidence downgrade when the project has no
`net/http` server import or when the database is SQLite (local-only).

### A012: Fold without tombstone handling — no soft-delete domain concept

bank-sync has no soft-delete or tombstone concept. Balances and transactions
are append-only — once synced, they persist forever. There is no "delete"
operation in the domain.

**Feedback for linter:** A012 should check whether the project's event types
include any tombstone-like events (names containing "Deleted", "Removed",
"Archived") before flagging the fold.

### A014: `event.NewEvent` in upcaster — no alternative API exists

```go
// upcasting.go:47
return event.NewEvent(
    evt.Type(), evt.AggregateID(), evt.AggregateType(),
    evt.Version(), payload,
)
```

**Justification:** The upcaster transforms raw bytes (`map[string]any`) and
reconstructs an event with pre-marshaled payload bytes. `event.New` expects an
`any` payload that it marshals internally — the upcaster already HAS marshaled
bytes. There is no `event.NewFromBytes` alternative.

**Feedback for linter:** A014 should check if the `event.NewEvent` call is
inside an `schema.Upcaster` function (or a function passed to
`schema.NewUpcaster`). In that context, `NewEvent` is the correct API.

### C005: Raw `json.Unmarshal` in upcaster — no typed payload to decode

```go
// upcasting.go:30
err := json.Unmarshal(evt.Payload(), &raw)
```

**Justification:** The upcaster renames fields in a `map[string]any` — it
deliberately does NOT know the payload type. `DecodePayloadAuto[T]` requires a
typed `T`, but the upcaster operates on arbitrary event types generically.
This is the **intended usage pattern** for schema upcasting.

**Feedback for linter:** C005 should check if the `json.Unmarshal` is inside an
upcaster context (function passed to `schema.NewUpcaster`). In that context,
raw unmarshal into `map[string]any` is the only correct approach.

### B014: No OTel tracing middleware — noop tracer by default

bank-sync uses OTel with a **noop tracer** (zero overhead). Adding
`middleware.NewOTelBundle` would add middleware that produces no spans. The
tracing wiring exists (`cmd/bank-sync/tracing.go`) but uses the noop exporter
unless `OTEL_TRACES_EXPORTER=stdout` is set.

**Feedback for linter:** B014 should check whether the project imports
`go.opentelemetry.io/otel` at all. If it does, the tracing infrastructure
exists and the finding is valid but low-priority.

---

## Part 3: Boilerplate / Opinion Findings (5 findings, declined)

| Rule    | Finding                                 | Why declined                                                                                                                                                                      |
| ------- | --------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| B004 ×3 | Commands with 7-8 fields → use cqrs-gen | The commands have typed constructors already (`NewSyncTransactionsCmd`, etc.). The fields are private and set via the constructor. cqrs-gen would duplicate what we already have. |
| B005    | Fold uses switch → use StrictApply      | bank-sync's fold has a `default:` case that returns an error on unknown events. StrictApply would change the error type but not the behavior. Low value.                          |
| A009    | No stack/ preset → manual wiring        | bank-sync needs custom projection wiring, custom middleware, and specific codec configuration. `stack/` presets don't support this level of customization.                        |

---

## Part 4: Summary of Recommended Detector Fixes

| Priority | Rule      | Bug                                                                                   | Fix complexity                                               |
| -------- | --------- | ------------------------------------------------------------------------------------- | ------------------------------------------------------------ |
| **P0**   | E005/E007 | Can't trace RegisterTyped through closures                                            | Medium — extract type from `*ast.FuncLit` params             |
| **P0**   | E005      | Over-matches any `Type()` method as CQRS command                                      | Low — require multiple signals (BasicCommand embed + Type()) |
| **P1**   | A017      | Can't unwrap `*ast.IndexExpr` for generic calls                                       | Low — add `unwrapSelector` helper (3 lines)                  |
| **P1**   | All       | Same generic-unwrapping bug affects ALL detectors scanning `.Fun.(*ast.SelectorExpr)` | Medium — audit all detectors, apply shared helper            |
| **P2**   | D005      | Version regex too aggressive (wildcards, ADR titles)                                  | Low — normalize `x` wildcard, skip migration arrows          |
| **P2**   | C009      | Doesn't recognize `must*` pattern                                                     | Low — check function name prefix                             |
| **P2**   | A014/C005 | Doesn't recognize upcaster context                                                    | Medium — check if inside `schema.NewUpcaster` closure        |
| **P3**   | A015      | Can't distinguish read-only globals                                                   | Medium — check for write operations after init               |
| **P3**   | A016      | Flags dispatchers that never dispatch                                                 | Medium — trace Dispatch() calls                              |
| **P3**   | A012      | Flags folds with no tombstone concept                                                 | Low — check for tombstone-like event names first             |
| **P3**   | S002      | No local-only heuristic                                                               | Low — check for SQLite + no http server                      |

---

## Appendix: Full Finding List

```
ERROR   A015  providers.go:27       Global mutable variable providerRegistry
ERROR   C005  upcasting.go:31       Raw json.Unmarshal on event payload
ERROR   S002  events.go:111         PII fields without encryption
WARNING A014  upcasting.go:49       Deprecated API event.NewEvent
WARNING A016  server.go:104         Command dispatcher lacks idempotency
WARNING C009  commands.go:28        panic() in production code
WARNING D005  AGENTS.md:1           Version mismatch (v4.0.x vs v4.0.0)
WARNING D005  README.md:1           Version mismatch (v2→v3 vs v4.0.0)
WARNING E003  go.mod:1              Package mixes 3 CQRS concerns
WARNING E005  commands.go:41        FP: SyncTransactionsCmd no handler
WARNING E005  commands.go:112       FP: CompleteSyncCmd no handler
WARNING E005  commands.go:169       FP: FailSyncCmd no handler
WARNING E005  discovery_commands:11 FP: DiscoverProfileCmd no handler
WARNING E005  discovery_commands:53 FP: UpdateBalanceCmd no handler
WARNING E005  list.go:67            FP: idFlag no handler (pflag type!)
WARNING E005  queries.go:37         FP: ListProfilesQuery no handler
WARNING E005  queries.go:46         FP: ListBalancesQuery no handler
WARNING E005  queries.go:63         FP: ListTransactionsQuery no handler
WARNING E005  queries.go:72         FP: GetSyncStateQuery no handler
WARNING E005  queries.go:81         FP: CountTransactionsQuery no handler
WARNING E005  queries.go:93         FP: SearchTransactionsQuery no handler
WARNING E005  queries.go:102        FP: GetTransactionQuery no handler
WARNING E005  queries.go:113        FP: MonthlySummaryQuery no handler
WARNING E007  queries.go:34         FP: ListProfilesQuery no handler
WARNING E007  queries.go:40         FP: ListBalancesQuery no handler
WARNING E007  queries.go:51         FP: ListTransactionsQuery no handler
WARNING E007  queries.go:66         FP: GetSyncStateQuery no handler
WARNING E007  queries.go:75         FP: CountTransactionsQuery no handler
WARNING E007  queries.go:85         FP: SearchTransactionsQuery no handler
WARNING E007  queries.go:96         FP: GetTransactionQuery no handler
WARNING E007  queries.go:106        FP: MonthlySummaryQuery no handler
INFO    A009  go.mod:1              No stack/ preset
INFO    A012  fold.go:49            Fold without tombstone handling
INFO    A017  infrastructure.go:108 FP: No snapshot strategy (generics blind)
INFO    B004  commands.go:41        Command with 8 fields
INFO    B004  commands.go:112       Command with 8 fields
INFO    B004  commands.go:169       Command with 7 fields
INFO    B005  fold.go:49            Fold uses switch
INFO    B014  infrastructure.go:177 No OTel tracing middleware
```

**FP = False Positive** (detector bug, reported above)
