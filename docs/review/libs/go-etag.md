# go-etag — Utilization Audit · go-cqrs-lite

> Audited 2026-11-22 · Library `github.com/larsartmann/go-etag` @ 39f9f37 (latest local tags: root v0.7.1, server v0.6.2, entitytag v0.6.2) · Consumer go-cqrs-lite @ feec9bc74 · Group G2

**Verdict:** UNDERUSED-MODERATE
**Score:** 2.5/6 capabilities fully leveraged = 42% (low)

go-etag is adopted at exactly the boundary the doctrine wants — app/recipe
code, not library modules — and the part used (the RFC 7232 domain API:
`ETag`, `NewETag`, `Strong`, `MatchesIfNoneMatch`) is used correctly,
deterministically, and test-pinned as the cache-validation leg of recipe
§2.47. What is left on the table: the consumer's own production HTTP surface
(docserver) serves its OpenAPI/catalog assets with zero conditional-request
support, and a CQRS library whose whole pitch includes optimistic concurrency
ships a PATCH endpoint recipe with no `If-Match`/`MatchesIfMatch` story.
Version-wise the consumer is on the **latest nested-module train** — the
v0.7.x root tags belong to the deprecated alias shim, not the modules in use.

## 1. What the library is

Two-sided RFC 7232 entity-tag library, five modules (go-etag `README.md:9-17`):

- **`go-etag/server`** (package `etag`) — middleware `etag.New` that buffers
  GET/HEAD bodies, hashes them (FNV-64a default, `HashFunc` injectable),
  answers `If-None-Match` with RFC 7232 weak comparison → 304, respects
  handler-set tags (`SkipIfPresent`), skips routes (`Skip`), degrades to
  streaming past `MaxBufferSize`, classifies errors via go-error-family, and
  fires `OnETagGenerated`/`On304`/`OnBufferOverflow` observability hooks.
  Re-exports the entitytag domain surface (parity-guarded by
  `go-etag/server/entity_parity_test.go:71-80`).
- **`go-etag/entitytag`** — the zero-dependency domain type: `ETag`,
  `Strength`, `StrongEqual`/`WeakEqual`, `ParseETag`/`ParseETagList`,
  `MatchesIfNoneMatch` (weak), `MatchesIfMatch` (strong, lost-update
  prevention) (go-etag/entitytag/entity_tag.go:52,91,111,143,176,187).
- **`go-etag/client`** — `http.RoundTripper` conditional-GET cache: replays
  validators, rebuilds 304s as 200s, RFC 9111 freshening, invalidation after
  mutations, bounded memory.
- **`go-etag/metrics`** — `metrics.Attach(cfg)` atomic counters with
  `HitRatio()`.
- **root** — deprecated alias shim for server, deleted at v1.0.0.

## 2. Usage census in go-cqrs-lite (evidence)

Exactly one module pins it:

- `example/taskmanager/go.mod` — `go-etag/server v0.6.2` direct;
  `go-etag/entitytag v0.6.2` indirect (pulled by server:
  `go mod graph` shows `go-etag/server@v0.6.2 → go-etag/entitytag@v0.6.2`;
  httputil v1.8.0 also requires `go-etag/server v0.6.1` transitively).

One import site, four symbols, all domain-tier (the middleware is NOT used):

- `example/taskmanager/httpx.go:17` — import.
- `example/taskmanager/httpx.go:77` — `etag.ETag` as the return type of
  `dispatchEncode`.
- `example/taskmanager/httpx.go:88-90` — SHA-256 of the deterministic codec
  output, wrapped: `etag.NewETag(hex.EncodeToString(sum[:]), etag.Strong)`.
- `example/taskmanager/httpx.go:57` — `etag.MatchesIfNoneMatch(tag,
  r.Header.Get("If-None-Match"))` → `304` with no body.
- Pinned by tests: `example/taskmanager/query_http_test.go:104-113` asserts a
  strong (unprefixed, quoted) tag, and the conditional GET returns 304 with
  the same tag and no body (`query_http_test.go:118-149`).

Doc-tier references:

- Recipe §2.47 credits `go-etag/server` for cache validation
  (`.agents/skills/go-cqrs-lite/references/recipes.md:3432`) and its helper
  fence imports it (`recipes.md:3513,3546`); compile-gated at
  `cmd/doc-check/v4/recipes_catalog_meta3.go:286,300,313,320,333`.
