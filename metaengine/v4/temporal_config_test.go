package metaengine_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// TestRetentionPolicyValidate pins the loud half of the ADR-0141 config
// surface (issue #131): retention knobs must be zero (unlimited) or
// positive — negatives read as "keep minus N versions", which no engine
// implements.
func TestRetentionPolicyValidate(t *testing.T) {
	cases := []struct {
		name    string
		policy  metaengine.RetentionPolicy
		wantErr string
	}{
		{"zero keeps everything", metaengine.RetentionPolicy{}, ""},
		{"max versions only", metaengine.RetentionPolicy{MaxVersions: 3}, ""},
		{"max age only", metaengine.RetentionPolicy{MaxAge: time.Hour}, ""},
		{"both knobs", metaengine.RetentionPolicy{MaxVersions: 3, MaxAge: time.Hour}, ""},
		{"negative versions", metaengine.RetentionPolicy{MaxVersions: -1}, "max_versions"},
		{"negative age", metaengine.RetentionPolicy{MaxAge: -time.Hour}, "max_age"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.policy.Validate()

			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("Validate() = %v, want nil", err)
				}

				return
			}

			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("Validate() = %v, want error naming %s", err, tc.wantErr)
			}
		})
	}
}

// TestRejectCellVersioning pins the construction-time contract for engines
// that cannot version cells: nil passes (versioning off is the default),
// any non-nil policy fails loudly with ErrUnsupportedCellVersioning, and an
// invalid policy fails on validation instead.
func TestRejectCellVersioning(t *testing.T) {
	if err := metaengine.RejectCellVersioning("pebble", metaengine.DriverConfig{}); err != nil {
		t.Fatalf("nil policy must pass, got %v", err)
	}

	policy := metaengine.RetentionPolicy{MaxVersions: 5}
	err := metaengine.RejectCellVersioning("pebble", metaengine.DriverConfig{CellVersioning: &policy})
	if !errors.Is(err, metaengine.ErrUnsupportedCellVersioning) {
		t.Fatalf("errors.Is(ErrUnsupportedCellVersioning) = false, got %v", err)
	}

	bad := metaengine.RetentionPolicy{MaxVersions: -2}
	err = metaengine.RejectCellVersioning("pebble", metaengine.DriverConfig{CellVersioning: &bad})
	if err == nil || !strings.Contains(err.Error(), "max_versions") {
		t.Fatalf("invalid policy: got %v, want the validation error", err)
	}
}

// TestMemoryDriver_CellVersioningFlowsThrough pins the driver-model half of
// the config surface: the registered memory factory translates
// DriverConfig.CellVersioning into the versioning constructor and applies
// retention (MaxVersions=1 collapses to latest-only), while a nil policy
// keeps the plain unversioned engine.
func TestMemoryDriver_CellVersioningFlowsThrough(t *testing.T) {
	factory, err := metaengine.LookupDriver("memory")
	if err != nil {
		t.Fatalf("LookupDriver: %v", err)
	}

	ctx := context.Background()

	policy := metaengine.RetentionPolicy{MaxVersions: 1}
	versioned, err := factory(ctx, metaengine.DriverConfig{CellVersioning: &policy})
	if err != nil {
		t.Fatalf("versioned factory: %v", err)
	}
	defer versioned.Close() //nolint:errcheck // test teardown

	if !metaengine.EngineVersionsCells(versioned) {
		t.Fatal("EngineVersionsCells(versioned) = false, want the policy to enable versioning")
	}

	writer, ok := versioned.(metaengine.VersionedWriter)
	if !ok {
		t.Fatal("versioned memory engine does not implement VersionedWriter")
	}

	now := time.Now()
	if err := writer.MapSetAt(ctx, "users", "u1", "old", now.Add(-time.Hour)); err != nil {
		t.Fatalf("MapSetAt old: %v", err)
	}

	if err := writer.MapSetAt(ctx, "users", "u1", "new", now); err != nil {
		t.Fatalf("MapSetAt new: %v", err)
	}

	reader, ok := versioned.(metaengine.CellHistoryReader)
	if !ok {
		t.Fatal("versioned memory engine does not implement CellHistoryReader")
	}

	history, err := reader.MapHistory(ctx, "users", "u1", time.Time{}, now.Add(time.Hour))
	if err != nil {
		t.Fatalf("MapHistory: %v", err)
	}

	if len(history) != 1 {
		t.Fatalf("len(history) = %d, want 1 (MaxVersions=1 trims the older version)", len(history))
	}

	plain, err := factory(ctx, metaengine.DriverConfig{})
	if err != nil {
		t.Fatalf("plain factory: %v", err)
	}
	defer plain.Close() //nolint:errcheck // test teardown

	if metaengine.EngineVersionsCells(plain) {
		t.Fatal("EngineVersionsCells(plain) = true, want nil policy to leave versioning off")
	}
}
