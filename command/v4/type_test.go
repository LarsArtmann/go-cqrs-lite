package command_test

import (
	"errors"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

func TestType_IsZero(t *testing.T) {
	t.Parallel()

	if command.Type("").IsZero() != true {
		t.Error("empty Type should be zero")
	}

	if command.Type("user.create").IsZero() != false {
		t.Error("non-empty Type should not be zero")
	}
}

func TestType_ParseType(t *testing.T) {
	t.Parallel()

	got, err := command.ParseType("user.create")
	if err != nil {
		t.Fatalf("ParseType: %v", err)
	}

	if got != "user.create" {
		t.Errorf("ParseType = %q, want %q", got, "user.create")
	}

	if got.IsZero() {
		t.Error("IsZero should be false for valid type")
	}
}

func TestType_ParseType_Empty(t *testing.T) {
	t.Parallel()

	_, err := command.ParseType("")
	if !errors.Is(err, command.ErrEmptyCommandType) {
		t.Errorf("empty type err = %v, want ErrEmptyCommandType", err)
	}
}

// TestType_IsAliasOfRecord locks the ADR-0111 alias: command.Type must
// remain assignment-compatible with record.Type — the cross-type comparison
// below only compiles while Type is an alias. Reverting to a standalone
// defined type fails this file at compile time.
func TestType_IsAliasOfRecord(t *testing.T) {
	t.Parallel()

	if command.Type("user.create") != record.Type("user.create") {
		t.Error("command.Type must be comparable to record.Type unchanged")
	}
}
