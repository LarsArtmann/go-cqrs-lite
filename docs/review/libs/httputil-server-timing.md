# httputil/server_timing — Utilization Audit · go-cqrs-lite

> Audited 2026-11-22 · Library `github.com/larsartmann/httputil/server_timing` @ 34dbf05 (repo HEAD; latest module tag v1.0.3) · Consumer go-cqrs-lite @ feec9bc74 · Group G2

**Verdict:** UNDERUSED-MODERATE
**Score:** 2/5 capabilities fully leveraged = 40% (low)

The canonical Server-Timing loop is adopted end-to-end and for real: the
example app's production HTTP server wraps its mux in
`ServerTimingMiddleware()` (outermost, so `total` covers everything), the
query handler records a named `query` phase via `MeasureServerTiming` with
the stop-before-write contract respected, and the tests assert both phases
byte-for-byte on the wire. Recipe §2.47's credit of server_timing is **real
code usage**, not doc-only — the recipe helper contains the same
`MeasureServerTiming` call and is compile-gated. The underuse is in the
capability fringe that matters in production: timing is exposed
unconditionally (no gating), phases carry no descriptions, and the repo's
SSE surface — where the module's own README documents a defer-idiom pattern —
measures nothing. Version pin is current at v1.0.3.

## 1. What the library is

W3C Server-Timing header instrumentation as a zero-dependency, stdlib-only
module (package `servertiming`), separate from the root httputil module so it
can be adopted without the middleware dependency set
(httputil/server_timing/README.md:1-20). Surface:

- `ServerTimingMiddleware()` — wraps the writer, stores a concurrency-safe
  collector in the request context, finalizes the auto `total` metric (time
  to first byte) and stamps the header exactly once at first write
  (httputil/server_timing/server_timing.go:400).
- `ServerTimingMiddlewareWhen(pred)` — identical, but only for matching
  requests; non-matching requests pass through unwrapped. The README's
  security note: "Server-Timing can leak internal performance details — gate
  it in production."
- `MeasureServerTiming(ctx, name) func()` — nil-safe timed region for
  handlers (server_timing.go:258); `RecordServerTiming(ctx, name, desc, dur)`
  for already-measured durations (server_timing.go:237); `Measure`/
  `MeasureWithDesc` collector methods for descriptions
  (server_timing.go:94,105).
- `ServerTimingFromContext`, `WrapServerTiming(w, r)`,
  `NewServerTiming()` + `WithServerTiming` for frameworks that own the chain
  (server_timing.go:230,369,55,222).
- Injection-safe header values (RFC 7230 token sanitization, fuzz-tested),
  fractional-millisecond precision, transparent `Flush`/`Hijack`/`Push`/
  `Unwrap` delegation so SSE and HTTP/2 keep working.

## 2. Usage census in go-cqrs-lite (evidence)

One module pins it — the same module as root httputil:

- `example/taskmanager/go.mod` — `github.com/larsartmann/httputil/server_timing
  v1.0.3` direct (alongside `httputil v1.8.0`).

Four call sites, all real:

