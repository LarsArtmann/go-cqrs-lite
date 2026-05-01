package event

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

type nopCheckpointStore struct{}

func (nopCheckpointStore) Load(_ context.Context, _ string) (id.EventID, error) {
	return id.EventID{}, nil
}

func (nopCheckpointStore) Save(_ context.Context, _ string, _ id.EventID) error {
	return nil
}

func TestInMemoryRunner_NilCheckpoint(t *testing.T) {
	t.Parallel()

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic for nil CheckpointStore")
		}

		msg, ok := r.(string)
		if !ok {
			t.Fatalf("panic value = %v (%T), want string", r, r)
		}

		if msg != "event: nil CheckpointStore" {
			t.Errorf("panic message = %q, want %q", msg, "event: nil CheckpointStore")
		}
	}()

	NewInMemoryRunner(nil)
}

func TestInMemoryRunner_RegisterNilProjection(t *testing.T) {
	t.Parallel()

	runner := NewInMemoryRunner(nopCheckpointStore{})

	err := runner.Register(nil)
	if err == nil {
		t.Fatal("expected error for nil projection")
	}
}

func TestInMemoryRunner_RegisterDuplicateName(t *testing.T) {
	t.Parallel()

	runner := NewInMemoryRunner(nopCheckpointStore{})

	proj := NewProjection("dup", func(_ context.Context, _ Event) error { return nil }, nil)

	err := runner.Register(proj)
	if err != nil {
		t.Fatalf("first Register: %v", err)
	}

	err = runner.Register(proj)
	if err == nil {
		t.Fatal("expected error for duplicate projection name")
	}
}
