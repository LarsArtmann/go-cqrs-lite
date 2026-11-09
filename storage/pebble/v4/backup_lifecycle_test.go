package pebble_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/snapshot/v4"
	"github.com/larsartmann/go-cqrs-lite/storage/backuptest/v4"
	"github.com/larsartmann/go-cqrs-lite/storage/pebble/v4"
)

type backupBackend struct{ b *pebble.Backend }

func (a backupBackend) EventStore() event.Store                { return a.b.EventStore() }
func (a backupBackend) SnapshotStore() snapshot.SnapshotStore  { return a.b.SnapshotStore() }
func (a backupBackend) CheckpointStore() event.CheckpointStore { return a.b.CheckpointStore() }
func (a backupBackend) Close() error                           { return a.b.Close() }

func pebbleBackupFactory() backuptest.Factory {
	return backuptest.Factory{
		New: func(t *testing.T) backuptest.Backend {
			b, err := pebble.Open(t.TempDir(), pebble.DefaultOptions(), nil)
			if err != nil {
				t.Fatalf("Open source: %v", err)
			}

			return backupBackend{b}
		},
		Backup: func(t *testing.T, src backuptest.Backend, destPath string) {
			t.Helper()

			b := src.(backupBackend).b

			if err := b.Flush(); err != nil {
				t.Fatalf("Flush before checkpoint: %v", err)
			}

			if err := b.Checkpoint(destPath); err != nil {
				t.Fatalf("Checkpoint: %v", err)
			}
		},
		Restore: func(t *testing.T, backupPath string) backuptest.Backend {
			b, err := pebble.Open(backupPath, pebble.DefaultOptions(), nil)
			if err != nil {
				t.Fatalf("Open restored: %v", err)
			}

			return backupBackend{b}
		},
	}
}

func TestBackupRestore_FullLifecycle(t *testing.T) {
	backuptest.RunFullLifecycle(t, pebbleBackupFactory())
}

func TestBackupRestore_IncrementalCheckpoints(t *testing.T) {
	backuptest.RunIncrementalCheckpoints(t, pebbleBackupFactory())
}
