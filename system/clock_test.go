package system_test

import (
	"context"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

// TestSystem_ClockDefaultsToReal pins the default time source: an unoptioned
// New yields a RealClock — the production behavior is unchanged (ADR-0153
// D4's no-behavior-change requirement).
func TestSystem_ClockDefaultsToReal(t *testing.T) {
	t.Parallel()

	sys, err := system.New(context.Background(), system.DomainConfig{}, system.DeploymentConfig{
		Engines:   map[string]system.EngineConfig{"primary": {Driver: "memory"}},
		Instances: []system.InstanceConfig{{Role: system.RoleSourceOfTruth, Engine: "primary"}},
	})
	if err != nil {
		t.Fatalf("system.New: %v", err)
	}

	defer func() { _ = sys.Close() }()

	if _, ok := sys.Clock().(system.RealClock); !ok {
		t.Fatalf("default Clock = %T, want system.RealClock", sys.Clock())
	}
}

// TestSystem_WithClockInjectsManualClock pins the injection surface and the
// ManualClock semantics: frozen reads, monotonic Advance, Set.
func TestSystem_WithClockInjectsManualClock(t *testing.T) {
	t.Parallel()

	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	manual := system.NewManualClock(start)

	sys, err := system.New(
		context.Background(),
		system.DomainConfig{},
		system.DeploymentConfig{
			Engines:   map[string]system.EngineConfig{"primary": {Driver: "memory"}},
			Instances: []system.InstanceConfig{{Role: system.RoleSourceOfTruth, Engine: "primary"}},
		},
		system.WithClock(manual),
	)
	if err != nil {
		t.Fatalf("system.New: %v", err)
	}

	defer func() { _ = sys.Close() }()

	if got := sys.Clock().Now(); !got.Equal(start) {
		t.Fatalf("injected clock reads %s, want frozen %s", got, start)
	}

	manual.Advance(90 * time.Minute)
	if got := sys.Clock().Now(); !got.Equal(start.Add(90 * time.Minute)) {
		t.Fatalf("after Advance clock reads %s, want %s", got, start.Add(90*time.Minute))
	}

	manual.Set(start.AddDate(1, 0, 0))
	if got := sys.Clock().Now(); !got.Equal(start.AddDate(1, 0, 0)) {
		t.Fatalf("after Set clock reads %s, want %s", got, start.AddDate(1, 0, 0))
	}
}
