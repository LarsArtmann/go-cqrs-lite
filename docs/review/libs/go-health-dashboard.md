# go-health-dashboard — Utilization Audit · go-cqrs-lite

> Audited 2026-11-22 · Library `github.com/larsartmann/go-health-dashboard` @ 19af9cf (latest local tag v0.13.3) · Consumer go-cqrs-lite @ feec9bc74 · Group G3

**Verdict:** NOT-ADOPTED-JUSTIFIED (blocked behind the go-health seam — flip is cheap once that lands)
**Score:** 0/4 capabilities fully leveraged = 0% (low)

Zero `require`s, and correctly so: the dashboard mounts on a go-health
`*health.Probe` (or aggregate), which go-cqrs-lite does not expose anywhere
(see `go-health.md`, this directory). Adopting it today would mean mounting it
on nothing. But the blocking dependency is one ~5-line example change away,
and once it lands the dashboard — and especially its zero-code federation hub
binary — deletes the per-app status-page glue the fleet would otherwise
re-write in every project. The repo's current answer to "is it healthy?" is a
stdout text dump (`Doctor`) that no browser, no NOC monitor, and no hub can
consume.

## 1. What the library is

A real-time health dashboard for go-health, powered by Datastar SSE: one
`dashboard.New(probe)` + `dash.RegisterRoutes(mux)` gives a live status page
on `/health` (HTML for browsers, JSON via `Accept: application/json`) while
kubelet probes (`/healthz`, `/readyz`, `/startupz`) stay go-health JSON
(go-health-dashboard/README.md:15-19,100-122). Severity-grouped service
tables with pass/warn/fail badges and "since/duration" per check, trend
sparklines + CSV/JSON export (`WithTrend`), Prometheus exposition
(`WithMetrics`), webhook push with HMAC signing, auth middleware + native
OIDC login, rate limiting, CSP nonce handling, aggregate multi-probe
grouping, and a **federation hub binary** (`nix run .#health-hub`) that
merges remote services' go-health documents into one fleet dashboard with
zero code — env vars only (go-health-dashboard/README.md:393-443). Heavy by
design and honestly so: templ + templ-components + go-datastar + go-sse
(go-health-dashboard/README.md:533-549) — which is exactly why it lives in a
separate repo from the single-dependency go-health
(go-health-dashboard/README.md:22-27). HEAD 19af9cf (2026-11-20), v0.13.3.

## 2. Usage census in go-cqrs-lite (evidence)

`rg -n 'go-health-dashboard' -g go.mod` over the repo: **zero hits**; no
imports. The alternative surface that exists instead:

- **`Doctor` is the de-facto status page.** `metaengine/v4/explain.go:256-259`
  renders `--- Health ---` and `--- Engine Health (ADR-0137) ---` sections
  into a plain-text report; `example/goal-shaped-app/main.go:62` prints it to
  stdout. Human-readable, machine-hostile: no JSON, no badge semantics, no
  history.
- **`HealthCheckDetailed` is the would-be JSON source** — its own doc says
  "suitable for detailed dashboards and debugging"
  (`system/v4/introspection_extended.go:55`) — but the `EngineHealth` shape
  (`system/v4/introspection_extended.go:12-13`) is bespoke: no
  pass/warn/fail, no `since`, no uptime/version scalars. Nothing in the repo
  renders it (grep: only tests and the method itself).
- **The example serves no status page at all**: the only health route is the
  static `{"status":"ok"}` handler (`example/taskmanager/http.go:63-65`).
- **docserver is a docs server, not an ops page**
  (`catalog/v4/docserver/`): it serves generated documentation HTML; health
  is not among its routes.
- The repo already standardizes the dashboard's _transport_ for other
  purposes: taskmanager streams query updates over go-sse via
  `metaengine.ServeSSE` (`example/taskmanager/http.go:55-60`) — the same
  go-sse that powers the dashboard's live patches.

## 3. Capability inventory

