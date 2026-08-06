package benchkit

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/stack/memory/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/v4"
)

// TestSkippedPhases_MetaEngineMissing verifies that when a bundle has no
// MetaEngine, the result records a skip warning instead of silently passing.
func TestSkippedPhases_MetaEngineMissing(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	factory := func() (*stack.Bundle, error) {
		return memory.New()
	}

	result, err := Run(ctx, Config{
		Profile:     ProfileDev,
		PayloadSize: 64,
		Backend:     "memory",
	}, factory)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if !slices.Contains(result.SkippedPhases, "metaengine phase") {
		t.Errorf("expected 'metaengine phase' in SkippedPhases, got %v",
			result.SkippedPhases)
	}

	found := false
	for _, w := range result.Warnings {
		if strings.Contains(w, "metaengine") {
			found = true

			break
		}
	}
	if !found {
		t.Errorf("expected a warning about metaengine, got %v", result.Warnings)
	}
}

// TestSkippedPhases_ConfigFlags verifies that config-level skip flags are
// recorded in SkippedPhases.
func TestSkippedPhases_ConfigFlags(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	factory := func() (*stack.Bundle, error) {
		return memory.New()
	}

	result, err := Run(ctx, Config{
		Profile:        ProfileDev,
		PayloadSize:    64,
		Backend:        "memory",
		SkipReads:      true,
		SkipMetaEngine: true,
	}, factory)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if !slices.Contains(result.SkippedPhases, "read phase") {
		t.Errorf("expected 'read phase' in SkippedPhases, got %v",
			result.SkippedPhases)
	}
	if !slices.Contains(result.SkippedPhases, "metaengine phase") {
		t.Errorf("expected 'metaengine phase' in SkippedPhases, got %v",
			result.SkippedPhases)
	}
}

// TestStrictMode_FailsOnSkip verifies that Strict mode returns an error when
// phases are skipped.
func TestStrictMode_FailsOnSkip(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	factory := func() (*stack.Bundle, error) {
		return memory.New()
	}

	_, err := Run(ctx, Config{
		Profile:     ProfileDev,
		PayloadSize: 64,
		Backend:     "memory",
		Strict:      true,
	}, factory)
	if err == nil {
		t.Fatal("expected error in strict mode (metaengine will be skipped)")
	}
	if !errors.Is(err, ErrStrictSkip) {
		t.Errorf("expected ErrStrictSkip, got: %v", err)
	}
}

// TestSkippedPhases_MinimalBundle verifies that nil-guard warnings fire when
// a bundle lacks required components (no SnapshotStore, no CheckpointStore,
// no ReadModels, no SeekableJournal). A minimal EventStore-only bundle should
// record skips for projection, checkpoint, snapshot, query, journey, and
// read-model phases.
func TestSkippedPhases_MinimalBundle(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	factory := func() (*stack.Bundle, error) {
		return stack.New(
			stack.WithEventStore(newMinimalEventStore()),
		)
	}

	result, err := Run(ctx, Config{
		Profile:        ProfileDev,
		PayloadSize:    64,
		Backend:        "minimal",
		SkipMetaEngine: true,
	}, factory)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	expectedSkips := []string{
		"read model phase",
		"projection phase",
		"checkpoint phase",
		"snapshot phase",
		"query phase",
		"journey phase",
	}

	for _, expected := range expectedSkips {
		found := slices.Contains(result.SkippedPhases, expected)
		if !found {
			t.Errorf("expected SkippedPhases to contain %q, got: %v",
				expected, result.SkippedPhases)
		}
	}
}

type minimalEventStore struct{}

func (m *minimalEventStore) Save(_ context.Context, _ id.StreamRef, _ []event.Event, _ event.Version) error {
	return nil
}

func (m *minimalEventStore) AppendBatch(_ context.Context, _ id.StreamRef, _ []event.Event) error {
	return nil
}

func (m *minimalEventStore) Load(_ context.Context, _ id.StreamRef) ([]event.Event, error) {
	return nil, nil
}

