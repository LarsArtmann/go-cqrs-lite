package command_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/command/v2"
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
	if err == nil {
		t.Fatal("expected error for empty type")
	}
}

func TestType_MustParseType(t *testing.T) {
	t.Parallel()

	got := command.MustParseType("user.create")
	if got != "user.create" {
		t.Errorf("MustParseType = %q, want %q", got, "user.create")
	}
}

func TestType_MustParseType_Panics(t *testing.T) {
	t.Parallel()

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic for empty type")
		}
	}()

	command.MustParseType("")
}
