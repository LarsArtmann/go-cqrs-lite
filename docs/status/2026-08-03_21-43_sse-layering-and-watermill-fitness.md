# Status Report — 2026-08-03 21:43

> **Session scope:** Analytical only. Two architecture questions about SSE in
> go-cqrs-lite (metaengine vs transport/http), plus a watermill fitness check.
> **No code was changed.** This report covers what was investigated, what was
> found, what was done poorly, and what should follow.

---

## a) FULLY DONE

1. **Explained `metaengine/sse.go`'s existence.** It is the read-side push
   transport for the metaengine's reactive query layer: `Watcher[V]` subscribes
   to a collection's changes, `ServeSSE` streams those `SeqValue[V]` mutations
   to browsers as SSE data events, with `Last-Event-ID` reconnection backed by
   `SSEReplay[V]` (journal ring buffer) + dedup ring for replay→live overlap.
   Distinct from `transport/http.SSEBroker`, which bridges `event.Bus` → HTTP
   (raw domain events, ULID-based resume).

2. **Verified the metaengine layer leak is real** (not a guess). `metaengine/go.mod`
   lines 6-7 require `go-sse` + `dedup` as **production** deps. ADR-0062 declares
   the metaengine core as "zero production deps (stdlib + `database/sql` only)".
   `sse.go` violates that boundary by pulling transport-tier concerns into a
   Tier-0 primitive. Confirmed, not speculation.

3. **Ruled out watermill as a general SSE bus.** Watermill is a broker adapter
   (Kafka/NATS/Redis/GoChannel) optimizing for routed, Ack-based, multi-consumer
   server-to-server pub/sub. SSE is single-consumer, fire-and-forget,
   connection-scoped browser push. The two have incompatible consumer models,
   payload types (`*message.Message` vs typed `V`), replay mechanisms
   (`SeekableJournal` vs `SSEReplay[V]` ring), and dependency weight. See the
   table in the session conversation.

---

## b) PARTIALLY DONE

1. **Comparison of the two SSE implementations.** I correctly identified that
   `transport/http` and `metaengine` each have their own SSE code, and that the
   doc comments on both `SSEConfig` and `SSEBroker` deliberately cross-reference
   each other (good — the split is documented). **Not done:** I did not measure
   how much loop logic is actually duplicated between `transport/http/sse.go`
   and `metaengine/sse.go` (heartbeat, timeout, flush, drop-old backpressure).
   That measurement is the real input to the "should we extract a shared loop?"
   question.

2. **Proposed `SSESource[V]` interface extraction.** I sketched moving
   `metaengine/sse.go` into `transport/http` behind a tiny typed source adapter
   interface. **Not done:** I did not verify this is feasible against the
   `Watcher[V]` API surface, did not check whether `transport/http` already
   has a generic SSE helper that could absorb it, and did not ask whether the
   user wants metaengine to stay self-contained for consumer-side convenience.

---

## c) NOT STARTED

1. **No actual refactor.** Nothing moved, no interface extracted, no ADR drafted.
2. **No measurement of duplication.** Did not run `art-dupl` or a diff between
   the two SSE files.
3. **No check of whether any consumer outside this repo imports
   `metaengine.ServeSSE` directly** (which would make moving it a breaking
   change requiring a v-bump or re-export alias).

---

## d) TOTALLY FUCKED UP

1. **Asserted before verifying.** I claimed metaengine "drags `net/http`,
   `go-sse`, and `dedup` into that module" before checking `go.mod`. I was
   correct, but the order was wrong — claim first, evidence second. The user
   had to prompt me to actually look. This is the "stale GREEN" anti-pattern's
   cousin: assertive architecture claims without grounding in the file.
2. **Did not examine watermill before answering "why no general SSE bus."**
   Watermill is the obvious candidate and I skipped it. The user had to
   explicitly say "checkout /watermill." I should have ruled it in or out as
   part of the original answer.
3. **Overstated sharing.** I said "the shared loop logic is already generic
   and reusable" — true *inside* metaengine (`sseMainLoop[T]`), but I implied
   `transport/http` shares it. It does not; they are separate reimplementations.
   I conflated "could be shared" with "is shared."
4. **Speculation presented as recommendation.** The `SSESource[V]` interface
   proposal was not validated against the codebase. It's a plausible direction,
   not a recommendation I earned the right to make.

---

## e) WHAT WE SHOULD IMPROVE

### Process (this session)

- **Verify deps before claiming layer leaks.** `go.mod` is one tool call.
- **Examine all plausible candidates before ruling.** "Why no X?" requires
  checking the existing things that look like X (watermill, transport/http,
  command.MemoryBus) before answering.
- **Measure duplication before proposing extraction.** `art-dupl` exists in
  this repo specifically for this. Run it.
- **Separate "is" from "should be."** Saying the loop *is* shared when it
  *could be* shared is a lie. Say "could be" until measured.

### Codebase (real findings)

1. **`metaengine/sse.go` breaks ADR-0062's dependency boundary.** The metaengine
   core is documented as zero-production-deps. `sse.go` + its go.mod lines pull
   `go-sse` and `dedup` in. Either: (a) move SSE out to `transport/http` behind
   a source adapter, (b) split metaengine into `metaengine` (core, no SSE) +
   `metaengine/sse` (transport, separate go.mod), or (c) amend ADR-0062 to
   acknowledge the boundary exception. Picking is a real decision; ignoring
   is the worst option.

