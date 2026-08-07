package system

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// unhealthyEngine is a minimal Engine that implements HealthChecker and always
// returns an error from HealthCheck.
type unhealthyEngine struct {
	profile metaengine.EngineProfile
}

func (e *unhealthyEngine) Profile() metaengine.EngineProfile { return e.profile }
func (e *unhealthyEngine) Close() error                      { return nil }
func (e *unhealthyEngine) HealthCheck(_ context.Context) error {
	return errors.New("simulated engine failure")
}

func TestSystem_HealthCheck_EngineUnhealthy(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	sys, err := New(ctx, DomainConfig{}, DeploymentConfig{
		Engines: map[string]EngineConfig{"primary": {Driver: "memory"}},
		Instances: []InstanceConfig{
			{Role: RoleSourceOfTruth, Engine: "primary"},
		},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer sys.Close()

	// Inject an unhealthy engine into the system's engine list.
	sys.mu.Lock()
	sys.engines = append(sys.engines, &unhealthyEngine{
		profile: metaengine.EngineProfile{Name: "unhealthy-mock"},
	})
	sys.mu.Unlock()

	err = sys.HealthCheck(ctx)
	if err == nil {
		t.Fatal("expected HealthCheck to return error for unhealthy engine")
	}

	if !strings.Contains(err.Error(), "unhealthy-mock") {
		t.Fatalf("expected error to name the unhealthy engine, got: %v", err)
	}

	if !strings.Contains(err.Error(), "simulated engine failure") {
		t.Fatalf("expected error to contain the engine's error message, got: %v", err)
	}
}
