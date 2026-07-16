# cqrs-lint

A domain-aware linter for [go-cqrs-lite](https://github.com/larsartmann/go-cqrs-lite) consumers.

It finds bugs, API misuse, and boilerplate that generic linters cannot detect — because these are CQRS-specific patterns that require understanding the library's types, conventions, and architecture.

## Quickstart

```bash
# Install
go install github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint@latest

# Or build from source
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

# Filter by severity and confidence
cqrs-lint --min-severity error --min-confidence high ./...

# Apply auto-fixes (with dry-run preview first)
cqrs-lint --fix --dry-run ./...
cqrs-lint --fix ./...

# List available rules
cqrs-lint rules

# Config file support (auto-loaded from .cqrs-lint.json)
echo '{"min-severity":"warning","format":"json"}' > .cqrs-lint.json
cqrs-lint ./...
```

## Rule Count

**60 rules** across 6 categories: correctness (12), API misuse (19), boilerplate (15), consistency (4), architecture (7), security (3).

## Correctness Rules (bugs)

| ID   | Rule                             | Severity | Description                                                      |
| ---- | -------------------------------- | -------- | ---------------------------------------------------------------- |
| C001 | missing-tx-commit                | Critical | Transaction wrapper returns nil instead of tx.Commit()           |
| C002 | broken-command-id                | Critical | Command ID() returns zero value — breaks idempotency             |
| C003 | silent-unknown-event-fold        | Error    | Fold function silently ignores unknown event types               |
| C004 | checkpoint-before-async-complete | Error    | Projection launches async work — checkpoint may save early       |
| C005 | raw-json-unmarshal-payload       | Error    | Raw json.Unmarshal on event payload instead of DecodePayloadAuto |
| C006 | manual-version-arithmetic        | Warning  | event.Version(x.Int()+1) instead of x.Increment()                |
| C007 | time-now-in-decider              | Warning  | time.Now() inside decider — non-deterministic                    |
| C008 | float64-for-money                | Warning  | float64 field with monetary name — use decimal or cents          |
| C009 | panic-in-production              | Warning  | panic() in production code — use error returns                   |
| C010 | swallowed-error-in-fold          | Warning  | Error from decode/unmarshal discarded in fold                    |
| C011 | nondeterministic-decider         | Warning  | rand.* call inside decider — non-deterministic replay            |
| C012 | missing-error-return-in-with-tx  | Critical | withTx ignores body error — failures silently lost               |

## API Misuse Rules

| ID   | Rule                                        | Severity | Description                                              |
| ---- | ------------------------------------------- | -------- | -------------------------------------------------------- |
| A001 | manual-command-interface                    | Error    | Manual Type()/ID()/AggregateID() instead of BasicCommand |
| A002 | newevent-manual-marshal                     | Warning  | event.NewEvent with json.Marshal — use event.New         |
| A003 | explicit-codec-in-decode                    | Info     | Explicit codec — use DecodePayloadAuto                   |
| A004 | untyped-dispatch-register                   | Warning  | Type assertion in handler — use RegisterTyped            |
| A005 | custom-projection-runner                    | Warning  | Manual bus.SubscribeAll — use projectionhost             |
| A006 | adapter-layer-wrapping                      | Info     | WrapEvent/UnwrapEvent adapter methods                    |
| A007 | dual-model-oo-functional                    | Error    | Both OO aggregates and functional deciders               |
| A008 | parallel-type-system                        | Error    | Custom AggregateID/Version types duplicating library     |
| A009 | missing-stack-preset                        | Info     | No stack/ preset — manual wiring is error-prone          |
| A010 | custom-error-types                          | Warning  | Custom error interface duplicating go-error-family       |
| A011 | inconsistent-json-key-casing-event-payloads | Info     | Event payload structs with mixed JSON key casing         |
| A012 | missing-tombstone-handling                  | Info     | Fold function does not check for tombstone events        |
| A013 | pointer-vs-value-basic-command              | Info     | Embeds *BasicCommand (pointer) instead of value          |
| A014 | deprecated-api-usage                        | Warning  | Calls to deprecated APIs (event.NewEvent, Register)      |
| A015 | global-mutable-state                        | Error    | Global mutable variable — race condition risk            |
| A016 | missing-idempotency-middleware              | Warning  | Command dispatcher lacks idempotency middleware          |
| A017 | missing-snapshot-strategy                   | Info     | Repository without snapshot strategy — slow aggregates   |
| A018 | no-actual-event-sourcing                    | Info     | Imports go-cqrs-lite but never calls Save/Publish        |
| A019 | vendored-cqrs                               | Warning  | Vendored copy of go-cqrs-lite detected                   |

## Boilerplate Rules

| ID   | Rule                            | Severity | Description                                     |
| ---- | ------------------------------- | -------- | ----------------------------------------------- |
| B001 | single-event-helper             | Info     | Use event.Single() instead                      |
| B002 | manual-repository-wiring        | Info     | Use stack preset instead                        |
| B003 | subscribeall-large-switch       | Info     | Split into separate projections                 |
| B004 | command-constructor-boilerplate | Info     | Command with many fields — use cqrs-gen         |
| B005 | fold-switch-boilerplate         | Info     | Fold uses switch — consider decider.StrictApply |
| B006 | duplicate-fk-stub-sql           | Info     | Duplicated foreign-key SQL — centralize         |
| B007 | repeated-handler-registration   | Info     | 3+ consecutive registrations — table-driven     |
| B008 | manual-retry-implementation     | Warning  | Manual retry loop — use retry.Do                |
| B009 | emit-function-boilerplate       | Info     | Hand-written emit helper wrapping event.New     |
| B010 | catalog-event-list-boilerplate  | Info     | 3+ catalog.Event calls — use cqrs-gen           |
| B011 | must-marshal-helper             | Info     | mustMarshal helper — use event.New              |
| B012 | make-event-helper               | Info     | Hand-written makeEvent helper — use event.New   |
| B013 | missing-correlation-enricher    | Warning  | Repository without correlation enricher         |
| B014 | missing-otel-middleware         | Info     | Bus/dispatcher lacks OTel tracing               |
| B015 | missing-test-utilities          | Info     | Project has tests but no testutil imports       |

## Consistency Rules

| ID   | Rule                         | Severity | Description                                  |
| ---- | ---------------------------- | -------- | -------------------------------------------- |
| D001 | inconsistent-event-naming    | Info     | Mixed dot notation and PascalCase            |
| D002 | inconsistent-json-casing     | Info     | Mixed camelCase and snake_case JSON tags     |
| D003 | inconsistent-logging-library | Info     | Project mixes multiple logging libraries     |
| D005 | stale-documentation-version  | Warning  | Docs reference different version than go.mod |

## Architecture Rules

| ID   | Rule                     | Severity | Description                                |
| ---- | ------------------------ | -------- | ------------------------------------------ |
| E001 | layer-violation          | Error    | Tier-0 module imports Tier-3+ module       |
| E002 | circular-dependency      | Error    | Two modules import each other              |
| E003 | missing-module-boundary  | Warning  | All CQRS code in one package               |
| E004 | event-not-in-catalog     | Info     | Event type emitted but not in catalog      |
| E005 | command-without-handler  | Warning  | Command type defined but never registered  |
| E006 | event-without-projection | Info     | Event emitted but no projection handles it |
| E007 | query-without-handler    | Warning  | Query type defined but never registered    |

## Security Rules

| ID   | Rule                                      | Severity | Description                                      |
| ---- | ----------------------------------------- | -------- | ------------------------------------------------ |
| S001 | hardcoded-secrets                         | Critical | Potential hardcoded secret in string literal     |
| S002 | missing-encryption-for-sensitive-payloads | Error    | PII event payloads without encryption middleware |
| S003 | missing-event-signing                     | Warning  | Event store without signing middleware           |

## CLI

Built with [cmdguard](https://github.com/larsartmann/cmdguard) for type-safe flag parsing, config file support, and subcommands.

### Flags

| Flag               | Short | Default | Description                                      |
| ------------------ | ----- | ------- | ------------------------------------------------ |
| `--format`         | `-o`  | text    | Output format: text, json, sarif, markdown       |
| `--min-severity`   |       | info    | Minimum severity: info, warning, error, critical |
| `--min-confidence` |       | low     | Minimum confidence: low, medium, high            |
| `--fix`            |       | false   | Apply auto-fixes                                 |
| `--dry-run`        |       | false   | Show fixes without applying                      |
| `--fast`           |       | false   | Run only Critical correctness rules              |
| `--health-score`   |       | false   | Print the health score after findings            |
| `--only`           |       |         | Filter by category or rule IDs (comma-separated) |
| `--exclude`        |       |         | Exclude paths (comma-separated)                  |
| `--color`          |       | auto    | Colored output: auto, always, never              |
| `--verbose`        |       | false   | Verbose output (module grouping, stats)          |
| `--quiet`          | `-q`  | false   | Suppress non-finding output                      |
| `--config`         | `-c`  |         | Path to config file                              |

### Config File

A `.cqrs-lint.json` file in the project root is auto-loaded:

```json
{
  "format": "json",
  "min-severity": "warning",
  "min-confidence": "medium",
  "fast": false
}
```

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

C001 (missing tx commit), C003 (silent fold), and C006 (manual version arithmetic) are auto-fixable.

## Architecture

Built on three foundation libraries:

- **[go-finding](https://github.com/larsartmann/go-finding)** — Finding model, Detector interface, Pipeline (parallel execution, fix engine), output formats (SARIF/JSON/text/markdown)
- **[cmdguard](https://github.com/larsartmann/cmdguard)** — Type-safe CLI with struct-tag flags, config file loading, subcommands
- **go-cqrs-lite AST patterns** — CQRS-specific AST scanning

## Library Functions

cqrs-lint also ships library functions that eliminate boilerplate at the source:

### event.Single()

```go
return event.Single("user.created", cmd.AggregateID(), "User", s.Version.Increment(), UserCreated{Name: cmd.Name})
```

### decider.StrictApply()

```go
d := decider.Decider[State]{
    Initial: State{},
    Apply: decider.StrictApply(fold, []event.Type{"user.created", "user.updated"}),
}
```

## Rule Development

Each rule is a `finding.Detector` in its own file (`c001.go`, `a001.go`, etc.):

```go
func NewC006Detector(ctx *analyzer.AnalysisContext) finding.Detector {
    return finding.NamedDetectorFunc("C006-manual-version-arithmetic", func(_ context.Context) ([]finding.Finding, error) {
        // AST inspection logic
    })
}
```

Register it in `pkg/rules/register.go`. Write tests using `analyzer.BuildContextFromSource`.

## CI Integration

```yaml
# .github/workflows/cqrs-lint.yml
- name: Run cqrs-lint
  run: cqrs-lint --format sarif ./... > results.sarif
- uses: github/codeql-action/upload-sarif@v3
  with:
    sarif_file: results.sarif
```
