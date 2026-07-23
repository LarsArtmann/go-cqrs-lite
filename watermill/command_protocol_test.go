package watermill_test

import (
	"testing"

	"github.com/ThreeDotsLabs/watermill/message"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	wm "github.com/larsartmann/go-cqrs-lite/watermill/v4"
)

func TestCommandRoundTrip(t *testing.T) {
	t.Parallel()

	aggID := id.NewStreamID()
	correlationID := id.NewCorrelationID()
	causationID := id.NewCausationID()
	userID := id.NewUserID()
	requestID := id.NewRequestID()

	original, err := command.New(
		"user.create", aggID,
		command.WithCorrelationID(correlationID),
		command.WithCausationID(causationID),
		command.WithUserID(userID),
		command.WithRequestID(requestID),
		command.WithCustomMetadata("tenant", "acme"),
		command.WithCustomMetadata("source", "web"),
	)
	if err != nil {
		t.Fatalf("create command: %v", err)
	}

	msg := wm.CommandToMessage(original)

	if msg.Metadata.Get("command_type") != "user.create" {
		t.Fatalf("command_type: got %q, want %q",
			msg.Metadata.Get("command_type"), "user.create")
	}
	if msg.Metadata.Get("aggregate_id") != aggID.String() {
		t.Fatalf("aggregate_id mismatch")
	}
	if msg.Metadata.Get("correlation_id") != correlationID.String() {
		t.Fatalf("correlation_id mismatch")
	}
	if msg.Metadata.Get("causation_id") != causationID.String() {
		t.Fatalf("causation_id mismatch")
	}
	if msg.Metadata.Get("user_id") != userID.String() {
		t.Fatalf("user_id mismatch")
	}
	if msg.Metadata.Get("request_id") != requestID.String() {
		t.Fatalf("request_id mismatch")
	}
	if msg.Metadata.Get("custom.tenant") != "acme" {
		t.Fatalf("custom.tenant: got %q", msg.Metadata.Get("custom.tenant"))
	}
	if msg.Metadata.Get("custom.source") != "web" {
		t.Fatalf("custom.source: got %q", msg.Metadata.Get("custom.source"))
	}

	reconstructed, err := wm.MessageToCommand("user.create", msg)
	if err != nil {
		t.Fatalf("MessageToCommand: %v", err)
	}

	if reconstructed.Type() != original.Type() {
		t.Fatalf("type: got %q, want %q", reconstructed.Type(), original.Type())
	}
	if reconstructed.StreamID() != original.StreamID() {
		t.Fatalf("aggregate_id mismatch after round-trip")
	}

	md := reconstructed.Metadata()
	if md.CorrelationID != correlationID {
		t.Fatalf("correlation_id mismatch after round-trip")
	}
	if md.CausationID != causationID {
		t.Fatalf("causation_id mismatch after round-trip")
	}
	if md.UserID != userID {
		t.Fatalf("user_id mismatch after round-trip")
	}
	if md.RequestID != requestID {
		t.Fatalf("request_id mismatch after round-trip")
	}
	if md.Custom["tenant"] != "acme" {
		t.Fatalf("custom.tenant mismatch after round-trip")
	}
	if md.Custom["source"] != "web" {
		t.Fatalf("custom.source mismatch after round-trip")
	}
}

func TestMessageToCommand_MissingAggregateID(t *testing.T) {
	t.Parallel()

	msg := message.NewMessage("test-1", nil)
	msg.Metadata.Set("command_type", "user.create")

	_, err := wm.MessageToCommand("user.create", msg)
	if err == nil {
		t.Fatal("expected error for missing aggregate_id")
	}
}

func TestMessageToCommand_EmptyType(t *testing.T) {
	t.Parallel()

	msg := message.NewMessage("test-2", nil)
	msg.Metadata.Set("aggregate_id", id.NewStreamID().String())

	_, err := wm.MessageToCommand("", msg)
	if err == nil {
		t.Fatal("expected error for empty command type")
	}
}

func TestMessageToCommand_TopicFallback(t *testing.T) {
	t.Parallel()

	aggID := id.NewStreamID()
	msg := message.NewMessage("test-3", nil)
	msg.Metadata.Set("aggregate_id", aggID.String())

	cmd, err := wm.MessageToCommand("user.delete", msg)
	if err != nil {
		t.Fatalf("MessageToCommand: %v", err)
	}

	if cmd.Type() != "user.delete" {
		t.Fatalf("type: got %q, want %q (from topic)", cmd.Type(), "user.delete")
	}
}

