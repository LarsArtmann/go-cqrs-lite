package record_test

import (
	"errors"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

func TestType_String(t *testing.T) {
	t.Parallel()

	if got := record.Type("user.created").String(); got != "user.created" {
		t.Errorf("String() = %q, want %q", got, "user.created")
	}

	if got := record.Type("").String(); got != "" {
		t.Errorf("empty String() = %q, want %q", got, "")
	}
}

func TestType_IsZero(t *testing.T) {
	t.Parallel()

	if !record.Type("").IsZero() {
		t.Error("empty Type must be zero")
	}

	if record.Type("x").IsZero() {
		t.Error("non-empty Type must not be zero")
	}
}

func TestParseType(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("empty")

	got, err := record.ParseType("user.created", sentinel)
	if err != nil {
		t.Fatalf("ParseType: %v", err)
	}

	if got != record.Type("user.created") {
		t.Errorf("ParseType = %q, want %q", got, "user.created")
	}

	if _, err := record.ParseType("", sentinel); !errors.Is(err, sentinel) {
		t.Errorf("empty input err = %v, want the caller's sentinel", err)
	}
}
