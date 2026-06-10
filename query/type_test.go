package query_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/query/v2"
)

func TestType_IsZero(t *testing.T) {
	t.Parallel()

	if query.Type("").IsZero() != true {
		t.Error("empty Type should be zero")
	}

	if query.Type("user.getById").IsZero() != false {
		t.Error("non-empty Type should not be zero")
	}
}

func TestType_ParseType(t *testing.T) {
	t.Parallel()

	got, err := query.ParseType("user.getById")
	if err != nil {
		t.Fatalf("ParseType: %v", err)
	}

	if got != "user.getById" {
		t.Errorf("ParseType = %q, want %q", got, "user.getById")
	}

	if got.IsZero() {
		t.Error("IsZero should be false for valid type")
	}
}

func TestType_ParseType_Empty(t *testing.T) {
	t.Parallel()

	_, err := query.ParseType("")
	if err == nil {
		t.Fatal("expected error for empty type")
	}
}

func TestType_MustParseType(t *testing.T) {
	t.Parallel()

	got := query.MustParseType("user.getById")
	if got != "user.getById" {
		t.Errorf("MustParseType = %q, want %q", got, "user.getById")
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

	query.MustParseType("")
}
