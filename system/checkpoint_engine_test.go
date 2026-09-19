package system_test

import (
	"context"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/sqliteengine/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

// engineCheckpointOpener opens the checkpoint store the same way the System
// constructor does, against a freshly opened engine on the given DSN — the
// restart harness (close engine, reopen, read back).
func engineCheckpointOpener(t *testing.T) func(string) event.CheckpointStore {
	t.Helper()

	return func(dsn string) event.CheckpointStore {
		t.Helper()

		eng, err := sqliteengine.NewSQLiteEngineFromDSN(dsn)
		if err != nil {
			t.Fatalf("NewSQLiteEngineFromDSN: %v", err)
		}

		t.Cleanup(func() { _ = eng.Close() })

		return newEngineCheckpoints(t, eng)
	}
}

func newEngineCheckpoints(t *testing.T, eng metaengine.Engine) event.CheckpointStore {
	t.Helper()

	sys, err := system.New(context.Background(), system.DomainConfig{}, system.DeploymentConfig{
		Engines: map[string]system.EngineConfig{
			"primary":      {Driver: "memory"},
			"checkpoints":  {Driver: "sqlite", DSN: ""},
		},
		Instances: []system.InstanceConfig{{Role: system.RoleSourceOfTruth, Engine: "primary"}},
	})
	if err != nil {
		t.Fatalf("system.New: %v", err)
	}

	_ = sys

	backend, ok := eng.(metaengine.MapBackend)
	if !ok {
		t.Fatal("sqlite engine must carry the Map ADT")
	}

	return newStoreFromBackend(t, backend)
}