- `example/taskmanager/setup.go:275` — production wiring:
  `Handler: servertiming.ServerTimingMiddleware()(s.routes())` in
  `StartHTTP`, deliberately outermost ("Timing middleware outermost so
  `total` covers every phase inside", example/taskmanager/setup.go:272-274).
- `example/taskmanager/httpx.go:45` — `stop :=
  servertiming.MeasureServerTiming(r.Context(), "query")` inside the §2.47
  query-binding helper, with the stop-before-write contract honored and
  documented: `stop() // stop BEFORE any write: the header is injected at
  first Write` (httpx.go:48).
- `example/taskmanager/query_http_test.go:33` — `queryServer` wraps the mux
  "exactly like Server.StartHTTP does, so the Server-Timing header lands on
  every response"; `:215` a second test server wraps a bare mux the same way.
- `example/taskmanager/query_http_test.go:103-108` — wire assertions: the
  header must contain both `query;` (the helper's phase) and `total;` (the
  middleware's). **The header is emitted on real responses and pinned.**

Recipe tier:

- `.agents/skills/go-cqrs-lite/references/recipes.md:3433` credits
  `httputil/server_timing` ("Server-Timing: query;dur=… on every response,
  visible in browser devtools") in §2.47's job table; the helper fence calls
  `servertiming.MeasureServerTiming(r.Context(), "query")` with the same
  stop-before-write comment (`recipes.md:3537-3539`); import at
  `recipes.md:3514`. Compile-gated at
  `cmd/doc-check/v4/recipes_catalog_meta3.go:288,302,315,322,334,348,355`
  against `server_timing v1.0.2` (`recipes_compile_test.go:25`).

So the answer to "real code or doc-only": **real** — example production
wiring + test assertions + a compile-gated recipe that carries the same call.

## 3. Capability inventory

| # | Capability | Grade | Evidence / reason |
|---|---|---|---|
| 1 | Middleware + auto `total` (TTFB) | **Full** | example/taskmanager/setup.go:275 (outermost); `total;` asserted example/taskmanager/query_http_test.go:107 |
| 2 | Named handler phases (`MeasureServerTiming`) | **Full** | httpx.go:45 with stop-before-write respected (httpx.go:48); `query;` asserted query_http_test.go:105 |
| 3 | Conditional exposure (`ServerTimingMiddlewareWhen`) | Missed | timing is emitted unconditionally on every response including health checks; the module's own README says to gate it in production (server_timing/README.md "Features" bullet 3) |
| 4 | Phase descriptions (`RecordServerTiming`/`MeasureWithDesc`) | Missed | the query phase ships bare (`query;dur=…`) with no `desc=` — one `MeasureWithDesc` call would make DevTools output self-documenting |
| 5 | SSE/streaming phase timing (defer idiom) | Missed | `metaengine.ServeSSE` measures nothing; its replay phase (metaengine/v4/sse.go:244-285) is exactly the slow part a "replay" metric would expose, and the module README documents the defer idiom as correct for SSE (header stamped at stream start) |
| 6 | Direct collector access (`ServerTimingFromContext`) | N/A | no consumer needs collector-level control today — flip trigger: recording a phase from a goroutine the handler spawns |
| 7 | `WrapServerTiming` per-writer wrapping | N/A | the app owns a plain middleware chain; nothing framework-owned to hook into |
| 8 | Manual collector lifetime (`NewServerTiming` + `WithServerTiming`) | N/A | full-lifetime control for exotic chains; no such chain exists here |

Score derivation: rows N=8, N/A rows 6–8 → **A=5**. Full = 2 × 1.0,
Partial = 0, Missed = 3 × 0 (rows 3, 4, 5).
Score = round(100 × 2.0 / 5) = **40% → low band**.

## 4. Gap analysis

1. **Ungated timing in the golden path.** The example that fleet apps copy
   emits `Server-Timing` to every client, forever. The module ships the
   fix (`ServerTimingMiddlewareWhen`) and the guidance; the recipe and
   example just don't use it. Since §2.47 is the copy-me template, the
   fleet inherits unconditional exposure. One `When(func(r) bool { ... })`
   predicate (debug flag, admin route) in the example + one sentence in the
   recipe closes it.
2. **Replay phase is a blind spot at the worst place.** ServeSSE's
   reconnection path (`metaengine/v4/sse.go:244-285`) can replay hundreds of
   journal events before live streaming — the exact "why is this request
   slow" moment Server-Timing exists for, on the repo's flagship streaming
   surface, unmeasured. The module README's SSE note says the defer idiom is
   correct there precisely because the header commits at stream start.
   metaengine taking a `server_timing` dependency is a budget decision
   (stdlib-only module, zero transitive deps — the cheapest possible), but a
   recipe-level variant ("time your SSE replay") costs nothing in the
   library.
3. **Cosmetic: no descriptions.** `query` is self-evident; a fleet of copied
   helpers growing `db`, `encode`, `dispatch` phases will not stay
   self-evident. `MeasureWithDesc` at definition time is free.

## 5. Version currency

**Current.** Pinned v1.0.3 (`example/taskmanager/go.mod`); the repo's
latest `server_timing/*` tag is exactly **v1.0.3**, and httputil's CHANGELOG
records "server_timing has zero drift since v1.0.3" (CHANGELOG.md:89) — no
newer module tag exists to chase. v1.0.3 was the 2026-10-09 coordinated cut
with root v1.5.0 (CHANGELOG.md:102). The only skew is doc-tier: the recipe
compile-gate pins **v1.0.2** (`cmd/doc-check/v4/recipes_compile_test.go:25`)
— one patch behind, harmless (the recipe uses the stable core), alignable in
the next doc sweep.

## 6. The fleet argument

Server timing is a fleet-debugging primitive: when the owner gets "the CQRS
app feels slow" from any project, DevTools-visible phase timings are the
zero-agent diagnostic. go-cqrs-lite's §2.47 is where fleet apps learn to bind
queries to HTTP — by embedding `MeasureServerTiming` in the copied helper,
the library guarantees every fleet app that copies the recipe gets phase
timing for free. That part is done and test-pinned. What the fleet argument
adds: the recipe should also teach the production posture (gating), or the
first fleet app that ships timings to the internet learns it from an incident
instead of the template. The core is fully leveraged; the doctrine-fringe
(gating, SSE replay) is where collective benefit is being left behind.

## 7. Ranked opportunities

1. **Gate timing in the example + recipe** (medium): switch example/taskmanager/setup.go:275 to
   `ServerTimingMiddlewareWhen` behind a debug predicate (env flag or
   `?debug`), keep the ungated form in tests; one sentence in §2.47's job
   table noting the production posture.
2. **Time the SSE replay phase** (medium): either a recipe addendum showing
   the defer idiom on a ServeSSE-style handler, or — if metaengine's budget
   allows a stdlib-only dep — a `replay` metric inside
   `metaengine/v4/sse.go:244`.
3. **Add a `desc` to the query phase** (low): `MeasureWithDesc("query",
   "dispatch + encode")` in httpx.go:45.
4. **Align the recipe compile-pin v1.0.2 → v1.0.3** (trivial):
  `cmd/doc-check/v4/recipes_compile_test.go:25`.

## 8. Reconsider when

- The example twin or a fleet app deploys publicly (opportunity 1 flips from
  posture to requirement).
- ServeSSE replay latency ever surfaces as a support question (opportunity 2
  becomes the diagnostic that answers it).
- A handler needs to record a phase from a spawned goroutine (row 6's N/A
  flips; the collector is documented concurrency-safe).
- httputil cuts a new `server_timing/*` tag (none exists past v1.0.3; a new
  one likely follows a root-module coordinated release).

## 9. How to verify

```
rg -n 'server_timing' -g go.mod /home/lars/projects/go-cqrs-lite
rg -n 'servertiming\.' /home/lars/projects/go-cqrs-lite/example/taskmanager
sed -n '40,60p' /home/lars/projects/go-cqrs-lite/example/taskmanager/httpx.go
sed -n '270,280p' /home/lars/projects/go-cqrs-lite/example/taskmanager/setup.go
sed -n '100,110p' /home/lars/projects/go-cqrs-lite/example/taskmanager/query_http_test.go
git -C /home/lars/projects/httputil tag --list 'server_timing/*' | sort -V | tail -3
rg -n 'zero drift since' /home/lars/projects/httputil/CHANGELOG.md
sed -n '3530,3545p' /home/lars/projects/go-cqrs-lite/.agents/skills/go-cqrs-lite/references/recipes.md
sed -n '23,26p' /home/lars/projects/go-cqrs-lite/cmd/doc-check/v4/recipes_compile_test.go
```
