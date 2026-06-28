package watermill

import (
	"context"
	"slices"

	"github.com/larsartmann/go-cqrs-lite/command/v3"
)

func (b *CommandBus) rebuildHandlerChain() {
	allHandlers := make([]command.Handler, len(b.allHandlers))
	copy(allHandlers, b.allHandlers)

	typeSnapshot := make(map[command.Type][]command.Handler, len(b.typeHandlers))
	for k, v := range b.typeHandlers {
		cp := make([]command.Handler, len(v))
		copy(cp, v)
		typeSnapshot[k] = cp
	}

	inner := command.Handler(func(ctx context.Context, cmd command.Command) error {
		for _, h := range allHandlers {
			if err := h(ctx, cmd); err != nil {
				return err
			}
		}

		for _, h := range typeSnapshot[cmd.Type()] {
			if err := h(ctx, cmd); err != nil {
				return err
			}
		}

		return nil
	})

	for _, v := range slices.Backward(b.middleware) {
		inner = v(inner)
	}

	b.cachedHandler = inner
}

func (b *CommandBus) dispatchLocal(ctx context.Context, cmd command.Command) error {
	b.mu.Lock()
	handler := b.cachedHandler
	b.mu.Unlock()

	return handler(ctx, cmd)
}

func (b *CommandBus) ensureSubscriptionLocked() {
	if b.subStarted {
		return
	}

	b.subCtx, b.subCancel = context.WithCancel(context.Background())

	msgs, err := b.subscriber.Subscribe(b.subCtx, b.topic)
	if err != nil {
		b.logger.ErrorContext(b.subCtx, "watermill: subscribe failed",
			"error", err, "topic", b.topic)
		b.subCancel()
		b.subCtx = nil
		b.subCancel = nil

		return
	}

	b.subStarted = true

	go func() {
		for {
			select {
			case msg, ok := <-msgs:
				if !ok {
					return
				}

				cmd, decodeErr := MessageToCommand(b.topic, msg)
				if decodeErr != nil {
					b.logger.ErrorContext(b.subCtx, "watermill: decode command message failed",
						"error", decodeErr)
					msg.Ack()

					continue
				}

				if dispatchErr := b.dispatchLocal(b.subCtx, cmd); dispatchErr != nil {
					b.logger.ErrorContext(b.subCtx, "watermill: dispatch command failed",
						"command_type", cmd.Type(), "error", dispatchErr)
					msg.Ack()

					continue
				}

				msg.Ack()
			case <-b.subCtx.Done():
				return
			}
		}
	}()
}
