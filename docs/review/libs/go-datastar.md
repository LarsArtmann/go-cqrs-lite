# go-datastar — Utilization Audit · go-cqrs-lite

> Audited 2026-11-22 · Library `github.com/larsartmann/go-datastar` @ 24a808f (latest local tag v0.7.0, 2026-11-07, four-module lockstep) · Consumer go-cqrs-lite @ feec9bc74 · Group G2

**Verdict:** NOT-ADOPTED-JUSTIFIED
**Score:** 0/1 applicable capabilities leveraged = 0% (low)

Zero requires across all 104 go.mod files. The justification is structural,
not lazy: go-cqrs-lite serves no HTML to browsers anywhere — its browser
story is JSON-over-SSE (`metaengine.ServeSSE`) consumed by whatever client an
app brings, its one templ-based surface (docserver) renders static docs, and
the transport-agnostic doctrine (ADR-0127) keeps browser delivery at the
example/recipe tier rather than in library modules. Datastar patch values
have no DOM to patch here. The one honest Missed row is dogfood-tier: the
fleet owns both halves of the Datastar-over-SSE bridge (go-sse already
documents the Datastar keyed-lines wire format and is production-adopted in
metaengine), and a live-UI example twin would cost little — but with zero
browser surface today, "not yet" is the correct verdict, with named flip
triggers below.

## 1. What the library is

A DataStar protocol library for Go whose core design decision is that
**patches are values, not method calls**: every patch implements
`Patch interface { Event() sse.Event }` (go-datastar/patch.go:25), so a
patch can be constructed without an open connection, then stored, queued,
filtered, replayed, or broadcast through go-sse's infrastructure — the
explicit architectural difference from the official connection-bound
`datastar-go` SDK (go-datastar README.md:9-55). Surface by module:

- **root** — `NewElementsPatch` with selector/mode/namespace/view-transition
  options and sugar (`WithModeInner`, `WithSelectorID`, `NewRemovePatch`, …;
  go-datastar/elements.go:97, go-datastar/sugar.go:13-67);
  `NewSignalsPatch`/`NewSignalsIfMissingPatch` (typed reactive-signal
  updates); `ScriptPatch` plus the convenience family `NewRedirectPatch`,
  `NewConsoleLogPatch`, `NewConsoleErrorPatch`, `NewDispatchCustomEventPatch`,
  `NewReplaceURLPatch`, `NewPrefetchPatch`
  (go-datastar/script_convenience.go:16-195); inbound `ReadSignals`
  (JSON body or query params) and `LastEventID` (go-datastar/inbound.go:26,107);
  `GetSSE`/`PostSSE`/… action strings (go-datastar/http.go:6-26); templ and
  GoStar adapters (go-datastar/adapters.go:22,42).
- **broadcast** — `Broadcaster` with `WithReplayCapacity`/`WithStore`
  (multi-instance replay)/`WithHeartbeatInterval`/`WithHeartbeatDisabled`
  (go-datastar/broadcast/broadcaster.go:57; go-datastar/broadcast/options.go:25).
- **datastartest** — E2E helpers for consumer handlers (typed decoding,
  diffs, filters).
- **static** — embedded DataStar JS client (v1.0.4) served by `ScriptHandler`
  with ETag + Cache-Control.

Latest release v0.7.0 (2026-11-07) is the first full four-module lockstep
cut; its one breaking change drops `ScriptHandlerWith`'s dead version
parameter (go-datastar/docs/migration-guide.md:100-107).

## 2. Usage census in go-cqrs-lite (evidence)

**None.** Verified by:

```
rg -n 'larsartmann/go-datastar' -g go.mod /home/lars/projects/go-cqrs-lite   # no hits
rg -rn 'larsartmann/go-datastar' /home/lars/projects/go-cqrs-lite --type go  # no hits
```

For contrast, the browser-adjacent surfaces that DO exist:

- `metaengine/v4/sse.go` — `ServeSSE` streams JSON-encoded read-model values
  over SSE (wire via go-sse; Last-Event-ID replay via `Watcher.WithReplay`).