| # | Capability                                                                                | Grade  | Evidence / reason                                                                                                                                       |
| - | ----------------------------------------------------------------------------------------- | ------ | ------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1 | Live HTML status page on `/health` (content-negotiated)                                   | Missed | no status page exists; closest artifact is stdout `Doctor` text (`metaengine/v4/explain.go:256-259`, `example/goal-shaped-app/main.go:62`)              |
| 2 | K8s probe JSON endpoints (`/healthz`,`/readyz`,`/startupz`) via the dashboard's route set | Missed | inherited from unadopted go-health; example's `/health` is static (`example/taskmanager/http.go:63-65`) — see `go-health.md` rows 1–2                   |
| 3 | Trend history + CSV/JSON export                                                           | Missed | no health history retained anywhere; projection lag (`projectionhost/v4/host.go:332-338`) is the obvious trend feed and goes unrecorded                 |
| 4 | Federation hub: one fleet-wide dashboard, zero app code                                   | Missed | `HEALTH_HUB_REMOTES`-driven hub (go-health-dashboard/README.md:393-443) has nothing to federate: no fleet service currently serves a go-health document |
| 5 | Prometheus metrics (`WithMetrics`) / webhook push                                         | N/A    | meaningless until a probe exists; consumer's `prometheus` module is the eventual sink — flip with rows 1–2                                              |
| 6 | Auth middleware / native OIDC login                                                       | N/A    | no public-facing status surface in the fleet today — flip when a hub is exposed beyond localhost                                                        |
| 7 | Rate limiting, CSP nonce, SSE connection caps                                             | N/A    | hardening options for internet-facing mounts; example/docserver are loopback-grade — flip on first public deployment                                    |
| 8 | Aggregate multi-probe grouping (`GroupBySource`)                                          | N/A    | requires multiple in-process probes; consumer has one `System` per process                                                                              |
| 9 | `dash.HealthCheck` / samber/do registration                                               | N/A    | consumer does not use samber/do                                                                                                                         |

Score derivation: rows N=9, N/A rows 5–9 → **A=4**. Full=0, Partial=0,
Missed rows 1–4 (4 × 0). Score = round(100 × 0 / 4) = **0% → low band**.
Low is honest: the block is upstream (go-health), not this library's fault.

## 4. Gap analysis

1. **The chain, not the link, is missing.** Dashboard → go-health probe →
   `System.HealthCheck` — only the last link exists. Every dashboard
   capability grades Missed for the same root cause, so the fix belongs in
   `go-health.md` opportunity 1–2 (bridge + example), after which the
   dashboard is a 4-line mount (`dashboard.New(probe)` +
   `RegisterRoutes`, go-health-dashboard/README.md:41-78).
2. **`Doctor` and the dashboard answer different questions and should
   coexist.** Doctor is deep diagnostics (explain plans, capability audits,
   quarantine state — `metaengine/v4/explain.go:256-259`); the dashboard is
   triage (badges, since-stamps, trends — go-health-dashboard/README.md:205-243).
   The mistake to avoid is growing Doctor toward HTML/JSON rendering, or
   growing the dashboard toward explain-plans. A thin bridge that feeds
   engine health + projection lag into go-health checks gives the dashboard
   the CQRS-specific rows (per-engine state, `LagDuration`) without either
   drift.
3. **The hub is the fleet's collective asset and has zero prerequisites
   beyond row 2 of `go-health.md`.** Once any fleet service (starting with
   the taskmanager example) serves go-health JSON, `health-hub` renders the
   whole fleet from env vars — the exact "fix bugs once, every project
   benefits" shape. Today the fleet would hand-roll status pages per app
   instead; nothing does, which is why nobody can see anything.

## 5. Version currency

No pin (zero requires). Latest local tag **v0.13.3**; HEAD 19af9cf
(2026-11-20). The dashboard pins `go-health v0.10.0`
(go-health-dashboard/README.md:546) — identical to go-health's latest tag,
so a combined adoption lands version-aligned with no reconciliation.
Dependency weight (templ, templ-components, go-datastar, go-sse,
prometheus/common — go-health-dashboard/go.mod:5-20) rules out any
production-module pin under dep budgets; the only sane mount points are
`example/` apps or fleet apps themselves, which is consistent with the
library's own separation rationale (go-health-dashboard/README.md:22-27).

