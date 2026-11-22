# httputil/endpoints — Utilization Audit · go-cqrs-lite

> Audited 2026-11-22 · Library `github.com/larsartmann/httputil/endpoints` @ 34dbf05 (repo HEAD; latest module tag v0.1.0, 2026-11-06) · Consumer go-cqrs-lite @ feec9bc74 · Group G2

**Verdict:** NOT-ADOPTED-GAP
**Score:** 0/3 applicable capabilities leveraged = 0% (low)

Zero requires anywhere in go-cqrs-lite's 104 go.mod files. The gap is narrow
but real and slightly embarrassing in shape: the repo's own
**self-describing documentation server** — whose entire purpose is telling
humans and tools what surfaces exist — mounts 18 routes bare on an
`http.ServeMux` with no route-level listing and no registry, which is
precisely the problem statement the endpoints module was extracted to solve
("the listing cannot drift from what is actually served"). Severity is low:
OpenAPI/AsyncAPI/catalog.json already describe the _system's_ contract, the
docserver route set is stable, and the module is a two-week-old v0.1.0. But
the adoption site is correct, the dependency cost is zero (stdlib-only), and
the repo is the fleet's template.

## 1. What the library is

A self-describing route-registration layer for Go's `http.ServeMux`, shipped
as a separate stdlib-only, zero-dependency module
(httputil/endpoints/README.md:1-10). Because `*http.ServeMux` cannot
enumerate its patterns, a `Registry` is the source of truth: every route
registered through it is **mounted on the mux AND recorded in one
declaration** (httputil/endpoints/endpoints.go:111,143), and
`endpoints.Register(mux, routes)` (endpoints.go:191) mounts a
`GET /endpoints` listing — JSON for `curl` (json/v2), an HTML table for
browsers, picked by RFC 9110 q-value negotiation that never 406s, with
`Vary: Accept` + `Cache-Control: no-store`. Wiring errors are values, not
panics (nil mux, duplicate pattern, non-UTF-8 summary → classified errors).
The HTML is CSP-clean and escaped; the README warns the listing documents
your attack surface and should be mounted deliberately on public services.
`Registry.Routes()` (endpoints.go:158) exposes the sorted, defensive-copied
route list for non-HTTP consumers. Released 2026-11-06 as v0.1.0 in the
owner's "pre-1.0 iteration tier" (httputil CHANGELOG [endpoints/v0.1.0]);
first external consumer: the CV repo.

## 2. Usage census in go-cqrs-lite (evidence)

**None.** Verified by:

```
rg -n 'httputil/endpoints' -g go.mod /home/lars/projects/go-cqrs-lite   # no hits
rg -n 'httputil/endpoints' --type go /home/lars/projects/go-cqrs-lite   # no hits
```

The only httputil requires in the repo are root `httputil v1.8.0` +
`server_timing v1.0.3`, both in `example/taskmanager/go.mod`. The docserver
registers its routes directly on the mux:
`catalog/v4/docserver/docserver.go:218-247` holds 18
`mux.HandleFunc("GET "+prefix+"/...")` calls (index, openapi HTML/JSON/YAML,
asyncapi ×3, catalog.json, eventcatalog + 5 subpages, d2 ×2, architecture),
each maintained by hand alongside whatever the index page links.

## 3. Capability inventory

| # | Capability                                                       | Grade  | Evidence / reason                                                                                                                                                                                                                                                      |
| - | ---------------------------------------------------------------- | ------ | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1 | Self-describing `GET /endpoints` listing on the docs server      | Missed | docserver is the fleet's canonical "describe the surfaces" HTTP surface yet has no machine-readable inventory of its own 18 routes (docserver.go:218-247); an adopter would register routes through `Registry.HandleFunc` and mount the listing behind the same prefix |
| 2 | Registry as single source of truth (mount + record, drift-proof) | Missed | today the docserver's index page, its route handlers, and any prose docs each enumerate the surface independently — the exact drift class the module exists to kill                                                                                                    |
| 3 | `Routes()` machine-readable inventory for fleet tooling          | Missed | docserver already emits `catalog.json`/`openapi.json` for the _system_; a route-level listing (or an `endpoints.Response` embedded in the docs index payload) would cover the _server_ tier that catalog.json cannot see                                               |
| 4 | Route listing for example/taskmanager                            | N/A    | examples minimize ceremony; a listing on the golden path is demo value, not correctness — flip trigger: the example grows enough routes that its README hand-maintains a route table                                                                                   |
| 5 | Adoption in other library modules                                | N/A    | docserver is the only HTTP-serving production module in the repo (transport-agnostic doctrine, ADR-0127); no other listing site exists                                                                                                                                 |
| 6 | Custom representations / listing customization                   | N/A    | v0.1.0 pre-1.0 iteration tier; no consumer requirement has been validated yet, here or elsewhere                                                                                                                                                                       |

