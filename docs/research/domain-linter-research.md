# Domain-Aware CQRS Linter: Research and Design

> **Date**: 2026-07-16
> **Scope**: Design a domain-aware linter for `go-cqrs-lite` consumers
> **Based on**: Analysis of 22 consumer projects, existing tooling (`cqrs-gen`, `api-stability`, `doc-check`), and the `go-structure-linter` architecture
> **Companion document**: [Consumer Projects Analysis](../../../docs/go-cqrs-lite-consumer-projects-analysis.md)

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [Architecture Options](#2-architecture-options)
3. [Rule Catalog](#3-rule-catalog)
4. [Detection Logic](#4-detection-logic)
5. [Auto-Fix Capabilities](#5-auto-fix-capabilities)
6. [Recommendation Engine](#6-recommendation-engine)
7. [Implementation Blueprint](#7-implementation-blueprint)
8. [Priority Matrix](#8-priority-matrix)

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

---

## 2. Architecture Options

### Option A: `go/analysis` Plugin (golangci-lint integration)

**Pros:**

- Integrates with existing CI (golangci-lint, go vet)
- Type information available (`pass.TypesInfo`)
- Per-package incremental analysis

**Cons:**

- Per-package scope - cannot check cross-module invariants
- No project-level context (directory structure, go.mod analysis)
- Requires golangci-lint knowledge for consumers
- Limited auto-fix support in the framework

**Best for:** Single-function semantic checks (purity, error handling)

### Option B: Standalone Tool (go-structure-linter architecture) - RECOMMENDED

**Pros:**

- Cross-file, cross-module analysis (the rules need this)
- Project-level context (go.mod, directory structure, module graph)
- Proven architecture across 2 existing tools
- Rich auto-fix support (DryRunFix, ApplyFix, ValidateFix)
- `go-finding` for unified output (SARIF, JSON, console, GitHub Annotations)
- SDK for programmatic use
- Already has CQRS-aware AST scanning (from `cqrs-gen`)

**Cons:**

- Consumers must install and run separately from golangci-lint
- More code to maintain than a single analyzer

### Option C: Hybrid (both)

Build the standalone tool for project-level rules AND a `go/analysis` plugin for per-function rules. The standalone tool can shell out to or embed the analyzer.

**Verdict:** Start with **Option B** (standalone tool). It covers all identified rules. Add `go/analysis` plugin later if golangci-lint integration is requested.

---

## 3. Rule Catalog

### Category 1: Correctness (Bug Detection) - 12 rules

These rules detect bugs that cause silent data loss, broken idempotency, or incorrect behavior.

| ID   | Rule                               | Severity | Auto-Fix | Projects Affected                           |
| ---- | ---------------------------------- | -------- | -------- | ------------------------------------------- |
| C001 | `missing-tx-commit`                | Critical | Yes      | DiscordSync                                 |
| C002 | `broken-command-id`                | Critical | Yes      | Kernovia                                    |
| C003 | `silent-unknown-event-fold`        | High     | Yes      | KeyCountdown                                |
| C004 | `checkpoint-before-async-complete` | High     | No       | DiscordSync                                 |
| C005 | `raw-json-unmarshal-payload`       | High     | Yes      | storbi, KeyCountdown                        |
| C006 | `manual-version-arithmetic`        | Medium   | Yes      | SBTS, Standup-Killer, KeyCountdown, Zlota44 |
| C007 | `time-now-in-decider`              | Medium   | No       | Zlota44, Kernovia, SEC                      |
| C008 | `float64-for-money`                | Medium   | No       | Zlota44                                     |
| C009 | `panic-in-production-path`         | Medium   | No       | DiscordSync (4 sites)                       |
| C010 | `swallowed-error-in-fold`          | Medium   | Yes      | SEC                                         |
| C011 | `nondeterministic-decider`         | Medium   | No       | SEC (diceService)                           |
| C012 | `missing-error-return-in-with-tx`  | Critical | Yes      | DiscordSync                                 |

#### C001: Missing Transaction Commit

**Detects:** A `withTx` or similar transaction helper whose success path returns `nil` instead of `tx.Commit()`.

**Pattern:**

```go
// BAD: if body returns nil without committing, data is silently lost
func withTx(ctx, db, body) error {
    tx, _ := db.BeginTx(ctx, nil)
    if err := body(tx); err != nil { _ = tx.Rollback(); return err }
    return nil  // BUG: should be tx.Commit()
}
```

**Auto-fix:** Change `return nil` to `return tx.Commit()` in the success path of transaction wrapper functions.

**Detection logic:**

1. Find functions that call `db.BeginTx` or `sql.DB.BeginTx`
2. Check if the function returns `error` as last return value
3. Check if the success path (no error from body) returns `nil` instead of `tx.Commit()`

#### C002: Broken Command ID

**Detects:** Command structs whose `ID()` method returns a zero-value `CommandID`.

**Pattern:**

```go
// BAD: ID() returns zero value, breaking idempotency + tracing
func (c LoadPluginCommand) ID() cqrsid.CommandID {
    return cqrsid.CommandID{}  // always zero
}
```

**Auto-fix:** Replace manual interface implementation with `*command.BasicCommand` embedding.

**Detection logic:**

1. Find types implementing `ID() CommandID` (or returning a `CommandID`)
2. Check if the return statement returns a composite literal `{}` or zero value
3. Flag as broken

#### C003: Silent Unknown Event in Fold

**Detects:** Fold functions that silently return unchanged state for unrecognized event types instead of returning an error.

**Pattern:**

```go
// BAD: corrupt event streams go undetected
func fold(state State, evt event.Event) (State, error) {
    switch evt.Type() {
    case eventCreated: // ...
    case eventUpdated: // ...
    default:
        return state, nil  // BUG: should return error
    }
}
```

**Auto-fix:** Change `return state, nil` to `return state, fmt.Errorf("unknown event type: %s", evt.Type())` in the default case.

**Detection logic:**

1. Find functions matching the fold signature: `func(Type, event.Event) (Type, error)`
2. Check if there's a `switch evt.Type()` with a `default:` case
3. Check if the default case returns `nil` error

#### C004: Checkpoint Before Async Complete

**Detects:** Projection handlers that enqueue async work and return nil immediately, allowing the checkpoint to advance before the work completes.

**Pattern:**

```go
// BAD: checkpoint advances before download completes
func (p *Projection) Handle(ctx, evt) error {
    p.queue <- evt  // enqueue async work
    return nil      // checkpoint advances, work may never complete if process crashes
}
```

**Detection logic:**

1. Find projection `Handle` methods
2. Check if they write to a channel or start a goroutine
3. Check if they return nil immediately after

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

**Auto-fix:** Replace with `event.DecodePayloadAuto[T](evt)` call.

**Detection logic:**

1. Find call expressions to `json.Unmarshal` or `json.NewDecoder` where the first argument is `.Payload()` on an event type
2. Flag and suggest `DecodePayloadAuto`

#### C006: Manual Version Arithmetic

**Detects:** `event.Version(version.Int()+1)` instead of `version.Increment()`.

**Pattern:**

```go
// BAD: manual arithmetic
evt, _ := event.NewEvent(type, id, aggType, event.Version(version.Int()+1), payload)

// GOOD: use the method
evt, _ := event.NewEvent(type, id, aggType, version.Increment(), payload)
```

**Auto-fix:** Replace `event.Version(X.Int()+1)` with `X.Increment()`.

**Detection logic:**

1. Find `event.Version(` calls
2. Check if the argument is `X.Int() + 1` pattern
3. Suggest `.Increment()`

#### C007: time.Now() in Decider

**Detects:** Calls to `time.Now()` inside decider functions (decide/Apply/fold).

**Pattern:**

```go
// BAD: non-deterministic, untestable
func decideX(...) func(State, Version) ([]Event, error) {
    return func(state State, ver Version) ([]Event, error) {
        evt := NewEvent(..., time.Now())  // BUG: non-deterministic
        return []Event{evt}, nil
    }
}
```

**Detection logic:**

1. Find functions matching decide signature (returning `func(State, Version) ([]Event, error)`)
2. AST-walk the closure body for `time.Now()` calls
3. Also check fold functions for `time.Now()`

#### C008: float64 for Monetary Values

**Detects:** `float64` fields in event payloads or state structs where the field name suggests money (Amount, Price, Cost, Value, Balance, etc.).

**Detection logic:**

1. Find struct fields with `float64` type
2. Check if field name matches monetary patterns: `Amount|Price|Cost|Value|Balance|Fee|Total|Subtotal|Tax|Discount|Payment|Charge|Salary|Wage|Revenue|Profit|Loss|Deposit|Withdrawal`
3. Suggest `int64` (cents) or `decimal.Decimal`

#### C009: Panic in Production Code

**Detects:** `panic()` calls in non-test, non-init Go files.

**Auto-fix:** None (requires manual judgment), but can suggest `return error`.

**Detection logic:**

1. Find `panic(` call expressions
2. Exclude `_test.go` files and `init()` functions
3. Flag with context about what would need to change

#### C010: Swallowed Error in Fold

**Detects:** Fold function cases that ignore decode errors.

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

**Auto-fix:** Propagate the error: `if err != nil { return state, fmt.Errorf("...: %w", err) }`.

**Detection logic:**

1. Find assignments with `_ :=` in fold function bodies
2. Check if the right side is a function that returns `(T, error)`

#### C011: Nondeterministic Decider

**Detects:** Decider decide functions that take service dependencies (non-pure inputs).

**Pattern:**

```go
// QUESTIONABLE: diceService makes decider impure
func DecidePlayRound(state State, cmd Cmd, dice DiceService) ([]Event, error)
```

**Detection logic:**

1. Find decide functions (returning `([]event.Event, error)` or `func(...) ([]event.Event, error)`)
2. Check if they take non-value parameters (interfaces, function types)
3. Flag as potentially impure (informational, not necessarily wrong)

#### C012: Missing Error Return in withTx

**Detects:** Transaction wrapper functions that don't commit on success (more general version of C001).

**Detection logic:** Same as C001 but also catches patterns where the commit is conditionally skipped.

---

### Category 2: API Misuse - 20 rules

These rules detect incorrect use of go-cqrs-lite APIs.

| ID   | Rule                             | Severity      | Auto-Fix | Projects Affected          |
| ---- | -------------------------------- | ------------- | -------- | -------------------------- |
| A001 | `manual-command-interface`       | High          | Yes      | storbi, Kernovia           |
| A002 | `event-newevent-manual-marshal`  | High          | Yes      | Zlota44                    |
| A003 | `explicit-codec-in-decode`       | Medium        | Yes      | Zlota44                    |
| A004 | `untyped-dispatch-register`      | Medium        | Yes      | storbi, Standup-Killer     |
| A005 | `custom-projection-runner`       | High          | No       | SBTS, DiscordSync          |
| A006 | `adapter-layer-wrapping`         | Medium        | No       | PapDashboard, KeyCountdown |
| A007 | `dual-model-oo-plus-functional`  | Critical      | No       | SBTS, KeyCountdown         |
| A008 | `parallel-type-system`           | Critical      | No       | SBTS                       |
| A009 | `missing-stack-preset`           | Low           | No       | All v3 projects            |
| A010 | `custom-error-types`             | Medium        | No       | SBTS                       |
| A011 | `inconsistent-json-key-casing`   | Low           | No       | Kernovia                   |
| A012 | `missing-tombstone-handling`     | Medium        | No       | Various                    |
| A013 | `pointer-vs-value-basic-command` | Low           | No       | KeyCountdown, KeyHolderAI  |
| A014 | `deprecated-api-usage`           | Low           | Yes      | cqrs-htmx                  |
| A015 | `global-mutable-state`           | High          | No       | cqrs-htmx, SEC             |
| A016 | `missing-idempotency-middleware` | Medium        | No       | Various                    |
| A017 | `missing-snapshot-strategy`      | Low           | No       | Large aggregate projects   |
| A018 | `no-actual-event-sourcing`       | Informational | No       | KeyHolderAI, Cyberdom      |
| A019 | `vendored-cqrs`                  | Medium        | No       | StopTube                   |
| A020 | `event-store-as-append-log`      | Medium        | No       | DiscordSync                |

#### A001: Manual Command Interface

**Detects:** Command structs that manually implement `Type()`, `AggregateID()`, `ID()` instead of embedding `*command.BasicCommand`.

**Pattern:**

```go
// BAD: 4 methods of boilerplate per command
type CreateItemCommand struct {
    id    id.AggregateID
    itemType string
}
func (c CreateItemCommand) Type() command.Type { return cmdCreateItem }
func (c CreateItemCommand) AggregateID() id.AggregateID { return c.id }
func (c CreateItemCommand) ID() command.CommandID { return c.cmdID }

// GOOD: embedding provides all three
type CreateItemCommand struct {
    *command.BasicCommand
    itemType string
}
```

**Auto-fix:** Add `*command.BasicCommand` embedding, remove manual method implementations, update constructor to call `command.New()`.

**Detection logic:**

1. Find structs with methods matching `Type() command.Type` or `Type() string`
2. Check if the struct embeds `*command.BasicCommand` or `command.BasicCommand`
3. If not embedded but methods exist, flag

#### A002: event.NewEvent with Manual Marshal

**Detects:** Using `event.NewEvent` (which requires pre-marshaled `[]byte`) instead of `event.New` (which takes typed payload and auto-marshals).

**Pattern:**

```go
// BAD: manual marshaling
payload, _ := json.Marshal(data)
evt, _ := event.NewEvent(type, id, aggType, version, payload)

// GOOD: typed, auto-marshaled
evt, _ := event.New(type, id, aggType, version, data)
```

**Auto-fix:** Replace `event.NewEvent(type, id, aggType, ver, marshaledBytes)` with `event.New(type, id, aggType, ver, typedPayload)`.

**Detection logic:**

1. Find calls to `event.NewEvent`
2. Check if the payload argument is a `[]byte` (from `json.Marshal` or similar)
3. Suggest `event.New` with the original typed value

#### A003: Explicit Codec in Decode

**Detects:** `event.DecodePayload[T](evt, codecInstance)` instead of `event.DecodePayloadAuto[T](evt)`.

**Auto-fix:** Replace with `DecodePayloadAuto[T](evt)`.

**Detection logic:** Find `DecodePayload[` calls with a second codec argument.

#### A004: Untyped Dispatch Register

**Detects:** `dispatcher.Register(type, handler)` with manual type assertion instead of `command.RegisterTyped` / `query.RegisterTyped`.

**Pattern:**

```go
// BAD: untyped, manual assertion
disp.Register(cmdType, func(ctx context.Context, cmd command.Command) error {
    c, ok := cmd.(*MyCmd)  //nolint:forcetypeassert
    if !ok { return err }
    // ...
})

// GOOD: type-safe
command.RegisterTyped(dispatcher, cmdType, func(ctx context.Context, c *MyCmd) error {
    // ...
})
```

**Auto-fix:** Convert to `RegisterTyped` call.

**Detection logic:**

1. Find `.Register(` calls on a dispatcher
2. Check if the handler function contains a type assertion on the command argument
3. Suggest `RegisterTyped`

#### A005: Custom Projection Runner

**Detects:** Manual `bus.SubscribeAll` + switch-based event handling loops instead of `projectionhost.Host`.

**Pattern:**

```go
// BAD: no checkpoint, no DLQ, no crash recovery
bus.SubscribeAll(func(ctx, evt) {
    switch evt.Type() {
    case eventX: // handle
    }
})

// GOOD: managed lifecycle
host, _ := projectionhost.New(journal, checkpointStore, opts...)
host.Register(&MyProjection{})
go host.Start(ctx)
```

**Detection logic:**

1. Find `bus.SubscribeAll` calls
2. Check if the callback contains a `switch evt.Type()` or `if evt.Type() ==` pattern
3. Check if `projectionhost` is imported in the same module
4. If not using projectionhost, flag

#### A006: Adapter Layer Wrapping

**Detects:** Types that wrap go-cqrs-lite interfaces and add conversion methods (WrapEvent/UnwrapEvent, ToEvent/FromEvent).

**Detection logic:**

1. Find types with methods like `Wrap*`, `Unwrap*`, `To*`, `From*` that convert to/from go-cqrs-lite types
2. Find types implementing go-cqrs-lite interfaces (Store, Bus, EventSink) through delegation to a wrapped type
3. Flag as unnecessary indirection

#### A007: Dual Model (OO + Functional)

**Detects:** Projects that have both an OO aggregate pattern (struct with `uncommittedEvents`, `addEvent`, `MarkEventsAsCommitted`) AND a functional decider pattern.

**Detection logic:**

1. Find structs with fields named `uncommittedEvents`, `pendingEvents`, or similar
2. Find methods named `addEvent`, `markEventsAsCommitted`, `apply`, `getUncommittedEvents`
3. Cross-reference: if the same project also has `decider.Decider[T]` or `Fold*` functions
4. Flag as dual model

#### A008: Parallel Type System

**Detects:** Custom types that duplicate go-cqrs-lite domain types (AggregateID, Version, CommandType, EventType, EventID, CommandID).

**Detection logic:**

1. Find type definitions for types named `AggregateID`, `Version`, `CommandType`, `EventType`, `EventID`, `CommandID`
2. Check if they are NOT aliases (`= id.AggregateID` etc.)
3. Flag as parallel type system

#### A009: Missing Stack Preset

**Detects:** Projects that manually wire event store + bus + repository instead of using `stack/sqlite`, `stack/postgres`, etc.

**Detection logic:**

1. Find code that calls `storage.NewSQLiteEventStore` + `watermill.NewEventBus` + `decider.NewRepository` in sequence
2. Check if `stack/` is NOT imported
3. Suggest using stack preset

#### A010: Custom Error Types

**Detects:** Custom error struct types (`ValidationError`, `DomainError`) instead of using `go-error-family`.

**Detection logic:**

1. Find types implementing `Error() string` that are NOT from go-error-family
2. Check if they have `Field`, `Value`, `Message` or `Operation`, `Entity` fields
3. Suggest using `errorfamily.NewRejection`, `errorfamily.NewConflict`, etc.

#### A015: Global Mutable State

**Detects:** Package-level mutable variables (not constants) that are modified by exported setter functions.

**Pattern:**

```go
// BAD: package-level singleton, unsafe for multiple instances
var globalRegistry *Registry
func SetRegistry(r *Registry) { globalRegistry = r }
```

**Detection logic:**

1. Find `var` declarations at package level (not `const`)
2. Find exported functions that assign to them
3. Flag as global mutable state

---

### Category 3: Boilerplate Detection - 15 rules

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

**Detects:** The universal pattern of creating a single event from a typed payload.

**Pattern:**

```go
// This exists in EVERY project under different names:
func singleEvent(t event.Type, aggID id.AggregateID, aggType string, ver event.Version, payload any, opts ...event.Option) ([]event.Event, error) {
    evt, err := event.NewEvent(t, aggID, aggType, ver, payload, opts...)
    if err != nil {
        return nil, fmt.Errorf("create event %s: %w", t, err)
    }
    return []event.Event{evt}, nil
}
```

**Detection logic:**

1. Find functions that call `event.NewEvent` or `event.New` and wrap the result in `[]event.Event{}`
2. Check function signature: takes event type, aggregate ID, version, payload -> returns `([]event.Event, error)`
3. Flag and suggest using a shared library helper

**Auto-fix:** Replace all call sites with a library-provided `event.Single(...)` function (if added to the library).

#### B002: Repository Wiring Boilerplate

**Detects:** The repeated 5-step repository setup pattern.

**Detection logic:**

1. Find functions that call 3+ of these in sequence: `storage.NewSQLiteEventStore`, `watermill.NewEventBus`, `bus.Use(`, `decider.NewRepository`, `snapshot.New*Store`
2. Check if `stack/sqlite` or `stack/postgres` is imported
3. If not, suggest migrating to stack preset

#### B003: Read Model Projection Boilerplate

**Detects:** Manual `bus.SubscribeAll` + switch-on-event-type projection implementations.

**Detection logic:**

1. Find `bus.SubscribeAll` callbacks
2. Check for `switch evt.Type()` pattern
3. Check if `projectionhost` is used anywhere in the module
4. Suggest `projectionhost.Host` with typed projections

---

### Category 4: Consistency - 5 rules

| ID   | Rule                             | Severity      | Auto-Fix |
| ---- | -------------------------------- | ------------- | -------- |
| D001 | `inconsistent-command-embedding` | Low           | No       |
| D002 | `version-fragmentation`          | Informational | No       |
| D003 | `inconsistent-logging-library`   | Low           | No       |
| D004 | `inconsistent-json-key-casing`   | Low           | No       |
| D005 | `stale-documentation-version`    | Medium        | No       |

---

### Category 5: Architecture - 7 rules

| ID   | Rule                        | Severity | Auto-Fix |
| ---- | --------------------------- | -------- | -------- |
| E001 | `layer-violation`           | High     | No       |
| E002 | `circular-dependency`       | High     | No       |
| E003 | `missing-module-boundary`   | Medium   | No       |
| E004 | `event-type-not-in-catalog` | Medium   | Yes      |
| E005 | `command-without-handler`   | High     | No       |
| E006 | `event-without-projection`  | Medium   | No       |
| E007 | `query-without-handler`     | High     | No       |

#### E004: Event Type Not in Catalog

**Detects:** Event types used in deciders but not registered in the catalog.

**Auto-fix:** Add `catalog.Event[PayloadType](eventType, ...)` call.

**Detection logic:**

1. Collect all `event.Type` constants
2. Collect all `event.NewEvent`/`event.New` calls to see which types are emitted
3. Collect all `catalog.Event[` calls
4. Diff: types emitted but not in catalog

#### E005: Command Without Handler

**Detects:** Command types that are defined but never registered with a dispatcher.

**Detection logic:**

1. Find all command type constants and structs embedding `BasicCommand`
2. Find all `command.RegisterTyped` calls
3. Diff: commands defined but never registered

---

### Category 6: Security - 3 rules

| ID   | Rule                                        | Severity | Auto-Fix |
| ---- | ------------------------------------------- | -------- | -------- |
| S001 | `hardcoded-secrets-in-events`               | Critical | No       |
| S002 | `missing-encryption-for-sensitive-payloads` | High     | No       |
| S003 | `missing-event-signing`                     | Medium   | No       |

---

## 4. Detection Logic

### 4.1 AST Analysis Layer

The linter needs three levels of analysis:

#### Level 1: File-level AST (from `cqrs-gen`)

Already implemented in `cmd/cqrs-gen/main.go`. Scans for:

- `//cqrs:command`, `//cqrs:event`, `//cqrs:query` markers
- Struct tags: `cqrs:"command:CreateUser"`
- Type declarations with their fields and methods
- Function signatures matching CQRS patterns

**Enhancement needed:** Also scan for:

- `event.NewEvent` vs `event.New` calls
- `json.Unmarshal(evt.Payload()` patterns
- `time.Now()` inside specific function types
- `panic()` calls
- Manual interface implementations (Type/AggregateID/ID methods)

#### Level 2: Type-level cross-reference

Using `go/types` package to resolve types across files:

- Which types embed `*command.BasicCommand`?
- Which functions have the fold signature `func(S, event.Event) (S, error)`?
- Which types implement `projection.Projection`?
- Which types implement `command.Command`?

**Implementation:** Use `golang.org/x/tools/go/packages` to load type information:

```go
cfg := &packages.Config{Mode: packages.NeedName | packages.NeedTypes | packages.NeedTypesInfo | packages.NeedSyntax}
pkgs, _ := packages.Load(cfg, "./...")
```

#### Level 3: Project-level structural

Using filesystem analysis (from `go-structure-linter`):

- Module dependencies (parse `go.mod` files)
- Directory structure (aggregate/command/event/projection organization)
- Presence of stack presets vs manual wiring
- Version of go-cqrs-lite being used

### 4.2 Rule Implementation Pattern

Each rule follows this structure:

```go
type CQRSLintRule struct {
    Name        string
    Description string
    Category    RuleCategory
    Severity    Severity
    EnableByDefault bool
}

func (r *CQRSLintRule) Check(ctx *AnalysisContext) []Issue {
    // ctx provides:
    // - Parsed AST for all Go files
    // - Type information (go/types)
    // - Project structure (go.mod, directory tree)
    // - CQRS type registry (commands, events, queries, projections)

    var issues []Issue
    for _, file := range ctx.GoFiles {
        ast.Inspect(file.AST, func(n ast.Node) bool {
            // Pattern-specific detection logic
            if r.matches(n, ctx) {
                issues = append(issues, Issue{
                    Rule:     r.Name,
                    File:     file.Path,
                    Line:     position(n),
                    Message:  r.formatMessage(n),
                    Fix:      r.autoFix(n, ctx),
                })
            }
            return true
        })
    }
    return issues
}
```

### 4.3 CQRS Type Registry

The linter builds a registry of all CQRS types in the project:

```go
type CQRSRegistry struct {
    Commands   []CommandInfo    // structs embedding BasicCommand or implementing Command
    Events     []EventInfo      // event type constants + payload types
    Queries    []QueryInfo      // query type constants + result types
    Deciders   []DeciderInfo    // Decider[T] instances + decide/fold functions
    Projections []ProjectionInfo // Projection implementations
    Aggregates []AggregateInfo  // inferred from deciders
}

type CommandInfo struct {
    Name          string
    Type          command.Type
    StructName    string
    File          string
    HasBasicCmd   bool   // embeds *command.BasicCommand?
    HasHandler    bool   // registered with RegisterTyped?
    HasIdempotency bool  // has IdempotencyKey()?
}
```

This registry is built once during analysis and shared across all rules.

---

## 5. Auto-Fix Capabilities

### 5.1 Auto-Fixable Rules (18 rules)

| Rule      | Auto-Fix Description                                          | Risk Level |
| --------- | ------------------------------------------------------------- | ---------- |
| C001      | `return nil` -> `return tx.Commit()` in tx wrappers           | Low        |
| C002      | Add `*command.BasicCommand` embedding, remove manual methods  | Medium     |
| C003      | Add error return in fold default case                         | Low        |
| C005      | `json.Unmarshal` -> `DecodePayloadAuto[T]`                    | Medium     |
| C006      | `Version(x.Int()+1)` -> `x.Increment()`                       | Low        |
| C010      | Propagate swallowed decode errors                             | Low        |
| A001      | Add `*command.BasicCommand` embedding                         | Medium     |
| A002      | `event.NewEvent` -> `event.New` with typed payload            | Medium     |
| A003      | `DecodePayload[T](evt, codec)` -> `DecodePayloadAuto[T](evt)` | Low        |
| A004      | `Register` -> `RegisterTyped`                                 | Medium     |
| B001      | Replace `singleEvent` helpers with library function           | Low        |
| B003      | Wrap manual projections in `projectionhost`                   | High       |
| B006      | Centralize FK stub SQL                                        | Medium     |
| B008      | Replace custom retry with `retry.Do`                          | Medium     |
| B011/B012 | Replace `mustMarshal`/`makeEvent` with library helper         | Low        |
| E004      | Add missing catalog entries                                   | Low        |

### 5.2 Auto-Fix Safety Model

Following the `go-structure-linter` pattern:

```go
type FixableRule interface {
    DryRunFix(path string, issue Issue) (diff string, err error)  // preview
    ApplyFix(path string, issue Issue) (outcome FixOutcome, err error)  // apply
    ValidateFix(path string, issue Issue) error  // verify
}
```

**Safety levels:**

- **Safe**: Only adds code (missing error handling, missing catalog entries)
- **Moderate**: Modifies code patterns (json.Unmarshal -> DecodePayloadAuto, manual version -> Increment)
- **Risky**: Restructures types (adds BasicCommand embedding, removes manual methods) - requires compilation verification

All risky fixes must:

1. Run `go build` after applying
2. Revert on compilation failure
3. Require `--force` flag or interactive confirmation

### 5.3 Fix Transaction

```go
type FixTransaction struct {
    backups map[string][]byte  // original file content
    applied []FixOperation
}

func (t *FixTransaction) Commit() error {
    // Verify all changes compile
    if err := goBuild(); err != nil {
        return t.Rollback()  // restore all files
    }
    return nil
}
```

---

## 6. Recommendation Engine

Beyond finding violations, the linter should provide actionable recommendations.

### 6.1 Pattern Recommendations

| Trigger                                     | Recommendation                                            |
| ------------------------------------------- | --------------------------------------------------------- |
| Project has >5 commands                     | Consider using `cqrs-gen` for handler registration        |
| Project has >3 projections                  | Migrate to `projectionhost.Host`                          |
| Project manually wires event store + bus    | Migrate to `stack/sqlite` or `stack/postgres`             |
| Project uses `fmt.Errorf` for domain errors | Adopt `go-error-family` for classification                |
| Project on v3                               | Plan migration to v4                                      |
| Aggregate has >100 events without snapshots | Add `snapshot.EveryNEvents(50)`                           |
| Projection handles events synchronously     | Consider `projectionhost.WithBatchSize(100)`              |
| Decider doesn't use correlation enricher    | Add `decider.WithEnricher(correlation.ContextEnricher())` |

### 6.2 Health Score

The linter can compute a "CQRS Health Score" per project:

```
Score = 100
- 10 points per Critical violation
- 5 points per High violation
- 2 points per Medium violation
- 1 point per Low violation
- 5 points per category of universal boilerplate
```

Projects scoring <60 should be flagged for refactoring.

### 6.3 Migration Path Suggestions

For projects with systemic issues (dual model, parallel types), suggest a step-by-step migration plan:

1. Add the correct pattern alongside the old one
2. Migrate consumers one at a time
3. Remove the old pattern
4. Verify with the linter (score should improve)

---

## 7. Implementation Blueprint

### 7.1 Module Structure

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
```

### 7.2 Dependencies

```
golang.org/x/tools/go/packages   # Type-aware package loading
golang.org/x/tools/go/ast/       # AST inspection helpers
github.com/larsartmann/go-finding  # Unified finding model + pipeline
github.com/larsartmann/go-error-family  # Classified errors
github.com/spf13/cobra            # CLI
```

### 7.3 Integration Points

1. **golangci-lint**: Can wrap as a `go/analysis` plugin for per-function rules
2. **CI**: SARIF output for GitHub Code Scanning
3. **Pre-commit**: Fast mode (correctness rules only) for commit hooks
4. **Editor**: LSP integration for real-time feedback
5. **cqrs-gen**: The linter and generator share the same AST scanning foundation

### 7.4 Execution Modes

```bash
# Full analysis (all rules)
cqrs-lint ./...

# Correctness only (fast, for pre-commit)
cqrs-lint --only correctness ./...

# Specific rule
cqrs-lint --rule C001 ./...

# Auto-fix (safe fixes only)
cqrs-lint --fix --safe-only ./...

# Auto-fix (all fixes, requires confirmation)
cqrs-lint --fix ./...

# Dry-run (preview fixes)
cqrs-lint --fix --dry-run ./...

# Health score
cqrs-lint --health-score ./...

# SARIF for CI
cqrs-lint --format sarif ./... > results.sarif
```

### 7.5 Rule Configuration

```yaml
# .cqrs-lint.yml
rules:
  correctness:
    all: true # enable all correctness rules
  api:
    all: true
    exclude: [A018] # don't flag "no actual event sourcing" (intentional)
  boilerplate:
    all: false # boilerplate is informational by default
    include: [B002, B003] # but check for missing stack presets
  consistency:
    all: true
  architecture:
    all: true
  security:
    all: true

severity:
  # Override default severity per rule
  A018: informational # "no actual ES" -> just info

fix:
  safe_only: false
  require_compilation: true # verify fixes compile
  backup: true # keep .bak files
```

---

## 8. Priority Matrix

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

| Project        | Critical Issues | High Issues      | Medium Issues          | Low Issues |
| -------------- | --------------- | ---------------- | ---------------------- | ---------- |
| DiscordSync    | C001, C012      | C004, C009       | A020, B006, B008       | B009, B010 |
| SBTS           | A007, A008      | A005, A010, C006 | A009, B002             | B005, B011 |
| KeyCountdown   | A007            | C003             | A006, A013             | B005       |
| Kernovia       | C002            | A015             | C007, A001, D004       | B010       |
| storbi         | A018            | A001, C005       | A010, D003             | D005       |
| Zlota44        | -               | A002             | C007, C008, A003, C006 | -          |
| PapDashboard   | -               | A006             | B003                   | -          |
| Standup-Killer | -               | -                | A004, C006             | B012       |
| SEC            | -               | C011             | A015                   | -          |
| KeyHolderAI    | -               | A018             | A013                   | -          |
| Cyberdom       | -               | A018             | A006                   | -          |
| StopTube       | -               | A019             | -                      | -          |
| crush-daily    | -               | -                | -                      | -          |
| bank-sync      | -               | -                | -                      | -          |
| InboxClean     | -               | -                | -                      | -          |
| go-localsync   | -               | -                | C002(panic)            | -          |
| cqrs-htmx      | -               | A015             | A014                   | -          |

## Appendix B: Rule to Library Improvement Feedforward

Some linter findings suggest library-level improvements that would eliminate entire rule categories:

| Linter Rule                       | Library Improvement                                                               | Eliminates                       |
| --------------------------------- | --------------------------------------------------------------------------------- | -------------------------------- |
| B001 (single-event helper)        | Add `event.Single(type, id, aggType, version, payload, opts...) ([]Event, error)` | B001, B011, B012 in ALL projects |
| B002 (repository wiring)          | Better `stack/` preset documentation + `stack.QuickSetup()`                       | B002 in ALL v3 projects          |
| A013 (pointer vs value embedding) | Document canonical form in `command.BasicCommand` docs                            | A013                             |
| C006 (manual version arithmetic)  | Consider deprecating `Version.Int()` (force `.Increment()`)                       | C006                             |
| A009 (missing stack preset)       | Add migration guide in release notes                                              | A009                             |
