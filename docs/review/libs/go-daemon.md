# go-daemon — Utilization Audit · go-cqrs-lite

> Audited 2026-11-22 · Library `github.com/LarsArtmann/go-daemon` @ b6af9f1 (latest local tag v0.4.0) · Consumer go-cqrs-lite @ feec9bc74 · Group G3

**Verdict:** NOT-ADOPTED-JUSTIFIED
**Score:** 0/1 capabilities fully leveraged = 0% (low)

Zero `require`s — and for once the zero is defensible almost across the
board: go-cqrs-lite is a library, its examples are foreground HTTP servers,
and nothing in the repo speaks unix domain sockets or systemd. The one row
that survives the N/A filter is lifecycle ordering, and it is telling: the
golden-path example hand-rolls a ~50-line signal/drain choreography that
duplicates the library's core product, and the hand-roll already contains
the exact class of ordering wart (drain consumers _before_ stopping intake)
that go-daemon's explicit `Startup → OnListen → OnShutdown` contract exists
to make structurally impossible.

## 1. What the library is

"Production-proven building blocks for superb long-lived daemons served over
unix domain sockets" — mechanism, not policy
(go-daemon/README.md:3-9). The pieces: `SocketServer` owns the full socket
lifecycle — stale-socket cleanup, directory creation, `0600` permissions,
`/run` placement, socket removal on every exit path — behind an explicit
ordered `Lifecycle` hook contract (go-daemon/README.md:25);
`NewServer("tcp", …)` / `WithListener` give the same lifecycle on any
network (go-daemon/README.md:26); `NotifyReady`/`NotifyStopping`/`Watchdog`
are standalone sd_notify integration with watchdog keepalive at
`WatchdogSec/2` (go-daemon/README.md:27); `NewHTTPClient` is a unix-socket
HTTP transport (go-daemon/README.md:28); `ResolveRequestCodec`/
`ResolveResponseCodec`/`WriteEncoded` do JSON/CBOR content negotiation;
`ParseSSEData` is a WPT-corpus-pinned WHATWG SSE parser for clients;
`FlightRecorder` wraps go-flightrecorder with slow-request auto-trigger,
Prometheus metrics, and a rate-limited `TraceHandler`
(go-daemon/README.md:32); `IsSocketAlive` probes socket liveness
(go-daemon/README.md:33). Deps: `coreos/go-systemd/v22`, `go-codec`,
`go-flightrecorder`, `go-sse/sseparse`, `prometheus/client_golang`
(go-daemon/go.mod:5-11). Note the module path is
`github.com/LarsArtmann/go-daemon` (capitalized). Go 1.27.2; v0.4.0.

## 2. Usage census in go-cqrs-lite (evidence)

`rg -n 'go-daemon' -g go.mod` over the repo: **zero hits**; no imports. The
alternative surface that exists instead:

- **The example hand-rolls daemonization.** `example/taskmanager/setup.go:368-374`:
  `signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)`, receive, log,
  return into a deferred `Stop()`. `Stop()`
  (`example/taskmanager/setup.go:290-320`) then hand-sequences the drain:
  projection host stop at :296, HTTP `Shutdown` at :301, OTel provider,
  watcher, work queue, DLQ. That is a bespoke re-implementation of the
  lifecycle-ordering product (go-daemon/README.md:25-26).
- **The drain order has a wart.** Projections stop (`example/taskmanager/setup.go:296`) _before_
  the HTTP server stops accepting (`example/taskmanager/setup.go:301`): during the drain window
  the API is still serving, and any query arriving between the two hits a
  stopped projection host. Small, real, and precisely the bug class the
  library's ordered hook contract prevents.
- **A second hand-roll exists in a demo:**
  `metaengine/irohengine/quic/v4/demo/main.go:258-263` (`waitForSignal`) —
  signal wait, print, die; no drain at all (acceptable for a demo, but it is
  the second independent copy of the pattern).
- **The repo polices this itself:** lint rule E017 flags "signal.Notify
  without GracefulClose/Stop — in-flight events and projections are killed
  without draining" (`cmd/cqrs-lint/v4/pkg/rules/architecture/e017.go:14-17,72`).
  taskmanager passes only because of its hand-written `Stop`.
- **No unix-socket or systemd surface anywhere**: the only "unix socket"
  mentions in the repo are prose about engine network locality
  (`metaengine/pgengine/v4/probe.go:9`); no `net.Listen("unix"`, no
  `sd_notify`, no systemd units.

