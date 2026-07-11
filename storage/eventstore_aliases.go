package storage

import (
	"github.com/larsartmann/go-cqrs-lite/storage/v4/eventstore"
)

// Re-exports for backward compatibility. Consumers importing
// storage.SQLEventStore etc. continue to work. New code should import
// the eventstore package directly.

type (
	SQLEventStore      = eventstore.SQLEventStore
	SQLSnapshotStore   = eventstore.SQLSnapshotStore
	SQLCheckpointStore = eventstore.SQLCheckpointStore
	EventByIDLoader    = eventstore.EventByIDLoader
)

var (
	NewSQLEventStore                 = eventstore.NewSQLEventStore
	NewSQLiteEventStore              = eventstore.NewSQLiteEventStore
	NewSQLEventStoreWithDialect      = eventstore.NewSQLEventStoreWithDialect
	NewSQLSnapshotStore              = eventstore.NewSQLSnapshotStore
	NewSQLiteSnapshotStore           = eventstore.NewSQLiteSnapshotStore
	NewSQLSnapshotStoreWithDialect   = eventstore.NewSQLSnapshotStoreWithDialect
	NewSQLCheckpointStore            = eventstore.NewSQLCheckpointStore
	NewSQLiteCheckpointStore         = eventstore.NewSQLiteCheckpointStore
	NewSQLCheckpointStoreWithDialect = eventstore.NewSQLCheckpointStoreWithDialect
	SnapshotSchema                   = eventstore.SnapshotSchema
	SQLiteSnapshotSchema             = eventstore.SQLiteSnapshotSchema
	CheckpointSchema                 = eventstore.CheckpointSchema
	SQLiteCheckpointSchema           = eventstore.SQLiteCheckpointSchema
)
