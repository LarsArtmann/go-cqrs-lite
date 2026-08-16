package bbolt

import (
	"os"
	"path/filepath"
	"testing"

	bolt "go.etcd.io/bbolt"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/snapshot/v4"
	"github.com/larsartmann/go-cqrs-lite/storage/backuptest/v4"
)

type backupBackend struct{ b *Backend }

func (a backupBackend) EventStore() event.Store                { return a.b.EventStore() }
func (a backupBackend) SnapshotStore() snapshot.SnapshotStore  { return a.b.SnapshotStore() }
func (a backupBackend) CheckpointStore() event.CheckpointStore { return a.b.CheckpointStore() }
func (a backupBackend) Close() error                           { return a.b.Close() }

func bboltBackupFactory() backuptest.Factory {
	return backuptest.Factory{
		New: func(t *testing.T) backuptest.Backend {
			b, err := Open(filepath.Join(t.TempDir(), "source.db"), nil)
			if err != nil {
				t.Fatalf("open source: %v", err)
			}

			return backupBackend{b}
		},
		Backup: func(t *testing.T, src backuptest.Backend, destPath string) {
			t.Helper()

			db := src.(backupBackend).b.DB()

			f, err := os.Create(destPath)
			if err != nil {
				t.Fatalf("create backup file: %v", err)
			}

			defer f.Close()

			if err := db.View(func(tx *bolt.Tx) error {
				_, err := tx.WriteTo(f)
				return err
			}); err != nil {
				t.Fatalf("backup: %v", err)
			}
		},
		Restore: func(t *testing.T, backupPath string) backuptest.Backend {
			b, err := Open(backupPath, nil)
			if err != nil {
				t.Fatalf("open restored: %v", err)
			}

			return backupBackend{b}
		},
	}
}

func TestBackupRestore_FullLifecycle(t *testing.T) {
	backuptest.RunFullLifecycle(t, bboltBackupFactory())
}

func TestBackupRestore_IncrementalCheckpoints(t *testing.T) {
	backuptest.RunIncrementalCheckpoints(t, bboltBackupFactory())
}
