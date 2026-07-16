# cqrs-lint

A domain-aware linter for [go-cqrs-lite](https://github.com/larsartmann/go-cqrs-lite) consumers.

It finds bugs, API misuse, and boilerplate that generic linters cannot detect — because these are CQRS-specific patterns that require understanding the library's types, conventions, and architecture.

## Quickstart

```bash
# Build
go build -o cqrs-lint ./cmd/cqrs-lint

# Lint your project
cqrs-lint ./...

# JSON output for CI
cqrs-lint ./... --format json

# SARIF for GitHub Code Scanning
cqrs-lint ./... --format sarif > results.sarif

# Fast mode (Critical/High rules only, for pre-commit)
cqrs-lint --fast ./...

# Health score
cqrs-lint --health-score ./...

# Apply auto-fixes (with dry-run preview first)
cqrs-lint --fix --dry-run ./...
cqrs-lint --fix ./...

# List available rules
cqrs-lint rules
```

## What It Detects

### Correctness Rules (bugs)

| ID   | Rule                            | Severity | Description                                                      |
| ---- | ------------------------------- | -------- | ---------------------------------------------------------------- |
| C001 | missing-tx-commit               | Critical | Transaction wrapper returns nil instead of tx.Commit()           |
| C002 | broken-command-id               | Critical | Command ID() returns zero value — breaks idempotency             |
| C003 | silent-unknown-event-fold       | Error    | Fold function silently ignores unknown event types               |
| C005 | raw-json-unmarshal-payload      | Error    | Raw json.Unmarshal on event payload instead of DecodePayloadAuto |
| C006 | manual-version-arithmetic       | Warning  | event.Version(x.Int()+1) instead of x.Increment()                |
| C007 | time-now-in-decider             | Warning  | time.Now() inside decider — non-deterministic                    |
| C008 | float64-for-money               | Warning  | float64 field with monetary name — use decimal or cents          |
| C009 | panic-in-production             | Warning  | panic() in production code — use error returns                   |
| C010 | swallowed-error-in-fold         | Warning  | Error from decode/unmarshal discarded in fold                    |
| C012 | missing-error-return-in-with-tx | Critical | withTx ignores body error — failures silently lost               |

### API Misuse Rules (wrong API)

| ID   | Rule                      | Severity | Description                                              |
| ---- | ------------------------- | -------- | -------------------------------------------------------- |
| A001 | manual-command-interface  | Error    | Manual Type()/ID()/AggregateID() instead of BasicCommand |
| A002 | newevent-manual-marshal   | Warning  | event.NewEvent with json.Marshal — use event.New         |
| A003 | explicit-codec-in-decode  | Info     | Explicit codec — use DecodePayloadAuto                   |
| A004 | untyped-dispatch-register | Warning  | Type assertion in handler — use RegisterTyped            |
| A005 | custom-projection-runner  | Warning  | Manual bus.SubscribeAll — use projectionhost             |
| A006 | adapter-layer-wrapping    | Info     | WrapEvent/UnwrapEvent adapter methods                    |
| A007 | dual-model-oo-functional  | Error    | Both OO aggregates and functional deciders               |
| A008 | parallel-type-system      | Error    | Custom AggregateID/Version types duplicating library     |

### Boilerplate Rules (repetitive code)

| ID   | Rule                      | Severity | Description                     |
| ---- | ------------------------- | -------- | ------------------------------- |
| B001 | single-event-helper       | Info     | Use event.Single() instead      |
| B002 | manual-repository-wiring  | Info     | Use stack preset instead        |
| B003 | subscribeall-large-switch | Info     | Split into separate projections |

### Architecture Rules (cross-module invariants)

| ID   | Rule                    | Severity | Description                               |
| ---- | ----------------------- | -------- | ----------------------------------------- |
| E004 | event-not-in-catalog    | Info     | Event type emitted but not in catalog     |
| E005 | command-without-handler | Warning  | Command type defined but never registered |

### Other Categories

| ID   | Rule                      | Severity | Description                                  |
| ---- | ------------------------- | -------- | -------------------------------------------- |
| D001 | inconsistent-event-naming | Info     | Mixed dot notation and PascalCase            |
| D002 | inconsistent-json-casing  | Info     | Mixed camelCase and snake_case JSON tags     |
| S001 | hardcoded-secrets         | Critical | Potential hardcoded secret in string literal |

## Suppression

Inline suppression for false positives:

```go
//cqrs-lint:ignore(C007) wall-clock is domain logic
now := time.Now()
```

## Auto-Fix

Rules with `[auto-fixable]` can be applied automatically:

```bash
cqrs-lint --fix --dry-run ./...  # preview
cqrs-lint --fix ./...            # apply
```

C006 (manual version arithmetic) and C003 (silent fold) are auto-fixable.

## Architecture

Built on three foundation libraries:

- **[go-finding](https://github.com/larsartmann/go-finding)** — Finding model, Detector interface, Pipeline (parallel execution, fix engine), output formats (SARIF/JSON/text/markdown)
- **[go-error-family](https://github.com/larsartmann/go-error-family)** — Error classification, exit codes
- **go-cqrs-lite AST patterns** — Built on the `cqrs-gen` scanning foundation

The linter writes only CQRS-specific rule logic (~50-100 lines per rule). All infrastructure (finding model, pipeline, output, fix engine) comes from go-finding.

## Library Functions

cqrs-lint also ships library functions that eliminate boilerplate at the source:

### event.Single()

```go
// Instead of writing a singleEvent helper in every project:
func decideCreate(s State, cmd CreateCmd) ([]event.Event, error) {
    return event.Single("user.created", cmd.AggregateID(), "User", s.Version.Increment(), UserCreated{Name: cmd.Name})
}
```

### decider.StrictApply()

```go
// Prevents silent data corruption from unknown event types:
d := decider.Decider[State]{
    Initial: State{},
    Apply: decider.StrictApply(fold, []event.Type{
        "user.created",
        "user.updated",
    }),
}
```

## Rule Development

Each rule is a `finding.Detector` implementation:

```go
func NewC006Detector(ctx *analyzer.AnalysisContext) finding.Detector {
    return finding.NamedDetectorFunc("C006-manual-version-arithmetic", func(_ context.Context) ([]finding.Finding, error) {
        // AST inspection logic
        // Return []finding.Finding
    })
}
```

Register it in `pkg/rules/register.go`. Write tests in `pkg/rules/<category>/rules_test.go` using `analyzer.BuildContextFromSource`.

## CI Integration

```yaml
# .github/workflows/cqrs-lint.yml
- name: Run cqrs-lint
  run: cqrs-lint --format sarif ./... > results.sarif
- uses: github/codeql-action/upload-sarif@v3
  with:
    sarif_file: results.sarif
```
