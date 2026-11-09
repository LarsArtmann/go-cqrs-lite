package system_test

import (
	"context"
	"strings"
	"testing"
	"time"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

// deduplessEngine is the minimal Engine without the DedupStore capability —
// the error-path fixture for System.Dedup.
type deduplessEngine struct{}

func (deduplessEngine) Profile() metaengine.EngineProfile {
	return metaengine.EngineProfile{Name: "dedupless"}
}

func (deduplessEngine) Close() error { return nil }

func dedupDeployment(t *testing.T, engines map[string]system.EngineConfig) system.DeploymentConfig {
	t.Helper()

	return system.DeploymentConfig{
		Engines: engines,
		Instances: []system.InstanceConfig{
			{Role: system.RoleSourceOfTruth, Engine: "primary"},
			{Role: system.RoleProjections, Engine: "primary"},
		},
	}
}

func dedupDomain() system.DomainConfig {
	return system.DomainConfig{}
}

// TestSystem_Dedup_NamedEnginePreferred pins the resolution order: an engine
// named "dedup" wins over the first-configured engine (ADR-0158).
func TestSystem_Dedup_NamedEnginePreferred(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	sys, err := system.New(ctx, dedupDomain(), dedupDeployment(t, map[string]system.EngineConfig{
		"primary": {Driver: "memory"},
		"dedup":   {Driver: "memory"},
	}))
	if err != nil {
		t.Fatalf("system.New: %v", err)
	}

	defer sys.Close()

	dedup, err := sys.Dedup()
	if err != nil {
		t.Fatalf("Dedup: %v", err)
	}

	seen, err := dedup.DedupCheckAndRecord(ctx, "probe", "k1", time.Minute, time.Now())
	if err != nil {
		t.Fatalf("DedupCheckAndRecord: %v", err)
	}

	if seen {
		t.Fatal("first CAS for a fresh key must return seen=false")
	}
}

// TestSystem_Dedup_FallsBackToFirstEngine: no "dedup" name — the first
// configured engine is capability-asserted (memory implements DedupStore).
func TestSystem_Dedup_FallsBackToFirstEngine(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	sys, err := system.New(ctx, dedupDomain(), dedupDeployment(t, map[string]system.EngineConfig{
		"primary": {Driver: "memory"},
	}))
	if err != nil {
		t.Fatalf("system.New: %v", err)
	}

	defer sys.Close()

	if _, err := sys.Dedup(); err != nil {
		t.Fatalf("Dedup fallback: %v", err)
	}
}

// TestSystem_Dedup_NoCapabilityNamesTheEngines: an engine without the
// capability produces an error that names the inspected engines — a
// deployment-declaration fix, not a mystery (ADR-0158).
func TestSystem_Dedup_NoCapabilityNamesTheEngines(t *testing.T) {
	t.Parallel()

	driverName := "dedupless-" + t.Name()
	metaengine.RegisterDriver(
		driverName,
		func(_ context.Context, _ metaengine.DriverConfig) (metaengine.Engine, error) {
			return deduplessEngine{}, nil
		},
	)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	sys, err := system.New(ctx, dedupDomain(), dedupDeployment(t, map[string]system.EngineConfig{
		"primary": {Driver: "memory"},
		"dedup":   {Driver: driverName},
	}))
	if err != nil {
		t.Fatalf("system.New: %v", err)
	}

	defer sys.Close()

	_, err = sys.Dedup()
	if err == nil {
		t.Fatal("Dedup must fail on a capability-less engine")
	}

	for _, want := range []string{"engine \"dedup\"", "does not implement metaengine.DedupStore"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error must name %q, got: %v", want, err)
		}
	}
}