func (m *minimalEventStore) LoadFromVersion(_ context.Context, _ id.StreamRef, _ event.Version) ([]event.Event, error) {
	return nil, nil
}

func (m *minimalEventStore) LoadToVersion(_ context.Context, _ id.StreamRef, _ event.Version) ([]event.Event, error) {
	return nil, nil
}

func (m *minimalEventStore) LoadToTimestamp(_ context.Context, _ id.StreamRef, _ time.Time) ([]event.Event, error) {
	return nil, nil
}

func newMinimalEventStore() event.Store {
	return &minimalEventStore{}
}
// TestStrictMode_PassesWhenNothingSkipped verifies strict mode succeeds when
// all phases run (using SkipMetaEngine to avoid the nil-MetaEngine skip, plus
// all other skip flags false).
// NOTE: This can only work if the bundle has all required components. Memory
// backend has no MetaEngine, so we skip MetaEngine via config and set Strict.
// But Strict checks SkippedPhases — so we need a way to exclude config skips
// from strict? No — strict means NO skips at all. So this test verifies that
// strict mode correctly fails even for config-initiated skips.
func TestStrictMode_ConfigSkipAlsoFails(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	factory := func() (*stack.Bundle, error) {
		return memory.New()
	}

	_, err := Run(ctx, Config{
		Profile:        ProfileDev,
		PayloadSize:    64,
		Backend:        "memory",
		Strict:         true,
		SkipMetaEngine: true,
	}, factory)
	if err == nil {
		t.Fatal("expected error in strict mode with SkipMetaEngine=true")
	}
	if !errors.Is(err, ErrStrictSkip) {
		t.Errorf("expected ErrStrictSkip, got: %v", err)
	}
}

// TestSkippedPhases_MinimalBundle verifies that nil-guard warnings fire when
// a bundle lacks required components (no SnapshotStore, no CheckpointStore,
// no ReadModels, no SeekableJournal). A minimal EventStore-only bundle should
// record skips for projection, checkpoint, snapshot, query, journey, and
// read-model phases.
func TestSkippedPhases_MinimalBundle(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	factory := func() (*stack.Bundle, error) {
		return stack.New(
			stack.WithEventStore(newMinimalEventStore()),
		)
	}

	result, err := Run(ctx, Config{
		Profile:        ProfileDev,
		PayloadSize:    64,
		Backend:        "minimal",
		SkipMetaEngine: true,
	}, factory)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	expectedSkips := []string{
		"read model phase",
		"projection phase",
		"checkpoint phase",
		"snapshot phase",
		"query phase",
		"journey phase",
	}

	for _, expected := range expectedSkips {
		found := slices.Contains(result.SkippedPhases, expected)
		if !found {
			t.Errorf("expected SkippedPhases to contain %q, got: %v",
				expected, result.SkippedPhases)
		}
	}
}

type minimalEventStore struct{}

func (m *minimalEventStore) Save(_ context.Context, _ id.StreamRef, _ []event.Event, _ event.Version) error {
	return nil
}

func (m *minimalEventStore) AppendBatch(_ context.Context, _ id.StreamRef, _ []event.Event) error {
	return nil
}

func (m *minimalEventStore) Load(_ context.Context, _ id.StreamRef) ([]event.Event, error) {
	return nil, nil
}

func (m *minimalEventStore) LoadFromVersion(_ context.Context, _ id.StreamRef, _ event.Version) ([]event.Event, error) {
	return nil, nil
}

func (m *minimalEventStore) LoadToVersion(_ context.Context, _ id.StreamRef, _ event.Version) ([]event.Event, error) {
	return nil, nil
}

func (m *minimalEventStore) LoadToTimestamp(_ context.Context, _ id.StreamRef, _ time.Time) ([]event.Event, error) {
	return nil, nil
}

func newMinimalEventStore() event.Store {
	return &minimalEventStore{}
}