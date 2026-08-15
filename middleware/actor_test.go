package middleware_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/middleware/v4"
)

func TestCommandActorContext(t *testing.T) {
	t.Parallel()

	actor := id.NewBotActor("ci-runner")

	cmd, err := command.New("deploy.release", id.NewStreamID(), command.WithActor(actor))
	if err != nil {
		t.Fatalf("create command: %v", err)
	}

	var gotCtx context.Context

	handler := middleware.CommandActorContext()(
		func(ctx context.Context, _ command.Command) error {
			gotCtx = ctx
			return nil
		},
	)

	if err := handler(context.Background(), cmd); err != nil {
		t.Fatalf("handle: %v", err)
	}

	got, ok := event.ActorFromContext(gotCtx)
	if !ok {
		t.Fatal("expected actor in handler context")
	}

	if !got.Equal(actor) {
		t.Errorf("actor = %q, want %q", got.PrefixedString(), actor.PrefixedString())
	}
}

func TestCommandActorContext_NoActor(t *testing.T) {
	t.Parallel()

	cmd, err := command.New("deploy.release", id.NewStreamID())
	if err != nil {
		t.Fatalf("create command: %v", err)
	}

	var gotCtx context.Context

	handler := middleware.CommandActorContext()(
		func(ctx context.Context, _ command.Command) error {
			gotCtx = ctx
			return nil
		},
	)

	if err := handler(context.Background(), cmd); err != nil {
		t.Fatalf("handle: %v", err)
	}

	if _, ok := event.ActorFromContext(gotCtx); ok {
		t.Error("expected no actor in handler context for command without actor")
	}
}