## 3. Capability inventory

| # | Capability                                                                                  | Grade  | Evidence / reason                                                                                                                                                                                                                                                            |
| - | ------------------------------------------------------------------------------------------- | ------ | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1 | Lifecycle ordering (Startup → OnListen → OnShutdown; drain before/after intake by contract) | Missed | hand-rolled ~50-line choreography (`example/taskmanager/setup.go:290-320,368-374`) that duplicates the lib's product and already drains projections before stopping intake (`example/taskmanager/setup.go:296` vs `:301`) — duplication with a wart, not absence of function |
| 2 | `SocketServer` unix-socket lifecycle (stale cleanup, 0600, /run, removal on all exits)      | N/A    | zero unix-socket listeners in the repo — flip: first CLI tool gaining a server/daemon mode over UDS (e.g. a cqrs-gen watch server)                                                                                                                                           |
| 3 | sd_notify (READY/STOPPING/WATCHDOG)                                                         | N/A    | nothing ships systemd units — flip: first fleet deployment under systemd `Type=notify`, where the example's log line (`example/taskmanager/setup.go:372`) must become a real notify or the unit hangs                                                                        |
| 4 | Unix-socket HTTP client (`NewHTTPClient`)                                                   | N/A    | no UDS clients exist — arrives together with row 2                                                                                                                                                                                                                           |
| 5 | Content negotiation codecs (`ResolveRequestCodec`, `WriteEncoded`)                          | N/A    | owned by the go-codec audit (`go-codec.md`, this directory); graded there to avoid double-counting — go-daemon re-exports the same capability                                                                                                                                |
| 6 | `ParseSSEData` (client-side WHATWG SSE framing)                                             | N/A    | consumer _serves_ SSE via go-sse (`example/taskmanager/http.go:55-60`); no SSE client dispatch exists — flip: any fleet tool that consumes another service's SSE feed                                                                                                        |
| 7 | `FlightRecorder` (slow-request trace capture)                                               | N/A    | no long-running production daemon in this repo to instrument; benchmarks use benchkit — flip: first fleet service with intermittent slow engine folds in production, where the lighter path is `go-flightrecorder` directly (go-daemon/go.mod:8)                             |
| 8 | `IsSocketAlive` probe                                                                       | N/A    | no socket files to probe — arrives together with row 2                                                                                                                                                                                                                       |

Score derivation: rows N=8, N/A rows 2–8 → **A=1**. Full=0, Partial=0,
Missed row 1 (0). Score = round(100 × 0 / 1) = **0% → low band**. Low is
the honest reading of "leveraged": the consumer achieves the row-1
_outcome_ hand-rolled, so the deficit is duplication-plus-wart, not missing
function.

## 4. Gap analysis

1. **One example, one demo, two hand-rolled signal waits — and a lint rule
   policing the pattern.** E017 (`cmd/cqrs-lint/v4/pkg/rules/architecture/e017.go:14-17`)
   exists because the fleet knows SIGTERM drains go wrong; the golden path
   answers it with a bespoke 30-line `Stop()` whose ordering already has a
   window where live traffic meets stopped projections
   (`example/taskmanager/setup.go:296,301`). The gap is not "adopt
   go-daemon in the example" (its socket-systemd weight is wrong for a
   teaching HTTP example) but "the example should stop intake first" — a
   two-line reorder — and the lifecycle lesson should be recorded where E017
   points consumers.
2. **The socket/systemd core is genuinely out of scope today.** No consumer
   surface speaks UDS or systemd; forcing adoption would be
   dependency-driven architecture. The flip triggers are concrete (rows
   2–4, 7–8) and the first one to fire is likely a `cqrs-gen` watch-server
   or a fleet systemd deployment.
3. **FlightRecorder is the sleeper capability for CQRS.** Slow folds are
   exactly the "slow-request auto-trigger" use case
   (go-daemon/README.md:32), and the consumer already has the timing
   signals (`projectionhost` staleness) to feed a trigger. But the right
   first move is the underlying `go-flightrecorder` (one dep), not this
   five-dep wrapper — unless the hosting app is already a go-daemon
   service.

## 5. Version currency

