package main

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/stack/v4"
)

func TestParseDurability_Valid(t *testing.T) {
	t.Parallel()

	cases := []struct {
		input    string
		tier     stack.DurabilityTier
		isSet    bool
	}{
		{"strict", stack.DurabilityStrict, true},
		{"STRICT", stack.DurabilityStrict, true},
		{" strict ", stack.DurabilityStrict, true},
		{"normal", stack.DurabilityNormal, true},
		{"relaxed", stack.DurabilityRelaxed, true},
		{"", stack.DurabilityNormal, false},
	}

	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			t.Parallel()

			tier, isSet, err := parseDurability(tc.input)
			if err != nil {
				t.Fatalf("parseDurability(%q) error: %v", tc.input, err)
			}

			if tier != tc.tier {
				t.Errorf("tier = %q, want %q", tier, tc.tier)
			}

			if isSet != tc.isSet {
				t.Errorf("isSet = %v, want %v", isSet, tc.isSet)
			}
		})
	}
}

func TestParseDurability_InvalidReturnsError(t *testing.T) {
	t.Parallel()

	_, _, err := parseDurability("bogus")
	if err == nil {
		t.Fatal("parseDurability(\"bogus\") should return error, not call os.Exit")
	}
}

func TestMakeFactory_MemoryWithDurability(t *testing.T) {
	t.Parallel()

	// makeFactory calls fatalf on invalid durability, which calls os.Exit.
	// This test verifies valid input doesn't crash and returns a working
	// factory closure.
	factory, _, cleanup := makeFactory("memory", "", "", "strict")
	if cleanup != nil {
		defer cleanup()
	}

	if factory == nil {
		t.Fatal("factory is nil")
	}

	b, err := factory()
	if err != nil {
		t.Fatalf("factory(): %v", err)
	}

	defer func() { _ = b.Close() }()

	if b.Durability() != stack.DurabilityStrict {
		t.Fatalf("Durability() = %q, want %q", b.Durability(), stack.DurabilityStrict)
	}
}

func TestMakeFactory_TursoBackend(t *testing.T) {
	t.Parallel()

	factory, _, cleanup := makeFactory("turso", "", "", "")
	if cleanup == nil {
		t.Fatal("turso factory should have cleanup for temp dir")
	}

	defer cleanup()

	if factory == nil {
		t.Fatal("factory is nil")
	}

	b, err := factory()
	if err != nil {
		t.Fatalf("factory(): %v", err)
	}

	defer func() { _ = b.Close() }()
}
