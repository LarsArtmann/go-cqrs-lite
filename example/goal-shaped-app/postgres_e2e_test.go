package main

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

// TestGoal_PostgresSwapEndToEnd boots the WHOLE app on a real Postgres —
// the boot-proven half of the operator-swap story (the config-level swap is
// pinned driver-less by TestGoal_OperatorSwapsDriverByConfig). Runs only
// when a Postgres DSN is in the environment; the standing playbook is
// `nix run .#integration-pg` (sets POSTGRES_TEST_DSN/DATABASE_URL).
func TestGoal_PostgresSwapEndToEnd(t *testing.T) {
	dsn := os.Getenv("POSTGRES_TEST_DSN")
	if dsn == "" {
		dsn = os.Getenv("DATABASE_URL")
	}

	if dsn == "" {
		t.Skip("POSTGRES_TEST_DSN/DATABASE_URL not set — skipping Postgres e2e (use nix run .#integration-pg)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	deployment := system.DeploymentConfig{
		Engines: map[string]system.EngineConfig{
			"primary": {Driver: "postgres", DSN: dsn},
		},
		Instances: []system.InstanceConfig{
			{Role: system.RoleSourceOfTruth, Engine: "primary"},
			{Role: system.RoleProjections, Engine: "primary"},
		},
	}

	sys, err := system.New(ctx, Domain(), deployment)
	if err != nil {
		t.Fatalf("system.New on postgres: %v", err)
	}

	defer sys.Close()

	// runStory starts the system itself.
	view, err := runStory(ctx, sys)
	if err != nil {
		t.Fatalf("story on postgres: %v", err)
	}

	// Doctor must name the operator's engine, not sqlite's.
	if doctor := sys.MetaEngine().Doctor(ctx); !strings.Contains(doctor, "postgres") {
		t.Fatalf("Doctor should name postgres:\n%s", doctor)
	}

	_ = view
}
