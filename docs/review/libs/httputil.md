# httputil — Utilization Audit · go-cqrs-lite

> Audited 2026-11-22 · Library `github.com/larsartmann/httputil` @ 34dbf05 (latest local tag v1.9.0) · Consumer go-cqrs-lite @ feec9bc74 · Group G2

**Verdict:** UNDERUSED-SEVERE
**Score:** 1/12 capabilities fully leveraged = 8% (low)

go-cqrs-lite pins the full httputil middleware suite in exactly one module — the
blessed example app — and uses **one function out of ~60 exported symbols**.
The shape of the adoption is right (thin, recipe-driven, transport stays in app
code per ADR-0127); the depth is not. The golden-path example that fleet apps
copy omits panic recovery, body-size limits, and request IDs, and hand-rolls
health handlers that the library ships — so every fleet app that copies the
recipe inherits the gaps.

## 1. What the library is

Composable `net/http` middleware and server primitives: CORS, security headers,
CSP nonce, panic recovery, request IDs, timeouts, structured logging, response
compression, request decompression with bomb protection, keyed rate limiting,
content-language negotiation, CSRF (nosurf), max-body-size enforcement,
response recording, client-IP extraction, a configurable `http.Server` wrapper,
Kubernetes health handlers, a 19-spec behavioral conformance suite
(`httpspec`), and error classification throughout. Four dependencies
(`go-error-family`, `go-etag/server`, `golang.org/x/time`, `justinas/nosurf`),
pure stdlib `net/http`, Go 1.27+. The repo is multi-module: root
`httputil`, plus submodules `server_timing`, `endpoints` (each covered by its
own report in this directory) and an `httpspec` subpackage of the root.

Sources: `httputil README.md` (feature walkthrough + API table),
`httputil/go.mod:1-13` (dep set), `httputil/CHANGELOG.md:34` ([1.9.0]).

## 2. Usage census in go-cqrs-lite (evidence)

Exactly one module pins the root module:

- `example/taskmanager/go.mod`: `github.com/larsartmann/httputil v1.8.0` (direct).

Exactly one import site, one symbol:

- `example/taskmanager/http.go:13` — import.
- `example/taskmanager/http.go:118` — `httputil.ParseUintQuery(r, "limit")` in
  `decodeListTasks`. That is the entire production call surface.

Indirect/doc references (no code usage):

- `.agents/skills/go-cqrs-lite/references/recipes.md:3418` — recipe §2.47
  ("Bind a system Query to an HTTP endpoint") imports `httputil` for the same
  `ParseUintQuery` pattern; its job table (`recipes.md:3429-3434`) credits
  `go-error-family`, `go-etag/server`, and `httputil/server_timing` — the root
  module appears only as the query-param parser.
