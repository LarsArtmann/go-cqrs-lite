package sqliteengine

import (
	"context"
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// TestDriverFactory_MaxOpenConnsFlowsThrough pins the DriverConfig ->
// engine pool plumbing (issue #112, the Kith read-storm class): the
// serialized MaxOpenConns(1) default is DELIBERATE, zero keeps it, and a
// raised ceiling reaches the owned database/sql handle. The factory cannot
// hand the handle out, so the observable is DBStats on the returned engine.
func TestDriverFactory_MaxOpenConnsFlowsThrough(t *testing.T) {
	factory, err := metaengine.LookupDriver("sqlite")
	if err != nil {
		t.Fatalf("LookupDriver: %v", err)
	}

	eng, err := factory(context.Background(), metaengine.DriverConfig{DSN: t.TempDir() + "/pool-default.db"})
	if err != nil {
		t.Fatalf("default: %v", err)
	}
	defer eng.Close() //nolint:errcheck // test teardown

	if got := DBStats(eng).MaxOpenConnections; got != 1 {
		t.Fatalf("default MaxOpenConnections = %d, want the deliberate 1", got)
	}

	eng, err = factory(context.Background(), metaengine.DriverConfig{
		DSN:         t.TempDir() + "/pool-raised.db",
		MaxOpenConns: 8,
	})
	if err != nil {
		t.Fatalf("raised: %v", err)
	}
	defer eng.Close() //nolint:errcheck // test teardown

	if got := DBStats(eng).MaxOpenConnections; got != 8 {
		t.Fatalf("raised MaxOpenConnections = %d, want 8", got)
	}
}
