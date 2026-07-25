# ADR-0064: Extract `retry/` into Standalone `go-retry` Repository

| Field      | Value                                                                       |
| ---------- | --------------------------------------------------------------------------- |
| Status     | Proposed                                                                    |
| Date       | 2026-07-25                                                                  |
| Deciders   | Lars Artmann                                                                |
| Related    | ADR-0046 (seven-tier model), ROADMAP (module extraction), `retry/README.md` |
| Supersedes | —                                                                           |

## Context

The `retry/` module is a **zero-coupling** package: it depends only on
`go-error-family` and the standard library. It provides exponential backoff with
jitter, classified errors, and a simple `Do(ctx, config, fn)` API. No CQRS, event
sourcing, or storage concepts appear anywhere in its 217 lines of source code.

Currently `retry/v4` lives inside the `go-cqrs-lite` monorepo. Its single
internal consumer is `middleware/retry.go`, which wraps it as
`middleware.Retry()` for command/event/query pipelines. External consumers who
need retry logic but not CQRS must import the entire `go-cqrs-lite` module graph
to get a 217-line backoff helper.

The ROADMAP calls for extracting zero-coupling modules into standalone repos so
they can be consumed independently.

## Current State

```
retry/
├── doc.go          (9 lines — package doc)
├── retry.go        (125 lines — Do, Backoff, ComputeDelay, ErrExhausted, ErrCanceled)
├── config.go       (83 lines — Config struct, DefaultConfig, Validate)
└── retry_test.go   (351 lines — 14 test functions)
```

Total: 217 source + 351 test = 568 lines.

### Dependencies

| Dependency        | Type | Notes                                                 |
| ----------------- | ---- | ----------------------------------------------------- |
| `go-error-family` | Prod | `ErrExhausted`, `ErrCanceled` use `NewInfrastructure` |
| `onsi/ginkgo/v2`  | Test | BDD test framework                                    |
| `onsi/gomega`     | Test | Match library                                         |

### Public API (must be preserved)

| Symbol                                               | Type     |
| ---------------------------------------------------- | -------- |
| `Do(ctx, config, fn) error`                          | Function |
| `Backoff(config, attempt) Duration`                  | Function |
| `ComputeDelay(initial, max, mult, attempt) Duration` | Function |
| `DefaultConfig() Config`                             | Function |
| `Config.Validate() error`                            | Method   |
| `Config` struct                                      | Type     |
| `AttemptFunc` type                                   | Type     |
| `ErrExhausted`                                       | Sentinel |
| `ErrCanceled`                                        | Sentinel |

### Internal Consumers

| Consumer      | File                     | Import alias          |
| ------------- | ------------------------ | --------------------- |
| `middleware/` | `middleware/retry.go:15` | `retrypkg "retry/v4"` |

Single consumer. The `middleware` package wraps `retry.Do` as
`middleware.Retry()` and `middleware.RetryWithConfig()`.

## Decision

**Extract `retry/` into a standalone `github.com/larsartmann/go-retry` repository
and re-export from `go-cqrs-lite/retry/` for backward compatibility.**

### Extraction Plan

#### Phase 1: Create the standalone repo

1. Create `github.com/larsartmann/go-retry` repository
2. Copy the 4 source files verbatim (no API changes)
3. Set module path to `github.com/larsartmann/go-retry`
4. Run `go mod tidy` — only dep is `go-error-family`
5. Tag `v1.0.0` (stable API, zero breaking changes planned)
6. Verify `go list -m github.com/larsartmann/go-retry@v1.0.0` fetches

#### Phase 2: Re-export alias in go-cqrs-lite

Replace the contents of `retry/retry.go`, `retry/config.go`, and `retry/doc.go`
with re-export type aliases so existing import paths keep working:

```go
// Package retry re-exports github.com/larsartmann/go-retry for backward compat.
// New consumers should import go-retry directly.
package retry

import (
    goretry "github.com/larsartmann/go-retry"
)

type Config = goretry.Config
type AttemptFunc = goretry.AttemptFunc

var (
    ErrExhausted = goretry.ErrExhausted
    ErrCanceled  = goretry.ErrCanceled
)

func Do(ctx context.Context, config Config, fn AttemptFunc) error {
    return goretry.Do(ctx, config, fn)
}

func Backoff(config Config, attempt int) time.Duration {
    return goretry.Backoff(config, attempt)
}

func ComputeDelay(initial, max time.Duration, multiplier float64, attempt int) time.Duration {
    return goretry.ComputeDelay(initial, max, multiplier, attempt)
}

func DefaultConfig() Config {
    return goretry.DefaultConfig()
}
```

The `retry/go.mod` changes from:

```
module github.com/larsartmann/go-cqrs-lite/retry/v4
```

to:

```
module github.com/larsartmann/go-cqrs-lite/retry/v4
require github.com/larsartmann/go-retry v1.0.0
```

#### Phase 3: Update internal consumer

`middleware/go.mod` already requires `retry/v4` — the version doesn't change
(the re-export alias keeps the same import path). No consumer code changes needed.

### Tagging

| Repo                    | Tag      | Notes                                |
| ----------------------- | -------- | ------------------------------------ |
| `go-retry`              | `v1.0.0` | Stable, SemVer-compliant             |
| `go-cqrs-lite/retry/v4` | `v4.2.0` | Minor bump (re-export, no API break) |

## Alternatives Considered

### A. Keep retry/ in the monorepo forever

**Rejected.** The ROADMAP explicitly calls for extraction. The module has zero
CQRS coupling — keeping it in `go-cqrs-lite` forces non-CQRS consumers to import
a massive module graph for a 217-line helper. It also inflates the monorepo's
module count (currently 57 `go.mod` files) without architectural benefit.

### B. Hard-replace (delete retry/, point consumers to go-retry)

**Rejected.** Breaking the `retry/v4` import path would force every consumer to
update their imports. The re-export alias pattern (used by `sync/atomic` →
`sync/atomic` type aliases in Go stdlib) provides zero-friction migration. New
consumers import `go-retry` directly; existing consumers update at their leisure.

### C. Extract without re-export, bump go-cqrs-lite major version

**Rejected.** A major version bump across 57 modules for one 217-line extraction
is disproportionate. The re-export alias costs nothing and preserves SemVer
stability.

## Consequences

**Positive:**

- `go-retry` becomes independently consumable by non-CQRS projects
- Monorepo module count decreases by 1 (if re-export alias replaces source)
- Clear ownership boundary: retry logic is its own project
- `go-retry` can evolve its own release cadence independent of CQRS

**Negative:**

- Two repos to maintain instead of one (mitigated by re-export alias)
- `go-cqrs-lite/retry/` becomes a thin wrapper (217 LOC → ~40 LOC of aliases)
- Slight indirection cost: consumers reading `retry.Do` must trace through alias

**Neutral:**

- `middleware` package continues to work unchanged (re-export preserves types)
- CI continues testing `retry/v4` (the alias module is in testModules)

## Open Questions

1. **Should `go-retry` adopt `go-error-family` or use plain `errors.New`?**
   The current code uses `errorfamily.NewInfrastructure` for classified errors.
   Keeping this dependency means `go-retry` depends on `go-error-family`. This is
   acceptable (it's a small, stable lib), but alternative is to use plain
   `errors.New` and let consumers classify if needed. **Recommendation:** keep
   `go-error-family` — error classification is valuable and the dep is tiny.

2. **When to execute?** This ADR is the design step (M14). Execution requires
   creating the new repo, which is a manual step outside this codebase.