- `cmd/doc-check/v4/recipes_compile_test.go:23` pins the fences to
  `go-etag/server v0.6.1` (one patch behind the example's v0.6.2).

No library production module requires go-etag (doctrine + dep budgets).

## 3. Capability inventory

| # | Capability | Grade | Evidence / reason |
|---|---|---|---|
| 1 | ETag domain type + strength semantics | **Full** | `NewETag(..., etag.Strong)` at httpx.go:90; strength asserted by query_http_test.go:110-112 |
| 2 | Conditional GET: `MatchesIfNoneMatch` → 304 | **Full** | httpx.go:57-61; 304-no-body asserted query_http_test.go:118-130 |
| 3 | Validator generation (`HashFunc`/FNV-64a middleware shape) | **Partial** | the hash job is hand-rolled as SHA-256 (httpx.go:88) — defensible (the bytes are already in hand; `etag.New` would re-buffer), but the "generate a tag from bytes" capability is re-implemented, and the recipe pins consumers to SHA-256+hex rather than the library's pluggable `HashFunc` |
| 4 | Auto-ETag middleware (`etag.New`) | Missed | zero call sites; the obvious home is docserver — `catalog/v4/docserver` serves `openapi.json`, `openapi.yaml`, `catalog.json`, eventcatalog pages, and a rewritten `scalar.js` with no `ETag`/`Cache-Control` anywhere in the package (verified: `rg 'ETag|Cache-Control' catalog/v4/docserver/*.go` → no production hits), so every docs-page reload re-downloads unchanged specs |
| 5 | `MatchesIfMatch` lost-update prevention (RFC 7232 § 3.1) | Missed | `PATCH /api/tasks/{id}` (example/taskmanager/http.go:42) applies commands with no `If-Match` precondition; a CQRS library teaching optimistic concurrency ships no If-Match recipe even though the library exports exactly this helper (go-etag/entitytag/entity_tag.go:187) |
| 6 | `SkipIfPresent` handler-tag respect | N/A | single-source tags in the one adopted pattern — flip trigger: mixing handler-set revision tags with computed tags |
| 7 | Observability hooks + `metrics` module | N/A | example-grade app has no cache telemetry consumer — flip trigger: first production fleet app on recipe §2.47 (then `metrics.Attach` is one line for hit-ratio) |
| 8 | HEAD-compliant conditional responses | N/A | no HEAD-specific routes; behavior would arrive free with row 4's middleware |
| 9 | `go-etag/client` conditional-GET transport | N/A | consumer is server-side — flip trigger: a systemtest/systemscenario harness that polls REST endpoints could cut re-download of unchanged query responses |
| 10 | `entitytag` direct for projection/query caching | N/A | projection versioning correctly uses seq numbers (metaengine `Watcher`/`SSEReplay`), which is the right mechanism at the store layer; ETags are an HTTP-representation concern — flip trigger: an HTTP-layer projection cache in front of metaengine reads |

Score derivation: rows N=10, N/A rows 6–10 → **A=6** (adjusted: rows 1–5).
Full = 2 × 1.0, Partial = 1 × 0.5, Missed = 2 × 0.
Score = round(100 × 2.5 / 6) = **42% → low band**.

## 4. Gap analysis

1. **docserver is an unconditional-download server.** Eighteen routes
   (`catalog/v4/docserver/docserver.go:218-247`) serve large, deterministic,
   rarely-changing documents (OpenAPI/AsyncAPI JSON+YAML, catalog.json, a
   ~MB-scale rewritten `scalar.js` from
   `catalog/v4/docserver/serve.go:24-53`) with no validators and no
   cache-control. `etag.New(etag.DefaultETagConfig())(mux)` — or even plain
   content-hash headers using the already-adopted domain API — would make
   every repeat visit a 304. This is the single highest-value miss: it is
   production code (not example), the representations are hash-stable by
   construction, and the dep is already in the fleet. Budget note:
   `go-etag/server` carries `go-error-family` (already ubiquitous in this
   repo), so the graph cost is near zero.
2. **No If-Match story on the write path.** Recipe §2.47's own framing is
   "hand-rolled endpoints reliably go wrong"; lost updates on
   read-modify-write UIs are that class of wrong. The pieces exist on both
   sides: `etag.MatchesIfMatch` for the check, errorfamily's documented
   428/412 posture (httputil README "Error Classification" → go-error-family
   "Conditional Requests") for the response. A short recipe addendum would
   close it.
3. **Hash choice is baked into the recipe rather than the library.**
   SHA-256-of-encoded-bytes is a good default, but it lives in every
   consumer's copy of the §2.47 helper. If the tag scheme ever needs to
   change (e.g., truncation for header-size limits), every copied helper
   drifts — the fleet-lens argument for pushing the generation into
   `HashFunc`-shaped shared code.

## 5. Version currency

**Current on the modules in use.**

- Pinned: `go-etag/server v0.6.2` + `entitytag v0.6.2` (indirect) — and
  `server/v0.6.2` / `entitytag/v0.6.2` ARE the latest nested-module tags
  (verified via `git tag --list 'server/v*' 'entitytag/v*'`). The v0.7.0 /
  v0.7.1 root tags are the deprecated alias shim's own train: go-etag's
  CHANGELOG records `[0.7.1]` (2026-11-18) as a staircase release whose
  nested-module train is exactly **v0.6.2**, with root staying one minor
  ahead only until v1.0.0 deletes the shim. Nothing to bump.
- v0.6.2's nested deltas are maintenance (go-error-family v0.12.0 bump,
  internal test layout) — zero public-API change
  (go-etag CHANGELOG `[0.6.2]`).
