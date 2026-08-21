# go-cqrs-lite — Consumer Feedback: otel/v4 Adoption

**Consumer:** [bank-sync](https://github.com/LarsArtmann/bank-sync) — bank transaction sync CLI + dashboard (Wise API + Qonto CSV → SQLite)
**SDK version:** otel/v4 v4.3.0
**Date:** 2026-08-21
**Severity:** n/a — positive adoption report; one observation for the OTLP story
**Status:** Verified against bank-sync source (`cmd/bank-sync/tracing.go`, `internal/cqrs/infrastructure.go`, `internal/server/patchstream.go`).

---

## TL;DR

bank-sync runs otel/v4's `Setup()` in EVERY process — while production tracing is
**noop by design** (no OTLP endpoint exists yet). That is only possible because
the module made noop the cheap path: one `Setup` call wires naming, propagation
and CQRS-tuned views, and skipping the exporter entirely costs nothing. The
adoption is now load-bearing in two places beyond the decider: the OTel
middleware bundle on dispatch, and hand-instrumented spans around the dashboard's
patch-stream pump.

---

## 1. The adoption shape (verified against source)

**Tracer bootstrap** — a single Setup call, stdout exporter only when debugging:

```go
// cmd/bank-sync/tracing.go
provider, err := cqrsotel.Setup(
    cqrsotel.WithService("bank-sync", "", ""),
    cqrsotel.WithStdoutExporter(os.Stderr),
)
```

When `OTEL_TRACES_EXPORTER != "stdout"`, bank-sync never calls `Setup` at all —
the otel-default noop provider takes over, spans are created and discarded at
zero cost. This "instrument everywhere, pay only when observing" shape is why
the wiring shipped into production before any collector existed.

**Middleware bundle** — dispatch and query paths are wrapped once:

```go
// internal/cqrs/infrastructure.go:89
otelBundle, err := middleware.NewOTelBundle(...)
```

**Hand instrumentation** — the patch-stream pump (the dashboard's live-update
core, ~200 lines of coalescing/flush logic that no SDK layer knows about) opens
its own span per flush:

```go
// internal/server/patchstream.go:327
ctx, span := tracer.Start(ctx, "patchstream.flush")
```

With `OTEL_TRACES_EXPORTER=stdout` this makes pump latency observable without
any collector — it directly measured the 250 ms debounce / 2 s max-delay
behaviour during the Datastar adoption work.

## 2. What worked well

1. **noop-first economics.** The decision to adopt was made *before* having an
   endpoint to ship spans to — possible only because unobserved spans are free.
   Consumers with "OTLP someday" roadmaps (most small services) get the
   instrumentation now and the observability later, with no re-instrumentation.
2. **`WithSpanExporter` takes an interface, not a config dialect.** bank-sync
   passes `WithStdoutExporter(os.Stderr)` today and can pass an OTLP exporter
   tomorrow without touching Setup structure — the standard
   `sdktrace.SpanExporter` type is the whole contract.
3. **Consistent span naming for free.** The decider/middleware spans coming out
   of `Setup` matched what `middleware.NewOTelBundle` emits — one service
   attribute, no per-layer naming to reconcile.

## 3. Observation for the OTLP story (not a blocker)

When bank-sync eventually adopts a real collector (currently a user-gated
decision — the deployment is a single-user LAN service), the likely gap is
**exporter lifecycle documentation**: `Setup` returns a `*Provider` whose
`Shutdown` must flush buffered spans before exit, and the correct pattern for
"stdout today, OTLP tomorrow, noop in tests" deserves a doc example. The API
surface already supports it (`WithSpanExporter`, `WithoutGlobalRegistration` for
parallel tests); nothing needs to change — a recipe in the module README would
save each consumer the source-archaeology.

Related consumer record: bank-sync explicitly decided **against** wrapping HTTP
handlers in otelhttp spans while the tracer is noop (spans nobody collects are
pure middleware overhead) — tracked in bank-sync's gated-adoption watchlist. The
otel/v4 core adoption was NOT deferred: only the HTTP-side wrapper was. That
distinction — core adoption cheap, edge instrumentation deferrable — is the
pattern worth knowing when advising other consumers on adoption sequencing.
