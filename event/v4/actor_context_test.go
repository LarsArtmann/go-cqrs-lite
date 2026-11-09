package event_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

func TestWithActorContext(t *testing.T) {
	t.Parallel()

	actor := id.NewSystemActor("scheduler")
	ctx := event.WithActorContext(context.Background(), actor)

	got, ok := event.ActorFromContext(ctx)
	if !ok {
		t.Fatal("ActorFromContext: expected ok=true after WithActorContext")
	}

	if !got.Equal(actor) {
		t.Errorf("ActorFromContext = %q, want %q", got.PrefixedString(), actor.PrefixedString())
	}
}

func TestActorFromContext_Absent(t *testing.T) {
	t.Parallel()

	if _, ok := event.ActorFromContext(context.Background()); ok {
		t.Error("expected ok=false without WithActorContext")
	}

	zeroCtx := event.WithActorContext(context.Background(), id.ActorID{})
	if _, ok := event.ActorFromContext(zeroCtx); ok {
		t.Error("expected ok=false for zero actor")
	}
}

func TestActorEnricher(t *testing.T) {
	t.Parallel()

	actor := id.NewUserActor(id.NewUserID())
	ctx := event.WithActorContext(context.Background(), actor)

	evt, err := event.NewEvent(
		"user.created", id.NewStreamID(), "User", 1,
		[]byte(`{}`),
		event.ActorEnricher(ctx)...,
	)
	if err != nil {
		t.Fatalf("create event: %v", err)
	}

	if got := evt.Metadata().ActorID; !got.Equal(actor) {
		t.Errorf("ActorID = %q, want %q", got.PrefixedString(), actor.PrefixedString())
	}
}

func TestActorEnricher_NoActor(t *testing.T) {
	t.Parallel()

	if opts := event.ActorEnricher(context.Background()); opts != nil {
		t.Errorf("expected nil options without actor, got %v", opts)
	}
}