- `cmd/doc-check/v4/recipes_catalog_meta3.go:287` and
  `cmd/doc-check/v4/recipes_compile_test.go:24` — the recipe fences are
  compile-gated against released `httputil v1.5.0` (two minors behind the
  example's v1.8.0 pin; see §5).

No library production module (metaengine, catalog, system, …) requires
httputil — consistent with the transport-agnostic doctrine and per-module dep
budgets (`nix run .#check-arch`).

## 3. Capability inventory

| # | Capability | Grade | Evidence / reason |
|---|---|---|---|
| 1 | Query-param parsing (`ParseUintQuery`) | **Full** | `example/taskmanager/http.go:118`, recipe §2.47 |
| 2 | Middleware composition (`Chain`, `Compose`, `MiddlewareStack`) | Missed | taskmanager wires exactly one middleware manually (`example/taskmanager/setup.go:275`); no chain anywhere |
| 3 | Panic recovery (`Recovery`) | Missed | no recovery middleware in the example server; a handler panic drops the connection without a 500 |
| 4 | Request logging (`Logging`/`LoggingFromContext`) | Missed | `StartHTTP` logs only start/stop (`example/taskmanager/setup.go:271-283`); no per-request line |
| 5 | Request ID (`RequestID`) | Missed | no request ID generation/propagation; nothing correlates a response to a log line |
| 6 | Security headers (`SecurityHeaders`) | Missed | taskmanager sets none; docserver sets `X-Content-Type-Options` once by hand (`catalog/v4/docserver/architecture.go:74`) |
| 7 | CSP nonce (`Nonce`, `NonceAttr`) | Missed | docserver hand-rolls 128-bit nonce minting (`catalog/v4/docserver/csp.go:18-29`) — the exact job of `httputil.Nonce` |
| 8 | Body size limit (`MaxBodySize`) | Missed | `decodeJSON` decodes `r.Body` unbounded (`example/taskmanager/http.go:344-349`); the recipe's own intro warns about `?limit=100000000`-class inputs but the body path has no ceiling |
| 9 | Request timeout (`Timeout`) | Missed | `StartHTTP` sets only `ReadHeaderTimeout` (`example/taskmanager/setup.go:274-276`) |
| 10 | Server lifecycle (`NewServer`/`NewServerListener`) | Missed | hand-rolled `&http.Server{...}` (`example/taskmanager/setup.go:271-277`). Note: the default `WriteTimeout: 30s` would kill SSE streams — a fair reason to not adopt blindly, but then the omission deserves a comment, not silence |
| 11 | Health handlers (`RegisterHealth`, `ReadyHandlerWithProbe`) | Missed | taskmanager hand-rolls `/health` → `{"status":"ok"}` (`example/taskmanager/http.go:63-65`) — a re-implementation of `httputil.HealthHandler()` without the `/live` + `/ready` K8s pair |
| 12 | Behavioral spec suite (`httpspec.Run`) | Missed | taskmanager and docserver HTTP behavior is pinned only by hand-written tests; `httpspec.Run(t, mux)` is one line |
| 13 | CORS | N/A | no browser origins in scope today — flip trigger: first fleet app with a browser frontend on the example twin |
| 14 | CSRF (`CSRFMiddleware`) | N/A | tokenless JSON API, no cookies — flip trigger: cookie-authenticated browser app |
| 15 | Compression / Decompression | N/A | proxy-tier job in the fleet's posture (same ruling as go-datastar's README); flip trigger: bandwidth-sensitive direct-to-browser API without a proxy |
| 16 | Keyed rate limiting | N/A | example/docs servers are not internet-facing — flip trigger: public docserver or fleet-app deployment |
| 17 | Language negotiation (`Language`) | N/A | single-locale surfaces — flip trigger: i18n docs UI |
| 18 | Client IP / trusted proxies | N/A | no documented reverse-proxy tier in the examples |
| 19 | Metrics middleware (`Metrics`) | N/A | observability goes through cqrsotel (OTel), which is the deeper integration |
| 20 | ResponseRecorder / capability detection | N/A | a middleware-construction building block; the consumer constructs no middleware |

Score derivation: rows N=20, N/A rows 13–20 → **A=12**. Full=1 (1.0), Partial=0,
Missed rows 2–12 (11 × 0). Score = round(100 × 1.0 / 12) = **8% → low band**.

## 4. Gap analysis

The one-miss-per-line table above clusters into three findings:

1. **The golden path teaches gaps.** Recipe §2.47's own preamble
   (`recipes.md:3420-3427`) says hand-rolled endpoints "reliably go wrong" and
   sells the helper as done-right-once. Yet the twin example lacks recovery,
   body limits, request IDs, and timeouts — the exact classes the library
   ships one-liners for. Fleet apps copy the example, so the misses propagate;
   under the owner's fleet lens this is the highest-leverage fix in this
   report: one example diff, every future fleet app inherits it.
2. **The repo's own production HTTP surface re-implements httputil pieces.**
   docserver mints CSP nonces by hand (`catalog/v4/docserver/csp.go:18`) and
   sets `nosniff` ad hoc (`catalog/v4/docserver/architecture.go:74`). The
   counterweight is real: catalog/v4 has a strict production dep budget, and
   root httputil drags four deps for a nonce. The budget-honest fix is to keep
   the hand-rolled nonce but record it as a deliberate budget decision, or
   extract nothing and accept the duplication consciously.
3. **`ParseUintQuery` alone does not justify the dependency.** The recipe
   imports the full middleware suite (with nosurf and x/time in the module
   graph) to parse one integer. Either use more of it (finding 1) or, in
   leaner fleet apps, drop the import — a 3-line `strconv.ParseUint` beats a
   4-dependency import. As pinned, it is the worst of both worlds.

## 5. Version currency

- Pinned: **v1.8.0** (`example/taskmanager/go.mod`); latest local tag
  **v1.9.0** (2026-11-14). Behind one minor.
