package command_test

import (
	"errors"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/core/command"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

func validRef() command.AggregateRef {
	return command.NewAggregateRef("User", id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95"))
}

func TestNewPersistedCommand_Success(t *testing.T) {
	t.Parallel()

	ref := validRef()
	payload := []byte(`{"name":"Alice"}`)

	cmd, err := command.NewPersistedCommand("CreateUser", ref, payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cmd.Type() != "CreateUser" {
		t.Errorf("Type() = %q, want %q", cmd.Type(), "CreateUser")
	}

	if cmd.AggregateRef() != ref {
		t.Errorf("AggregateRef() = %v, want %v", cmd.AggregateRef(), ref)
	}

	if cmd.AggregateID() != ref.ID {
		t.Errorf("AggregateID() = %v, want %v", cmd.AggregateID(), ref.ID)
	}

	if cmd.AggregateType() != ref.Type {
		t.Errorf("AggregateType() = %v, want %v", cmd.AggregateType(), ref.Type)
	}

	if cmd.ID().IsZero() {
		t.Error("ID() should not be zero")
	}

	if cmd.ReceivedAt().IsZero() {
		t.Error("ReceivedAt() should not be zero")
	}

	if (command.Metadata{}) != cmd.Metadata() {
		t.Error("Metadata() should return zero value")
	}

	gotPayload := cmd.Payload()
	if string(gotPayload) != `{"name":"Alice"}` {
		t.Errorf("Payload() = %q, want %q", gotPayload, `{"name":"Alice"}`)
	}
}

func TestNewPersistedCommand_PayloadIsolation(t *testing.T) {
	t.Parallel()

	ref := validRef()
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

func TestNewPersistedCommand_EmptyType(t *testing.T) {
	t.Parallel()

	ref := validRef()

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

	ref := command.NewAggregateRef("", id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95"))

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

	ref := command.NewAggregateRef("User", id.AggregateID{})

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

	ref := validRef()
	ts := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)

	cmd, err := command.NewPersistedCommand("CreateUser", ref, nil, command.WithReceivedAt(ts))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !cmd.ReceivedAt().Equal(ts) {
		t.Errorf("ReceivedAt() = %v, want %v", cmd.ReceivedAt(), ts)
	}
}

func TestNewPersistedCommand_WithCommandID(t *testing.T) {
	t.Parallel()

	ref := validRef()
	cmdID := id.MustParseCommandID("01HK1540X0841Y0A6BSX1VKR95")

	cmd, err := command.NewPersistedCommand("CreateUser", ref, nil, command.WithCommandID(cmdID))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cmd.ID() != cmdID {
		t.Errorf("ID() = %v, want %v", cmd.ID(), cmdID)
	}
}

func TestNewPersistedCommand_NilPayload(t *testing.T) {
	t.Parallel()

	ref := validRef()

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

	ref := validRef()
	cmdID := id.MustParseCommandID("01HK1540X0841Y0A6BSX1VKR95")

	cmd, err := command.NewPersistedCommand("CreateUser", ref, nil, command.WithCommandID(cmdID))
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

	ref := command.NewAggregateRef("User", id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95"))
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

func TestAggregateType_MustParse_Panics(t *testing.T) {
	t.Parallel()

	defer func() {
		r := recover()
		if r == nil {
			t.Error("expected panic for empty aggregate type")
		}
	}()

	command.MustParseAggregateType("")
}

func TestAggregateType_IsZero(t *testing.T) {
	t.Parallel()

	if command.AggregateType("").IsZero() != true {
		t.Error("empty AggregateType should be zero")
	}

	if command.AggregateType("User").IsZero() != false {
		t.Error("non-empty AggregateType should not be zero")
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

func TestCommandID_MustParse_Panics(t *testing.T) {
	t.Parallel()

	defer func() {
		r := recover()
		if r == nil {
			t.Error("expected panic for invalid command ID")
		}
	}()

	id.MustParseCommandID("not-a-valid-ulid")
}

func TestStoreInterface_CompileTime(t *testing.T) {
	t.Parallel()

	// Store must embed both CommandSink and CommandSource.
	// This is a structural check — Store = CommandSink + CommandSource.
	var _ command.CommandSink = command.Store(nil)
	var _ command.CommandSource = command.Store(nil)
}
