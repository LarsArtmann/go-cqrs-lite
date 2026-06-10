package snapshot_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/snapshot/v2"
)

func mustEveryN(n int) snapshot.SnapshotStrategy {
	s, err := snapshot.EveryNEvents(n)
	if err != nil {
		panic(err)
	}
	return s
}


func TestEveryNEvents_ReturnsErrorForZero(t *testing.T) {
	t.Parallel()

	_, err := snapshot.EveryNEvents(0)
	if err == nil {
		t.Error("EveryNEvents(0) should return error")
	}
}

func TestEveryNEvents_ReturnsErrorForNegative(t *testing.T) {
	t.Parallel()

	_, err := snapshot.EveryNEvents(-5)
	if err == nil {
		t.Error("EveryNEvents(-5) should return error")
	}
}

func TestEveryNEvents_Success(t *testing.T) {
	t.Parallel()

	strategy, err := snapshot.EveryNEvents(5)
	if err != nil {
		t.Fatalf("EveryNEvents(5) err = %v", err)
	}

	tests := []struct {
		version  int
		expected bool
	}{
		{0, false},
		{1, false},
		{4, false},
		{5, true},
		{10, true},
		{15, true},
	}

	for _, tt := range tests {
		v, _ := event.ParseVersion(tt.version)
		got := strategy.ShouldSnapshot("User", v)
		if got != tt.expected {
			t.Errorf("ShouldSnapshot(User, %d) = %v, want %v", tt.version, got, tt.expected)
		}
	}
}

func TestMustEveryNEvents_PanicsOnZero(t *testing.T) {
	t.Parallel()

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic for MustEveryNEvents(0)")
		}
	}()

	mustEveryN(0)
}

func TestMustEveryNEvents_Success(t *testing.T) {
	t.Parallel()

	s := mustEveryN(3)
	if s == nil {
		t.Fatal("MustEveryNEvents(3) returned nil")
	}
}
