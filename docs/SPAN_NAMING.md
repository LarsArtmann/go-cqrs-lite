# Span Naming Convention

> Canonical naming for OpenTelemetry spans across go-cqrs-lite modules.

## Pattern

All spans follow the `{component}.{action}` or `{component}.{phase}.{action}`
convention:

- **component** — the module or integration layer (command, event, query,
  decider, grpc, sse, watermill, etc.)
- **action** — the verb describing what the span covers (handle, publish,
  load, save, execute, dispatch, replay, fanout)

```
{component}.{action}
{component}.{phase}.{action}   (when a multi-phase operation needs sub-spans)
```

### Examples

| Span                            | Component   | Action   | Meaning                                       |
| ------------------------------- | ----------- | -------- | --------------------------------------------- |
| `command.handle`                | command     | handle   | Processing a command through middleware       |
| `event.handle`                  | event       | handle   | Processing an event through middleware        |
| `event.publish`                 | event       | publish  | Publishing events to the bus                  |
| `query.handle`                  | query       | handle   | Processing a query through middleware         |
| `decider.execute`               | decider     | execute  | Full stream execute (load → decide → save)   |
| `decider.load`                  | decider     | load     | Loading stream state from events             |
| `event.store.load`              | event.store | load     | SQL store load                                |
| `event.store.save`              | event.store | save     | SQL store save                                |
| `grpc.command.dispatch`         | grpc        | dispatch | gRPC server handling a command                |
| `grpc.query.ask`                | grpc        | ask      | gRPC server handling a query                  |
| `watermill.event.publish`       | watermill   | publish  | Publishing events to a Watermill topic        |
| `watermill.command.publish`     | watermill   | publish  | Publishing commands to a Watermill topic      |
| `watermill.replay.from_journal` | watermill   | replay   | CatchUpSubscriber replaying events            |
| `sse.fanout`                    | sse         | fanout   | Broadcasting an event to SSE clients          |
| `sse.replay`                    | sse         | replay   | Last-Event-ID reconnection replay             |
| `retry.attempt.N`               | retry       | attempt  | Nth retry attempt (child span)                |

## Span Kinds

| Kind         | When                                                                   |
| ------------ | ---------------------------------------------------------------------- |
| **Server**   | Inbound request handling (command.handle, grpc.\*.dispatch)            |
| **Consumer** | Event consumption (event.handle, sse.fanout)                           |
| **Producer** | Event/command publishing (event.publish, watermill.\*.publish)         |
| **Client**   | SQL store operations (event.store.\*)                                  |
| **Internal** | Library-internal operations (decider.\*, sse.replay, watermill.replay) |

## Attributes

All spans use the `cqrs.*` attribute namespace:

| Attribute                | Example                     | Used by                  |
| ------------------------ | --------------------------- | ------------------------ |
| `cqrs.message.kind`      | `command`, `event`, `query` | All middleware spans     |
| `cqrs.command.type`      | `user.create`               | Command spans            |
| `cqrs.event.type`        | `user.created`              | Event spans              |
| `cqrs.query.type`        | `user.get`                  | Query spans              |
| `cqrs.aggregate.type`    | `User`                      | Store/decider spans      |
| `cqrs.aggregate.id`      | `01HK...`                   | Most spans               |
| `cqrs.aggregate.version` | `5`                         | Store spans              |
| `cqrs.event.count`       | `3`                         | Batch publish/save spans |
| `cqrs.status`            | `success`, `error`          | Metrics spans            |
| `cqrs.projection.name`   | `users`                     | Replay spans             |

## Rules

1. **Dot-separated, lowercase.** No camelCase, no underscores.
2. **Component first, action last.** Spans group by component in trace UIs.
3. **One action per span.** If a span covers two actions, split it.
4. **Retry attempts are children.** Each retry creates a child span
   (`retry.attempt.N`) under the parent operation span.
5. **Error recording.** All spans call `cqrsotel.RecordError(span, err)` on
   error paths, setting the OTel error status.
6. **No dynamic span names.** Span names must be static strings for
   cardinality control. Dynamic values go in attributes.

## Cross-Process Trace Linking

Producer spans (publish) inject W3C trace context into message metadata.
Consumer spans (handle) extract it to link into the same trace tree:

- **Watermill:** `EventPublisher`/`CommandPublisher` inject on publish;
  `watermill.ExtractContext()` / `watermill.TraceContextMiddleware()` extract
  on consume.
- **gRPC:** Use `otelgrpc` server/client interceptors at the transport layer.
- **SSE:** Browser clients reconnect with `Last-Event-ID`; trace context
  flows through W3C headers on the HTTP layer.
