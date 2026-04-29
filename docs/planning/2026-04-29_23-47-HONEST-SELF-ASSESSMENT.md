# Honest Self-Assessment and Execution Plan

**Date:** 2026-04-29  
**Session:** Session 10 Post-Mortem + Fixes

---

## What I Forgot / Messed Up

1. **`go mod tidy` is BROKEN in 3 modules** — `catalog`, `middleware`, and `testhelpers` all fail `go mod tidy` because `core` has test dependencies on `memory` and `testhelpers` that propagate transitively. No replace directives in those modules = resolution failure. **This is a real build hygiene bug.**

2. **Tracing middleware creates a new `Tracer()` on EVERY call** — `tracerProvider.Tracer(instrumentationName)` is called on every single handler invocation. Tracers are meant to be cached. This is a performance bug.

3. **Tracing tests can't run in parallel** — The global `tracerProvider` variable forces sequential test execution. This reflects the global state code smell.

4. **`middleware/go.mod` has test-only dependencies as indirect** — `go.opentelemetry.io/otel/sdk` is pulled in as an indirect dependency because the test file imports it. The production code doesn't need it.

5. **No structured logging (slog) integration** — We have a custom `Logger` interface in `middleware/logging.go` but no `log/slog` (Go 1.21+ stdlib) adapter. Users want structured logging out of the box.

6. **TODO_LIST.md is stale** — Still shows completed tasks as "not started."

7. **LSP cache has false positives** — Deleted `core/internal/testhelpers` and `xtypes` still show up in LSP errors.

---

## What I Could Have Done Better

1. Should have run `go mod tidy` in every module after adding OTel dependencies
2. Should have benchmarked the tracing middleware before committing
3. Should have made tracer provider injectable per-instance instead of global
4. Should have updated TODO_LIST.md immediately after completing tasks

---

## Execution Plan

| Step | Task | Effort | Impact | Rationale |
|------|------|--------|--------|-----------|
| 1 | Fix `go mod tidy` in all modules | ~15m | HIGH | Blocks independent module maintenance |
| 2 | Cache tracer in tracing middleware | ~15m | HIGH | Performance bug — Tracer() every call |
| 3 | Make tracer injectable (not global) | ~20m | MEDIUM | Enables parallel tests, better architecture |
| 4 | Add slog structured logging middleware | ~30m | MEDIUM | Standard library integration |
| 5 | Update TODO_LIST.md | ~5m | LOW | Documentation accuracy |
| 6 | Verify everything | ~10m | — | Tests, lint, build, flake check |

**Total: ~95 minutes**

---

## Key Decisions

### Tracer caching approach
Instead of global `tracerProvider`, pass `trace.Tracer` directly to middleware constructors:
```go
func CommandTracing(tracer trace.Tracer) command.Middleware
```

This eliminates global state, enables parallel tests, and avoids `Tracer()` call overhead.

### go mod tidy fix approach
Add `replace` directives for `memory` and `testhelpers` to `catalog/go.mod`, `middleware/go.mod`, and `testhelpers/go.mod`. This is the pragmatic fix. The long-term fix (moving test files to integration/) is documented in AGENTS.md and can be done later.

### slog middleware approach
Create `middleware/slog.go` with `CommandSlog`, `EventSlog`, `QuerySlog` that use `log/slog` from stdlib. Follow the same pattern as existing logging middleware but with structured logging.
