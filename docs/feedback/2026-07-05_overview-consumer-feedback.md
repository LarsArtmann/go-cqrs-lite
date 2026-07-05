# go-cqrs-lite — SDK Feedback from Overview

**Consumer:** [Overview](https://github.com/larsartmann/overview) — local project dashboard
**Date:** 2026-07-05
**Version used:** v3.5.0 (transitive — all modules indirect via `cqrs-htmx/v4`)
**Session:** Indirect consumer through cqrs-htmx middleware/SSE stack; direct user of `go-error-family`

---

## Context: Overview is an indirect consumer

Overview does **not** import any `go-cqrs-lite/*/v3` module directly. All 12 go-cqrs-lite modules in `go.mod` are indirect, pulled in by `cqrs-htmx/v4`:

```
github.com/larsartmann/go-cqrs-lite/codec/v3 v3.5.0 // indirect
github.com/larsartmann/go-cqrs-lite/command/v3 v3.5.0 // indirect
github.com/larsartmann/go-cqrs-lite/event/v3 v3.5.0 // indirect
github.com/larsartmann/go-cqrs-lite/id/v3 v3.5.0 // indirect
...
```

Overview uses cqrs-htmx only for its **middleware/SSE/embedded-asset surface** — no CQRS dispatch, no event sourcing, no projections. This feedback is therefore about the **transitive dependency experience** and the adjacent `go-error-family` library (which shares design DNA).

---

## What worked superbly

### 1. `event/v3` error families — inherited via `go-error-family`

Overview uses `go-error-family` directly (which is the standalone evolution of go-cqrs-lite's error family system). The five-family model (Rejection, Conflict, Transient, Infrastructure, Corruption) is the right taxonomy for HTTP error classification:

```go
errorfamily.WrapInfrastructure(err, codeServerCreate, "create HTTP server")
errorfamily.WrapTransient(err, codeDiscovery, "sdk discover")
```

`errorfamily.Classify(err).String()` powers structured logging (`family` field in slog attrs) and the `errorpage.FromError()` family-colored rendering. This error vocabulary is the backbone of Overview's error UX.

### 2. `id/v3` — ULID-based request IDs (via cqrs-htmx)

`ContextEnrichmentMiddleware` uses go-cqrs-lite's `id` package to generate ULID request IDs. These flow through the entire middleware chain and appear in structured logs. The branded type system (`id.Of[T]`) prevents mixing request IDs with other ID types at compile time — even though Overview only uses one ID type, the type safety is architecturally sound.

### 3. Minimal transitive footprint for non-CQRS consumers

Despite pulling in 12 indirect modules, the actual compile-time surface for Overview is just: `id` (request IDs), `event` (error families via go-error-family), and `codec` (indirect via event). The CQRS dispatch, projection, storage, and schema modules don't leak into the consumer's API surface. This is good module isolation.

---

## Pain points

### 1. `go-error-family` vs `event/v3` error constructors — unclear relationship

**Severity:** Medium (architecture clarity)

Overview uses `go-error-family` for error construction, but `cqrs-htmx`'s skill says to use `event.New*`/`event.Wrap*` from `go-cqrs-lite/event/v3` (not `fmt.Errorf`). These appear to be the same five families (Rejection, Conflict, Transient, Infrastructure, Corruption) with potentially different APIs.

It's unclear:

- Is `go-error-family` a standalone extraction of `event/v3`'s error system?
- Should non-CQRS apps use `go-error-family` or `event/v3` for errors?
- Are they interchangeable? Can you `Classify()` an error created by `event.NewRejection()`?

Overview chose `go-error-family` because it doesn't need event payloads or codec integration — just the error taxonomy. This works, but the relationship between the two packages should be documented in both places.

**Suggestion:** Add a section to `go-error-family` README: "Relationship to go-cqrs-lite/event/v3" — explain that go-error-family is the standalone error classification, and event/v3 wraps the same families with event-store context. Non-CQRS consumers should use go-error-family.

### 2. Indirect dependency weight — 12 modules for middleware-only usage

**Severity:** Low (dependency hygiene)

Pulling in `cqrs-htmx/v4` for just middleware + SSE + embedded assets transitively brings all 12 go-cqrs-lite modules into `go.mod`:

```
codec/v3, command/v3, dispatcher/v3, event/v3, id/v3, idempotency/v3,
kv/v3, otel/v3, query/v3, transport/http/v3
```

None of these are imported by Overview's code. The dependency weight is cosmetic (Go modules don't compile unused imports), but it inflates `go.mod` and makes `go mod tidy` surface confusing.

This is likely unavoidable given cqrs-htmx's architecture (it genuinely depends on go-cqrs-lite). If cqrs-htmx ever splits its middleware/SSE surface into a lighter sub-module that doesn't depend on the CQRS core, that would reduce the transitive footprint for read-only consumers.

### 3. `id/v3` branded types require `.String()` conversion at boundaries

**Severity:** Low (ergonomic)

`cqrshtmx.RequestIDFromContext(ctx)` returns a branded `id` type. Overview wraps it:

```go
func requestIDFromContext(ctx context.Context) string {
    return cqrshtmx.RequestIDFromContext(ctx).String()
}
```

This `.String()` conversion appears in every log call site. The branded type is correct for compile-time safety, but a `requestIDString(ctx) string` convenience helper in cqrs-htmx would reduce boilerplate for the most common consumer pattern (logging).

---

## Summary

As a **transitive consumer**, Overview's experience with go-cqrs-lite is mostly invisible — and that's a good thing. The module isolation works; the CQRS machinery doesn't leak into the middleware/SSE surface.

The most impactful improvement is **clarifying the `go-error-family` ↔ `event/v3` relationship** — Overview chose go-error-family because it's standalone, but the lack of documentation about when to use which creates uncertainty for every non-CQRS consumer of cqrs-htmx.

The error family taxonomy itself (Rejection/Conflict/Transient/Infrastructure/Corruption) is excellent and powers Overview's entire error UX — from structured logging to family-colored error pages. This is the most valuable design exported by the go-cqrs-lite ecosystem for read-only apps.
