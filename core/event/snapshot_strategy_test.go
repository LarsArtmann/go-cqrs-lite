package event_test

import (
	"errors"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/core/event"
)

func TestEveryNEvents_ReturnsErrorForZero(t *testing.T) {
	t.Parallel()

	_, err := event.EveryNEvents(0)
	if !errors.Is(err, event.ErrInvalidSnapshotInterval) {
		t.Errorf("EveryNEvents(0) err = %v, want ErrInvalidSnapshotInterval", err)
	}
}

func TestEveryNEvents_ReturnsErrorForNegative(t *testing.T) {
	t.Parallel()

	_, err := event.EveryNEvents(-5)
	if !errors.Is(err, event.ErrInvalidSnapshotInterval) {
		t.Errorf("EveryNEvents(-5) err = %v, want ErrInvalidSnapshotInterval", err)
	}
}

func TestEveryNEvents_Success(t *testing.T) {
	t.Parallel()

	strategy, err := event.EveryNEvents(5)
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

	event.MustEveryNEvents(0)
}

func TestMustEveryNEvents_Success(t *testing.T) {
	t.Parallel()

	s := event.MustEveryNEvents(3)
	if s == nil {
		t.Fatal("MustEveryNEvents(3) returned nil")
	}
}