- The only real skew is doc-tier: the recipe compile-gate pins
  `go-etag/server v0.6.1` (`cmd/doc-check/v4/recipes_compile_test.go:23`)
  while the example twin runs v0.6.2. Harmless (the recipe uses stable
  surface), worth aligning in the next doc sweep.

## 6. The fleet argument

go-etag is the fleet's single conditional-request implementation, and
go-cqrs-lite's §2.47 is its loudest distribution channel — every fleet app
that binds a query to HTTP copies the ETag+Server-Timing helper. That makes
the recipe the leverage point: fixing the missing If-Match leg or the
hard-wired hash scheme once in the recipe beats fixing it in N apps. The
asymmetry that hurts: the consumer adopted the domain API (good) but not the
middleware (row 4), so its one production HTTP surface re-downloads specs on
every visit — the exact waste conditional requests exist to remove, in the
repo best positioned to demonstrate it.

## 7. Ranked opportunities

1. **ETag the docserver assets** (high): wrap the docserver mux (or just the
   JSON/YAML/js handlers) with `etag.New(etag.DefaultETagConfig())`, or
   compute content-hash tags via the already-adopted domain API in
   `serveOpenAPIJSON` et al. 18 routes, one wrap, every repeat visit 304s.
2. **Add the If-Match leg to recipe §2.47** (medium): a short addendum —
   read the tag from the GET, send it back as `If-Match` on PATCH,
   `etag.MatchesIfMatch` gate → 412 via errorfamily — with the taskmanager
   twin demonstrating it on `PATCH /api/tasks/{id}`
   (example/taskmanager/http.go:42).
3. **Align the recipe compile-pin to v0.6.2** (low): one line at
   `cmd/doc-check/v4/recipes_compile_test.go:23`.
4. **Consider `HashFunc` in the recipe helper** (low, design): parameterize
   the helper's tag scheme so copied helpers don't hard-code SHA-256+hex.

## 8. Reconsider when

- docserver gets deployed anywhere with humans or CI reloading specs
  (opportunity 1 flips from "high" to "urgent").
- A fleet app builds a read-modify-write UI on the taskmanager pattern
  (opportunity 2 becomes a correctness requirement, not a recipe nicety).
- The first production deployment of §2.47 wants cache telemetry (row 7's
  N/A flips; `metrics.Attach` + `HitRatio()` is the one-liner).
- go-etag ships a nested-module release with behavior changes (root v0.7.x
  history shows staircase cuts; nested bumps so far are API-neutral).

## 9. How to verify

```
rg -n 'larsartmann/go-etag' -g go.mod /home/lars/projects/go-cqrs-lite
rg -n 'etag\.' /home/lars/projects/go-cqrs-lite/example/taskmanager/httpx.go
sed -n '45,61p;77,91p' /home/lars/projects/go-cqrs-lite/example/taskmanager/httpx.go
sed -n '103,130p' /home/lars/projects/go-cqrs-lite/example/taskmanager/query_http_test.go
rg -n 'ETag|Cache-Control' /home/lars/projects/go-cqrs-lite/catalog/v4/docserver/*.go
sed -n '218,247p' /home/lars/projects/go-cqrs-lite/catalog/v4/docserver/docserver.go
cd /home/lars/projects/go-etag && git tag --list | sort -V | tail -6 && git tag --list 'server/v*' | sort -V | tail -2
rg -n '0\.7\.1|0\.6\.2' /home/lars/projects/go-etag/CHANGELOG.md | head -6
cd /home/lars/projects/go-cqrs-lite/example/taskmanager && go mod graph | grep -E 'go-etag|httputil' | head -8
```
