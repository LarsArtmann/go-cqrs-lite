# go-health — Utilization Audit · go-cqrs-lite

> Audited 2026-11-22 · Library `github.com/larsartmann/go-health` @ 3ca6644 (latest local tag v0.10.0) · Consumer go-cqrs-lite @ feec9bc74 · Group G3

**Verdict:** NOT-ADOPTED-GAP
**Score:** 0/8 capabilities fully leveraged (2 more partial) = 13% (low)

Zero `require`s — correct for the library's core modules, which must stay
transport- and platform-agnostic under per-module dep budgets. But the gap is
real and cheap to close: the repo has a rich *diagnostics* health surface
(`System.HealthCheck`, `Doctor`, ADR-0137 quarantine, projection lag) and an
empty *probe* layer on top of it. The blessed example answers `/health` with a
hard-coded `{"status":"ok"}` that never consults any of it, and the repo's own
lint rule (E016) mandates Kubernetes probes for server-mode projects that no
example or recipe demonstrates. `health.NewChecks` accepts exactly the
`func(ctx) error` shape the repo already exposes, so the bridge is ~5 lines in
the example — and until it exists, every fleet app cloned from the template
ships the same lying health endpoint.

## 1. What the library is

A Kubernetes health-probe SDK implementing the three-probe pattern
(liveness / readiness / startup) as one `Probe` type: liveness never checks
dependencies, readiness gates only on *critical* services (non-critical
failures degrade to `warn`/200 instead of restarting the pod), startup latches
after boot. Background caching (1s default), shutdown-aware readiness drain,
method-set enforcement, panic-hardened check execution, a programmatic API
(`Ready()`, `AwaitReady(ctx)`, `Healthz()`), an evaluation-hook observability
seam for Prometheus/OTel, an `aggregate` sub-package for merging in-process
probes, a `federation` sub-package for merging remote probes over HTTP, and a
zero-dependency `checks` battery set (Disk, Memory, HTTP, Database). Explicit
non-goal (its own words): it "is a **probe**, not a diagnostics page"
(go-health/README.md:234). Three constructors matter here: injector-based
`New`, `NewWithHealthCheck(fn)`, and `NewChecks(map[string]CheckFunc)` where
`CheckFunc` is `func(ctx context.Context) error` (go-health/checks.go:15) —
run concurrently, timed, panic-recovered, batch-deadlined
(go-health/README.md:226-228). Deps: `samber/do/v2` v2.1.0, `go-branded-id`,
`go-type-to-string` (go-health/go.mod:5-9); Go 1.27+; v0.10.0 is alpha-stable
("three-probe API surface is stable; internal details may change before v1.0").

## 2. Usage census in go-cqrs-lite (evidence)

`rg -n 'go-health' -g go.mod` over the 104 go.mod files: **zero hits**. No
import sites. The real evidence is the alternative surface that exists
instead — the repo's own health stack, which is substantial but probe-shaped
nowhere:

- `system/v4/introspection.go:138-154` — `System.Health(ctx) string` returns a
  joined string (`"ok projections:running"`); stringly-typed, no
  pass/warn/fail classification, no per-component detail.
- `system/v4/introspection.go:156-190` — `System.HealthCheck(ctx) error`:
  first-error-only aggregation across engines + projection workers; its doc
  comment says "Use this for Kubernetes liveness/readiness probes"
  (`introspection.go:163`) — the exact single-endpoint conflation go-health's
  opening argument rejects: "A single `/health` endpoint conflates 'process
  alive' with 'dependencies reachable.' … a restart cascade follows"
  (go-health/README.md:45).
- `system/v4/introspection_extended.go:12-13,53-87` — `HealthCheckDetailed`
  returns a bespoke `[]EngineHealth` for "detailed dashboards and debugging"
  (its own doc, `introspection_extended.go:55`) — a hand-rolled, less
  classified cousin of go-health's `Response.Checks`.
- `metaengine/v4/stats.go:78-83` — `HealthChecker` interface
  (`HealthCheck(ctx) error`), "for liveness/readiness probes"; engines
  implement it so "consumers can wire Kubernetes-style health checks".
- `metaengine/v4/engine_health.go:24-42,169-176` — ADR-0137 quarantine states
  (`EngineActive`/`EngineQuarantined`) and `HealthSnapshot()`;
  `metaengine/v4/engine_stats.go:47-50,68-77` folds them into
  `GetEngineStats`; `metaengine/v4/explain.go:256-259` renders them into the
  `Doctor` report. This is *diagnostics*, i.e. the layer go-health explicitly
  does not want to be — complementary, not duplicate.
- `projectionhost/v4/host.go:332-338` — `LagDuration()`, whose doc suggests
  wiring it to a gauge: a health signal with no probe surface.
- `example/taskmanager/http.go:63-65` — the blessed example's entire probe
  layer: `handleHealth` writes `{"status":"ok"}` unconditionally, never
  calling `s.HealthCheck`/`HealthCheckDetailed`. A dead store still reports
  ok. `example/taskmanager/main.go:16` advertises it as "liveness probe".
