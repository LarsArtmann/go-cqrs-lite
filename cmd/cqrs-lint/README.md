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

## Feature Profiles

cqrs-lint auto-detects which go-cqrs-lite features your project uses and adapts
its rules accordingly. Run `cqrs-lint doctor` to see what was detected:

```bash
cqrs-lint doctor
```

You can override auto-detection in `.cqrs-lint.json`:

```json
{
	"features": {
		"store": "sqlite",
		"command-flow": "sync",
		"server": false,
		"soft-delete": true,
		"tracing": "off",
		"snapshot": "off"
	}
}
```

Each flag maps to a go-cqrs-lite module. Rules that depend on deployment
context (S002 encryption, S003 signing, A015 global mutable, A016 idempotency,
B014 OTel) consult these flags instead of guessing.

### Presets

For convenience, named presets set common flag combinations:

```json
{
	"preset": "local-cli"
}
```

| Preset       | Effect                                                                      |
| ------------ | --------------------------------------------------------------------------- |
| `local-cli`  | `server: false`, `tracing: off`                                             |
| `production` | `server: true`, `tracing: on`                                               |
| `library`    | `server: false`, `command-flow: read-only`, `tracing: off`, `snapshot: off` |
| `read-only`  | `command-flow: read-only`                                                   |

Explicit `features` flags always override preset values.

## Rule Count

**65 rules** across 6 categories: correctness (16), API misuse (19), boilerplate (15), consistency (5), architecture (7), security (3).

## Correctness Rules (bugs)

| ID   | Rule                             | Severity | Description                                                                                          |
| ---- | -------------------------------- | -------- | ---------------------------------------------------------------------------------------------------- |
| C001 | missing-tx-commit                | Critical | Transaction wrapper returns nil instead of tx.Commit()                                               |
| C002 | broken-command-id                | Critical | Command ID() returns zero value — breaks idempotency                                                 |
| C003 | silent-unknown-event-fold        | Error    | Fold function silently ignores unknown event types                                                   |
| C004 | checkpoint-before-async-complete | Error    | Projection launches async work — checkpoint may save early                                           |
| C005 | raw-json-unmarshal-payload       | Error    | Raw json.Unmarshal on event payload instead of DecodePayloadAuto                                     |
| C006 | manual-version-arithmetic        | Warning  | event.Version(x.Int()+1) instead of x.Increment()                                                    |
| C007 | time-now-in-decider              | Warning  | time.Now() inside decider — non-deterministic                                                        |
| C008 | float64-for-money                | Warning  | float64 field with monetary name — use decimal or cents                                              |
| C009 | panic-in-production              | Warning  | panic() in production code — use error returns                                                       |
| C010 | swallowed-error-in-fold          | Warning  | Error from decode/unmarshal discarded in fold                                                        |
| C011 | nondeterministic-decider         | Warning  | rand.* call inside decider — non-deterministic replay                                                |
| C012 | missing-error-return-in-with-tx  | Critical | withTx ignores body error — failures silently lost                                                   |
| C013 | time-time-in-event-payload       | Warning  | time.Time field in event payload loses timezone via CBOR epoch encoding                              |
| C014 | time-local-usage                 | Warning  | time.Local causes silent data corruption across timezone boundaries                                  |
| C015 | unchecked-close                  | Warning  | Close() error discarded — resource leak or silent data loss risk                                     |
| C016 | background-in-handler            | Warning  | context.Background()/TODO() in a handler with a ctx param — discards cancellation, timeouts, tracing |

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