Score derivation: rows N=6, N/A rows 4–6 → **A=3**. Full = 0, Partial = 0,
Missed = 3 × 0 (rows 1–3). Score = round(100 × 0 / 3) = **0% → low band**.

## 4. Gap analysis

1. **The docs server doesn't describe itself.** Everything docserver serves
   is about the CQRS system's contracts; the docserver's own surface — 18
   routes, prefix-parameterized — is discoverable only by reading
   `docserver.go`. For a team (or an AI agent) wiring docs into a service,
   "what does this docs server expose, and where" is a legitimate API
   question that a `/endpoints` listing would answer with zero drift risk.
   The registry pattern also fits the existing code shape: the 18
   registrations are already centralized in one function, so converting them
   to `routes.HandleFunc(mux, pattern, handler, summary)` is mechanical.
2. **Index-page drift risk is unpinned.** The docs index
   (`serveIndex`, docserver.go:218-222) links to pages by hand; nothing
   fails if a route is added but not linked (or linked but renamed). The
   module's core promise — "the listing cannot drift from what is actually
   served" — is the pin. (Counterpoint kept honest: no drift bug has been
   observed; this is prevention, not repair.)
3. **What adoption does NOT buy.** OpenAPI/AsyncAPI listings describe the
   fleet's _contracts_ — richer than any route table. The endpoints listing
   covers the deployment tier (which routes exist, which methods). Those are
   complements, not substitutes; adopting one does not retire the other.

## 5. Version currency

Not pinned (no require), so nothing to be current or behind on. Latest
module tag: **v0.1.0** (2026-11-06), cut from the same repo commit era as
root v1.8.0. Post-v0.1.0, the repo's [Unreleased] section carries one
endpoints fix (multiple `Accept` header lines now honored —
httputil CHANGELOG.md:24-37), i.e. **the fix is not yet in any tag**; an
adopter today gets the first-line-only negotiation behavior unless they pin
the repo HEAD. For a v0.1.0 pre-1.0 module this is normal, but it means
"go grab it now" should wait for v0.2.0 unless the fix matters.

## 6. The fleet argument

The owner built this module so that every fleet service can self-describe
once instead of N times, and CV already validates the pattern. go-cqrs-lite's
docserver is the fleet's most "describing things is the product" HTTP
surface and is the natural second consumer: adopting there (a) gives the
module its first library-grade consumer, (b) dogfoods the v0.1.0 API while
it still has iteration freedom (the owner's stated purpose for the tier),
and (c) turns the repo's template story ("here is how a docs server should
behave") complete. The cost side is as small as it gets: stdlib-only module,
mechanical migration of 18 already-centralized registrations, no new dep
budget entry. The residual argument against — "route listings are
nice-to-have on an internal docs server" — is fair, which is why the verdict
is a mild GAP, not severe.

## 7. Ranked opportunities

1. **Register docserver routes through `endpoints.Registry` + mount the
   listing** (medium): mechanical conversion at docserver.go:218-247; decide
   exposure (the README's attack-surface warning applies if docserver is
   internet-facing — listing can sit behind the same middleware stack).
   Consider waiting for the v0.2.0 tag that includes the multi-Accept-line
   fix (§5).
2. **Derive the index page's link set from `Registry.Routes()`** (low):
   kills the hand-maintained link list with the same single source of truth.
3. **Show the pattern in the taskmanager example** (low, optional): one
   `endpoints.Register` line on the golden path teaches fleet apps the
   habit; skip if example ceremony is the bigger cost.

## 8. Reconsider when

- The docserver index and its registered routes ever actually drift (turns
  prevention into repair; opportunity 1 becomes a bugfix).
- A fleet app or the docs server is exposed to third parties who need a
  machine-readable route inventory (the `/endpoints` JSON becomes an
  integration surface, not a nicety).
- endpoints ships v0.2.0+ with the negotiation fix and (per the owner's
  iteration-tier ruling) any consumer-driven API hardening — the adoption
  cost drops further and the "pre-1.0 churn" objection weakens.
- The taskmanager example grows a README-maintained route table (row 4's
  flip trigger).

## 9. How to verify

```
rg -n 'httputil/endpoints' -g go.mod /home/lars/projects/go-cqrs-lite
rg -rn 'httputil/endpoints' /home/lars/projects/go-cqrs-lite --type go
sed -n '218,247p' /home/lars/projects/go-cqrs-lite/catalog/v4/docserver/docserver.go
rg -c 'HandleFunc' /home/lars/projects/go-cqrs-lite/catalog/v4/docserver/docserver.go
git -C /home/lars/projects/httputil tag --list 'endpoints/*' | sort -V
rg -n 'MultipleAcceptLines|combinedAccept' /home/lars/projects/httputil/CHANGELOG.md | head -3
rg -n 'func .*Registry.*Handle|func Register|func NewRegistry' /home/lars/projects/httputil/endpoints/endpoints.go
```
