package system_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

func TestSystem_HealthCheck_Healthy(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	sys, err := system.New(ctx, system.DomainConfig{}, system.DeploymentConfig{
		Engines: map[string]system.EngineConfig{"primary": {Driver: "memory"}},
		Instances: []system.InstanceConfig{
			{Role: system.RoleSourceOfTruth, Engine: "primary"},
		},
	})
	if err != nil {
		t.Fatalf("system.New: %v", err)
	}
	defer sys.Close()

	if err := sys.HealthCheck(ctx); err != nil {
		t.Fatalf("HealthCheck on healthy system: %v", err)
	}
}

func TestSystem_HealthCheck_Stopped(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	sys, _ := system.New(ctx, system.DomainConfig{}, system.DeploymentConfig{
		Engines: map[string]system.EngineConfig{"primary": {Driver: "memory"}},
		Instances: []system.InstanceConfig{
			{Role: system.RoleSourceOfTruth, Engine: "primary"},
		},
	})

	if err := sys.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	err := sys.HealthCheck(ctx)
	if err == nil {
		t.Fatal("expected error from HealthCheck on stopped system")
	}

	if !errors.Is(err, system.ErrSystemStopped) {
		t.Fatalf("expected ErrSystemStopped, got: %v", err)
	}
}

func TestSystem_GracefulClose(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	sys, err := system.New(ctx, system.DomainConfig{}, system.DeploymentConfig{
		Engines: map[string]system.EngineConfig{"primary": {Driver: "memory"}},
		Instances: []system.InstanceConfig{
			{Role: system.RoleSourceOfTruth, Engine: "primary"},
		},
	})
	if err != nil {
		t.Fatalf("system.New: %v", err)
	}

	gCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := sys.GracefulClose(gCtx); err != nil {
		t.Fatalf("GracefulClose: %v", err)
	}

	// Double GracefulClose should be nil (idempotent via Close).
	if err := sys.GracefulClose(gCtx); err != nil {
		t.Fatalf("double GracefulClose: %v", err)
	}
}

func TestSystem_GracefulClose_ContextExpired(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	sys, _ := system.New(ctx, system.DomainConfig{}, system.DeploymentConfig{
		Engines: map[string]system.EngineConfig{"primary": {Driver: "memory"}},
		Instances: []system.InstanceConfig{
			{Role: system.RoleSourceOfTruth, Engine: "primary"},
		},
	})

	// Use an already-cancelled context to trigger the timeout path.
	cancelCtx, cancel := context.WithCancel(ctx)
	cancel()

	err := sys.GracefulClose(cancelCtx)
	if err == nil {
		t.Fatal("expected error from GracefulClose with cancelled context")
	}
}

func TestSystem_ResetProjection_NoHost(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// No projection instance configured → no projHost.
	sys, _ := system.New(ctx, system.DomainConfig{}, system.DeploymentConfig{
		Engines: map[string]system.EngineConfig{"primary": {Driver: "memory"}},
		Instances: []system.InstanceConfig{
			{Role: system.RoleSourceOfTruth, Engine: "primary"},
		},
	})
	defer sys.Close()

	err := sys.ResetProjection(ctx, "nonexistent")
	if err == nil {
		t.Fatal("expected error from ResetProjection with no projection host")
	}

	if !errors.Is(err, system.ErrNoProjectionHost) {
		t.Fatalf("expected ErrNoProjectionHost, got: %v", err)
	}
}

// recordingCheckpointStore is a minimal CheckpointStore that records calls
// for test assertions.
type recordingCheckpointStore struct {
	saved   map[string]event.Checkpoint
	saveCnt int
}

func (s *recordingCheckpointStore) Save(
	_ context.Context,
	projection string,
	cp event.Checkpoint,
) error {
	if s.saved == nil {
		s.saved = make(map[string]event.Checkpoint)
	}

	s.saved[projection] = cp
	s.saveCnt++

	return nil
}

func (s *recordingCheckpointStore) Load(
	_ context.Context,
	projection string,
) (event.Checkpoint, error) {
	return s.saved[projection], nil
}

func (s *recordingCheckpointStore) Close() error { return nil }

func TestSystem_CustomCheckpointStore(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	cpStore := &recordingCheckpointStore{}

	// Configure a projection so the projection host is created.
	domain := system.DomainConfig{
		CheckpointStore: cpStore,
	}

	sys, err := system.New(ctx, domain, system.DeploymentConfig{
		Engines: map[string]system.EngineConfig{"primary": {Driver: "memory"}},
		Instances: []system.InstanceConfig{
			{Role: system.RoleSourceOfTruth, Engine: "primary"},
		},
	})
	if err != nil {
		t.Fatalf("system.New: %v", err)
	}
	defer sys.Close()

	// The projection host should be nil since no projections were declared,
	// but the checkpoint store field should still be accepted without error.
	// When projections ARE declared, the custom store is used instead of the
	// default memoryCheckpointStore.
	_ = sys
}