| ID   | Rule                         | Severity | Description                                                                                                      |
| ---- | ---------------------------- | -------- | ---------------------------------------------------------------------------------------------------------------- |
| D001 | inconsistent-event-naming    | Info     | Mixed dot notation and PascalCase                                                                                |
| D002 | inconsistent-json-casing     | Info     | Mixed camelCase and snake_case JSON tags (excludes external-API mirrors — see [Rule Overrides](#rule-overrides)) |
| D003 | inconsistent-logging-library | Info     | Project mixes multiple logging libraries                                                                         |
| D005 | stale-documentation-version  | Warning  | Docs reference different version than go.mod                                                                     |
| D006 | missing-errorfamily          | Info     | errors.New or fmt.Errorf without %w bypasses the 6-family error taxonomy                                         |

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

| Flag                | Short | Default | Description                                                        |
| ------------------- | ----- | ------- | ------------------------------------------------------------------ |
| `--format`          | `-o`  | text    | Output format: text, json, sarif, markdown                         |
| `--min-severity`    |       | info    | Minimum severity: info, warning, error, critical                   |
| `--min-confidence`  |       | low     | Minimum confidence: low, medium, high                              |
| `--fix`             |       | false   | Apply auto-fixes                                                   |
| `--dry-run`         |       | false   | Show fixes without applying                                        |
| `--fast`            |       | false   | Run only Critical correctness rules                                |
| `--health-score`    |       | false   | Print the health score after findings                              |
| `--fp-suspects`     |       | false   | Show only low-confidence findings (likely FPs). Exit code always 0 |
| `--show-suppressed` |       | false   | Show suppressed findings with their suppression reason             |
| `--only`            |       |         | Filter by category or rule IDs (comma-separated)                   |
| `--exclude`         |       |         | Exclude paths (comma-separated)                                    |
| `--color`           |       | auto    | Colored output: auto, always, never                                |
| `--verbose`         |       | false   | Verbose output (module grouping, stats)                            |
| `--quiet`           | `-q`  | false   | Suppress non-finding output                                        |
| `--config`          | `-c`  |         | Path to config file                                                |

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

### Health Scoring

The `--health-score` flag computes a 0-100 score from findings. Two fairness
adjustments prevent heuristic noise from drowning real bugs:

- **Confidence weighting** — each finding's deduction is scaled by its confidence:

  | Confidence  | Multiplier | Example                  |
  | ----------- | ---------- | ------------------------ |
  | High / Full | 1.0 (100%) | Structural pattern match |
  | Medium      | 0.75 (75%) | Name + shape heuristic   |
  | Low         | 0.5 (50%)  | Coincidental field name  |
  | none        | 1.0 (100%) | Preserves prior behavior |

- **Severity deductions** — Critical: -10, Error: -5, Warning: -2, Info: -1
  (before confidence weighting).

- **Info cap** — total Info deductions are capped (default 20) so a chatty style
  rule can't outweigh a Critical correctness bug. Tunable via:

  ```json
  {
  	"health": {
  		"info-cap": 15
  	}
  }
  ```

  When the cap applies, `--verbose` output shows the raw vs capped deduction.

### Rule Overrides

Some rules read project-specific overrides from the `"rules"` key so you can
tell cqrs-lint about intentional patterns it would otherwise flag.

**D002 — external-API struct prefixes**

D002 flags files that mix `camelCase` and `snake_case` JSON tags. If your code
mirrors an external API (Discord, Stripe, GitHub) whose snake_case tags you
can't change, exclude those structs so they don't count toward the mix:

```json
{
	"rules": {
		"external-api-struct-prefixes": ["Discord", "Stripe", "GitHub"]
	}
}
```

Every struct whose name starts with a listed prefix is treated as an external
mirror. For one-off cases, use the in-source marker instead (see
[Suppression](#suppression)). Both mechanisms stack.

Run `cqrs-lint doctor` to confirm your overrides were loaded.

## Suppression

Inline suppression for false positives:

```go
//cqrs-lint:ignore(C007) wall-clock is domain logic
now := time.Now()
```

**D002 external-API marker** — excludes a single struct from the mixed-casing
check when its snake_case JSON tags mirror an external API:

```go
//cqrs-lint:external-api
type DiscordMessage struct {
	Content string `json:"content"`
	GuildID string `json:"guild_id"`
}
```

The marker works on both single `type Foo struct{}` declarations and grouped
`type ( ... )` blocks (place it on the struct's own doc line inside the group).
For bulk exclusion prefer the `rules.external-api-struct-prefixes` config above.

Suppressed findings are excluded from all output formats, the health score, and
the error-exit check. The lint run prints a count of suppressed findings to
stderr for visibility.

### Reviewing False Positives

Use `--fp-suspects` to surface only low-confidence findings — the ones most
likely to be false positives:

```bash
cqrs-lint --fp-suspects --path ./...
```

This filters to findings below Medium confidence and prints them with a
header explaining they are advisory. The exit code is always 0 in this mode,
making it safe to run in CI alongside the normal lint step. Review the
suspects and suppress confirmed false positives with `//cqrs-lint:ignore(RULE)`.

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

### GitHub Actions (SARIF upload)

```yaml
# .github/workflows/cqrs-lint.yml
name: cqrs-lint
on: [push, pull_request]
jobs:
  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: stable
      - name: Install cqrs-lint
        run: go install -tags "goexperiment.jsonv2" github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint@latest
      - name: Run cqrs-lint
        run: cqrs-lint --format sarif --path ./... > results.sarif
      - uses: github/codeql-action/upload-sarif@v3
        with:
          sarif_file: results.sarif
```

### Health-score gate

Fail CI when the score drops below a threshold:

```bash
cqrs-lint --health-score --path ./...
# Outputs just the numeric score — use in a script:
SCORE=$(cqrs-lint --health-score --path ./...)
[ "$SCORE" -ge 75 ] || exit 1
```

### Pre-commit hook

```bash
# .git/hooks/pre-commit
cqrs-lint --fast --path ./... || exit 1
```
