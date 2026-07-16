# Domain-Aware CQRS Linter: Research and Design

> **Date**: 2026-07-16 (revised)
> **Scope**: Design a domain-aware linter for `go-cqrs-lite` consumers
> **Based on**: Analysis of 22 consumer projects, existing tooling (`cqrs-gen`, `api-stability`, `doc-check`), and the `go-structure-linter` architecture
> **Companion document**: [Consumer Projects Analysis](../../../docs/go-cqrs-lite-consumer-projects-analysis.md)

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [Architecture Options](#2-architecture-options)
3. [Rule Catalog](#3-rule-catalog)
4. [Detection Logic](#4-detection-logic)
5. [False-Positive Mitigation](#5-false-positive-mitigation)
6. [Auto-Fix Capabilities](#6-auto-fix-capabilities)
7. [Recommendation Engine](#7-recommendation-engine)
8. [Test Fixture Strategy](#8-test-fixture-strategy)
9. [Performance and Caching](#9-performance-and-caching)
10. [Implementation Blueprint](#10-implementation-blueprint)
11. [Priority Matrix](#11-priority-matrix)
12. Appendix A: Issues by Project
13. Appendix B: Library Improvement Feedforward
14. Appendix C: Pipeline Architecture Diagram

---

## 1. Executive Summary

A domain-aware linter for go-cqrs-lite would address **systemic anti-patterns found in 11 of 22 consumer projects** and eliminate **5 universal boilerplate patterns** reimplemented in every project.

The linter should be built as a **standalone tool** using the `go-structure-linter` architecture (Rule interface + Registry + `go-finding` pipeline), enhanced with **AST analysis** from the `cqrs-gen` scanning foundation. This dual approach enables both architectural checks (cross-file, cross-module) and semantic checks (per-function, per-type).

**Key numbers:**

- **47 rules** identified across 6 categories
- **18 rules** are auto-fixable
- **12 rules** detect correctness bugs (silent data loss, broken idempotency)
- **20 rules** detect API misuse (wrong function, wrong pattern)
- **15 rules** detect boilerplate (redundant code, missing helpers)
- **11 of 22 projects** have at least one Critical or High severity violation

---

## 2. Architecture Options

### Option A: `go/analysis` Plugin (golangci-lint integration)

**Pros:**

- Integrates with existing CI (golangci-lint, go vet)
- Type information available (`pass.TypesInfo`, `pass.Pkg.Imports`)
- Per-package incremental analysis
- Mature ecosystem — consumers already know how to configure it

**Cons:**

- Per-package scope — `analysis.Pass` covers one package at a time. Cross-package rules (E004: event-not-in-catalog, E005: command-without-handler) require a `analysis.Pass` of `fact` types or a driver that aggregates multiple passes. This is possible but cumbersome.
- No project-level context (directory structure, go.mod analysis). The analyzer cannot inspect the filesystem layout.
- Auto-fix via `analysis.SuggestedFixes` is limited — no dry-run, no compile-check, no batch rollback.
- Consumers must configure golangci-lint to enable it.

**Best for:** Single-function semantic checks (C005, C006, C007) where type information and incremental analysis matter.

### Option B: Standalone Tool (go-structure-linter architecture) - RECOMMENDED

**Pros:**

- Cross-file, cross-module analysis — the rules need this (E004, E005, E006 require scanning all packages and cross-referencing)
- Project-level context (go.mod, directory structure, module graph)
- Proven architecture across 2 existing tools (`go-structure-linter` has 70 rules, `golangci-lint-auto-configure`)
- Rich auto-fix support (DryRunFix, ApplyFix, ValidateFix with compile-check)
- `go-finding` for unified output (SARIF, JSON, console, GitHub Annotations)
- SDK for programmatic use (`sdk.New -> Lint -> Result`)
- Already has CQRS-aware AST scanning (from `cqrs-gen`)
- Supports `go-finding/pipeline` for parallel rule execution with panic recovery

**Cons:**

- Consumers must install and run separately from golangci-lint
- More code to maintain than a single analyzer

### Option C: Hybrid (both)

Build the standalone tool for project-level rules AND ship a `go/analysis` plugin that wraps the per-function rules for golangci-lint consumers. The standalone tool uses `golang.org/x/tools/go/packages` with `packages.NeedTypes | packages.NeedTypesInfo | packages.NeedSyntax` to get full type information — the `go/analysis` plugin is a thin adapter.

**Verdict:** Start with **Option B** (standalone tool). It covers all identified rules and has the richest auto-fix support. Add `go/analysis` plugin (Option C) later if golangci-lint integration is requested.

### Type Information Strategy

The standalone tool uses `golang.org/x/tools/go/packages` to load type information:

```go
cfg := &packages.Config{
    Mode: packages.NeedName | packages.NeedTypes | packages.NeedTypesInfo |
          packages.NeedSyntax | packages.NeedImports,
    Tests: false,
}
pkgs, err := packages.Load(cfg, "./...")
```

This gives us:

- `pkg.TypesInfo` — type checking results (types of expressions, method sets)
- `pkg.Syntax` — parsed AST files (`[]*ast.File`)
- `pkg.Types` — the `*types.Package` for cross-referencing
- `pkg.Imports` — imported packages (to check which go-cqrs-lite modules are used)

All AST inspection builds on the patterns already proven in `cqrs-gen` (see Section 4).

---

## 3. Rule Catalog

### 3.1 Category 1: Correctness (Bug Detection) - 12 rules

These rules detect bugs that cause silent data loss, broken idempotency, or incorrect behavior.

| ID   | Rule                               | Severity | Auto-Fix | Confidence | Projects Affected                           |
| ---- | ---------------------------------- | -------- | -------- | ---------- | ------------------------------------------- |
| C001 | `missing-tx-commit`                | Critical | Yes      | High       | DiscordSync                                 |
| C002 | `broken-command-id`                | Critical | Yes      | High       | Kernovia                                    |
| C003 | `silent-unknown-event-fold`        | High     | Yes      | High       | KeyCountdown                                |
| C004 | `checkpoint-before-async-complete` | High     | No       | Medium     | DiscordSync                                 |
| C005 | `raw-json-unmarshal-payload`       | High     | Yes      | High       | storbi, KeyCountdown                        |
| C006 | `manual-version-arithmetic`        | Medium   | Yes      | High       | SBTS, Standup-Killer, KeyCountdown, Zlota44 |
| C007 | `time-now-in-decider`              | Medium   | No       | Medium     | Zlota44, Kernovia, SEC                      |
| C008 | `float64-for-money`                | Medium   | No       | Medium     | Zlota44                                     |
| C009 | `panic-in-production-path`         | Medium   | No       | High       | DiscordSync (4 sites)                       |
| C010 | `swallowed-error-in-fold`          | Medium   | Yes      | High       | SEC                                         |
| C011 | `nondeterministic-decider`         | Medium   | No       | Low        | SEC (diceService)                           |
| C012 | `missing-error-return-in-with-tx`  | Critical | Yes      | High       | DiscordSync                                 |

> **Confidence** indicates how certain the rule is that a finding is a real issue, not a false positive. High = near-zero false positives. Low = requires human judgment.

---

#### C001: Missing Transaction Commit

**Detects:** A `withTx` or similar transaction helper whose success path returns `nil` instead of `tx.Commit()`.

**Pattern:**

```go
// BAD: if body returns nil without committing, data is silently lost
func withTx(ctx context.Context, db *sql.DB, body func(*sql.Tx) error) error {
    tx, _ := db.BeginTx(ctx, nil)
    if err := body(tx); err != nil { _ = tx.Rollback(); return err }
    return nil  // BUG: should be tx.Commit()
}
```

**Detection logic (concrete AST):**

```
1. Walk ast.FuncDecl nodes with return type `error` (single return value)
2. In the function body, find a call to (*sql.DB).BeginTx or (*sql.DB).Begin
   — AST: ast.CallExpr where Fun is ast.SelectorExpr with Sel.Name == "BeginTx" or "Begin"
3. Identify the tx variable name from the LHS of the assignment
4. Walk all return statements in the function
5. If any return statement returns nil AND tx.Commit() is never called
   in the same function → FLAG
```

**AST selector (Go pseudo-code):**

```go
ast.Inspect(funcDecl, func(n ast.Node) bool {
    // Step 2: find BeginTx call
    call, ok := n.(*ast.CallExpr)
    if !ok { return true }
    sel, ok := call.Fun.(*ast.SelectorExpr)
    if !ok { return true }
    if sel.Sel.Name != "BeginTx" && sel.Sel.Name != "Begin" { return true }

    // Step 3: get tx variable name
    txVarName := getAssignedVarName(funcDecl, call)  // e.g., "tx"

    // Step 4: check all return statements
    hasCommit := containsCallTo(funcDecl, txVarName, "Commit")
    hasNilReturn := hasReturnNil(funcDecl)

    // Step 5: flag
    if hasNilReturn && !hasCommit {
        return Issue{Rule: "C001", ...}
    }
    return true
})
```

**False-positive scenarios:**

- Function delegates to `body(tx)` which calls `tx.Commit()` internally (this IS the bug — the convention is undocumented and fragile). Suppression: `//cqrs-lint:ignore(C001) body commits intentionally`
- Function uses `defer tx.Commit()` — this is valid, the linter should check for `defer` before flagging.

**Auto-fix:**

```go
// OLD: return nil
// NEW: return tx.Commit()
```

Risk: **Low** — unambiguous when tx variable is identified.

---

#### C002: Broken Command ID

**Detects:** Command structs whose `ID()` method returns a zero-value `CommandID`.

**Pattern:**

```go
// BAD: ID() returns zero value, breaking idempotency + tracing
func (c LoadPluginCommand) ID() cqrsid.CommandID {
    return cqrsid.CommandID{}  // always zero
}
```

**Detection logic (concrete AST):**

```
1. Walk ast.FuncDecl nodes where:
   - Recv != nil (it's a method)
   - Name.Name == "ID"
   - Return type is CommandID (resolved via packages.TypesInfo)
2. Check the function body for a return statement
3. If the return value is a composite literal with empty fields: ast.CompositeLit
   with Elts == nil or len(Elts) == 0 → FLAG
```

**AST selector:**

```go
for _, decl := range file.Decls {
    fn, ok := decl.(*ast.FuncDecl)
    if !ok || fn.Recv == nil || fn.Name.Name != "ID" { continue }

    // Check return type resolves to CommandID
    if !returnsCommandID(fn, typeInfo) { continue }

    // Check for zero-value return
    ast.Inspect(fn.Body, func(n ast.Node) bool {
        ret, ok := n.(*ast.ReturnStmt)
        if !ok { return true }
        for _, expr := range ret.Results {
            lit, ok := expr.(*ast.CompositeLit)
            if ok && (lit.Elts == nil || len(lit.Elts) == 0) {
                return Issue{Rule: "C002", ...}
            }
        }
        return true
    })
}
```

**False-positive scenarios:**

- Legitimate zero CommandID in test code — suppressed by `_test.go` exclusion.
- Struct that genuinely has no ID concept — unlikely if it implements `ID() CommandID`.

**Auto-fix:** Replace manual `Type()/AggregateID()/ID()` methods with `*command.BasicCommand` embedding. Risk: **Medium** — changes struct layout. Requires compile-check.

---

#### C003: Silent Unknown Event in Fold

**Detects:** Fold functions that silently return unchanged state for unrecognized event types.

**Pattern:**

```go
// BAD: corrupt event streams go undetected
func foldLock(state LockState, evt event.Event) (LockState, error) {
    next := state
    switch evt.Type() {
    case eventLockStarted: // ...
    case eventTimeAdded: // ...
    default:
        return state, nil  // BUG: should return error
    }
    return next, nil
}
```

**Detection logic (concrete AST):**

```
1. Identify fold functions:
   - Signature: func(X, event.Event) (X, error) where X is any type
   - Detection: ast.FuncDecl with exactly 2 params, 2 returns
     - Param 1 type resolves to X
     - Param 2 type resolves to event.Event (via typesInfo)
     - Return 1 type resolves to X (same as param 1)
     - Return 2 type is error
2. In the fold function body, find ast.SwitchStmt where the tag is a call
   to evt.Type() (SelectorExpr with Sel.Name == "Type")
3. Find the ast.CaseClause with List == nil (this is the `default:` case)
4. In the default case body, check for a return statement returning nil
   as the second return value → FLAG
```

**False-positive scenarios:**

- Intentional fallthrough for forward-compatibility (event stream from a newer version includes types the old code doesn't know). Suppression: `//cqrs-lint:ignore(C003) forward-compatible fold`
- Fold function that is not a real event fold (coincidental signature match). Mitigated by checking the second param type resolves to `event.Event`.

**Auto-fix:**

```go
// OLD:
default:
    return state, nil

// NEW:
default:
    return state, fmt.Errorf("fold: unknown event type: %s", evt.Type())
```

Risk: **Low** — unambiguous.

---

#### C004: Checkpoint Before Async Complete

**Detects:** Projection handlers that enqueue async work and return nil immediately.

**Pattern:**

```go
// BAD: checkpoint advances before download completes
func (p *AttachmentProjection) Handle(ctx context.Context, evt event.Event) error {
    p.queue <- evt  // enqueue async work
    return nil      // checkpoint advances, work may never complete if process crashes
}
```

**Detection logic:**

```
1. Find types implementing projection.Projection (Has method Handle(ctx, event.Event) error,
   Name() string, EventTypes() []event.Type)
2. In the Handle method body, look for:
   a. Send to channel: ast.SendStmt (ch <- value)
   b. Goroutine launch: ast.GoStmt
3. If either is found AND the method returns nil (ast.ReturnStmt with nil result) → FLAG
```

**False-positive scenarios:**

- Fire-and-forget notification that doesn't affect data integrity (e.g., logging). Suppression: `//cqrs-lint:ignore(C004) notification only`.
- Queue with its own checkpoint mechanism — needs manual review.

**Auto-fix:** None. Requires architectural decision (inline the work, or use projectionhost with a different projection that handles the async lifecycle).

---

#### C005: Raw json.Unmarshal for Event Payload

**Detects:** Direct `json.Unmarshal(evt.Payload(), ...)` instead of `event.DecodePayloadAuto[T](evt)`.

**Pattern:**

```go
// BAD: bypasses codec detection, schema versioning, upcasting
var p MyPayload
json.Unmarshal(evt.Payload(), &p)

// GOOD: auto-detects JSON/CBOR, applies upcasters
p, err := event.DecodePayloadAuto[MyPayload](evt)
```

**Detection logic (concrete AST):**

```
1. Walk ast.CallExpr nodes where Fun is ast.SelectorExpr:
   - X is an ast.Ident with Name == "json"
   - Sel.Name is "Unmarshal" or "NewDecoder"
2. For json.Unmarshal: check first argument is a method call to .Payload()
   — ast.CallExpr with Fun = ast.SelectorExpr{Sel.Name: "Payload"}
3. For json.NewDecoder: check the argument is a call to .Payload()
4. If the argument to .Payload() is an event type (resolved via typesInfo) → FLAG
```

**AST selector:**

```go
ast.Inspect(file, func(n ast.Node) bool {
    call, ok := n.(*ast.CallExpr)
    if !ok { return true }
    sel, ok := call.Fun.(*ast.SelectorExpr)
    if !ok { return true }

    // Check: json.Unmarshal(...)
    ident, ok := sel.X.(*ast.Ident)
    if !ok || ident.Name != "json" { return true }
    if sel.Sel.Name != "Unmarshal" && sel.Sel.Name != "NewDecoder" { return true }

    // Check: first arg is .Payload() call
    if len(call.Args) > 0 {
        if isPayloadCall(call.Args[0]) {
            return Issue{Rule: "C005", ...}
        }
    }
    return true
})
```

**False-positive scenarios:**

- Decoding a non-event payload (e.g., HTTP request body that happens to be in a file importing event types). Mitigated by checking the receiver of `.Payload()` resolves to `event.Event` or `*event.ImmutableEvent`.

**Auto-fix:**

```go
// OLD:
var p MyPayload
json.Unmarshal(evt.Payload(), &p)

// NEW:
p, err := event.DecodePayloadAuto[MyPayload](evt)
if err != nil {
    return fmt.Errorf("decode payload: %w", err)
}
```

Risk: **Medium** — requires knowing the payload type `T` and the event variable name. The linter can infer both from the AST context (the `&p` target gives the type, the `.Payload()` receiver gives the event variable).

---

#### C006: Manual Version Arithmetic

**Detects:** `event.Version(version.Int()+1)` instead of `version.Increment()`.

**Pattern:**

```go
// BAD: manual arithmetic, prone to off-by-one
evt, _ := event.NewEvent(type, id, aggType, event.Version(version.Int()+1), payload)

// GOOD: use the method
evt, _ := event.NewEvent(type, id, aggType, version.Increment(), payload)
```

**Detection logic (concrete AST):**

```
1. Walk ast.CallExpr nodes where Fun resolves to event.Version (type conversion)
   — ast.CallExpr with Fun = ast.SelectorExpr{X.Ident.Name: "event", Sel.Name: "Version"}
2. Check the single argument is a BinaryExpr with Op == token.ADD
3. Check the left side is a call to .Int() and the right side is a literal 1
   — BinaryExpr.X is ast.CallExpr{Fun: SelectorExpr{Sel.Name: "Int"}}
   — BinaryExpr.Y is ast.BasicLit with Value "1"
4. → FLAG
```

**AST selector:**

```go
ast.Inspect(file, func(n ast.Node) bool {
    call, ok := n.(*ast.CallExpr)
    if !ok { return true }
    sel, ok := call.Fun.(*ast.SelectorExpr)
    if !ok { return true }
    ident, ok := sel.X.(*ast.Ident)
    if !ok || ident.Name != "event" || sel.Sel.Name != "Version" { return true }

    if len(call.Args) != 1 { return true }
    binExpr, ok := call.Args[0].(*ast.BinaryExpr)
    if !ok || binExpr.Op != token.ADD { return true }

    // Left: X.Int()
    leftCall, ok := binExpr.X.(*ast.CallExpr)
    if !ok { return true }
    leftSel, ok := leftCall.Fun.(*ast.SelectorExpr)
    if !ok || leftSel.Sel.Name != "Int" { return true }

    // Right: literal 1
    lit, ok := binExpr.Y.(*ast.BasicLit)
    if !ok || lit.Value != "1" { return true }

    return Issue{Rule: "C006", ...}
})
```

**False-positive scenarios:** None. The pattern is unambiguous.

**Auto-fix:**

```go
// OLD: event.Version(version.Int()+1)
// NEW: version.Increment()
// The version variable name comes from leftCall.Fun.X (the receiver of .Int())
```

Risk: **Low** — mechanical text replacement.

---

#### C007: time.Now() in Decider

**Detects:** Calls to `time.Now()` inside decider closures or fold functions.

**Pattern:**

```go
// BAD: non-deterministic, untestable, non-replayable
func decideDiscover(aggID id.AggregateID, kwNumber string) func(property.State, event.Version) ([]event.Event, error) {
    return func(state property.State, version event.Version) ([]event.Event, error) {
        now := time.Now()  // BUG: different result every time
        evt, _ := event.New(eventDiscovered, aggID, aggregateType, version.Increment(), Payload{FoundAt: now})
        return []event.Event{evt}, nil
    }
}
```

**Detection logic:**

```
1. Identify decider decide functions:
   - Functions returning func(X, event.Version) ([]event.Event, error) — the closure factory
   - Or the closures themselves: func(X, event.Version) ([]event.Event, error)
2. Identify fold functions: func(X, event.Event) (X, error)
3. Walk the body of these functions/closures
4. Find ast.CallExpr to time.Now():
   — Fun = ast.SelectorExpr{X.Ident.Name: "time", Sel.Name: "Now"}
5. → FLAG
```

**False-positive scenarios:**

- **Domain where wall-clock IS the logic** — e.g., a timeout decider that checks `time.Now().After(deadline)`. This is semantically correct but makes the decider non-replayable. Suppression: `//cqrs-lint:ignore(C007) wall-clock is domain logic`.
- Time used for ordering/metadata, not for the decision itself. The linter cannot distinguish — hence Medium confidence.

**Auto-fix:** None. Requires injecting a clock interface or passing timestamps via command fields.

---

#### C008: float64 for Monetary Values

**Detects:** `float64` fields in event payload or state structs where the field name suggests money.

**Detection logic:**

```
1. Walk ast.StructType nodes (in event payload types or decider state types)
2. For each ast.Field, check if the type is ast.Ident{Name: "float64"}
3. Check the field name against a monetary pattern regex:
   ^(?i).*(amount|price|cost|value|balance|fee|total|subtotal|tax|discount|payment|charge|salary|wage|revenue|profit|loss|deposit|withdrawal).*
4. → FLAG
```

**False-positive scenarios:**

- `TaxRate float64` (0.0-1.0 ratio, not a monetary amount) — pattern match on "tax" is too broad. Mitigated by excluding field names ending in "Rate", "Percentage", "Ratio", "Factor".
- Non-monetary values named `Value` (e.g., `SensorValue float64`). Low confidence — hence Medium severity.

**Auto-fix:** None. Migration to `int64` cents changes the serialized format.

---

#### C009: Panic in Production Code

**Detects:** `panic()` calls in non-test, non-init Go files.

**Detection logic:**

```
1. Walk ast.CallExpr where Fun is ast.Ident{Name: "panic"}
2. Exclude _test.go files
3. Exclude init() functions (ast.FuncDecl.Name.Name == "init")
4. Exclude sync.OnceValue / sync.OnceFunc closures (acceptable for init-time panics)
5. → FLAG
```

**False-positive scenarios:**

- `mustXxx()` constructor functions where panic IS the documented contract (e.g., `template.Must`). Suppression: `//cqrs-lint:ignore(C009) must-constructor`. Or detect the `Must` prefix on the containing function and suppress automatically.

**Auto-fix:** None. Each case requires manual judgment.

---

#### C010: Swallowed Error in Fold

**Detects:** Fold function cases that ignore decode errors via `_ :=`.

**Pattern:**

```go
// BAD: silently ignores decode errors
case eventCreated:
    p, _ := decode(evt)  // error swallowed
    state.Name = p.Name

// GOOD: propagate errors
case eventCreated:
    p, err := decode(evt)
    if err != nil {
        return state, fmt.Errorf("decode eventCreated: %w", err)
    }
```

**Detection logic:**

```
1. Inside fold functions (same identification as C003)
2. Find ast.AssignStmt where the LHS includes ast.Ident{Name: "_"}
   and the RHS is a function call returning (T, error)
3. Check if the function called is a decode/decode-payload function:
   — Name matches DecodePayload, decodePayload, DecodePayloadAuto
   — Or resolves to event.DecodePayload[T] via typesInfo
4. → FLAG
```

**False-positive scenarios:**

- Intentional best-effort decode with fallback default. Suppression: `//cqrs-lint:ignore(C010) best-effort decode`.

**Auto-fix:** Replace `_, ` with `err, ` and add error check. Risk: **Low**.

---

#### C011: Nondeterministic Decider

**Detects:** Decider decide functions that take service dependencies (non-value parameters types).

**Detection logic:**

```
1. Find decide functions (returning func(X, event.Version) ([]event.Event, error))
2. Check the outer function's parameters (not the closure's)
3. If any parameter type is an interface, func type, or pointer to a non-domain type → FLAG
4. Exclude parameters of type context.Context, id.AggregateID, and value types
```

**False-positive scenarios:**

- Services like `Clock`, `Random` that are intentionally injected for testability. This IS the issue — the decider is impure by design. Low confidence because it's a design trade-off, not a bug.

**Auto-fix:** None. Design-level change required.

---

#### C012: Missing Error Return in withTx (general C001)

**Detects:** Same as C001 but also catches patterns where the commit is conditionally skipped (e.g., read-only transactions that don't need commit but don't rollback either).

**Detection logic:** Same as C001, but also flags functions that call `BeginTx` and return without calling either `Commit()` or `Rollback()`.

---

### 3.2 Category 2: API Misuse - 20 rules

These rules detect incorrect use of go-cqrs-lite APIs.

| ID   | Rule                             | Severity      | Auto-Fix | Confidence | Projects Affected          |
| ---- | -------------------------------- | ------------- | -------- | ---------- | -------------------------- |
| A001 | `manual-command-interface`       | High          | Yes      | High       | storbi, Kernovia           |
| A002 | `event-newevent-manual-marshal`  | High          | Yes      | High       | Zlota44                    |
| A003 | `explicit-codec-in-decode`       | Medium        | Yes      | High       | Zlota44                    |
| A004 | `untyped-dispatch-register`      | Medium        | Yes      | High       | storbi, Standup-Killer     |
| A005 | `custom-projection-runner`       | High          | No       | Medium     | SBTS, DiscordSync          |
| A006 | `adapter-layer-wrapping`         | Medium        | No       | Medium     | PapDashboard, KeyCountdown |
| A007 | `dual-model-oo-plus-functional`  | Critical      | No       | High       | SBTS, KeyCountdown         |
| A008 | `parallel-type-system`           | Critical      | No       | High       | SBTS                       |
| A009 | `missing-stack-preset`           | Low           | No       | Medium     | All v3 projects            |
| A010 | `custom-error-types`             | Medium        | No       | Medium     | SBTS                       |
| A011 | `inconsistent-json-key-casing`   | Low           | No       | Low        | Kernovia                   |
| A012 | `missing-tombstone-handling`     | Medium        | No       | Low        | Various                    |
| A013 | `pointer-vs-value-basic-command` | Low           | No       | High       | KeyCountdown, KeyHolderAI  |
| A014 | `deprecated-api-usage`           | Low           | Yes      | High       | cqrs-htmx                  |
| A015 | `global-mutable-state`           | High          | No       | High       | cqrs-htmx, SEC             |
| A016 | `missing-idempotency-middleware` | Medium        | No       | Low        | Various                    |
| A017 | `missing-snapshot-strategy`      | Low           | No       | Low        | Large aggregate projects   |
| A018 | `no-actual-event-sourcing`       | Informational | No       | High       | KeyHolderAI, Cyberdom      |
| A019 | `vendored-cqrs`                  | Medium        | No       | High       | StopTube                   |
| A020 | `event-store-as-append-log`      | Medium        | No       | Medium     | DiscordSync                |

---

#### A001: Manual Command Interface

**Detects:** Command structs that manually implement `Type()`, `AggregateID()`, `ID()` instead of embedding `*command.BasicCommand`.

**Pattern:**

```go
// BAD: 4 methods of boilerplate per command
type CreateItemCmd struct {
    id       id.AggregateID
    itemType string
}
func (c *CreateItemCmd) Type() command.Type      { return CommandCreateItem }
func (c *CreateItemCmd) AggregateID() id.AggregateID { return c.id }
func (c *CreateItemCmd) ID() command.CommandID   { return c.cmdID }

// GOOD: embedding provides all three
type CreateItemCmd struct {
    *command.BasicCommand
    itemType string
}
```

**Detection logic (concrete AST):**

```
1. Collect all struct types (ast.StructType in ast.TypeSpec)
2. For each struct, check if it has methods Type(), AggregateID(), ID():
   — Scan ast.FuncDecl with Recv != nil where Recv type matches the struct
   — Collect method names
3. If the struct has all three methods AND does NOT embed *command.BasicCommand
   (check ast.StructType.Fields for an embedded field whose type is
   ast.StarExpr{X: ast.SelectorExpr{Sel.Name: "BasicCommand"}})
   → FLAG
```

**False-positive scenarios:**

- Command type from a different framework (not go-cqrs-lite). Mitigated by checking the return types resolve to `command.Type`, `id.AggregateID`, `command.CommandID`.
- Third-party command interface that happens to use the same method names. Unlikely.

**Auto-fix:** Add `*command.BasicCommand` field, remove manual method implementations, update constructor. Risk: **Medium** — changes struct initialization.

---

#### A002: event.NewEvent with Manual Marshal

**Detects:** Using `event.NewEvent` (which requires pre-marshaled `[]byte`) instead of `event.New` (which takes typed payload and auto-marshals).

**Detection logic (concrete AST):**

```
1. Walk ast.CallExpr where:
   Fun = ast.SelectorExpr{X.Ident.Name: "event", Sel.Name: "NewEvent"}
2. Check if the 5th argument (payload position) is:
   a. A call to json.Marshal / cbor.Marshal / codec.Encode
   b. A []byte variable (resolved via typesInfo)
3. If the payload is pre-marshaled bytes → FLAG
4. If the payload is a typed value (struct, map) → DON'T FLAG (valid usage for
   raw byte payloads)
```

**False-positive scenarios:**

- Legitimate raw-byte payloads (e.g., pre-signed binary blobs). Suppression: `//cqrs-lint:ignore(A002) raw byte payload`.

**Auto-fix:** Replace with `event.New(type, id, aggType, version, originalValue)` using the pre-marshal value. Risk: **Medium**.

---

#### A003: Explicit Codec in Decode

**Detects:** `event.DecodePayload[T](evt, codecInstance)` instead of `event.DecodePayloadAuto[T](evt)`.

**Detection logic:**

```
1. Walk ast.CallExpr where:
   Fun = ast.IndexExpr or ast.IndexListExpr (generic instantiation)
   with X = ast.SelectorExpr{Sel.Name: "DecodePayload"}
2. Check if there are 2 arguments (evt, codec)
3. If second argument is a codec instance → FLAG
```

**Auto-fix:** Replace `DecodePayload[T](evt, codec)` with `DecodePayloadAuto[T](evt)`. Risk: **Low**.

---

#### A004: Untyped Dispatch Register

**Detects:** `dispatcher.Register(type, handler)` with manual type assertion.

**Detection logic (concrete AST):**

```
1. Walk ast.CallExpr where Fun = ast.SelectorExpr{Sel.Name: "Register"}
2. Check the second argument (the handler) is an ast.FuncLit (function literal)
3. In the function literal body, look for:
   a. Type assertion: ast.TypeAssertStmt
   b. Or: ast.TypeSwitchStmt with a case matching *ConcreteCmd
4. If a type assertion on the command parameter is found → FLAG
```

**False-positive scenarios:**

- Custom dispatcher that doesn't have a `RegisterTyped` equivalent. Check if `command.RegisterTyped` is available in the project's dependencies.

**Auto-fix:** Convert to `command.RegisterTyped(dispatcher, cmdType, func(ctx, c *ConcreteCmd) error { ... })`. Risk: **Medium**.

---

#### A005: Custom Projection Runner

**Detects:** Manual `bus.SubscribeAll` + switch-based event handling loops instead of `projectionhost.Host`.

**Detection logic:**

```
1. Walk ast.CallExpr where Fun = ast.SelectorExpr{Sel.Name: "SubscribeAll"}
2. Check the callback argument for a switch-on-event-type pattern:
   — ast.FuncLit containing ast.SwitchStmt with tag = call to .Type()
3. Check if projectionhost is imported in any file in the module
4. If SubscribeAll + switch pattern found AND no projectionhost import → FLAG
5. If projectionhost IS imported but this specific projection uses SubscribeAll
   instead of being registered with the host → FLAG (informational)
```

**False-positive scenarios:**

- Legitimate use of SubscribeAll for cross-cutting concerns (logging, audit). Suppression: `//cqrs-lint:ignore(A005) audit handler`.
- Projection that needs custom ordering or batching not supported by projectionhost.

**Auto-fix:** None. Architectural migration required.

---

#### A006: Adapter Layer Wrapping

**Detects:** Types that wrap go-cqrs-lite interfaces with conversion methods.

**Detection logic:**

```
1. Find types with methods matching pattern:
   - WrapEvent(), UnwrapEvent(), ToEvent(), FromEvent()
   - WrapStore(), ToStore(), etc.
2. Check if the type has a field that holds a go-cqrs-lite type
   (resolved via typesInfo)
3. If the type wraps a go-cqrs-lite interface and adds conversion → FLAG
```

**False-positive scenarios:**

- Legitimate adapter pattern (e.g., wrapping a remote store with retry). Suppression via comment.

---

#### A007: Dual Model (OO + Functional)

**Detects:** Projects that maintain both an OO aggregate pattern AND a functional decider pattern.

**Detection logic:**

```
1. Find OO aggregate markers:
   — Structs with fields named: uncommittedEvents, pendingEvents, events
   — Methods named: addEvent, markEventsAsCommitted, getUncommittedEvents, apply
2. Find functional decider markers:
   — Variables of type decider.Decider[T]
   — Functions with fold signature: func(X, event.Event) (X, error)
3. If BOTH found in the same project → FLAG (one issue per project, not per file)
```

**False-positive scenarios:**

- Migration in progress (the OO model is being phased out). The linter should report this as Critical with a migration suggestion, not suppress it.

---

#### A008: Parallel Type System

**Detects:** Custom types that duplicate go-cqrs-lite domain types.

**Detection logic:**

```
1. Find type declarations (ast.TypeSpec) where the name is one of:
   AggregateID, Version, CommandType, EventType, EventID, CommandID
2. Check if the type definition is NOT a type alias:
   — ast.TypeSpec with Assign != 0 means it's an alias (type X = Y)
   — Non-alias definitions (type X string, type X int64) → FLAG
3. Check if the project also imports id/, event/, command/
4. If parallel types exist alongside go-cqrs-lite imports → FLAG
```

**False-positive scenarios:** None. The pattern is unambiguous.

---

#### A009: Missing Stack Preset

**Detects:** Projects that manually wire event store + bus + repository.

**Detection logic:**

```
1. Scan go.mod for imports of stack/sqlite, stack/postgres, stack/pebble
2. Scan Go source for calls to:
   - storage.NewSQLiteEventStore
   - watermill.NewEventBus
   - decider.NewRepository
3. If all 3+ calls found AND no stack/ import → FLAG
```

**False-positive scenarios:**

- Project on v3 where stack presets don't exist or are incomplete. The linter should check the go-cqrs-lite version and only suggest stacks if available.

---

#### A010: Custom Error Types

**Detects:** Custom error struct types instead of `go-error-family`.

**Detection logic:**

```
1. Find structs implementing Error() string:
   — ast.FuncDecl with Name "Error", 0 params, 1 return (string)
2. Check the struct fields for error-pattern fields:
   Field, Value, Message, Operation, Entity, Code
3. If the struct is NOT from go-error-family package → FLAG
4. Exclude stdlib error types (errors.New, fmt.Errorf)
```

**False-positive scenarios:**

- Domain-specific errors that genuinely need custom structure. Suppression via comment.

---

#### A015: Global Mutable State

**Detects:** Package-level mutable variables modified by exported setter functions.

**Detection logic (concrete AST):**

```
1. Find ast.GenDecl with Tok == token.VAR at package level
   (top-level decl, not inside a function)
2. For each var, check if any exported function assigns to it:
   — Walk ast.FuncDecl with Name.IsExported()
   — In the body, look for ast.AssignStmt where the LHS is the var name
3. If an exported setter exists → FLAG
```

**False-positive scenarios:**

- `sync.Once` guarded initialization (acceptable). Check for `sync.Once` in the same file.
- Test-only mutable state in `_test.go` files (excluded).

---

### 3.3 Category 3: Boilerplate Detection - 15 rules

These rules detect repetitive code that could be replaced by library helpers or code generation.

| ID   | Rule                                | Severity      | Auto-Fix                     | Projects Affected     |
| ---- | ----------------------------------- | ------------- | ---------------------------- | --------------------- |
| B001 | `single-event-helper`               | Low           | Yes (suggest library fn)     | ALL (every project)   |
| B002 | `repository-wiring-boilerplate`     | Medium        | No                           | ALL v3 projects       |
| B003 | `read-model-projection-boilerplate` | Medium        | Yes (suggest projectionhost) | ALL with projections  |
| B004 | `command-constructor-boilerplate`   | Low           | Yes (suggest cqrs-gen)       | ALL with commands     |
| B005 | `fold-switch-boilerplate`           | Informational | No                           | ALL with ES           |
| B006 | `duplicate-fk-stub-sql`             | Low           | Yes                          | DiscordSync           |
| B007 | `repeated-handler-registration`     | Low           | Yes (suggest table-driven)   | ALL with commands     |
| B008 | `manual-retry-implementation`       | Medium        | Yes (suggest retry.Do)       | DiscordSync (3x)      |
| B009 | `emit-function-boilerplate`         | Low           | Yes (suggest generator)      | DiscordSync           |
| B010 | `catalog-event-list-boilerplate`    | Low           | Yes (suggest cqrs-gen)       | DiscordSync, Kernovia |
| B011 | `must-marshal-helper`               | Low           | Yes                          | 6+ projects           |
| B012 | `make-event-helper`                 | Low           | Yes                          | ALL                   |
| B013 | `missing-correlation-enricher`      | Medium        | No                           | Most projects         |
| B014 | `missing-otel-middleware`           | Low           | No                           | Most projects         |
| B015 | `missing-test-utilities`            | Low           | No                           | Various               |

#### B001: Single-Event Creation Helper

**Detects:** The universal pattern of creating a single event from a typed payload. Every project reimplements this under different names.

**Detection logic:**

```
1. Find functions with signature matching:
   (event.Type, id.AggregateID, string, event.Version, any, ...event.Option) -> ([]event.Event, error)
2. Check if the body calls event.New or event.NewEvent and wraps in []event.Event{}
3. Count call sites of this function
4. If >3 call sites → FLAG as boilerplate (suggest library function)
```

**Detection for variants** (named differently per project):

```
The function name pattern: ^(single|make|create|must).*[Ee]vent$
Or functions calling event.New/event.NewEvent where the return wraps to []event.Event{}
```

**Auto-fix:** Replace all call sites with `event.Single(...)` if that library function is added (see Appendix B). Risk: **Low**.

---

#### B002: Repository Wiring Boilerplate

**Detects:** The repeated 5-step repository setup pattern.

**Detection logic:**

```
1. Find functions that call 3+ of these in sequence (within the same function body):
   - storage.NewSQLiteEventStore / storage.NewSQLEventStore
   - watermill.NewEventBus / event.NewBus
   - bus.Use( (any middleware chain)
   - decider.NewRepository
   - snapshot.New*Store
2. Check if stack/sqlite or stack/postgres is imported in go.mod
3. If manual wiring + no stack import → FLAG
```

**Auto-fix:** None (migration to stack presets requires restructuring). Provide migration guide instead.

---

#### B003: Read Model Projection Boilerplate

**Detects:** Manual `bus.SubscribeAll` + switch-on-event-type projection implementations.

**Detection logic:**

```
1. Find bus.SubscribeAll callbacks (same as A005)
2. Count the number of cases in the switch evt.Type() statement
3. If >5 cases → FLAG as boilerplate (suggest typed Projection + projectionhost)
```

**Auto-fix:** Generate a typed `projection.Projection` implementation from the switch cases. Risk: **High** — generates new code that must be registered.

---

### 3.4 Category 4: Consistency - 5 rules

| ID   | Rule                             | Severity      | Auto-Fix | Description                                                                              |
| ---- | -------------------------------- | ------------- | -------- | ---------------------------------------------------------------------------------------- |
| D001 | `inconsistent-command-embedding` | Low           | No       | Some commands embed `*BasicCommand`, others use value `BasicCommand`, others are manual. |
| D002 | `version-fragmentation`          | Informational | No       | Project pins go-cqrs-lite at an older version when newer is available.                   |
| D003 | `inconsistent-logging-library`   | Low           | No       | Project mixes `log/slog`, `charm.land/log`, and `go.uber.org/zap`.                       |
| D004 | `inconsistent-json-key-casing`   | Low           | No       | Some event payloads use camelCase JSON keys, others use snake_case.                      |
| D005 | `stale-documentation-version`    | Medium        | No       | AGENTS.md or README references a different go-cqrs-lite version than go.mod.             |

#### D001: Inconsistent Command Embedding

**Detection logic:**

```
1. Collect all command structs in the project (same as A001)
2. Classify embedding: *command.BasicCommand (pointer), command.BasicCommand (value), manual
3. If the project has >1 style → FLAG (one issue per project)
```

#### D002: Version Fragmentation

**Detection logic:**

```
1. Parse go.mod for go-cqrs-lite version
2. Compare against latest tagged release (from git tags or module proxy)
3. If >2 minor versions behind → FLAG (informational)
```

#### D004: Inconsistent JSON Key Casing

**Detection logic:**

```
1. Collect all event payload struct fields with json: tags
2. Categorize: camelCase, snake_case, PascalCase
3. If >1 casing style in the same module → FLAG
```

---

### 3.5 Category 5: Architecture - 7 rules

| ID   | Rule                        | Severity | Auto-Fix | Description                                                             |
| ---- | --------------------------- | -------- | -------- | ----------------------------------------------------------------------- |
| E001 | `layer-violation`           | High     | No       | Tier 0 module (id, codec) imports Tier 3+ module (decider, middleware). |
| E002 | `circular-dependency`       | High     | No       | Two go-cqrs-lite modules import each other (should never happen).       |
| E003 | `missing-module-boundary`   | Medium   | No       | All code in one package without internal/ boundaries.                   |
| E004 | `event-type-not-in-catalog` | Medium   | Yes      | Event types emitted by deciders but missing from catalog registry.      |
| E005 | `command-without-handler`   | High     | No       | Command types defined but never registered with a dispatcher.           |
| E006 | `event-without-projection`  | Medium   | No       | Events emitted by deciders but no projection handles them.              |
| E007 | `query-without-handler`     | High     | No       | Query types defined but never registered with a dispatcher.             |

#### E001: Layer Violation

**Detection logic:**

```
1. Parse go.mod for the project's module path
2. Check which go-cqrs-lite modules are imported
3. Map modules to tiers (from ADR-0046 four-tier model):
   Tier 0: id/, dispatcher/, codec/, kv/, dedup/
   Tier 1: event/, command/, query/, scheduling/, metadata/
   Tier 2: schema/, snapshot/, projection/, idempotency/, deriver/
   Tier 3: decider/, graph/, scenario/, projectionhost/, listing/
   Tier 4: storage/, middleware/, signing/, encryption/, otel/, watermill/, transport/
   Tier 5: stack/
4. If a Tier N module imports a Tier N+2+ module → FLAG
   (Tier N importing N+1 is allowed — that's the normal dependency direction)
```

**Note:** This rule applies to go-cqrs-lite internal modules. For consumers, the rule checks whether a consumer's go.mod has unnecessary transitive deps due to importing high-tier modules when a low-tier module would suffice.

#### E004: Event Type Not in Catalog

**Detection logic:**

```
1. Collect all event types emitted (from event.New/event.NewEvent calls)
2. Collect all catalog registrations (catalog.Event[T](type, ...) calls)
3. Diff: emitted types not in catalog → FLAG
```

**Auto-fix:** Add missing `catalog.Event[PayloadType](eventType, ...)` call. Risk: **Low**.

#### E005: Command Without Handler

**Detection logic:**

```
1. Collect all command types (constants and structs embedding BasicCommand)
2. Collect all command.RegisterTyped / dispatcher.Register calls
3. Diff: commands defined but never registered → FLAG
```

---

### 3.6 Category 6: Security - 3 rules

| ID   | Rule                                        | Severity | Auto-Fix | Description                                                                             |
| ---- | ------------------------------------------- | -------- | -------- | --------------------------------------------------------------------------------------- |
| S001 | `hardcoded-secrets-in-events`               | Critical | No       | Event payload or metadata contains what looks like an API key, token, or password.      |
| S002 | `missing-encryption-for-sensitive-payloads` | High     | No       | Event payloads containing PII fields (email, SSN, phone) without encryption middleware. |
| S003 | `missing-event-signing`                     | Medium   | No       | Event store in production without signing middleware (tamper detection).                |

#### S001: Hardcoded Secrets in Events

**Detection logic:**

```
1. Walk event payload struct fields
2. Check field names against secret patterns:
   ^(?i).*(api[_-]?key|secret|token|password|passwd|credential|private[_-]?key|access[_-]?key).*
3. If field type is string or []byte and matches → FLAG
4. Also check string literals in event.New/event.NewEvent calls for
   high-entropy patterns (base64, hex, 32+ chars)
```

**False-positive scenarios:**

- Token hash (not the raw token). Suppression: `//cqrs-lint:ignore(S001) hashed value`.
- Test fixtures with fake tokens. Excluded by `_test.go` filter.

---

## 4. Detection Logic

### 4.1 Three Analysis Layers

The linter operates at three levels, each building on the previous:

#### Layer 1: File-level AST (from `cqrs-gen`)

Already proven in `cmd/cqrs-gen/main.go`. Key patterns:

**Type declaration scanning** (from `cqrs-gen:165-175`):

```go
for _, decl := range f.Decls {
    genDecl, ok := decl.(*ast.GenDecl)
    if !ok || genDecl.Tok != token.TYPE { continue }
    for _, spec := range genDecl.Specs {
        ts, ok := spec.(*ast.TypeSpec)
        if !ok { continue }
        // ts.Name.Name = type name
        // ts.Type = *ast.StructType, *ast.InterfaceType, etc.
    }
}
```

**Marker comment extraction** (from `cqrs-gen:205-217`):

```go
func extractMarker(doc *ast.CommentGroup, prefix string) string {
    if doc == nil { return "" }
    for _, c := range doc.List {
        if after, ok := strings.CutPrefix(c.Text, prefix); ok {
            return after
        }
    }
    return ""
}
```

**Struct tag extraction** (from `cqrs-gen:227-251`):

```go
func extractStructTag(st *ast.StructType, key string) string {
    for _, field := range st.Fields.List {
        if field.Tag == nil { continue }
        tagValue := strings.Trim(field.Tag.Value, "`")
        cqrsTag := reflect.StructTag(tagValue).Get("cqrs")
        // parse "command:CreateUser" format
    }
}
```

**Enhancement needed for the linter:** Also scan for:

- `event.NewEvent` vs `event.New` calls (ast.CallExpr pattern matching)
- `json.Unmarshal(evt.Payload()` patterns (call chain analysis)
- `time.Now()` inside specific function types (context-aware walk)
- `panic()` calls (simple ast.CallExpr match)
- Manual interface implementations (method set collection per type)
- `return nil` in functions that create transactions (return-statement analysis)

#### Layer 2: Type-level cross-reference

Using `go/types` (via `packages.Load`) to resolve types across files:

```go
// Resolve whether a type embeds *command.BasicCommand
func embedsBasicCommand(named *types.Named) bool {
    // Walk the struct's embedded fields
    s, ok := named.Underlying().(*types.Struct)
    if !ok { return false }
    for i := 0; i < s.NumFields(); i++ {
        f := s.Field(i)
        if f.Embedded() {
            // Check if the type path matches command.BasicCommand
            if named, ok := f.Type().(*types.Named); ok {
                obj := named.Obj()
                if obj.Pkg().Path() == "github.com/larsartmann/go-cqrs-lite/command" &&
                   obj.Name() == "BasicCommand" {
                    return true
                }
            }
            // Also check pointer-to-named for *BasicCommand
            if ptr, ok := f.Type().(*types.Pointer); ok {
                if named, ok := ptr.Elem().(*types.Named); ok {
                    obj := named.Obj()
                    if obj.Pkg().Path() == "..." && obj.Name() == "BasicCommand" {
                        return true
                    }
                }
            }
        }
    }
    return false
}
```

**Key type checks the linter performs:**

| Check                                  | How                                                                         |
| -------------------------------------- | --------------------------------------------------------------------------- |
| Is this a command struct?              | Embeds `*command.BasicCommand` or implements `Type() command.Type`          |
| Is this a fold function?               | Signature `func(S, event.Event) (S, error)` — checked via `types.Signature` |
| Is this a decide function?             | Returns `func(S, event.Version) ([]event.Event, error)`                     |
| Is this a projection?                  | Implements `projection.Projection` (method set check)                       |
| Is this an event payload?              | Used as 5th arg to `event.New` or `event.NewEvent`                          |
| Does this type come from go-cqrs-lite? | `types.Named.Obj().Pkg().Path()` contains `go-cqrs-lite`                    |

#### Layer 3: Project-level structural

Using filesystem analysis (from `go-structure-linter`):

- Module dependencies (parse `go.mod` files)
- Directory structure (aggregate/command/event/projection organization)
- Presence of stack presets vs manual wiring
- Version of go-cqrs-lite being used (from go.mod require/replace)
- Whether `projectionhost` is imported (for A005 rule)
- Whether `stack/` is imported (for A009 rule)

### 4.2 CQRS Type Registry

The linter builds a registry of all CQRS types in one pass, shared across all rules:

```go
type CQRSRegistry struct {
    Commands    []CommandInfo
    Events      []EventInfo
    Queries     []QueryInfo
    Deciders    []DeciderInfo
    Projections []ProjectionInfo
    Aggregates  []AggregateInfo
    Folds       []FoldInfo
}

type CommandInfo struct {
    Name           string
    Type           string         // command type constant value
    StructName     string
    File           string
    EmbedsBasicCmd bool           // *command.BasicCommand?
    EmbeddingStyle string         // "pointer", "value", "manual"
    HasHandler     bool           // registered with RegisterTyped?
    HasIdempotency bool           // has IdempotencyKey()?
}

type EventInfo struct {
    TypeConstant string          // event type string value
    PayloadType  string          // struct name used as payload
    File         string
    InCatalog    bool            // registered with catalog.Event?
    InProjection bool            // handled by at least one projection?
}

type FoldInfo struct {
    FuncName      string
    File          string
    StateType     string         // the S in func(S, event.Event) (S, error)
    HasDefaultErr bool           // does the switch default case return error?
    EventTypes    []string       // event types handled in switch cases
}
```

**Building the registry:**

```go
func BuildRegistry(pkgs []*packages.Package) *CQRSRegistry {
    reg := &CQRSRegistry{}
    for _, pkg := range pkgs {
        for _, file := range pkg.Syntax {
            // Scan type declarations (from cqrs-gen pattern)
            for _, decl := range file.Decls {
                switch d := decl.(type) {
                case *ast.GenDecl:
                    reg.scanGenDecl(d, pkg)
                case *ast.FuncDecl:
                    reg.scanFuncDecl(d, pkg)
                }
            }
            // Scan call expressions for registrations
            ast.Inspect(file, func(n ast.Node) bool {
                reg.scanCallExpr(n, pkg)
                return true
            })
        }
    }
    reg.crossReference()  // link commands to handlers, events to projections
    return reg
}
```

### 4.3 Rule Implementation Pattern

Each rule receives the pre-built registry + raw AST access:

```go
type Rule interface {
    Meta() RuleMeta
    Check(ctx *AnalysisContext) []Issue
}

type AnalysisContext struct {
    Registry  *CQRSRegistry
    Packages  []*packages.Package
    Project   *ProjectInfo    // go.mod data, directory structure
    Fset      *token.FileSet
}

type Issue struct {
    Rule       string
    File       string
    Line       int
    Column     int
    Message    string
    Suggestion string
    Confidence Confidence       // High, Medium, Low
    Fix        *SuggestedFix    // nil if not auto-fixable
}
```

### 4.4 Generated Code Detection

The linter skips generated files to avoid false positives from `cqrs-gen` output:

**Detection strategy:**

```go
func isGeneratedFile(path string, file *ast.File) bool {
    // Check 1: Standard "Code generated" header (Go convention)
    for _, cg := range file.Comments {
        for _, c := range cg.List {
            if strings.Contains(c.Text, "Code generated") &&
               strings.Contains(c.Text, "DO NOT EDIT") {
                return true
            }
        }
    }
    // Check 2: cqrs-gen output marker
    for _, cg := range file.Comments {
        if strings.Contains(cg.Text(), "// Code generated by cqrs-gen") {
            return true
        }
    }
    // Check 3: File name convention
    base := filepath.Base(path)
    if strings.HasSuffix(base, "_gen.go") || strings.HasSuffix(base, ".gen.go") {
        return true
    }
    return false
}
```

---

## 5. False-Positive Mitigation

### 5.1 Suppression Comments

Standard suppression syntax:

```go
//cqrs-lint:ignore(C007) wall-clock is domain logic
now := time.Now()
```

**Parsing:**

```go
func parseSuppression(file *ast.File) map[int][]string {
    // Returns map of line number -> []rule IDs to suppress
    suppressions := make(map[int][]string)
    for _, cg := range file.Comments {
        for _, c := range cg.List {
            if after, ok := strings.CutPrefix(c.Text, "//cqrs-lint:ignore("); ok {
                // Parse: C007) reason
                parts := strings.SplitN(after, ")", 2)
                ruleID := strings.TrimSpace(parts[0])
                line := getLineFromPos(c.Slash, fset)
                suppressions[line] = append(suppressions[line], ruleID)
            }
        }
    }
    return suppressions
}
```

**Scope:** Suppression applies to the next statement after the comment (same as `//nolint` in golangci-lint). For function-level suppression, place the comment on the function declaration:

```go
//cqrs-lint:ignore(C009) must-constructor panics by design
func MustNewRepository(...) *Repository {
    if err != nil { panic(err) }
}
```

### 5.2 File-Level and Package-Level Suppression

```go
//cqrs-lint:ignore-file(A018) command-bus pattern only, no event sourcing intended
package main
```

```yaml
# .cqrs-lint.yml — package-level suppression
ignore:
  rules:
    A018:
      - "internal/dispatch/**" # command bus only
    C007:
      - "**/*_time_test.go" # time-dependent tests
```

### 5.3 Confidence Scoring

Every issue carries a Confidence level. Rules with Low confidence (C007, C011, A011, A012, A016, A017) have inherent ambiguity. The linter should:

1. Default to reporting all confidences (user filters via `--min-confidence`)
2. In CI mode (`--ci`), only report Medium+ confidence
3. In the console output, visually distinguish confidence levels

### 5.4 Per-Rule Severity Override

Users can override severity per rule in `.cqrs-lint.yml`:

```yaml
severity:
  C007: informational # downgrade from Medium
  A018: informational # "no actual ES" -> just info, not actionable
```

---

## 6. Auto-Fix Capabilities

### 6.1 Auto-Fixable Rules (18 rules)

| Rule      | Auto-Fix Description                                          | Risk Level | Confidence |
| --------- | ------------------------------------------------------------- | ---------- | ---------- |
| C001      | `return nil` to `return tx.Commit()` in tx wrappers           | Low        | High       |
| C002      | Add `*command.BasicCommand` embedding, remove manual methods  | Medium     | High       |
| C003      | Add error return in fold default case                         | Low        | High       |
| C005      | `json.Unmarshal` to `DecodePayloadAuto[T]`                    | Medium     | High       |
| C006      | `Version(x.Int()+1)` to `x.Increment()`                       | Low        | High       |
| C010      | Propagate swallowed decode errors                             | Low        | High       |
| A001      | Add `*command.BasicCommand` embedding                         | Medium     | High       |
| A002      | `event.NewEvent` to `event.New` with typed payload            | Medium     | High       |
| A003      | `DecodePayload[T](evt, codec)` to `DecodePayloadAuto[T](evt)` | Low        | High       |
| A004      | `Register` to `RegisterTyped`                                 | Medium     | High       |
| B001      | Replace `singleEvent` helpers with library function           | Low        | High       |
| B003      | Wrap manual projections in `projectionhost`                   | High       | Medium     |
| B006      | Centralize FK stub SQL                                        | Medium     | High       |
| B008      | Replace custom retry with `retry.Do`                          | Medium     | High       |
| B011/B012 | Replace `mustMarshal`/`makeEvent` with library helper         | Low        | High       |
| E004      | Add missing catalog entries                                   | Low        | High       |

### 6.2 Auto-Fix Safety Model

Following the `go-structure-linter` `FixableRule` pattern:

```go
type FixableRule interface {
    DryRunFix(ctx *AnalysisContext, issue Issue) (diff string, err error)
    ApplyFix(ctx *AnalysisContext, issue Issue) (FixOutcome, error)
    ValidateFix(ctx *AnalysisContext, issue Issue) error
}
```

**Safety levels:**

| Level        | Description                | Examples                                   | Pre-checks                |
| ------------ | -------------------------- | ------------------------------------------ | ------------------------- |
| **Safe**     | Only adds code, no removal | C003 (add error), E004 (add catalog entry) | None                      |
| **Moderate** | Modifies code patterns     | C005, C006, A002, A003                     | Compile-check             |
| **Risky**    | Restructures types         | C002, A001, A004                           | Compile-check + `--force` |

### 6.3 Fix Transaction

All fixes are transactional — if any fix breaks compilation, ALL fixes in the batch are rolled back:

```go
type FixTransaction struct {
    backups map[string][]byte  // original file content keyed by path
    applied []FixOperation
}

func (t *FixTransaction) Apply(ctx *AnalysisContext, issue Issue) error {
    path := issue.File
    if _, ok := t.backups[path]; !ok {
        content, err := os.ReadFile(path)
        if err != nil { return err }
        t.backups[path] = content
    }
    // Apply the fix (modify file in place)
    if err := issue.Fix.Apply(ctx, issue); err != nil {
        return err
    }
    t.applied = append(t.applied, FixOperation{Issue: issue})
    return nil
}

func (t *FixTransaction) Commit() error {
    // Verify all changes compile
    if err := compileCheck(); err != nil {
        return t.Rollback()  // restore all backed-up files
    }
    return nil
}

func (t *FixTransaction) Rollback() error {
    for path, content := range t.backups {
        if err := os.WriteFile(path, content, 0644); err != nil {
            return err
        }
    }
    return nil
}
```

### 6.4 Concrete Auto-Fix Example: C006

```go
func (r *ManualVersionArithmeticRule) ApplyFix(
    ctx *AnalysisContext, issue Issue,
) (FixOutcome, error) {
    // Read the file
    content, _ := os.ReadFile(issue.File)

    // The issue carries the position of the event.Version( call
    // and the receiver variable name from the AST analysis
    oldText := fmt.Sprintf("event.Version(%s.Int()+1)", issue.Metadata["versionVar"])
    newText := fmt.Sprintf("%s.Increment()", issue.Metadata["versionVar"])

    newContent := strings.Replace(string(content), oldText, newText, 1)

    if err := os.WriteFile(issue.File, []byte(newContent), 0644); err != nil {
        return FixOutcome{}, err
    }
    return FixOutcome{Fixed: true, ChangedFiles: []string{issue.File}}, nil
}
```

---

## 7. Recommendation Engine

Beyond finding violations, the linter provides actionable recommendations.

### 7.1 Pattern Recommendations

| Trigger                                            | Recommendation                                            | Priority |
| -------------------------------------------------- | --------------------------------------------------------- | -------- |
| Project has >5 commands                            | Consider using `cqrs-gen` for handler registration        | Medium   |
| Project has >3 projections                         | Migrate to `projectionhost.Host`                          | High     |
| Project manually wires event store + bus           | Migrate to `stack/sqlite` or `stack/postgres`             | Medium   |
| Project uses `fmt.Errorf` for domain errors        | Adopt `go-error-family` for classification                | High     |
| Project on v3                                      | Plan migration to v4                                      | Medium   |
| Aggregate has >100 events without snapshots        | Add `snapshot.EveryNEvents(50)`                           | Medium   |
| Projection handles events synchronously            | Consider `projectionhost.WithBatchSize(100)`              | Low      |
| Decider doesn't use correlation enricher           | Add `decider.WithEnricher(correlation.ContextEnricher())` | Low      |
| No `scenario` BDD tests found                      | Consider `scenario.Given/When/Then` for decider tests     | Low      |
| No idempotency middleware on command dispatcher    | Add `middleware.CommandIdempotency(store, ttl, nil)`      | Medium   |
| No OTel middleware                                 | Add `middleware.NewOTelBundle(tracer, meter)`             | Low      |
| Event payload with sensitive fields, no encryption | Add `encryption.EncryptMiddleware`                        | High     |

### 7.2 Health Score

```
Score = 100
- 10 points per Critical violation
- 5 points per High violation
- 2 points per Medium violation
- 1 point per Low violation
- 5 points per category of universal boilerplate (capped at 15)

Minimum score: 0
```

| Score Range | Status          | Action                      |
| ----------- | --------------- | --------------------------- |
| 90-100      | Excellent       | No action needed            |
| 70-89       | Good            | Address Medium+ issues      |
| 50-69       | Needs Attention | Address High+ issues first  |
| 0-49        | Critical        | Systemic refactoring needed |

### 7.3 Migration Path Suggestions

For projects with systemic issues, the linter generates a step-by-step plan:

```json
{
  "migration_plan": {
    "project": "standard-bug-tracking-schema",
    "health_score": 25,
    "steps": [
      {
        "priority": 1,
        "rule": "A008",
        "action": "Replace types.AggregateID with id.AggregateID across all files",
        "estimated_files": 15,
        "risk": "Medium"
      },
      {
        "priority": 2,
        "rule": "A007",
        "action": "Delete OO aggregate model (types.Issue with uncommittedEvents), keep functional decider",
        "estimated_files": 8,
        "risk": "High"
      },
      {
        "priority": 3,
        "rule": "A005",
        "action": "Replace custom projectionRunner with projectionhost.Host",
        "estimated_files": 3,
        "risk": "Medium"
      }
    ]
  }
}
```

---

## 8. Test Fixture Strategy

### 8.1 Fixture Format

Each rule has a test fixture directory:

```
testdata/
  C001_missing-tx-commit/
    bad.go              # Input with the violation
    good.go             # Input with the fix applied
    expected.json       # Expected issues (rule, line, message)
    expected_fix.go     # Expected file content after auto-fix
  C003_silent-unknown-event-fold/
    bad.go
    good.go
    expected.json
  ...
```

### 8.2 Fixture Example: C006

**`testdata/C006_manual-version-arithmetic/bad.go`:**

```go
package testdata

import (
    "github.com/larsartmann/go-cqrs-lite/event"
)

func createEvent(version event.Version) {
    // BAD: manual arithmetic
    _ = event.Version(version.Int() + 1)
}
```

**`testdata/C006_manual-version-arithmetic/good.go`:**

```go
package testdata

import (
    "github.com/larsartmann/go-cqrs-lite/event"
)

func createEvent(version event.Version) {
    _ = version.Increment()
}
```

**`testdata/C006_manual-version-arithmetic/expected.json`:**

```json
[
  {
    "rule": "C006",
    "line": 8,
    "column": 9,
    "message": "Manual version arithmetic: use version.Increment() instead of event.Version(version.Int()+1)",
    "confidence": "High",
    "fixable": true
  }
]
```

### 8.3 Test Runner

```go
func TestRules(t *testing.T) {
    fixtures, _ := filepath.Glob("testdata/*")
    for _, fixture := range fixtures {
        ruleID := strings.Split(filepath.Base(fixture), "_")[0]
        t.Run(ruleID, func(t *testing.T) {
            // Run linter on bad.go
            issues := runLint(filepath.Join(fixture, "bad.go"))
            // Compare against expected.json
            expected := loadExpected(filepath.Join(fixture, "expected.json"))
            assertIssuesMatch(t, issues, expected)

            // If auto-fixable, verify the fix
            if len(issues) > 0 && issues[0].Fix != nil {
                applyFix(t, filepath.Join(fixture, "bad.go"), issues[0])
                fixedContent, _ := os.ReadFile(filepath.Join(fixture, "bad.go"))
                expectedFix, _ := os.ReadFile(filepath.Join(fixture, "expected_fix.go"))
                require.Equal(t, string(expectedFix), string(fixedContent))
            }
        })
    }
}
```

### 8.4 False-Positive Test Fixtures

Each rule also has a `false_positive/` subdirectory to verify it doesn't flag legitimate patterns:

```
testdata/C007_time-now-in-decider/
  bad.go                           # time.Now() in decider → flagged
  good.go                          # time injected via command → not flagged
  false_positive/
    wall_clock_logic.go            # time.Now() with suppression → not flagged
    non_decider_context.go         # time.Now() outside decider → not flagged
```

---

## 9. Performance and Caching

### 9.1 Package Loading Strategy

`packages.Load` is the expensive step. For a 900-file project like Kernovia:

```
Naive:        packages.Load(cfg, "./...")       ~8-12 seconds
Cached:       packages.Load with build cache     ~2-4 seconds (warm)
Incremental:  Only changed packages              ~0.5-1 second
```

**Strategy:**

1. Use `packages.NeedSyntax | packages.NeedTypesInfo` (minimum required for type resolution)
2. Cache the `CQRSRegistry` to disk (`.cqrs-lint-cache/`) keyed by Go file hash
3. On subsequent runs, only re-parse changed files
4. For CI (fresh checkout), no cache — full load is acceptable

### 9.2 Parallel Rule Execution

Using `go-finding/pipeline` (from `go-structure-linter`):

```go
// Each rule becomes a pipeline.Detector
detectors := rulesToDetectors(enabledRules, registry, fset)
cfg := pipeline.Config{
    Workers:    runtime.NumCPU(),
    StopOnPanic: false,  // collect all findings even if one rule panics
}
result := pipeline.New(cfg, projectPath, detectors...).Run(ctx)
```

**Panic recovery:** Each rule runs in a goroutine with `recover()`. Panics are converted to synthetic high-severity findings: `"rule C007 panicked: <stack trace>"`.

### 9.3 Fast Mode

For pre-commit hooks, a `--fast` flag runs only Critical and High correctness rules:

```
Full mode:     All 47 rules               ~8-12 seconds (cold), ~4 seconds (warm)
Fast mode:     C001, C002, C003, C005     ~2 seconds (cold), ~1 second (warm)
Health score:  Count violations only       ~2 seconds (cold), ~1 second (warm)
```

### 9.4 Memory Budget

For large projects, the AST + type info can consume significant memory. Strategy:

1. Parse files lazily (only when a rule needs them)
2. Drop type info for packages that have no CQRS types
3. Cap concurrency at `runtime.NumCPU()` to avoid OOM on large repos

---

## 10. Implementation Blueprint

### 10.1 Module Structure

```
cmd/cqrs-lint/
    main.go                    # CLI entry point (cobra + charm.land/fang)
    flags.go                   # --fix, --dry-run, --only, --severity, --format
pkg/
    analyzer/
        analyzer.go            # Orchestrator: load packages -> build registry -> run rules
        context.go             # AnalysisContext (shared state for all rules)
        registry.go            # CQRSRegistry builder
        types.go               # CommandInfo, EventInfo, etc.
    rules/
        correctness/           # C001-C012
        api/                   # A001-A020
        boilerplate/           # B001-B015
        consistency/           # D001-D005
        architecture/          # E001-E007
        security/              # S001-S003
        interface.go           # Rule interface (from go-structure-linter)
        fixable.go             # FixableRule interface
        registry.go            # RuleRegistry
    pipeline/
        adapter.go             # Rule -> go-finding/pipeline.Detector adapter
    fix/
        transaction.go         # FixTransaction (backup, apply, verify, rollback)
        diff.go                # Diff generation for dry-run
    output/
        console.go             # Human-readable console output
        json.go                # JSON output
        sarif.go               # SARIF for CI integration
        github.go              # GitHub Actions annotations
internal/
    ast/                       # AST helpers (CQRS type detection, pattern matching)
    packages/                  # go/packages loader configuration
    suppression/               # Suppression comment parsing
    generated/                 # Generated file detection
testdata/                      # Test fixtures per rule
```

### 10.2 Dependencies

```
golang.org/x/tools/go/packages     # Type-aware package loading
golang.org/x/tools/go/ast/         # AST inspection helpers (astutil.Apply)
github.com/larsartmann/go-finding  # Unified finding model + pipeline
github.com/larsartmann/go-error-family  # Classified errors
github.com/spf13/cobra             # CLI
github.com/charmbracelet/fang      # CLI UX
```

### 10.3 Integration Points

| Integration                 | How                                                                 | Priority |
| --------------------------- | ------------------------------------------------------------------- | -------- |
| **CI (GitHub Actions)**     | SARIF output via `--format sarif`, uploaded to GitHub Code Scanning | Phase 1  |
| **Pre-commit hook**         | `cqrs-lint --fast` for instant feedback on correctness rules        | Phase 1  |
| **golangci-lint**           | Wrap per-function rules as `go/analysis.Analyzer` plugin            | Phase 3  |
| **cqrs-gen**                | Share AST scanning code; linter validates what generator produces   | Phase 2  |
| **Editor (LSP)**            | `cqrs-lint lsp` mode for real-time diagnostics                      | Phase 4  |
| **CI on go-cqrs-lite repo** | Lint all 22 consumer projects on version bumps                      | Phase 3  |

### 10.4 Execution Modes

```bash
# Full analysis (all rules)
cqrs-lint ./...

# Correctness only (fast, for pre-commit)
cqrs-lint --only correctness ./...

# Specific rule
cqrs-lint --rule C001 ./...

# Fast mode (Critical + High correctness only)
cqrs-lint --fast ./...

# Auto-fix (safe fixes only)
cqrs-lint --fix --safe-only ./...

# Auto-fix (all fixes, requires confirmation for risky ones)
cqrs-lint --fix ./...

# Dry-run (preview fixes as diff)
cqrs-lint --fix --dry-run ./...

# Health score
cqrs-lint --health-score ./...

# SARIF for CI
cqrs-lint --format sarif ./... > results.sarif

# Migration plan (JSON output)
cqrs-lint --migration-plan --format json ./...

# Specific directory only
cqrs-lint ./internal/domain/...
```

### 10.5 Rule Configuration

```yaml
# .cqrs-lint.yml
rules:
  correctness:
    all: true
  api:
    all: true
    exclude: [A018] # don't flag "no actual event sourcing" (intentional)
  boilerplate:
    all: false
    include: [B002, B003] # but check for missing stack presets
  consistency:
    all: true
  architecture:
    all: true
  security:
    all: true

severity:
  C007: informational # downgrade time.Now() in decider
  A018: informational # "no actual ES" -> just info

ignore:
  rules:
    C009:
      - "**/must*.go" # Must constructors panic by design
    A005:
      - "internal/audit/**" # Audit handler uses SubscribeAll

fix:
  safe_only: false
  require_compilation: true
  backup: true
```

---

## 11. Priority Matrix

### Phase 1: Critical Correctness Rules (MVP)

| Rule                            | Effort | Impact   | Projects Helped      |
| ------------------------------- | ------ | -------- | -------------------- |
| C001 (missing tx commit)        | Low    | Critical | DiscordSync          |
| C002 (broken command ID)        | Low    | Critical | Kernovia             |
| C003 (silent unknown event)     | Low    | High     | KeyCountdown         |
| C005 (raw json.Unmarshal)       | Medium | High     | storbi, KeyCountdown |
| A001 (manual command interface) | Medium | High     | storbi, Kernovia     |
| A007 (dual model detection)     | Medium | Critical | SBTS, KeyCountdown   |

### Phase 2: API Misuse + Auto-Fix

| Rule                             | Effort | Impact |
| -------------------------------- | ------ | ------ |
| A002 (NewEvent -> New)           | Low    | Medium |
| A003 (explicit codec -> auto)    | Low    | Medium |
| A004 (untyped register)          | Medium | Medium |
| A005 (custom projection runner)  | Medium | High   |
| C006 (manual version arithmetic) | Low    | Low    |
| C007 (time.Now in decider)       | Medium | Medium |

### Phase 3: Boilerplate + Architecture

| Rule                           | Effort | Impact |
| ------------------------------ | ------ | ------ |
| B001 (single-event helper)     | Low    | Low    |
| B002 (repository wiring)       | Medium | Medium |
| B003 (read model boilerplate)  | Medium | Medium |
| E004 (event not in catalog)    | Medium | Medium |
| E005 (command without handler) | Medium | High   |

### Phase 4: Polish + Recommendations

| Feature               | Effort | Impact |
| --------------------- | ------ | ------ |
| Health score          | Low    | Medium |
| Migration suggestions | Medium | Medium |
| SARIF output          | Low    | Medium |
| golangci-lint plugin  | High   | Medium |
| LSP integration       | High   | Low    |

---

## Appendix A: Cross-Reference - Issues by Project

| Project        | Critical   | High             | Medium                 | Low        |
| -------------- | ---------- | ---------------- | ---------------------- | ---------- |
| DiscordSync    | C001, C012 | C004, C009       | A020, B006, B008       | B009, B010 |
| SBTS           | A007, A008 | A005, A010, C006 | A009, B002             | B005, B011 |
| KeyCountdown   | A007       | C003             | A006, A013             | B005       |
| Kernovia       | C002       | A015             | C007, A001, D004       | B010       |
| storbi         | A018       | A001, C005       | A010, D003             | D005       |
| Zlota44        | -          | A002             | C007, C008, A003, C006 | -          |
| PapDashboard   | -          | A006             | B003                   | -          |
| Standup-Killer | -          | -                | A004, C006             | B012       |
| SEC            | -          | C011             | A015                   | -          |
| KeyHolderAI    | -          | A018             | A013                   | -          |
| Cyberdom       | -          | A018             | A006                   | -          |
| StopTube       | -          | A019             | -                      | -          |
| crush-daily    | -          | -                | -                      | -          |
| bank-sync      | -          | -                | -                      | -          |
| InboxClean     | -          | -                | -                      | -          |
| go-localsync   | -          | -                | C002(panic)            | -          |
| cqrs-htmx      | -          | A015             | A014                   | -          |

## Appendix B: Library Improvement Feedforward

Some linter findings suggest library-level improvements that would eliminate entire rule categories:

| Linter Rule                          | Library Improvement                                                               | Eliminates                       |
| ------------------------------------ | --------------------------------------------------------------------------------- | -------------------------------- |
| B001 (single-event helper)           | Add `event.Single(type, id, aggType, version, payload, opts...) ([]Event, error)` | B001, B011, B012 in ALL projects |
| B002 (repository wiring)             | Better `stack/` preset documentation + `stack.QuickSetup()`                       | B002 in ALL v3 projects          |
| A013 (pointer vs value embedding)    | Document canonical form in `command.BasicCommand` docs                            | A013                             |
| C006 (manual version arithmetic)     | Consider deprecating `Version.Int()` (force `.Increment()`)                       | C006                             |
| A009 (missing stack preset)          | Add migration guide in release notes                                              | A009                             |
| C003 (silent unknown event fold)     | Add `decider.StrictFold[T](foldFunc)` that errors on unknown events by default    | C003                             |
| B007 (repeated handler registration) | Add `command.RegisterAll(dispatcher, registrations...)` variadic helper           | B007                             |
| A004 (untyped register)              | Deprecate `dispatcher.Register` in favor of `RegisterTyped` only                  | A004                             |

## Appendix C: Pipeline Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────┐
│                         cqrs-lint pipeline                          │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────────────┐  │
│  │  CLI / SDK   │───▶│  Config      │───▶│  Package Loader      │  │
│  │  (cobra)     │    │  (.cqrs-lint │    │  (go/packages)       │  │
│  │              │    │   .yml)      │    │  AST + Types + Info  │  │
│  └──────────────┘    └──────────────┘    └──────────┬───────────┘  │
│                                                     │              │
│                                          ┌──────────▼───────────┐  │
│                                          │  CQRSRegistry Builder │  │
│                                          │  (commands, events,   │  │
│                                          │   deciders, folds,    │  │
│                                          │   projections, etc.)  │  │
│                                          └──────────┬───────────┘  │
│                                                     │              │
│  ┌──────────────────────────────────────────────────▼───────────┐  │
│  │              go-finding/pipeline (parallel)                   │  │
│  │                                                               │  │
│  │  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐           │  │
│  │  │ C-rules │ │ A-rules │ │ B-rules │ │ E-rules │ ...        │  │
│  │  │ (12)    │ │ (20)    │ │ (15)    │ │ (7)     │            │  │
│  │  └────┬────┘ └────┬────┘ └────┬────┘ └────┬────┘            │  │
│  │       │           │           │           │                   │  │
│  │       └───────────┴───────────┴───────────┘                  │  │
│  │                       │                                       │  │
│  │              ┌────────▼────────┐                             │  │
│  │              │ Suppression     │                             │  │
│  │              │ Filter          │                             │  │
│  │              └────────┬────────┘                             │  │
│  └───────────────────────┼─────────────────────────────────────┘  │
│                          │                                          │
│               ┌──────────▼──────────┐                              │
│               │  []Issue            │                              │
│               │  (with SuggestedFix)│                              │
│               └──────────┬──────────┘                              │
│                          │                                          │
│           ┌──────────────┼──────────────┐                          │
│           ▼              ▼              ▼                           │
│  ┌──────────────┐ ┌────────────┐ ┌──────────────┐                 │
│  │ Console      │ │ JSON       │ │ SARIF        │                 │
│  │ (human)      │ │ (machine)  │ │ (CI/GitHub)  │                 │
│  └──────────────┘ └────────────┘ └──────────────┘                 │
│                                                                     │
│  ┌──────────────────────────────────────────────────────────────┐  │
│  │  Auto-Fix Path (--fix)                                       │  │
│  │                                                              │  │
│  │  Issues ──▶ FixTransaction ──▶ Apply ──▶ Compile Check       │  │
│  │                                        │                     │  │
│  │                                   ┌────┴────┐                │  │
│  │                                   │         │                │  │
│  │                                 Pass      Fail               │  │
│  │                                   │         │                │  │
│  │                              Commit    Rollback              │  │
│  └──────────────────────────────────────────────────────────────┘  │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```
