package event

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

func TestWithCommandCausality_RoundTrip(t *testing.T) {
	t.Parallel()

	cmdID := id.NewCommandID()
	ctx := WithCommandCausality(context.Background(), "create_user", cmdID)

	gotType, gotID, ok := CommandCausalityFromContext(ctx)
	if !ok {
		t.Fatal("expected causality to be present, got ok=false")
	}

	if gotType != "create_user" {
		t.Errorf("commandType = %q, want %q", gotType, "create_user")
	}

	if gotID != cmdID {
		t.Errorf("commandID = %v, want %v", gotID, cmdID)
	}
}

func TestCommandCausalityFromContext_NotSet(t *testing.T) {
	t.Parallel()

	t.Run("empty context", func(t *testing.T) {
		t.Parallel()

		gotType, gotID, ok := CommandCausalityFromContext(context.Background())
		if ok {
			t.Error("expected ok=false on empty context")
		}

		if gotType != "" {
			t.Errorf("commandType = %q, want empty", gotType)
		}

		if !gotID.IsZero() {
			t.Errorf("commandID = %v, want zero", gotID)
		}
	})

	t.Run("unrelated context value", func(t *testing.T) {
		t.Parallel()

		ctx := context.WithValue(context.Background(), ctxKeyCausality{}, "wrong-type")
		_, _, ok := CommandCausalityFromContext(ctx)
		if ok {
			t.Error("expected ok=false when context value is wrong type")
		}
	})
}

func TestWithCommandCausality_Overwrites(t *testing.T) {
	t.Parallel()

	firstID := id.NewCommandID()
	secondID := id.NewCommandID()

	ctx := WithCommandCausality(context.Background(), "create_user", firstID)
	ctx = WithCommandCausality(ctx, "update_user", secondID)

	gotType, gotID, ok := CommandCausalityFromContext(ctx)
	if !ok {
		t.Fatal("expected causality to be present after overwrite")
	}

	if gotType != "update_user" {
		t.Errorf("commandType = %q, want %q (latest)", gotType, "update_user")
	}

	if gotID != secondID {
		t.Errorf("commandID = %v, want %v (latest)", gotID, secondID)
	}
}

func TestWithCommandCausality_DoesNotMutateParent(t *testing.T) {
	t.Parallel()

	cmdID := id.NewCommandID()
	parent := context.Background()
	child := WithCommandCausality(parent, "create_user", cmdID)

	// Parent must remain causality-free.
	_, _, ok := CommandCausalityFromContext(parent)
	if ok {
		t.Error("parent context was mutated by WithCommandCausality")
	}

	// Child has the value.
	_, _, ok = CommandCausalityFromContext(child)
	if !ok {
		t.Error("child context lost the causality value")
	}
}

func TestCommandCausalityEnricher_WithCausality(t *testing.T) {
	t.Parallel()

	cmdID := id.NewCommandID()
	ctx := WithCommandCausality(context.Background(), "create_user", cmdID)

	opts := CommandCausalityEnricher(ctx)
	if opts == nil {
		t.Fatal("expected non-nil options when causality is set")
	}

	if len(opts) != 3 {
		t.Fatalf("expected 3 options (causation + type + id), got %d", len(opts))
	}

	// Apply options to a real event and inspect the resulting metadata.
	streamID := id.NewStreamID()
	evt, err := NewEvent("user.created", streamID, "User", Version(1), nil, opts...)
	if err != nil {
		t.Fatalf("NewEvent: %v", err)
	}

	meta := evt.Metadata()

	if meta.Custom[MetadataKeyCommandType] != "create_user" {
		t.Errorf("metadata[%s] = %q, want %q",
			MetadataKeyCommandType, meta.Custom[MetadataKeyCommandType], "create_user")
	}

	if meta.Custom[MetadataKeyCommandID] != cmdID.String() {
		t.Errorf("metadata[%s] = %q, want %q",
			MetadataKeyCommandID, meta.Custom[MetadataKeyCommandID], cmdID.String())
	}
}

func TestCommandCausalityEnricher_WithoutCausality(t *testing.T) {
	t.Parallel()

	opts := CommandCausalityEnricher(context.Background())
	if opts != nil {
		t.Fatalf("expected nil options when no causality is set, got %d options", len(opts))
	}
}

func TestCommandCausalityEnricher_EndToEnd(t *testing.T) {
	t.Parallel()

	// Simulate the realistic flow: handler sets causality, decider applies
	// the enricher options, the resulting event carries command metadata.
	cmdID := id.NewCommandID()
	ctx := WithCommandCausality(context.Background(), "create_user", cmdID)

	// Decider would call the enricher and append its options to NewEvent.
	streamID := id.NewStreamID()
	evt, err := NewEvent(
		"user.created", streamID, "User", Version(1),
		[]byte(`{"name":"Alice"}`),
		CommandCausalityEnricher(ctx)...,
	)
	if err != nil {
		t.Fatalf("NewEvent: %v", err)
	}

	meta := evt.Metadata()

	// Typed Causation field (ADR-0031).
	if meta.Causation == nil {
		t.Fatal("expected typed Causation to be set")
	}

	if meta.Causation.CommandType != "create_user" {
		t.Errorf("Causation.CommandType = %q, want %q",
			meta.Causation.CommandType, "create_user")
	}

	if meta.Causation.CommandID != cmdID {
		t.Errorf("Causation.CommandID = %v, want %v",
			meta.Causation.CommandID, cmdID)
	}

	// Backward-compatible Custom entries.
	if meta.Custom[MetadataKeyCommandType] != "create_user" {
		t.Errorf("command.type = %q, want %q",
			meta.Custom[MetadataKeyCommandType], "create_user")
	}

	if meta.Custom[MetadataKeyCommandID] != cmdID.String() {
		t.Errorf("command.id = %q, want %q",
			meta.Custom[MetadataKeyCommandID], cmdID.String())
	}
}

func TestCommandCausalityEnricher_PropagatesAcrossGoroutines(t *testing.T) {
	t.Parallel()

	// Context is immutable, so passing it to goroutines must preserve the
	// causality value without races.
	cmdID := id.NewCommandID()
	ctx := WithCommandCausality(context.Background(), "create_user", cmdID)

	done := make(chan struct{})

	const goroutines = 10
	for range goroutines {
		go func() {
			defer func() { done <- struct{}{} }()

			gotType, gotID, ok := CommandCausalityFromContext(ctx)
			if !ok {
				t.Error("causality missing in goroutine")
				return
			}

			if gotType != "create_user" || gotID != cmdID {
				t.Errorf("goroutine saw wrong causality: type=%q id=%v", gotType, gotID)
			}
		}()
	}

	for range goroutines {
		<-done
	}
}
