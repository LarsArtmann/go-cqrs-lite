package storage_test

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/storage/v2/sql"
)

var update = flag.Bool("update", false, "update golden files")

func TestGolden_PostgresEventSchema(t *testing.T) {
	got := sql.PostgresDialect{}.EventSchema()
	assertStorageGolden(t, filepath.Join("testdata", "golden", "postgres-events.sql"), []byte(got))
}

func TestGolden_PostgresCommandSchema(t *testing.T) {
	got := sql.PostgresDialect{}.CommandSchema()
	assertStorageGolden(
		t,
		filepath.Join("testdata", "golden", "postgres-commands.sql"),
		[]byte(got),
	)
}

func TestGolden_PostgresSnapshotSchema(t *testing.T) {
	got := sql.PostgresDialect{}.SnapshotSchema()
	assertStorageGolden(
		t,
		filepath.Join("testdata", "golden", "postgres-snapshots.sql"),
		[]byte(got),
	)
}

func TestGolden_PostgresCheckpointSchema(t *testing.T) {
	got := sql.PostgresDialect{}.CheckpointSchema()
	assertStorageGolden(
		t,
		filepath.Join("testdata", "golden", "postgres-checkpoints.sql"),
		[]byte(got),
	)
}

func TestGolden_SQLiteEventSchema(t *testing.T) {
	got := sql.SQLiteDialect{}.EventSchema()
	assertStorageGolden(t, filepath.Join("testdata", "golden", "sqlite-events.sql"), []byte(got))
}

func TestGolden_SQLiteCommandSchema(t *testing.T) {
	got := sql.SQLiteDialect{}.CommandSchema()
	assertStorageGolden(t, filepath.Join("testdata", "golden", "sqlite-commands.sql"), []byte(got))
}

func TestGolden_SQLiteSnapshotSchema(t *testing.T) {
	got := sql.SQLiteDialect{}.SnapshotSchema()
	assertStorageGolden(t, filepath.Join("testdata", "golden", "sqlite-snapshots.sql"), []byte(got))
}

func TestGolden_SQLiteCheckpointSchema(t *testing.T) {
	got := sql.SQLiteDialect{}.CheckpointSchema()
	assertStorageGolden(
		t,
		filepath.Join("testdata", "golden", "sqlite-checkpoints.sql"),
		[]byte(got),
	)
}

func assertStorageGolden(t *testing.T, path string, got []byte) {
	t.Helper()

	if *update {
		if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
			t.Fatalf("mkdir: %v", err)
		}

		if err := os.WriteFile(path, append(got, '\n'), 0o644); err != nil {
			t.Fatalf("write golden: %v", err)
		}

		return
	}

	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden %s (run with -update to create): %v", path, err)
	}

	if strings.TrimSpace(string(got)) != strings.TrimSpace(string(want)) {
		t.Errorf("golden mismatch for %s (run with -update to refresh)", path)
	}
}