- `example/taskmanager/http.go:57-61` — `/events` endpoint on that watcher;
  the stream is consumed only by tests (`example/taskmanager/sse_test.go`)
  and hypothetical app clients. No `EventSource`, HTML, or JS exists in any
  example in the repo.
- `catalog/v4/docserver/` — templ-rendered docs UI, but entirely static
  pages (no live updates).

Sibling precedent for the adoption shape: go-aichat pins `go-datastar
v0.6.2` + `static v0.6.1` and builds a typed signal vocabulary
(`go-aichat/datastar/scope.go:22-40`: `SignalText`, `SignalThinking`,
`SignalActive`, …, namespaced per chat panel) — i.e., typed Datastar signal
patches over SSE are a proven pattern elsewhere in the owner's fleet.

## 3. Capability inventory

| # | Capability | Grade | Reason (N/A rows carry the flip trigger) |
|---|---|---|---|
| 1 | Elements patches (server-driven DOM updates) | N/A | no HTML-serving surface exists in repo or examples — flip trigger: the first fleet app that wants live UI without hand-written client JS |
| 2 | Signals patches (typed reactive-state pushes) | N/A | same trigger; go-aichat's `datastar/scope.go` is the ready-made pattern for typed signal vocabularies |
| 3 | Script patches + convenience family (redirect, console, custom events) | N/A | rides on the same trigger as rows 1–2 |
| 4 | broadcast submodule (fan-out + replay + hub sharing) | N/A | `metaengine.ServeSSE`'s watcher fan-out already covers read-model push at the store layer; note the parallel evolution — broadcast's `WithReplayCapacity` and metaengine's `Watcher.WithReplay` solve the same reconnection problem at different tiers |
| 5 | datastartest E2E helpers | N/A | nothing to test until a Datastar handler exists |
| 6 | static JS client serving (`ScriptHandler`, ETag'd) | N/A | no browser assets served anywhere |
| 7 | Inbound `ReadSignals` (query/body signal decoding) | N/A | the sanctioned inbound pattern is typed JSON REST decode (recipe §2.47); flip trigger: a Datastar UI round-tripping its signal state |
| 8 | templ/gostar adapters | N/A | docserver's templ renders static docs; flip trigger: a live-updating docs/dashboard view (e.g., projection lag page updating itself) |
| 9 | Live-UI example twin on the golden path (watcher → patch values → SSE) | Missed | the one place adoption would actually help today: a recipe/example showing `TaskView` changes as `SignalsPatch`/`ElementsPatch` values over `ServeSSE`-class plumbing would (a) dogfood the owner's own go-datastar at library-example tier, (b) validate the go-sse `KeyedLines`/`SendKeyed` wire path (go-sse/doc.go "Keyed Data Lines") the fleet already owns, and (c) give fleet apps a copy-me alternative to hand-rolled EventSource JS. No doctrine violation: examples/recipes are where browser delivery lives (ADR-0127 sanctions cqrs-htmx for the same tier) |

Score derivation: rows N=9, N/A rows 1–8 → **A=1** (row 9). Full = 0,
Partial = 0, Missed = 1 × 0. Score = round(100 × 0 / 1) = **0% → low band**.

## 4. Gap analysis

1. **There is no production gap.** Nothing served by go-cqrs-lite renders to
   a browser dynamically; adopting Datastar patches in metaengine or any
   library module would couple the transport-agnostic core to one browser
   framework — precisely what ADR-0127's deletion of transport/* rejected.
   The doctrine holds.
2. **The dogfood gap is real but cheap and deferred.** The fleet owns the
   full vertical: go-sse (production-adopted here, Datastar-wire-aware) →
   go-datastar (values that emit `sse.Event`) → go-aichat (typed signal
   vocabulary precedent). The only missing piece is one example in
   go-cqrs-lite proving the vertical end-to-end on a CQRS read model. Under
   the owner's "collective improvements" lens this is the classic
   template-repo move: teach the pattern once, every fleet app copies it.
   Against it: examples cost maintenance, v4→v5 dual-support is in flight,
   and no fleet app has asked for it yet. Deferring is defensible; the
   decision should be recorded rather than silent.
3. **Coordination note for the eventual adoption.** go-datastar requires
   Go 1.27.1+ (README "Requirements") — same floor as the rest of the fleet,
   so no toolchain conflict. Adopt at v0.7.0+ (the `ScriptHandlerWith`
   signature break is already settled there).

## 5. Version currency

Not pinned; nothing to be behind. Latest tag **v0.7.0** (2026-11-07):
first four-module lockstep cut (root, static, broadcast, datastartest on one
commit), `WithHeartbeatDisabled` added, `ScriptTagVersioned` added, and the
one breaking change (`ScriptHandlerWith(bundle, version)` →
`ScriptHandlerWith(bundle)`, go-datastar/docs/migration-guide.md:100-112).
If/when adopted, start at v0.7.0; nothing in the consumer constrains the
choice.

## 6. The fleet argument

Judged strictly on go-cqrs-lite's current surface, non-adoption is correct —
there is no DOM. Judged through the owner's fleet lens, the interesting
asset is that **the bridge is pre-built**: metaengine already speaks
go-sse's wire format, go-sse already emits Datastar's keyed data lines, and
go-datastar already turns typed Go values into those events. The moment any
fleet app wants "live-updating UI over my CQRS read model without writing
client JS," the entire server side is assembling three of the owner's own
libraries. The recurring-cost argument says go-cqrs-lite's example tier is
where that assembly recipe should live so it is written once — but the
trigger for writing it is the first fleet app that actually needs it, not
speculation. Until then the honest posture is: justified non-adoption, one
named dogfood opportunity parked with its flip trigger.

## 7. Ranked opportunities

1. **Record the decision** (now, zero cost): a short paragraph (ADR-0127
   appendix or the recipes index) stating that Datastar-over-SSE is the
   sanctioned live-UI path for fleet apps on this library, with the
   go-aichat signal-vocabulary pattern as the reference — converts this
   audit's "deferred" into a documented posture.
2. **Live-UI example twin** (on the flip trigger): a `example/taskmanager`
   variant (or recipe §2.4x) streaming `SignalsPatch`-shaped TaskView
   updates over the existing watcher/SSE plumbing, E2E-tested with
   `datastartest` — dogfoods both go-datastar and go-sse's keyed-lines path.
3. **Nothing else** — every other row is N/A until a browser surface exists.

## 8. Reconsider when

- A fleet app (or the owner) wants live browser updates on a CQRS read model
  without bespoke client JS — rows 1–3, 7, 8 flip; opportunity 2 executes.
- docserver grows a live view (e.g., self-updating projection-lag or
  system-status page) — row 8 flips in the repo's own production tier.
- A fleet app needs broadcast/replay semantics richer than one-watcher-
  per-connection (row 4's tier separation may then be worth revisiting).
- go-datastar ships a consumer-driven v1.0 candidate — a good moment to
  decide the example twin before the API freezes.

## 9. How to verify

```
rg -n 'larsartmann/go-datastar' -g go.mod /home/lars/projects/go-cqrs-lite
rg -rn 'larsartmann/go-datastar' /home/lars/projects/go-cqrs-lite --type go
rg -rn 'EventSource|text/html' /home/lars/projects/go-cqrs-lite/example --type go
git -C /home/lars/projects/go-datastar log -1 --format='%H %cs' && git -C /home/lars/projects/go-datastar tag --list | sort -V | tail -3
sed -n '20,32p' /home/lars/projects/go-datastar/patch.go
sed -n '100,112p' /home/lars/projects/go-datastar/docs/migration-guide.md
rg -n 'go-datastar' /home/lars/projects/go-aichat/go.mod /home/lars/projects/go-aichat/datastar/scope.go | head -6
rg -n 'Keyed Data Lines' /home/lars/projects/go-sse/doc.go
```
