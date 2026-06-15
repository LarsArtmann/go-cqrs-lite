# watermill — Watermill Protocol Adapter

[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/go-cqrs-lite/watermill/v2.svg)](https://pkg.go.dev/github.com/larsartmann/go-cqrs-lite/watermill/v2)

Protocol adapters between go-cqrs-lite event interfaces and the [Watermill](https://watermill.io/) message bus library.

```bash
go get github.com/larsartmann/go-cqrs-lite/watermill/v2
```

## Adapters

| Adapter             | From              | To                   | Description                                      |
| ------------------- | ----------------- | -------------------- | ------------------------------------------------ |
| `PublisherAdapter`  | `event.Publisher` | `message.Publisher`  | Publish go-cqrs-lite events via Watermill        |
| `SubscriberAdapter` | `event.Bus`       | `message.Subscriber` | Receive Watermill messages from go-cqrs-lite bus |

## Usage

```go
import (
    "github.com/larsartmann/go-cqrs-lite/memory/v2"
    "github.com/larsartmann/go-cqrs-lite/watermill/v2"
)

bus := memory.NewMemoryBus()

publisher := watermill.NewPublisherAdapter(bus)
_ = publisher.Publish("user.created", watermillMessage)

subscriber := watermill.NewSubscriberAdapter(bus)
messages, _ := subscriber.Subscribe(ctx, "user.created")
```

## Dependencies

| Dependency                | Purpose                                                       |
| ------------------------- | ------------------------------------------------------------- |
| `ThreeDotsLabs/watermill` | Message bus interface (message.Publisher, message.Subscriber) |
| `event`                   | Event interfaces and error types                              |
