package storage_test

import (
	"flag"
	"path/filepath"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4/eventtest"
	"github.com/larsartmann/go-cqrs-lite/storage/v4/sql"
)

var update = flag.Bool("update", false, "update golden files")

func TestGolden_PostgresEventSchema(t *testing.T) {
	got := sql.PostgresDialect{}.EventSchema()
	eventtest.AssertGolden(
		t,
		filepath.Join("testdata", "golden", "postgres-events.sql"),
		[]byte(got),
		*update,
	)
}

func TestGolden_PostgresCommandSchema(t *testing.T) {
	got := sql.PostgresDialect{}.CommandSchema()
	eventtest.AssertGolden(
		t,
		filepath.Join("testdata", "golden", "postgres-commands.sql"),
		[]byte(got),
		*update,
	)
}

func TestGolden_PostgresSnapshotSchema(t *testing.T) {
	got := sql.PostgresDialect{}.SnapshotSchema()
	eventtest.AssertGolden(
		t,
		filepath.Join("testdata", "golden", "postgres-snapshots.sql"),
		[]byte(got),
		*update,
	)
}

func TestGolden_PostgresCheckpointSchema(t *testing.T) {
	got := sql.PostgresDialect{}.CheckpointSchema()
	eventtest.AssertGolden(
		t,
		filepath.Join("testdata", "golden", "postgres-checkpoints.sql"),
		[]byte(got),
		*update,
	)
}

func TestGolden_SQLiteEventSchema(t *testing.T) {
	got := sql.SQLiteDialect{}.EventSchema()
	eventtest.AssertGolden(
		t,
		filepath.Join("testdata", "golden", "sqlite-events.sql"),
		[]byte(got),
		*update,
	)
}

func TestGolden_SQLiteCommandSchema(t *testing.T) {
	got := sql.SQLiteDialect{}.CommandSchema()
	eventtest.AssertGolden(
		t,
		filepath.Join("testdata", "golden", "sqlite-commands.sql"),
		[]byte(got),
		*update,
	)
}

func TestGolden_SQLiteSnapshotSchema(t *testing.T) {
	got := sql.SQLiteDialect{}.SnapshotSchema()
	eventtest.AssertGolden(
		t,
		filepath.Join("testdata", "golden", "sqlite-snapshots.sql"),
		[]byte(got),
		*update,
	)
}

func TestGolden_SQLiteCheckpointSchema(t *testing.T) {
	got := sql.SQLiteDialect{}.CheckpointSchema()
	eventtest.AssertGolden(
		t,
		filepath.Join("testdata", "golden", "sqlite-checkpoints.sql"),
		[]byte(got),
		*update,
	)
}