2. **`Inspect()` and `InspectJSON()` are misplaced in `sse.go`.** Lines 372-398
   of `metaengine/sse.go` define `Store.Inspect()` and `Store.InspectJSON()` —
   store introspection methods that have nothing to do with SSE. They live in
   `sse.go` presumably because the file was a convenient dumping ground. They
   belong in `store.go` or a dedicated `inspect.go`. This is a naming/cohesion
   smell independent of the layering question.

3. **Two SSE implementations with no shared abstraction.** Whether to unify is
   a real open question. The data models (`event.Event` vs `SeqValue[V]`) are
   genuinely different, but the loop scaffolding (flush, heartbeat, timeout,
   drop-old, Last-Event-ID parsing, replay→live dedup handoff) is likely
   duplicated. Worth measuring before deciding.

---

## f) Up to 50 things to get done next

### High impact — layering & cohesion

1. **Decide metaengine SSE boundary.** Pick (a) move to transport/http,
   (b) split into metaengine/sse sub-module, or (c) amend ADR-0062. Document
   the decision in an ADR.
2. **Move `Inspect()` / `InspectJSON()` out of `sse.go`** into `inspect.go` or
   `store.go`. Zero behavior change, pure file cohesion fix.
3. **Run `art-dupl` between `transport/http/sse*.go` and `metaengine/sse.go`**
   to quantify actual duplication. Input to decisions 4-6.
4. **If duplication is high, extract a shared `sseloop` internal package**
   (heartbeat/timeout/flush/drop-old) usable by both, without forcing a shared
   source interface.
5. **If duplication is low, leave the two implementations separate** and add a
   one-line cross-reference comment in each pointing to the other (partially
   done already — verify completeness).

### Medium impact — verification & docs

6. **Check whether any external consumer imports `metaengine.ServeSSE`** (api-stability
   golden + grep consumer repos). Determines whether moving it is a breaking
   change.
7. **Verify the cross-reference comments in `SSEConfig` and `SSEBroker`** are
   symmetric and accurate (I read both doc comments but did not diff them line
   by line).
8. **Audit all other files in `metaengine/` for transport-tier imports**
   (`net/http`, `go-sse`, anything beyond stdlib + database/sql). `sse.go` may
   not be the only offender.
9. **Update AGENTS.md "Dependencies" section** if the metaengine production-dep
   list is inaccurate (it currently lists only `oklog/ulid`, `go-branded-id`,
   `go-error-family` for `event/` — the metaengine row is not called out as
   having extra deps).
10. **Write an ADR for the SSE layering decision** regardless of outcome —
    future contributors will ask the same question.

### Lower impact — polish

11. **Rename `metaengine/sse.go` → `metaengine/sse_serve.go`** if it stays, to
    signal it's the serve entry point, not the whole SSE story.
12. **Add a `doc.go` comment in metaengine** explaining why SSE lives there (if
    it stays) so the next reader doesn't have to re-derive it.
13. **Consider a `metaengine/transport/http` sub-module** mirroring the
    `transport/http` top-level package, as a home for metaengine-specific HTTP
    concerns (SSE, future REST handlers).
14. **Check if `dedup` is used anywhere else in metaengine** besides `sse.go`.
    If not, moving SSE out also removes the dedup dep — cleaner boundary.
15. **Verify `go-sse` version (v0.4.0) is current** — `metaengine/go.mod` pins
    it; check for upstream releases.

### Watermill-adjacent (from the second question)

16. **Document in `watermill/doc.go`** why Watermill is NOT used for SSE — a
    one-paragraph note prevents the next reader from trying to wire SSE through
    a Watermill router.
17. **Check if `CatchUpSubscriber` (watermill) and `SSEReplay` (metaengine)**
    share any replay/dedup logic that could be factored into `dedup/` or a
    shared replay helper. They solve the same problem (replay→live handoff with
    dedup) for different resume tokens.
18. **Audit whether `transport/http.SSEBroker` could optionally source from a
    Watermill subscriber** for multi-process SSE fan-out (browser ← SSE ←
    Watermill ← NATS). This is a real use case the current design doesn't
    address and may be the actual "general SSE bus" the user is probing for.

### Session-process improvements (meta)

19. **Add a "verify deps before claiming layer leaks" line to AGENTS.md** under
    Lint Conventions or Process Safety.
20. **Add a "measure duplication before proposing extraction" line** to the
    same section.

(20 items. The remaining 30 slots are intentionally left empty — padding a
list with busywork is worse than a shorter honest list.)

---

## g) Questions I cannot figure out myself

1. **Is `metaengine.ServeSSE` part of the published, supported public API that
   external consumers depend on?** If yes, moving it to `transport/http` is a
   breaking change requiring a re-export alias or major version bump. I can
   check the api-stability golden and git tags, but I cannot see consumer repos
   outside this one. **What's your intent for the metaengine SSE surface — is
   it stable public API or experimental?**

2. **Should metaengine remain self-contained (SSE included) for single-import
   consumer convenience, or is strict tier discipline (ADR-0062) the higher
   value?** This is a product/values call, not a technical one. Both are
   defensible; I can't pick for you. **Which matters more: one-import
   ergonomics, or the zero-dep primitive boundary?**

3. **Is there a real (planned or imagined) consumer that needs SSE fan-out
   across multiple processes — i.e., browser ← SSE ← Watermill ← broker?** If
   yes, the "general SSE bus" question becomes real and the answer is a
   transport-layer adapter that sources SSE from a Watermill subscriber (not
   the other way around). If no, the two current implementations are
   sufficient and the question is just about internal cleanliness. **Is
   multi-process SSE fan-out on the roadmap, or is single-process push the
   target?**

---

_Snapshot of an analytical session. No code changed; findings are
actionable but uncommitted._
