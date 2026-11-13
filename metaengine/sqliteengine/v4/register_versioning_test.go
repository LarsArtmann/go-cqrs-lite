package sqliteengine

import (
	"context"
	"strings"
	"testing"
	"time"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// TestDriverFactory_CellVersioningFlowsThrough pins the DriverConfig ->
// engine temporal plumbing (issue #131, ADR-0141): a non-nil
// CellVersioning policy builds a versioned engine with retention applied,
// a nil policy keeps the engine unversioned, and an invalid policy fails
// construction loudly.
func TestDriverFactory_CellVersioningFlowsThrough(t *testing.T) {
	factory, err := metaengine.LookupDriver("sqlite")
	if err != nil {
		t.Fatalf("LookupDriver: %v", err)
	}

	ctx := context.Background()

	policy := metaengine.RetentionPolicy{MaxVersions: 2}
	versioned, err := factory(ctx, metaengine.DriverConfig{
		DSN:            t.TempDir() + "/versioned.db",
		CellVersioning: &policy,
	})
	if err != nil {
		t.Fatalf("versioned factory: %v", err)
	}
	defer versioned.Close() //nolint:errcheck // test teardown

	if !metaengine.EngineVersionsCells(versioned) {
		t.Fatal("EngineVersionsCells(versioned) = false, want the policy to enable versioning")
	}

	writer, ok := versioned.(metaengine.VersionedWriter)
	if !ok {
		t.Fatal("versioned sqlite engine does not implement VersionedWriter")
	}

	now := time.Now()
	for i, value := range []string{"v1", "v2", "v3"} {
		if err := writer.MapSetAt(ctx, "users", "u1", value, now.Add(time.Duration(i)*time.Minute)); err != nil {
			t.Fatalf("MapSetAt %s: %v", value, err)
		}
	}

	reader, ok := versioned.(metaengine.CellHistoryReader)
	if !ok {
		t.Fatal("versioned sqlite engine does not implement CellHistoryReader")
	}

	history, err := reader.MapHistory(ctx, "users", "u1", time.Time{}, now.Add(time.Hour))
	if err != nil {
		t.Fatalf("MapHistory: %v", err)
	}

	if len(history) != 2 {
		t.Fatalf("len(history) = %d, want 2 (MaxVersions=2 trims the oldest)", len(history))
	}

	plain, err := factory(ctx, metaengine.DriverConfig{DSN: t.TempDir() + "/plain.db"})
	if err != nil {
		t.Fatalf("plain factory: %v", err)
	}
	defer plain.Close() //nolint:errcheck // test teardown

	if metaengine.EngineVersionsCells(plain) {
		t.Fatal("EngineVersionsCells(plain) = true, want nil policy to leave versioning off")
	}
}

// TestDriverFactory_CellVersioningRejectsInvalidPolicy pins the validation
// gate inside the driver factory: a negative knob never reaches the engine.
func TestDriverFactory_CellVersioningRejectsInvalidPolicy(t *testing.T) {
	factory, err := metaengine.LookupDriver("sqlite")
	if err != nil {
		t.Fatalf("LookupDriver: %v", err)
	}

	bad := metaengine.RetentionPolicy{MaxVersions: -3}
	_, err = factory(context.Background(), metaengine.DriverConfig{
		DSN:            t.TempDir() + "/bad.db",
		CellVersioning: &bad,
	})
	if err == nil || !strings.Contains(err.Error(), "max_versions") {
		t.Fatalf("factory = %v, want the validation error naming max_versions", err)
	}
}
