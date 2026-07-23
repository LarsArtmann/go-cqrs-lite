package command_test

import (
	"errors"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4/idtest"
)

func parseCommandID(tb testing.TB, s string) id.CommandID {
	tb.Helper()

	v, err := id.ParseCommandID(s)
	if err != nil {
		tb.Fatalf("parseCommandID %q: %v", s, err)
	}

	return v
}

func validRef(tb testing.TB) command.StreamRef {
	tb.Helper()

	return command.NewAggregateRef(
		"User",
		idtest.ParseAggregateID(tb, "01HK1540X0841Y0A6BSX1VKR95"),
	)
}

func TestNewPersistedCommand_Success(t *testing.T) {
	t.Parallel()

	ref := validRef(t)
	payload := []byte(`{"name":"Alice"}`)

	cmd, err := command.NewPersistedCommand("CreateUser", ref, payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cmd.Type() != "CreateUser" {
		t.Errorf("Type() = %q, want %q", cmd.Type(), "CreateUser")
	}

	if cmd.StreamRef() != ref {
		t.Errorf("StreamRef() = %v, want %v", cmd.StreamRef(), ref)
	}

	if cmd.StreamID() != ref.ID {
		t.Errorf("StreamID() = %v, want %v", cmd.StreamID(), ref.ID)
	}

	if cmd.StreamType() != ref.Type {
		t.Errorf("StreamType() = %v, want %v", cmd.StreamType(), ref.Type)
	}

	if cmd.ID().IsZero() {
		t.Error("ID() should not be zero")
	}

	if cmd.ReceivedAt().IsZero() {
		t.Error("ReceivedAt() should not be zero")
	}

	meta := cmd.Metadata()
	if !meta.CorrelationID.IsZero() || !meta.CausationID.IsZero() || !meta.UserID.IsZero() ||
		!meta.RequestID.IsZero() {
		t.Error("Metadata() should return zero tracing fields")
	}

	gotPayload := cmd.Payload()
	if string(gotPayload) != `{"name":"Alice"}` {
		t.Errorf("Payload() = %q, want %q", gotPayload, `{"name":"Alice"}`)
	}
}

func TestNewPersistedCommand_PayloadIsolation(t *testing.T) {
	t.Parallel()

	ref := validRef(t)
	payload := []byte(`{"name":"Alice"}`)

	cmd, err := command.NewPersistedCommand("CreateUser", ref, payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	returned := cmd.Payload()
	returned[0] = 'X'

	original := cmd.Payload()
	if original[0] == 'X' {
		t.Error("mutating Payload() return value should not affect internal state")
	}

	payload[0] = 'Y'
	again := cmd.Payload()
	if again[0] == 'Y' {
		t.Error("mutating original payload slice should not affect PersistedCommand")
	}
}

func TestNewPersistedCommand_MetadataIsolation(t *testing.T) {
	t.Parallel()

	ref := validRef(t)
	meta := command.Metadata{}
	meta.EnsureCustom()
	meta.Custom["key1"] = "value1"

	cmd, err := command.NewPersistedCommand(
		"CreateUser", ref, nil,
		command.WithCommandMetadata(meta),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	returned := cmd.Metadata()
	returned.Custom["key1"] = "modified"

	original := cmd.Metadata()
	if original.Custom["key1"] != "value1" {
		t.Error("mutating Metadata() return value should not affect internal state")
	}
}

func TestWithCommandMetadata_IntakeIsolation(t *testing.T) {
	t.Parallel()

	ref := validRef(t)
	meta := command.Metadata{}
	meta.EnsureCustom()
	meta.Custom["key"] = "original"

	cmd, err := command.NewPersistedCommand(
		"CreateUser", ref, nil,
		command.WithCommandMetadata(meta),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	meta.Custom["key"] = "mutated_after_passing"

	got := cmd.Metadata()
	if got.Custom["key"] != "original" {
		t.Error("WithCommandMetadata stored caller's map reference — mutation leaked into command")
	}
}

func TestNewPersistedCommand_EmptyType(t *testing.T) {
	t.Parallel()

	ref := validRef(t)

	_, err := command.NewPersistedCommand("", ref, nil)
	if err == nil {
		t.Fatal("expected error for empty command type")
	}

	if !errors.Is(err, command.ErrEmptyCommandType) {
		t.Errorf("errors.Is(err, ErrEmptyCommandType) = false, got: %v", err)
	}
}

func TestNewPersistedCommand_EmptyAggregateType(t *testing.T) {
	t.Parallel()

	ref := command.NewAggregateRef("", idtest.ParseAggregateID(t, "01HK1540X0841Y0A6BSX1VKR95"))

	_, err := command.NewPersistedCommand("CreateUser", ref, nil)
	if err == nil {
		t.Fatal("expected error for empty aggregate type")
	}

	if !errors.Is(err, command.ErrEmptyAggregateType) {
		t.Errorf("errors.Is(err, ErrEmptyAggregateType) = false, got: %v", err)
	}
}

func TestNewPersistedCommand_ZeroAggregateID(t *testing.T) {
	t.Parallel()

	ref := command.NewAggregateRef("User", id.StreamID{})

	_, err := command.NewPersistedCommand("CreateUser", ref, nil)
	if err == nil {
		t.Fatal("expected error for zero aggregate ID")
	}

	if !errors.Is(err, command.ErrNilAggregateID) {
		t.Errorf("errors.Is(err, ErrNilAggregateID) = false, got: %v", err)
	}
}

func TestNewPersistedCommand_WithReceivedAt(t *testing.T) {
	t.Parallel()

	ref := validRef(t)
	ts := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)

	cmd, err := command.NewPersistedCommand("CreateUser", ref, nil, command.WithReceivedAt(ts))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !cmd.ReceivedAt().Equal(ts) {
		t.Errorf("ReceivedAt() = %v, want %v", cmd.ReceivedAt(), ts)
	}
}

func TestNewPersistedCommand_WithPersistedCommandID(t *testing.T) {
	t.Parallel()

	ref := validRef(t)
	cmdID := parseCommandID(t, "01HK1540X0841Y0A6BSX1VKR95")

	cmd, err := command.NewPersistedCommand(
		"CreateUser",
		ref,
		nil,
		command.WithPersistedCommandID(cmdID),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cmd.ID() != cmdID {
		t.Errorf("ID() = %v, want %v", cmd.ID(), cmdID)
	}
}

func TestNewPersistedCommand_NilPayload(t *testing.T) {
	t.Parallel()

	ref := validRef(t)

	cmd, err := command.NewPersistedCommand("CreateUser", ref, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cmd.Payload() != nil {
		t.Errorf("Payload() should be nil for nil input, got %v", cmd.Payload())
	}
}

func TestPersistedCommand_String(t *testing.T) {
	t.Parallel()

	ref := validRef(t)
	cmdID := parseCommandID(t, "01HK1540X0841Y0A6BSX1VKR95")

	cmd, err := command.NewPersistedCommand(
		"CreateUser",
		ref,
		nil,
		command.WithPersistedCommandID(cmdID),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	s := cmd.String()
	if s == "" {
		t.Error("String() should not be empty")
	}

	if len(s) < 10 {
		t.Errorf("String() seems too short: %q", s)
	}
}

func TestAggregateRef_String(t *testing.T) {
	t.Parallel()

	ref := command.NewAggregateRef(
		"User",
		idtest.ParseAggregateID(t, "01HK1540X0841Y0A6BSX1VKR95"),
	)
	s := ref.String()

	if s == "" {
		t.Error("String() should not be empty")
	}
}

func TestAggregateType_Parse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{name: "valid", input: "User", wantErr: false},
		{name: "empty", input: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := command.ParseAggregateType(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error")
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if string(got) != tt.input {
				t.Errorf("ParseAggregateType(%q) = %q, want %q", tt.input, got, tt.input)
			}
		})
	}
}

func TestAggregateType_IsZero(t *testing.T) {
	t.Parallel()

	if command.StreamType("").IsZero() != true {
		t.Error("empty StreamType should be zero")
	}

	if command.StreamType("User").IsZero() != false {
		t.Error("non-empty StreamType should not be zero")
	}
}

func TestCommandID_NewAndParse(t *testing.T) {
	t.Parallel()

	cmdID := id.NewCommandID()
	if cmdID.IsZero() {
		t.Error("NewCommandID() should not be zero")
	}

	parsed, err := id.ParseCommandID(cmdID.String())
	if err != nil {
		t.Fatalf("ParseCommandID() error: %v", err)
	}

	if parsed != cmdID {
		t.Errorf("ParseCommandID roundtrip failed: got %v, want %v", parsed, cmdID)
	}
}

func TestCommandID_ParseEmpty(t *testing.T) {
	t.Parallel()

	_, err := id.ParseCommandID("")
	if err == nil {
		t.Error("expected error for empty string")
	}
}

func TestStoreInterface_CompileTime(t *testing.T) {
	t.Parallel()

	// Store must embed both CommandSink and CommandSource.
	// This is a structural check — Store = CommandSink + CommandSource.
	var _ command.CommandSink = command.Store(nil)
	var _ command.CommandSource = command.Store(nil)
}
