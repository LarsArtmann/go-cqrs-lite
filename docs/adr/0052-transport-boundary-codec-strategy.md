# ADR-0052: Transport Boundary Codec Strategy

## Status

**Accepted — 2026-07-11.** Four related decisions arising from the CBOR default
flip (ADR-0051). Each is evaluated independently below.

## Context

ADR-0051 changed `event.DefaultCodec` from JSON to CBOR. This surfaced four
architectural questions about the boundary between internal CBOR and external
(transport/browser) JSON:

1. **NewEvent() asymmetry** — `event.New()` stamps encoding from `DefaultCodec`
   (CBOR), but `event.NewEvent()` (raw `[]byte`) leaves the encoding field unset.
   The getter returns JSON for unset encoding. So raw-bytes events are ALWAYS
   JSON-stamped regardless of `DefaultCodec`.

2. **SSE default JSON-out** — SSE is a text-only channel. CBOR payloads sent
   raw are binary bytes in a text frame; browsers cannot parse them. Should the
   SSE broker auto-convert to JSON?

3. **WebSocket binary frames** — WebSocket supports binary frames natively.
   Should the library provide a WebSocket transport with CBOR-native support?

4. **fetch(arrayBuffer())** — HTTP clients can consume binary responses. Should
   REST endpoints support `Accept: application/cbor`?

## Decisions

### 1. NewEvent() asymmetry → v4 fix

**Keep for v3.** The asymmetry is intentional: raw bytes are pre-serialized by
the caller, so the library cannot know the encoding. Defaulting to JSON matches
existing caller behavior (all raw-bytes callers today send JSON).

**For v4:** Add `event.WithEncoding(codec.Encoding)` as a per-event option.
Callers sending raw CBOR bytes can stamp the encoding explicitly. The getter's
JSON fallback stays as the safe default for unset encoding.

This is a non-breaking addition (new option, no removed behavior).

### 2. SSE default JSON-out → No opinionated default

**Decision: The SSE broker will NOT auto-convert payloads.**

Rationale:

- **Design Principle #1 (Library, not framework):** The library does not impose
  transport-specific defaults. The consumer knows their transport shape.
- **`WithPayloadTransform` is the explicit translation point.** It exists
  precisely for this purpose. Auto-converting would surprise consumers who send
  pre-serialized payloads.
- **SSE penalizes CBOR via base64.** SSE is text-only; binary data must be
  base64-encoded, adding 33% overhead. This negates CBOR's ~35% size savings.
  JSON is the natural SSE codec.

Consumer guidance (documented in SKILL.md and AGENTS.md):

```
broker, _ := http.NewSSEBroker(bus, http.WithPayloadTransform(http.CBORToJSONTransform))
```

### 3. WebSocket binary frames → Deferred (no module)

**Decision: No WebSocket transport module will be built without consumer demand.**

Evaluation:

- WebSocket binary frames carry CBOR natively — no base64 tax. This is the
  ideal transport for CBOR-native clients.
- But: no current consumer needs it. Building it would be YAGNI.
- If built: a `transport/websocket` module would accept `event.Event` directly,
  encode via `codec.CBORCodec{}`, and send as binary frames. The read side
  would decode binary frames via `codec.ForEncoding()`.
- This would NOT change the core library — it's a pure transport addition.

### 4. fetch(arrayBuffer()) → Deferred (no module)

**Decision: No CBOR REST endpoint support will be built without consumer demand.**

Evaluation:

- HTTP clients can consume CBOR via `fetch(arrayBuffer())` + a CBOR decoder.
  This is slightly more efficient than JSON for large payloads.
- But: `transport/http` is SSE-only. REST API construction is a consumer concern.
  The library provides `BackfillHandler` for pull-based event access, which
  supports `WithPayloadTransform` for JSON output.
- Content negotiation (`Accept: application/cbor` → CBOR response) is a REST
  framework concern, not a library concern.

## Consequences

| Decision           | Impact                                               |
| ------------------ | ---------------------------------------------------- |
| NewEvent asymmetry | v4: add `WithEncoding()` option (non-breaking)       |
| SSE default        | Consumers MUST use `WithPayloadTransform` explicitly |
| WebSocket          | No action — deferred to consumer demand              |
| fetch CBOR         | No action — deferred to consumer demand              |

## Related

- [ADR-0051](0051-cbor-as-default-codec.md) — CBOR as default codec
- [ADR-0044](0044-blind-store-encoding-stamps.md) — Blind store encoding stamps
- [ADR-0015](0015-cbor-codec.md) — Original CBOR codec addition
