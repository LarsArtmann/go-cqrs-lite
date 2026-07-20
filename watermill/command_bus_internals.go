package watermill

import (
	"context"
	"slices"

	"github.com/ThreeDotsLabs/watermill/message"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
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
	return dispatchCached(&b.mu, b.cachedHandler, ctx, cmd)
}

func (b *CommandBus) ensureSubscriptionLocked() {
	b.ensureStarted(b.subscriber, b.topic, b.logger, b.runCommandLoop)
}

func (b *CommandBus) runCommandLoop(ctx context.Context, msgs <-chan *message.Message) {
	for {
		select {
		case msg, ok := <-msgs:
			if !ok {
				return
			}

			cmd, decodeErr := MessageToCommand(b.topic, msg)
			if decodeErr != nil {
				b.logger.ErrorContext(ctx, "watermill: decode command message failed",
					"error", decodeErr)
				msg.Ack()

				continue
			}

			if dispatchErr := b.dispatchLocal(ctx, cmd); dispatchErr != nil {
				b.logger.ErrorContext(ctx, "watermill: dispatch command failed",
					"command_type", cmd.Type(), "error", dispatchErr)
				msg.Ack()

				continue
			}

			msg.Ack()
		case <-ctx.Done():
			return
		}
	}
}