No pin (zero requires). Latest local tag **v0.4.0**; HEAD b6af9f1
(2026-11-18). Requires Go 1.27.2 — consumer modules are on go 1.27.2
(`example/taskmanager/go.mod:3`), so toolchain compatibility is trivial.
Dependency weight (systemd/v22, prometheus/client_golang, codec, flight
recorder — go-daemon/go.mod:5-11) makes any production-module pin a
non-starter under per-module dep budgets; the only sane mount points are
fleet _apps_ and repo _tools_ (cmd/), which is where the flip triggers aim.
Note the module path casing (`LarsArtmann`) differs from the fleet's usual
`larsartmann` — a go.mod typo waiting for whoever adopts first.

## 6. The fleet argument

The owner's fleet lens cuts both ways here, and honesty favors the library.
Against adoption: this consumer is a library plus teaching examples; nothing
daemon-shaped exists; pulling a systemd+prometheus dependency tree into
either would be architecture by dependency. For adoption: lifecycle
ordering is exactly the kind of thing every fleet app re-implements
slightly wrong (the repo's own E017 rule and the taskmanager drain-window
wart are the proof), and go-daemon is where that gets fixed once. The
fleet-correct posture is: keep go-cqrs-lite itself clean; when the first
fleet app genuinely needs a socket daemon or systemd notify, start it from
go-daemon rather than from a copy of `example/taskmanager/setup.go:368-374`
— and fix that example's drain order now, so the copies that inevitably
happen are at least ordered right.

## 7. Ranked opportunities

1. **Fix the example's drain order** (high, two lines): in
   `example/taskmanager/setup.go:290-320`, call `s.httpServer.Shutdown(ctx)`
   before `s.ProjHost.Stop()` — stop intake, then drain — and add a comment
   naming the ordering rule. Every clone inherits the fix; E017's teaching
   gets a correct reference answer.
2. **Point E017's remediation text at a canonical pattern** (medium): the
   rule suggests GracefulClose/Stop
   (`cmd/cqrs-lint/v4/pkg/rules/architecture/e017.go:72`); link the
   intake-first ordering rule there so consumers learn the order, not just
   the call.
3. **Record the flip triggers in fleet docs** (low): "first UDS-speaking
   tool → go-daemon `SocketServer`; first systemd `Type=notify` deployment
   → `NotifyReady`/`NotifyStopping`; first prod trace-capture need →
   `go-flightrecorder` directly". Makes the no-decision durable instead of
   re-litigated.
4. **Consolidate the second hand-roll** (low): `metaengine/irohengine/quic/v4/demo/main.go:258-263`
   can keep its trivial wait, but a one-line comment should say it is a
   demo shortcut, not the pattern.

## 8. Reconsider when

- Any cmd/ tool gains a long-running server mode (watch server, docserver
  daemon) — `SocketServer` + `WithListener` become the default answer
  (rows 2, 4, 8 flip).
- A fleet app is deployed under systemd `Type=notify` — rows 3 and 7 flip;
  READY/STOPPING semantics and watchdog keepalive stop being optional.
- A fleet service shows intermittent slow folds in production — trace
  capture flips from N/A to wanted; prefer `go-flightrecorder` directly
  unless the service already runs on go-daemon.
- The example's drain-order wart causes a real incident (requests failing
  during every deploy's drain window) — then opportunity 1 becomes urgent
  and the lifecycle-contract argument for the library strengthens.

## 9. How to verify

```
rg -n 'go-daemon' -g go.mod /home/lars/projects/go-cqrs-lite        # zero hits
rg -n 'LarsArtmann/go-daemon' --type go /home/lars/projects/go-cqrs-lite  # zero imports
sed -n '290,320p;368,374p' /home/lars/projects/go-cqrs-lite/example/taskmanager/setup.go
sed -n '256,263p' /home/lars/projects/go-cqrs-lite/metaengine/irohengine/quic/v4/demo/main.go
sed -n '14,17p;70,74p' /home/lars/projects/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/rules/architecture/e017.go
sed -n '19,36p' /home/lars/projects/go-daemon/README.md
head -11 /home/lars/projects/go-daemon/go.mod
git -C /home/lars/projects/go-daemon log -1 --format='%H %cs'   # b6af9f1 2026-11-18
git -C /home/lars/projects/go-daemon tag --list | sort -V | tail -1  # v0.4.0
```

Related: `go-codec.md` (content-negotiation capability, this directory) and
`go-health.md` (the probe side of daemon readiness).