- `cmd/cqrs-lint/v4/pkg/rules/architecture/e016.go:126,133` — lint rule E016
  warns "Server-mode project without HealthCheck — Kubernetes probes need
  stack.Bundle.HealthCheck()" and suggests adding it "to your /healthz or
  /readyz endpoint" — the repo *mandates* the pattern it never demonstrates.
- `example/goal-shaped-app/main.go:62` — prints `Doctor(ctx)` to stdout: the
  example story for health is "print diagnostics", not "serve probes".

## 3. Capability inventory

| # | Capability | Grade | Evidence / reason |
|---|---|---|---|
| 1 | Three-probe HTTP handlers (`/healthz`,`/readyz`,`/startupz`) | Missed | example serves one hand-rolled `/health` (`example/taskmanager/http.go:63-65`); E016 demands `/healthz` or `/readyz` (`cmd/cqrs-lint/v4/pkg/rules/architecture/e016.go:133`) |
| 2 | Critical vs non-critical classification (warn, don't restart) | Missed | `System.HealthCheck` fails on *any* engine error (`system/v4/introspection.go:172-180`); quarantined-but-failover-covered engines (ADR-0137) would flap readiness when the system is serving fine from a healthy engine — the exact case that should be `warn` |
| 3 | Startup latch (slow-boot tolerance) | Partial | `System` tracks `started` and reports `projections:not-started` (`system/v4/introspection.go:146-150`), but it is informational only — nothing gates traffic on it and there is no one-way latch |
| 4 | Shutdown-aware readiness drain (503 while stopping) | Partial | `HealthCheck` returns `ErrSystemStopped` when stopped (`system/v4/introspection.go:168-170`), but nothing in the examples flips probes during graceful drain |
| 5 | Background caching / throttle (kubelet poll ≠ dependency hammering) | Missed | `HealthCheck`/`HealthCheckDetailed` evaluate every engine synchronously on every call (`system/v4/introspection.go:172-189`) |
| 6 | Structured classified response (`pass`/`warn`/`fail`, per-check `since`, `duration_ns`) | Missed | string `Health` (`system/v4/introspection.go:139-154`) and bespoke `[]EngineHealth` (`system/v4/introspection_extended.go:12-13`) — same information, non-standard shape, no dashboard can consume it |
| 7 | Injector-free bridge (`NewChecks`/`NewWithHealthCheck`) | Missed | the seam is *structurally free*: `CheckFunc` = `func(ctx) error` (go-health/checks.go:15) ≡ `metaengine.HealthChecker` (`metaengine/v4/stats.go:81-83`) ≡ `stack.HealthChecker` (`stack/v4/health.go:14-16`); no example/recipe exploits it |
| 8 | Observability seam (evaluation hook → Prometheus) | Missed | repo has a `prometheus` module and a doc-suggested lag gauge (`projectionhost/v4/host.go:337`), but health status is exported to no metrics anywhere |
| 9 | Aggregation sub-package (multiple in-process probes) | N/A | one system per process in current examples — flip: an app embedding several independent `System`s |
| 10 | Federation sub-package (remote probes → one hub) | N/A | fleet-deployment concern, not a library one — covered in go-health-dashboard.md |
| 11 | `checks` batteries (Disk/Memory/HTTP/Database) | N/A | engine pings are domain checks, not the stdlib resource set — flip: first fleet app wanting disk ceilings under a SQLite/pebble data dir |
| 12 | samber/do injector path, version endpoint, method enforcement | N/A | consumer uses no `samber/do`; identity/method policy are app concerns that arrive for free with any future adoption |

Score derivation: rows N=12, N/A rows 9–12 → **A=8**. Full=0, Partial=2
(0.5 + 0.5 = 1.0), Missed rows 1,2,5,6,7,8 (6 × 0). Score =
round(100 × 1.0 / 8) = **13% → low band** (0 full rows, so the M/A
fraction reads 0/8; the two partials carry the score).

## 4. Gap analysis

1. **The example's probe lies.** `example/taskmanager/http.go:63-65` answers
   `ok` statically; the system underneath has three health surfaces
   (`Health`, `HealthCheck`, `HealthCheckDetailed`) none of which are called.
   Under the fleet lens this is the template bug every clone inherits: K8s
   will never restart a genuinely broken taskmanager because the liveness
   endpoint cannot fail.
2. **Duplication is at the roll-up layer, not the diagnostics layer.**
   `Doctor`, quarantine, `GetEngineStats` are diagnostics and *should* stay —
   go-health itself refuses that job (go-health/README.md:234). What the repo
   re-implements poorly is the probe tier: string `Health`, first-error
   aggregation, conflated liveness/readiness (its own doc comment,
   `system/v4/introspection.go:163`) — each a weaker version of a go-health
   primitive. The library core must not depend on go-health (ADR-0127
   transport-agnosticism + dep budgets) and doesn't need to: the seam is
   signature-identical.
3. **The lint rule points at a bridge that doesn't exist.** E016 tells every
   server-mode consumer to wire `bundle.HealthCheck(ctx)` into `/healthz` or
   `/readyz` (`cmd/cqrs-lint/v4/pkg/rules/architecture/e016.go:126,133`) with
   no recipe, no example, and no classification story (what is critical vs
   warn for a CQRS app? quarantined engine with healthy failover = warn;
   projection worker failed = critical, per `system/v4/introspection.go:182-189`).

## 5. Version currency

No pin (zero requires). Latest local tag **v0.10.0** (2026-11-20, two days
before this audit); HEAD 3ca6644. Self-declared alpha ("v0.10.0 alpha. The
three-probe API surface is stable; internal details may change before
v1.0" — go-health/README.md:11). Requires Go 1.27+ and `samber/do/v2` v2.1.0;
consumer modules already build on go 1.27.2 (e.g.
`system/v4/go.mod:3`), so toolchain compatibility is a non-issue. Adoption
cost is 3 deps for the example module only — inside every reasonable budget,
since it would land in `example/`, not a production module.

## 6. The fleet argument

The owner's lens: "if every project re-integrates everything I will only fix
bugs all day long." Health endpoints are the canonical re-integration bug:
every fleet app hand-rolls `/health`, each slightly wrong, none inheriting
the restart-cascade fix (go-health/README.md:45), the warn-tier semantics, or
the caching. go-health is where that gets fixed once. go-cqrs-lite already
did the hard part — it *has* the check functions (`System.HealthCheck`,
per-engine `HealthChecker`, quarantine, lag) — and lacks only the 5-line
bridge and one honest example. Until the template demonstrates it, E016
creates demand for a pattern the fleet's own reference repo never shows, so
every consumer reinvents the probe tier differently. That is precisely the
outcome the fleet exists to prevent.

## 7. Ranked opportunities

1. **Make the example's `/health` real** (high, ~30 min): replace
   `example/taskmanager/http.go:63-65` with a go-health probe over
   `NewChecks`:

   ```go
   // skip-validate
   probe := health.NewChecks(map[string]health.CheckFunc{
       "engines":     sys.HealthCheck, // engines + projections, first error
   }, health.WithCriticalServices("engines"))
   ```

   `sys.HealthCheck` (a method value of type `func(context.Context) error`)
   assigns directly to `health.CheckFunc` (go-health/checks.go:15). Register
   `probe.RegisterRoutes(mux, health.DefaultRoutes())`; keep the static
   handler only as liveness if desired.
2. **Split critical vs warn using signals the system already has** (medium):
   quarantined engine with live failover → warn (not 503); projection
   `WorkerFailed` (`system/v4/introspection.go:184-188`) → critical. This is
   a `NewWithHealthCheck` returning `map[string]error` — still no library dep.
3. **Add a recipe** ("Wire system health into K8s probes") beside the E016
   suggestion so the lint rule's fix has a canonical answer in
   recipes.md rather than N per-app inventions.
4. **Export health status through the existing prometheus module** (low):
   `WithEvaluationHook` is the library's sanctioned seam
   (go-health/README.md:453-467); alternatively document `LagDuration` +
   `HealthSnapshot` gauges now and reconcile later.
5. **Retire or justify `System.Health` the string** (low): with a real probe,
   `"ok projections:running"` is a third, redundant shape; fold into
   `HealthCheckDetailed` or mark deprecated.

## 8. Reconsider when

- Any fleet app deploys the example twin (or a clone) under Kubernetes — the
  lying `/health` flips from cosmetic to outage-hiding; opportunity 1 becomes
  mandatory.
- A second fleet app needs health: then the bridge belongs in a shared
  example/recipe once, not in each app.
- go-health hits v1.0 (API freeze) — removes the alpha caveat for depending
  on it from any module that ever wants to.
- An engine's `HealthCheck` is ever observed flapping under kubelet poll
  load — the missing cache (capability 5) stops being theoretical.

## 9. How to verify

```
rg -n 'go-health' -g go.mod /home/lars/projects/go-cqrs-lite          # zero hits
rg -n 'larsartmann/go-health' --type go /home/lars/projects/go-cqrs-lite  # zero imports
sed -n '138,190p' /home/lars/projects/go-cqrs-lite/system/v4/introspection.go
sed -n '53,87p'  /home/lars/projects/go-cqrs-lite/system/v4/introspection_extended.go
sed -n '78,83p'  /home/lars/projects/go-cqrs-lite/metaengine/v4/stats.go
sed -n '63,65p'  /home/lars/projects/go-cqrs-lite/example/taskmanager/http.go
sed -n '120,135p' /home/lars/projects/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/rules/architecture/e016.go
sed -n '43,60p;225,232p;415,452p' /home/lars/projects/go-health/README.md
sed -n '13,17p'  /home/lars/projects/go-health/checks.go
git -C /home/lars/projects/go-health log -1 --format='%H %cs'   # 3ca6644 2026-11-20
git -C /home/lars/projects/go-health tag --list | sort -V | tail -1  # v0.10.0
```

Related: `go-health-dashboard.md` (this directory) covers the consumer side
of the same seam.
