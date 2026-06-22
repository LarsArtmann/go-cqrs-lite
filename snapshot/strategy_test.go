package snapshot_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/snapshot/v3"
)

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
		v, _ := event.ParseVersion(uint64(tt.version))
		got := strategy.ShouldSnapshot("User", v)
		if got != tt.expected {
			t.Errorf("ShouldSnapshot(User, %d) = %v, want %v", tt.version, got, tt.expected)
		}
	}
}