func TestCommandToMessage_NoMetadata(t *testing.T) {
	t.Parallel()

	aggID := id.NewStreamID()
	cmd, err := command.New("user.create", aggID)
	if err != nil {
		t.Fatalf("create command: %v", err)
	}

	msg := wm.CommandToMessage(cmd)

	if msg.Metadata.Get("command_type") != "user.create" {
		t.Fatalf("command_type mismatch")
	}
	if msg.Metadata.Get("aggregate_id") != aggID.String() {
		t.Fatalf("aggregate_id mismatch")
	}
	if msg.Metadata.Get("correlation_id") != "" {
		t.Fatalf("correlation_id should be empty")
	}
	if msg.Metadata.Get("custom.tenant") != "" {
		t.Fatalf("custom.tenant should be empty")
	}
}

func TestCommandToMessage_GeneratesCommandID(t *testing.T) {
	t.Parallel()

	aggID := id.NewStreamID()
	cmd, _ := command.New("user.create", aggID)

	msg := wm.CommandToMessage(cmd)

	if msg.UUID == "" {
		t.Fatal("message UUID should be non-empty (command ID)")
	}
	if msg.Metadata.Get("command_id") == "" {
		t.Fatal("command_id metadata should be set")
	}
	if msg.UUID != msg.Metadata.Get("command_id") {
		t.Fatalf("UUID (%q) should match command_id (%q)",
			msg.UUID, msg.Metadata.Get("command_id"))
	}
}

func TestCommandToMessage_StableID(t *testing.T) {
	t.Parallel()

	aggID := id.NewStreamID()
	cmd, _ := command.New("user.create", aggID)

	msg1 := wm.CommandToMessage(cmd)
	msg2 := wm.CommandToMessage(cmd)

	// Same command instance → same message UUID (stable ID for dedup).
	if msg1.UUID != msg2.UUID {
		t.Fatalf("same command should produce stable UUID: %q != %q", msg1.UUID, msg2.UUID)
	}

	// Different command instances → different UUIDs (auto-minted in New).
	cmd2, _ := command.New("user.create", aggID)
	msg3 := wm.CommandToMessage(cmd2)
	if msg1.UUID == msg3.UUID {
		t.Fatal("different command instances should have different command IDs")
	}
}

func TestCommandRoundTrip_MinimalCommand(t *testing.T) {
	t.Parallel()

	aggID := id.NewStreamID()
	original, err := command.New("noop", aggID)
	if err != nil {
		t.Fatalf("create command: %v", err)
	}

	msg := wm.CommandToMessage(original)
	reconstructed, err := wm.MessageToCommand("noop", msg)
	if err != nil {
		t.Fatalf("MessageToCommand: %v", err)
	}

	if reconstructed.Type() != "noop" {
		t.Fatalf("type mismatch")
	}
	if reconstructed.StreamID() != aggID {
		t.Fatalf("aggregate_id mismatch")
	}
}

func TestCommandRoundTrip_MultipleCustomMetadata(t *testing.T) {
	t.Parallel()

	aggID := id.NewStreamID()
	original, err := command.New(
		"bulk.op", aggID,
		command.WithCustomMetadata("k1", "v1"),
		command.WithCustomMetadata("k2", "v2"),
		command.WithCustomMetadata("k3", "v3"),
	)
	if err != nil {
		t.Fatalf("create command: %v", err)
	}

	msg := wm.CommandToMessage(original)
	reconstructed, err := wm.MessageToCommand("bulk.op", msg)
	if err != nil {
		t.Fatalf("MessageToCommand: %v", err)
	}

	md := reconstructed.Metadata()
	for _, key := range []string{"k1", "k2", "k3"} {
		if md.Custom[command.MetadataKey(key)] == "" {
			t.Fatalf("custom metadata key %q lost in round-trip", key)
		}
	}
}

func TestCommandToMessage_InvalidCorrelationIDIgnored(t *testing.T) {
	t.Parallel()

	msg := message.NewMessage("test-4", nil)
	msg.Metadata.Set("command_type", "user.create")
	msg.Metadata.Set("aggregate_id", id.NewStreamID().String())
	msg.Metadata.Set("correlation_id", "not-a-valid-ulid")

	cmd, err := wm.MessageToCommand("user.create", msg)
	if err != nil {
		t.Fatalf("invalid correlation_id should not fail the whole parse: %v", err)
	}
	if !cmd.Metadata().CorrelationID.IsZero() {
		t.Fatal("correlation_id should be zero for invalid input")
	}
}
