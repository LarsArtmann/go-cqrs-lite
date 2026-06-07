package simulation

import (
	"testing"
)

func TestEventGenerator_Generate(t *testing.T) {
	gen := DefaultUserGenerator()
	events, err := gen.Generate(10)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	if len(events) != 10 {
		t.Fatalf("expected 10 events, got %d", len(events))
	}

	for i, evt := range events {
		if evt.Type() != "UserCreated" {
			t.Fatalf("event %d: expected UserCreated, got %s", i, evt.Type())
		}
		expectedVersion := i + 2 // NewEvents uses version.Add(i+1), so start+1, start+2, ...
		if int(evt.Version()) != expectedVersion {
			t.Fatalf("event %d: expected version %d, got %d", i, expectedVersion, evt.Version())
		}
	}
}

func TestEventGenerator_GenerateMulti(t *testing.T) {
	gen := DefaultUserGenerator()
	events, err := gen.GenerateMulti(5, 20)
	if err != nil {
		t.Fatalf("generate multi: %v", err)
	}

	if len(events) != 100 {
		t.Fatalf("expected 100 events, got %d", len(events))
	}
}

func BenchmarkEventGenerator_Generate(b *testing.B) {
	gen := DefaultUserGenerator()

	for b.Loop() {
		_, err := gen.Generate(100)
		if err != nil {
			b.Fatalf("generate: %v", err)
		}
	}
}