## 6. The fleet argument

The owner's fleet lens is at its sharpest here. A status page is the
single most re-implemented, least-differentiating piece of ops glue in
existence — and the fleet already owns a good one with auth, rate limiting,
CSP, trend, metrics, and a no-code multi-service hub. go-cqrs-lite is the
template its apps clone; if the template demonstrates
`dashboard.New(probe)` once, every fleet app gets a NOC-viewable page and
the hub gets a remotes list for the price of env vars. If it doesn't, each
app will either ship nothing (today's state: static `ok` JSON, invisible
outages) or hand-roll a page (the bug-fixing treadmill the owner explicitly
wants to escape). The dependency-ordering caveat justifies non-adoption
_today_; it does not justify non-adoption _by default_.

## 7. Ranked opportunities

1. **Land the go-health bridge first** (high; prerequisite): execute
   `go-health.md` opportunity 1 — a real probe over `System.HealthCheck` in
   the taskmanager example. Nothing dashboard-shaped should precede it.
2. **Mount the dashboard in the example behind the probe** (medium, ~15
   lines): `dash := dashboard.New(probe, dashboard.WithTitle("taskmanager"))`
   (go-health-dashboard/README.md:71) + `RegisterRoutes`; the browser gets a live page, the kubelet keeps JSON.
   Add per-engine rows (quarantine state from `HealthSnapshot`,
   `metaengine/v4/engine_health.go:169-176`) and a `projection-lag` check fed
   by `projectionhost/v4/host.go:332-338` so the page shows CQRS-native
   signals, not generic pings.
3. **Run `health-hub` for the fleet** (medium, ops-only): once two fleet
   services expose go-health JSON, a single host with
   `HEALTH_HUB_REMOTES=a=...,b=...` is the collective dashboard — zero code
   in any app. Record it in fleet runbook docs, not in this repo's modules.
4. **Feed the existing `prometheus` module from `WithMetrics`** (low): or
   point the dashboard's exposition at the existing scrape setup — one of
   the two, decided once.
5. **Do not grow Doctor toward HTML** (guardrail, zero cost): keep the
   diagnostics/dashboard boundary explicit in the Doctor doc comment so the
   next audit doesn't find drift.

## 8. Reconsider when

- The go-health seam lands in the example (flip trigger for opportunities
  2–4; this report's verdict flips from JUSTIFIED to GAP at that moment if
  the dashboard is still absent).
- A second fleet service needs monitoring — the hub's fixed cost amortizes
  across the fleet from the second remote onward.
- Anyone proposes hand-rolling a status/health HTML page in any fleet app —
  that PR is the signal this library should have been used.
- The dashboard ships a feature the fleet needs collectively (OIDC login,
  webhooks) that would otherwise be built per-app.

## 9. How to verify

```
rg -n 'go-health-dashboard' -g go.mod /home/lars/projects/go-cqrs-lite   # zero hits
rg -n 'larsartmann/go-health-dashboard' --type go /home/lars/projects/go-cqrs-lite  # zero imports
sed -n '256,259p' /home/lars/projects/go-cqrs-lite/metaengine/v4/explain.go
sed -n '62p'     /home/lars/projects/go-cqrs-lite/example/goal-shaped-app/main.go
sed -n '55,65p'  /home/lars/projects/go-cqrs-lite/example/taskmanager/http.go
sed -n '12,13p;53,58p' /home/lars/projects/go-cqrs-lite/system/v4/introspection_extended.go
sed -n '15,27p;100,122p;377,443p;533,549p' /home/lars/projects/go-health-dashboard/README.md
git -C /home/lars/projects/go-health-dashboard log -1 --format='%H %cs'  # 19af9cf 2026-11-20
git -C /home/lars/projects/go-health-dashboard tag --list | sort -V | tail -1  # v0.13.3
```

Related: `go-health.md` (this directory) — the prerequisite this report
inherits its verdict from.
