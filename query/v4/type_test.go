package query_test

import (
	"errors"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/query/v4"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
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
	if !errors.Is(err, query.ErrEmptyQueryType) {
		t.Errorf("empty type err = %v, want ErrEmptyQueryType", err)
	}
}

// TestType_IsAliasOfRecord locks the ADR-0111 alias: query.Type must remain
// assignment-compatible with record.Type — the cross-type comparison below
// only compiles while Type is an alias. Reverting to a standalone defined
// type fails this file at compile time.
func TestType_IsAliasOfRecord(t *testing.T) {
	t.Parallel()

	if query.Type("user.getById") != record.Type("user.getById") {
		t.Error("query.Type must be comparable to record.Type unchanged")
	}
}
