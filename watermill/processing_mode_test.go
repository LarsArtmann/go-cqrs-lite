package watermill_test

import (
	"context"
	"testing"

	"github.com/ThreeDotsLabs/watermill/message"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	wm "github.com/larsartmann/go-cqrs-lite/watermill/v4"
)

// TestProcessingModeMiddleware_ReplayFlagPropagates verifies that a message
// carrying processing_mode=replay metadata reconstructs ModeReplay into the
// handler context, so consumers can branch on event.IsReplay(ctx) even across
// the Watermill process boundary.
func TestProcessingModeMiddleware_ReplayFlagPropagates(t *testing.T) {
	t.Parallel()

	mw := wm.ProcessingModeMiddleware()

	called := false
	wrapped := mw(func(msg *message.Message) ([]*message.Message, error) {
		called = true
		if !event.IsReplay(msg.Context()) {
			t.Error("expected IsReplay(ctx) to be true after middleware reconstructed replay flag")
		}

		return nil, nil
	})

	msg := message.NewMessage("test-1", []byte(`{}`))
	msg.Metadata.Set("processing_mode", string(event.ModeReplay))

	if _, err := wrapped(msg); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	if !called {
		t.Fatal("middleware did not invoke the wrapped handler")
	}
}

// TestProcessingModeMiddleware_LiveIsDefault verifies that messages without
// the processing_mode metadata key default to ModeLive (IsReplay == false).
func TestProcessingModeMiddleware_LiveIsDefault(t *testing.T) {
	t.Parallel()

	mw := wm.ProcessingModeMiddleware()

	wrapped := mw(func(msg *message.Message) ([]*message.Message, error) {
		if event.IsReplay(msg.Context()) {
			t.Error(
				"expected IsReplay(ctx) to be false for a message without processing_mode metadata",
			)
		}

		return nil, nil
	})

	msg := message.NewMessage("test-2", []byte(`{}`))
	// No processing_mode metadata — should default to live.

	if _, err := wrapped(msg); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
}

// TestProcessingModeMiddleware_PreservesExistingContext verifies that the
// middleware preserves values already present in the message context.
func TestProcessingModeMiddleware_PreservesExistingContext(t *testing.T) {
	t.Parallel()

	type ctxKey struct{}
	val := "preserved"

	mw := wm.ProcessingModeMiddleware()
	wrapped := mw(func(msg *message.Message) ([]*message.Message, error) {
		if got := msg.Context().Value(ctxKey{}); got != val {
			t.Errorf("expected existing context value %q to be preserved, got %v", val, got)
		}

		return nil, nil
	})

	msg := message.NewMessage("test-3", []byte(`{}`))
	msg.SetContext(context.WithValue(context.Background(), ctxKey{}, val))

	if _, err := wrapped(msg); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
}
