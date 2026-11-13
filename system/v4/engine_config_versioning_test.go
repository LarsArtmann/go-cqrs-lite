package system

import (
	"context"
	"strings"
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	_ "github.com/larsartmann/go-cqrs-lite/metaengine/sqliteengine/v4" // registers "sqlite"
)

// TestCellVersioningConfigRetention pins the operator-config translation:
// valid knobs parse into the metaengine policy, a malformed max_age fails
// with an actionable duration error, and negative knobs fail validation.
func TestCellVersioningConfigRetention(t *testing.T) {
	t.Parallel()

	valid := &CellVersioningConfig{MaxVersions: 10, MaxAge: "720h"}
	policy, err := valid.retention()
	if err != nil {
		t.Fatalf("retention: %v", err)
	}

	if policy.MaxVersions != 10 || policy.MaxAge.String() != "720h0m0s" {
		t.Fatalf("policy = %+v, want MaxVersions 10 / MaxAge 720h", policy)
	}

	emptyAge := &CellVersioningConfig{MaxVersions: 1}
	if policy, err = emptyAge.retention(); err != nil || policy.MaxAge != 0 {
		t.Fatalf("empty max_age must mean unlimited, got (%+v, %v)", policy, err)
	}

	badAge := &CellVersioningConfig{MaxAge: "next tuesday"}
	if _, err = badAge.retention(); err == nil || !strings.Contains(err.Error(), "max_age") {
		t.Fatalf("malformed max_age: got %v, want a duration-parse error naming max_age", err)
	}

	negative := &CellVersioningConfig{MaxVersions: -1}
	if _, err = negative.retention(); err == nil || !strings.Contains(err.Error(), "max_versions") {
		t.Fatalf("negative max_versions: got %v, want the validation error", err)
	}
}

// TestCreateEngineFromDriver_CellVersioning pins the EngineConfig →
// DriverConfig → engine link of the ADR-0141 config surface (issue #131):
// a declared cell_versioning reaches the driver as a versioned engine,
// and the error path names the engine.
func TestCreateEngineFromDriver_CellVersioning(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	versioned, err := createEngineFromDriver(ctx, EngineConfig{
		Driver:         "sqlite",
		DSN:            t.TempDir() + "/versioned.db",
		CellVersioning: &CellVersioningConfig{MaxVersions: 5},
	}, "")
	if err != nil {
		t.Fatalf("createEngineFromDriver: %v", err)
	}
	defer versioned.Close() //nolint:errcheck // test teardown

	if !metaengine.EngineVersionsCells(versioned) {
		t.Fatal("EngineVersionsCells = false, want the config to enable versioning")
	}

	plain, err := createEngineFromDriver(ctx, EngineConfig{
		Driver: "sqlite",
		DSN:    t.TempDir() + "/plain.db",
	}, "")
	if err != nil {
		t.Fatalf("createEngineFromDriver (plain): %v", err)
	}
	defer plain.Close() //nolint:errcheck // test teardown

	if metaengine.EngineVersionsCells(plain) {
		t.Fatal("EngineVersionsCells(plain) = true, want nil config to leave versioning off")
	}

	_, err = createEngineFromDriver(ctx, EngineConfig{
		Driver:         "sqlite",
		DSN:            t.TempDir() + "/bad.db",
		CellVersioning: &CellVersioningConfig{MaxAge: "not-a-duration"},
	}, "")
	if err == nil || !strings.Contains(err.Error(), "cell_versioning") {
		t.Fatalf("malformed config: got %v, want a cell_versioning-labeled error", err)
	}
}