- v1.9.0 is additive for this consumer (Pusher delegation, rate-limiter clock
  injection, `httpspec.ExpectNotModifiedDynamic`, and a compression
  Content-Type sniff fix — `httputil/CHANGELOG.md:34-60`). Nothing used by
  go-cqrs-lite changed behavior; the bump is low-risk but also low-urgency.
- Recipe compile-gate pins are staler: `httputil v1.5.0`
  (`cmd/doc-check/v4/recipes_compile_test.go:24`) — two minors behind the
  example pin, so the docs and the example verify against different APIs.
  Harmless today (the recipe uses only stable surface), worth a deliberate
  bump alongside the v1.9.0 move.
- `httputil/CHANGELOG.md:9-22` [Unreleased] records a go1.27.4 CI-pin for
  GO-2026-6859 (`MaxBytesReader` crash) — relevant precisely to the
  `MaxBodySize` capability this consumer is not using yet.

## 6. The fleet argument

The owner's lens: "if every project re-integrates everything I will only fix
bugs all day long." httputil exists to be the one place HTTP middleware gets
fixed. go-cqrs-lite is the fleet's template repo: its example app is what
fleet apps clone. Right now the template uses 1/60th of the shared middleware
and hand-rolls health handlers, nonce generation (in docserver), and unbounded
body reads — each a place where a future httputil security fix will not reach
fleet apps that copied the template. The doctrine constraint cuts the other
way too: the library proper must stay transport-agnostic (ADR-0127), so
adoption belongs in `example/`, recipes, and docserver — which is exactly
where the gaps sit. There is no conflict between doctrine and adoption here;
there is only thinness where thinness is not required.

## 7. Ranked opportunities

1. **Harden the example twin** (high): add `Recovery`, `MaxBodySize`,
   `RequestID` (or a commented one-line `Chain` showing the fleet-standard
   stack) to `example/taskmanager` and mirror it in recipe §2.47's intro. One
   diff; every fleet app inherits it.
2. **Replace the hand-rolled `/health`** (high, trivial): swap
   `example/taskmanager/http.go:63-65` for `httputil.RegisterHealth(mux)` and
   demonstrate `/ready` with a store probe (`ReadyHandlerWithProbe`) — a CQRS
   app's readiness *is* its store.
3. **`httpspec.Run(t, mux)` in the example's HTTP tests** (medium): one line
   pins 19 behavioral conventions on the golden path.
4. **Decide docserver's nonce posture** (medium): record the
   `csp.go`-vs-`httputil.Nonce` choice as a budget decision in docserver's
   docs so the next audit doesn't re-litigate it.
5. **Bump v1.8.0 → v1.9.0** (low) and the recipe fence pin v1.5.0 → v1.9.0
   (low) in the same sweep so example and docs agree.

## 8. Reconsider when

- A fleet app puts the example twin on the internet (rate limiting, CORS,
  security headers flip from N/A to required).
- docserver is deployed publicly (the `endpoints` + `SecurityHeaders` +
  rate-limit trio becomes the default posture; see `httputil-endpoints.md`).
- Any browser frontend consumes the taskmanager API (CORS + CSRF + Language
  ordering rules in the httputil README become live).
- httputil lands a security fix the hand-rolled code can't inherit
  (GO-2026-6859 class) — then finding 1 stops being hygiene and becomes a
  patch gate.

## 9. How to verify

```
rg -n 'larsartmann/httputil' --type go /home/lars/projects/go-cqrs-lite
rg -n 'larsartmann/httputil' -g go.mod /home/lars/projects/go-cqrs-lite
git -C /home/lars/projects/httputil log -1 --format='%H' && git -C /home/lars/projects/httputil tag --list | sort -V | tail -3
sed -n '63,65p;118p;344,349p' /home/lars/projects/go-cqrs-lite/example/taskmanager/http.go
sed -n '271,283p' /home/lars/projects/go-cqrs-lite/example/taskmanager/setup.go
sed -n '18,29p' /home/lars/projects/go-cqrs-lite/catalog/v4/docserver/csp.go
sed -n '23,25p' /home/lars/projects/go-cqrs-lite/cmd/doc-check/v4/recipes_compile_test.go
sed -n '3418,3435p' /home/lars/projects/go-cqrs-lite/.agents/skills/go-cqrs-lite/references/recipes.md
```

Sibling modules of the same repo: `httputil-server-timing.md` (pinned, well
used), `httputil-endpoints.md` (not adopted), both in this directory.
