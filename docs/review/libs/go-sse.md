# go-sse — Utilization Audit · go-cqrs-lite

> Audited 2026-11-22 · Library `github.com/larsartmann/go-sse` @ 8acf04c (latest local tag v0.7.0; ssetest v0.5.0, sseparse v0.2.1) · Consumer go-cqrs-lite @ feec9bc74 · Group G2

**Verdict:** UNDERUSED-MODERATE
**Score:** 5/9 capabilities fully leveraged = 56% (moderate)

go-sse is the only library in this audit group with **production adoption
inside a library module**: `metaengine/v4` pins it directly and `ServeSSE` —
the sanctioned browser-delivery path since transport/* was deleted (ADR-0127)
— is built on it. But the adoption is confined to the wire-format layer: the
connection lifecycle, heartbeat loop, backpressure, reconnection header
parsing, and replay journal above it are all hand-rolled in metaengine, some
of it deliberately (documented ordering/backpressure semantics), some of it
plain duplication of go-sse API that sits one import away. Meanwhile the
consumer's SSE test harnesses mostly do NOT use the sibling ssetest/sseparse
modules the same repo ships — a raw-TCP parser and a hand-rolled bufio
scanner carry the load instead. Version pins are split across v0.6.2/v0.7.0.

## 1. What the library is

Server-Sent Events transport for Go in one root module plus two helper
modules (go-sse `README.md:1-25`):

- **Wire format** (`Event`, `EventID`, `WriteEvent`, `WriteHeartbeat`,
  `WriteRetry`, `SetHeaders`, `ContentType`) — allocation-minimized
  serialization, branded event IDs, multi-line data, CRLF handling
  (go-sse/event.go:33,167,239,248; go-sse/stream.go:66).
- **Stream** — one SSE connection: headers, `Send`/`SendData`/`SendJSON`/
  `SendLines`/`SendKeyed`, context, `Close`, `Heartbeat` (goroutine),
  `OnDisconnect`, `LastEventID`/`LastEventIDFromRequest`
  (go-sse/stream.go:83,111,227,246,279,305).
- **Broadcaster[T]** — generic subscriber fan-out with per-subscriber buffers
  (default `DefaultSubscriberBuffer` = 64), non-blocking broadcast with
  drop-newest per subscriber, `SubscribeFilter` predicates, hooks, `Shutdown`,
  `Health` (go-sse/broadcaster.go:68; go-sse/fanout.go:14,155,170,229).
- **Reconnection replay** — `EventStore` interface, `Replay`/
  `ReplayFiltered` (go-sse/replay.go:9,40,81).
- **DataStar wire support** — `KeyedLines`, `WriteKeyedLines`, `SendKeyed`
  (go-sse/lines.go:61,105; go-sse/doc.go "Keyed Data Lines").
- **ssetest** (v0.5.0) — one-liner handler testing: `Collect(t, handler)`,
  `WithLastEventID(sse.EventID)`, `NewStreamReader`, `Require*` assertions
  (go-sse/ssetest/collect.go:25; go-sse/ssetest/options.go:55).
- **sseparse** (v0.2.1) — the WHATWG § 9.2.6-conformant parser as a
  zero-dependency module, plus the WPT conformance corpus as data.

## 2. Usage census in go-cqrs-lite (evidence)

**Production (library code):**

- `metaengine/v4/go.mod` — `github.com/larsartmann/go-sse v0.6.2` **direct**
  (the only direct require in the whole repo).
- `metaengine/v4/sse.go:10` + `metaengine/v4/sse_loop.go:9` — imports.
  Symbols used: `sse.SetHeaders` (sse.go:140), `sse.WriteEvent` +
  `sse.Event{ID, Data}` (sse.go:179, 233-236, 274-277), `sse.NewEventID`
  (metaengine/v4/sse.go:234, 275), `sse.WriteHeartbeat` (metaengine/v4/sse_loop.go:127). Five symbols,
  all wire-format tier.
- Everything above the wire is metaengine's own: the event loop with
  timeout+heartbeat select (`sseMainLoop`, sse_loop.go:76-134), drop-old
  backpressure with a drop counter (`forwardWithDropOld`, sse_loop.go:33-71;
  `SSEDroppedEvents`, sse_loop.go:28), subscribe-before-header-commit
  ordering guarantee (metaengine/v4/sse.go:75-80, 135-145), the `SSEReplay` journal +
  dedup ring (metaengine/v4/sse_replay.go; metaengine/v4/sse.go:244-285), and Last-Event-ID parsing via
  `strconv.ParseUint` with malformed values silently ignored (metaengine/v4/sse.go:252-256).
- ADR-0127 (`docs/adr/0127-deprecate-transport-modules.md:31-32,53`) names
  go-sse as the sanctioned SSE path; cqrs-lint encodes it as transport
  detection (`cmd/cqrs-lint/v4/pkg/analyzer/feature_detect.go:253`) and in
  the deleted-module replacement text
  (`cmd/cqrs-lint/v4/pkg/rules/adoption/f030.go:53`).

**Test/example tier:**

- `example/taskmanager/go.mod` — `go-sse/ssetest v0.5.0` direct;
  `go-sse v0.6.2` and `go-sse/sseparse v0.2.1` indirect (pulled by ssetest).
- `example/taskmanager/sse_test.go:11,64` — `ssetest.NewStreamReader(resp.Body)`
  drives the one example SSE integration test. This is the repo's only
  ssetest/sseparse call site.
- `cmd/cqrs-lint/v4/pkg/rules/adoption/f030_test.go:77`,
  `.../architecture/e008_e011_test.go:140`,
  `.../analyzer/module_detect_test.go:183` — test-only imports feeding lint
  fixtures (go-sse v0.6.2 indirect in cqrs-lint's go.mod).

**Pin census:** 46 go.mod pin lines across the 104-module workspace — 41 at
v0.6.2 (metaengine/v4 direct + 40 indirect) and **5 at v0.7.0 (all
indirect)**: `integration/v4`, `scenario/v4`, `stack/bench/v4`,
`stack/memory/v4`, `stack/sqlite/v4`.

## 3. Capability inventory

| #  | Capability                                                                 | Grade       | Evidence / reason                                                                                                                                                                                                                                                                                   |
| -- | -------------------------------------------------------------------------- | ----------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1  | Wire format: `Event`/`WriteEvent`                                          | **Full**    | metaengine/v4/sse.go:179,233,274 (live + replay paths)                                                                                                                                                                                                                                              |
| 2  | Branded `EventID`/`NewEventID`                                             | **Full**    | every replay event carries `id: <seq>` (metaengine/v4/sse.go:234,275)                                                                                                                                                                                                                               |
| 3  | Heartbeats: `WriteHeartbeat`                                               | **Full**    | metaengine/v4/sse_loop.go:127 behind `WithSSEHeartbeat` (metaengine/v4/sse.go:59-62)                                                                                                                                                                                                                |
| 4  | `SetHeaders` + content type                                                | **Full**    | metaengine/v4/sse.go:140 (+ nginx `X-Accel-Buffering` alongside)                                                                                                                                                                                                                                    |
| 5  | `Stream` connection lifecycle (`Send`, `Context`, `Close`, `OnDisconnect`) | **Partial** | hand-rolled `sseMainLoop` instead; justified by drop-old + timeout + subscribe-before-commit semantics (metaengine/v4/sse.go:75-84), but `Stream` does not force header timing — the loop could still run on `Stream.Send`                                                                          |
| 6  | `LastEventIDFromRequest`/`ParseEventID`                                    | Missed      | re-rolled as `strconv.ParseUint` (metaengine/v4/sse.go:252-256); malformed IDs silently fall back to full replay instead of go-sse's validated `EventID`                                                                                                                                            |
| 7  | `Broadcaster[T]` fan-out                                                   | N/A         | the store-coupled `Watcher` hub (metaengine/v4/dx.go:29-36) is the correct layer for typed collection views — flip trigger: bridging one watcher to N HTTP subscribers without N store subscriptions                                                                                                |
| 8  | `EventStore` + `Replay`                                                    | N/A         | `SSEReplay[V]` is a typed value journal (records V + seq), the right granularity for read-model replay; deliberate layering per ADR-0127                                                                                                                                                            |
| 9  | `SubscribeFilter`                                                          | N/A         | key-filtered `Watch(ctx, key)` already pushes filtering into the store layer                                                                                                                                                                                                                        |
| 10 | `Event.Retry`/`WriteRetry` (reconnect pacing)                              | Missed      | ServeSSE never advertises a retry interval; reconnecting clients use the browser default (~3s) instead of a deliberate value                                                                                                                                                                        |
| 11 | ssetest in consumer's SSE tests                                            | **Partial** | 1 of 3 harnesses: taskmanager uses it (sse_test.go:64); metaengine's e2e tests speak raw TCP (`metaengine/v4/sse_replay_test.go:293,512` write hand-built `GET /events HTTP/1.0` + Last-Event-ID over `net.Conn` and substring-match the body); systemscenario hand-rolls a parser instead (row 12) |
| 12 | sseparse standalone parser                                                 | Missed      | `systemscenario/v4/sse.go:105-132` re-implements the SSE wire parser with `bufio.Scanner` (`data:` prefix + blank-line framing; drops `id:`/`event:`/`retry:` fields) — sseparse is the WHATWG-conformant, zero-dep replacement and would surface the `id:` field the harness currently discards    |
| 13 | KeyedLines/`SendKeyed` (DataStar wire)                                     | N/A         | no Datastar surface today — flip trigger: go-datastar adoption (see go-datastar.md)                                                                                                                                                                                                                 |

Score derivation: rows N=13, N/A rows 7–9, 13 → **A=9**. Full = 4 × 1.0,
Partial = 2 × 0.5 (rows 5, 11), Missed = 3 × 0 (rows 6, 10, 12).
Score = round(100 × 5.0 / 9) = **56% → moderate band**.

## 4. Gap analysis

1. **Test harnesses bypass the fleet's own SSE test kit.** The repo owns a
   100%-coverage, WPT-corpus-backed parser module and a typed assertion kit,
   and uses them once. `metaengine/v4/sse_replay_test.go:293` hand-writes
   HTTP/1.0 request bytes over `net.Conn`; `systemscenario/v4/sse.go:105`
   hand-rolls a parser that ignores `id:`/`retry:` fields — so the harness
   literally cannot assert on the Last-Event-ID wire behavior it exists to
   prove (it reconnects by header constant, not by parsed stream state).
   `ssetest.Collect(t, handler, ssetest.WithLastEventID(sse.NewEventID("2")))`
   is the one-liner replacement class.
2. **Malformed Last-Event-ID handling is an untested silent path.**
   `metaengine/v4/sse.go:252-256` ignores parse errors and replays
   everything. go-sse's `ParseEventID`/`LastEventIDFromRequest`
   (go-sse/stream.go:305, go-sse/event.go:59) return a validated branded ID
   or error, which would let ServeSSE reject a garbage header with 400
   instead of flooding a client with the full journal. This is a behavior
   decision made by accident, not by design.
3. **Duplicated constants across repos.** metaengine's default buffer
   (`SSEConfig{MaxBuffer: 64}`, metaengine/v4/sse.go:117) and drop-oldest ring duplicate
   go-sse's `DefaultSubscriberBuffer = 64` (go-sse/fanout.go:14) — exported
   in v0.7.0 precisely "so downstream hubs can size their channels by
   construction instead of duplicating the literal" (go-sse CHANGELOG
   [Unreleased]). Same number, two definitions, no lockstep test.
4. **Drop-policy divergence is undocumented at the boundary.** go-sse's
   `Broadcaster` drops the NEWEST event for a full subscriber
   (non-blocking `Broadcast`, go-sse/fanout.go:229); metaengine drops the
   OLDEST (`forwardWithDropOld`, sse_loop.go:33). Both are defensible —
   metaengine's "newest state wins" is right for read models — but nothing
   in metaengine's docs says "this is why we don't use Broadcaster," which
   is exactly the question the next auditor or contributor will ask.

## 5. Version currency

**Split-pin drift; v0.7.0 is non-breaking.**

- metaengine/v4 pins **v0.6.2 direct** (2026-10-03); latest tag **v0.7.0**
  (2026-11-04). Five modules (integration/v4, scenario/v4, stack/bench,
  stack/memory, stack/sqlite) record **v0.7.0 indirect** — their go.mods were
  tidied after v0.7.0 shipped, so the workspace already builds go-sse at
  v0.7.0 wherever MVS sees those modules, while metaengine's own standalone
  module build still resolves v0.6.2. The same production code is tested
  against two versions depending on entry point.
- What v0.7.0 changed (verified by `git diff v0.6.2..v0.7.0`): `DefaultSubscriberBuffer` exported, `eventBrand.Name` refactor, a
  `splitLines("")` edge-case removal, and dependency bumps. **No breaking
  API change** — and notably the go-sse CHANGELOG never cut a `[0.7.0]`
  section; the tag was taken from a chore commit ("fix: correct syntax error
  in verify.sh", 2026-11-04) with the content still sitting under
  `[Unreleased]`. That missing changelog section is likely why no sweep
  picked the new version up repo-wide.
- Fix is one command per module: `go get github.com/larsartmann/go-sse@v0.7.0
  && go mod tidy`, metaengine first, then the v0.6.2 indirects. ssetest
  (v0.5.0) and sseparse (v0.2.1) are the latest of their trains.

## 6. The fleet argument

go-cqrs-lite is the reference consumer of go-sse: ADR-0127 made it the
sanctioned transport, cqrs-lint coaches migrated projects toward it, and
`metaengine.ServeSSE` is the one blessed push surface. Under the owner's
lens — shared fixes reaching every project — that relationship makes
go-cqrs-lite the place where go-sse's bug fixes and semantic decisions get
validated first. Two consequences follow. First, the split pin means the
fleet's integration tier (integration/v4, scenario, stack/*) already runs
v0.7.0 while the library's own tests run v0.6.2 — the validation matrix is
accidental, not chosen. Second, every hand-rolled harness
(`sse_replay_test.go` raw TCP, systemscenario's parser) is a place where a
future go-sse wire-format fix will not be exercised by the fleet's flagship
consumer, even though the fix's own test kit (ssetest/sseparse) lives in the
same fleet. Using the library's test modules in the library's tests is the
cheapest dogfooding in this audit.

## 7. Ranked opportunities

1. **Unify the pin at v0.7.0** (high, trivial): metaengine direct bump +
   tidy sweep across the 40 v0.6.2 indirects. Removes split-brain testing;
   non-breaking by diff inspection.
2. **Adopt ssetest/sseparse in the two hand-rolled harnesses** (high):
   replace `systemscenario/v4/sse.go:105-132` with
   `sseparse.NewStreamReader` (keeps the zero-dep budget: sseparse has no
   go-sse in its graph), and rework `metaengine/v4/sse_replay_test.go`'s
   raw-TCP legs on `ssetest.Collect` + `WithLastEventID` so reconnect
   assertions read typed events instead of substring-matching bodies.
3. **Route Last-Event-ID through `sse.LastEventIDFromRequest`** (medium):
   one-line change at `metaengine/v4/sse.go:252` that turns malformed
   headers into a deliberate decision instead of a silent full replay.
4. **Use `sse.DefaultSubscriberBuffer` for the SSEConfig default** (low):
   delete metaengine's duplicate 64 literal (metaengine/v4/sse.go:117) once on v0.7.0.
5. **Advertise `retry:` on reconnect-capable streams** (low): one
   `sse.WriteRetry` at stream start gives clients deliberate reconnect
   pacing instead of the browser default.
6. **Document the drop-old vs Broadcaster rationale** (low): one paragraph
   in metaengine's SSE docs (or ADR-0127 appendix) closing the "why not
   go-sse Broadcaster" question permanently.

## 8. Reconsider when

- go-sse lands a wire-format or security fix (then the ssetest/sseparse
  adoption in opportunity 2 becomes mandatory dogfooding, not nice-to-have).
- A use case needs one watcher fanned to many HTTP subscribers through a
  go-sse `Broadcaster` bridge (row 7's flip trigger fires).
- go-datastar is adopted anywhere in the fleet's browser story (row 13:
  `KeyedLines`/`SendKeyed` become the shared wire path; see go-datastar.md).
- go-sse cuts a release WITH a changelog section and behavior notes (the
  missing `[0.7.0]` section is why this bump was easy to miss; a loud
  release should trigger the sweep habit).

## 9. How to verify

```
rg -n 'larsartmann/go-sse' -g go.mod /home/lars/projects/go-cqrs-lite | sort
rg -n 'sse\.' /home/lars/projects/go-cqrs-lite/metaengine/v4/sse.go /home/lars/projects/go-cqrs-lite/metaengine/v4/sse_loop.go
sed -n '250,256p;117p;140p' /home/lars/projects/go-cqrs-lite/metaengine/v4/sse.go
sed -n '100,135p' /home/lars/projects/go-cqrs-lite/systemscenario/v4/sse.go
sed -n '285,300p' /home/lars/projects/go-cqrs-lite/metaengine/v4/sse_replay_test.go
cd /home/lars/projects/go-sse && git tag --list | sort -V | tail -4
git -C /home/lars/projects/go-sse diff v0.6.2..v0.7.0 --stat -- '*.go'
rg -n 'DefaultSubscriberBuffer' /home/lars/projects/go-sse/fanout.go
cd /home/lars/projects/go-cqrs-lite/stack/memory/v4 && go mod graph | grep 'go-sse@v0.7.0'
```
