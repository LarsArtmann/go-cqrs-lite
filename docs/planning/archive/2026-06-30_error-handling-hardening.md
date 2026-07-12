# Error-Handling Hardening Plan

Point-in-time execution plan derived from a brutal self-review of the error-handling audit.
Goal: make the 5-family taxonomy actually hold across every module + boundary, so consumers
get correct retry decisions and recoverable typed errors end-to-end.

Sorted by **impact / effort**. Each task is a self-contained commit (≤ ~12 min).

## Tier S — Classification adoption (fixes the retry-correctness bug) [1% → 51%]

The retry middleware defaults unclassified errors to `Transient` (retryable). Every plain
`errors.New` sentinel below is therefore retried even when it represents a non-retryable
Rejection/Corruption. Converting them to `event.NewRejection` etc. fixes this and yields
machine-readable codes for transport/DLQ.

| ID  | Task                                                        | Impact | Effort |
| --- | ----------------------------------------------------------- | ------ | ------ |
| S1  | `graph/errors.go`: classify + export sentinels (~27)        | High   | M      |
| S2  | `storage/relational/errors.go`: classify + export (~13)     | High   | S      |
| S3  | `storage/view/options.go`: classify + export (~8)           | High   | S      |
| S4  | `stack/errors.go`: classify exported sentinels (~9)         | High   | S      |
| S5  | `projectionhost` inline `errors.New` → `event.NewRejection` | Med    | S      |
| S6  | `transport/grpc/client.go`: classify sentinels (~4)         | Med    | S      |
| S7  | `deriver/deriver.go`: classify 1 sentinel                   | Low    | XS     |
| S8  | `middleware/errors.go` `ErrMeterRequired`: classify         | Low    | XS     |
| S9  | `otel/setup.go`: classify + `errors.Join` (was `%v`)        | Med    | S      |

## Tier A — Correctness & modernization wins [4% → 64%]

| ID  | Task                                                                    | Impact | Effort |
| --- | ----------------------------------------------------------------------- | ------ | ------ |
| A1  | Find & fix the broken `errors.Is` chain (`run_projections_test.go:194`) | High   | S      |
| A2  | `storage/sql/duplicate.go`: `errors.As` → `errors.AsType[T]` (3 sites)  | Low    | XS     |
| A3  | `storage/turso/indexing/auto.go:319` string-match → typed/documented    | Med    | S      |
| A4  | `storage/turso/indexing/optimizations.go:125` string-match → typed      | Med    | S      |
| A5  | `storage/turso/indexing/hooks.go:102,120` swallowed hook errors         | Med    | S      |
| A6  | `id/errors.go`: export `errEmptyString`                                 | Low    | XS     |
| A7  | `storage/sql/duplicate.go`: cover string fallback with a test           | Low    | S      |

## Tier B — Boundary fidelity (typed errors survive transport + DLQ) [20% → 80%]

| ID  | Task                                                                         | Impact | Effort |
| --- | ---------------------------------------------------------------------------- | ------ | ------ |
| B1  | gRPC proto: add optional `error_code`/`error_family` (backward-compatible)   | High   | M      |
| B2  | gRPC servers: populate structured error from `event.Classify`/`*event.Error` | High   | S      |
| B3  | gRPC client: reconstruct typed error from structured fields                  | High   | S      |
| B4  | projectionhost DLQ: add `ErrorCode`/`ErrorFamily` (keep `Error string`)      | Med    | S      |
| B5  | middleware DLQ SQL: add `error_code`/`error_family` columns                  | Med    | M      |
| B6  | example/todo: map family → HTTP status; stop leaking `err.Error()`           | High   | S      |

## Tier C — Polish, consistency, docs, tests

| ID  | Task                                                                      | Impact | Effort |
| --- | ------------------------------------------------------------------------- | ------ | ------ |
| C1  | `catalog/simple/builder.go`: `Build()` panic → error (or remove dual API) | Med    | S      |
| C2  | `storage/view/auto.go`: `AutoMapper` panic → error                        | Med    | S      |
| C3  | `catalog` `ValidationError` split-brain → unify with taxonomy             | Med    | M      |
| C4  | Update `docs/error-taxonomy.md` with newly-classified modules             | Med    | S      |
| C5  | Add retry-classification tests for key sentinels                          | High   | M      |
| C6  | Convert key BDD assertions `ContainSubstring` → `errors.Is`               | Med    | M      |

## Guardrails

- **Library, not app** — prefer _adding_ optional fields over breaking existing public types.
- Reuse `event.NewRejection/NewConflict/NewTransient/NewCorruption/NewInfrastructure` +
  `event.Classify` / `event.IsRetryable`. No new dependencies (everything already present).
- Codes are namespaced: `module.subdomain.error` (per `docs/error-taxonomy.md`).
- Run `nix run .#build` + targeted `go test` after each commit.
