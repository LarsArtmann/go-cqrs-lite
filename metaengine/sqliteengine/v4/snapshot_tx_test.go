package sqliteengine_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/metaengine/sqliteengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

type snapshotBackend interface {
	SnapshotSave(ctx context.Context, collection, streamID string, version int64, data []byte) error
	SnapshotLoad(ctx context.Context, collection, streamID string) ([]byte, int64, error)
	SnapshotLoadAtVersion(
		ctx context.Context, collection, streamID string, maxVersion int64,
	) ([]byte, int64, error)
	SnapshotDelete(ctx context.Context, collection, streamID string) error
}

// With the constructors' default single-connection pool a snapshot call that
// bypasses the ambient transaction cannot get a connection: it blocks until
// the context is cancelled. The snapshot methods must join the transaction.
func TestSnapshot_MethodsJoinAmbientTransaction(t *testing.T) {
	t.Parallel()

	eng, err := sqliteengine.NewSQLiteEngineFromDSN(":memory:")
	if err != nil {
		t.Fatalf("NewSQLiteEngineFromDSN: %v", err)
	}

	t.Cleanup(func() { _ = eng.Close() })

	sb := eng.(snapshotBackend)
	tx := eng.(metaengine.Transactional)

	// A short deadline turns the blocked-forever bug into a fast failure.
	tctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = tx.RunInTx(tctx, func(ctx context.Context) error {
		if err := sb.SnapshotSave(ctx, "snaps_tx", "s1", 5, []byte("state")); err != nil {
			return err
		}

		data, version, err := sb.SnapshotLoad(ctx, "snaps_tx", "s1")
		if err != nil {
			return err
		}

		if string(data) != "state" || version != 5 {
			t.Errorf("SnapshotLoad inside tx: data=%q version=%d", data, version)
		}

		if _, _, err := sb.SnapshotLoadAtVersion(ctx, "snaps_tx", "s1", 4); err == nil {
			t.Error("SnapshotLoadAtVersion above maxVersion must be ErrNotFound")
		}

		return nil
	})
	if err != nil {
		t.Fatalf("snapshot calls inside RunInTx: %v", err)
	}

	data, _, err := sb.SnapshotLoad(context.Background(), "snaps_tx", "s1")
	if err != nil || string(data) != "state" {
		t.Fatalf("post-commit SnapshotLoad: data=%q err=%v", data, err)
	}

	err = tx.RunInTx(context.Background(), func(ctx context.Context) error {
		if err := sb.SnapshotDelete(ctx, "snaps_tx", "s1"); err != nil {
			return err
		}

		if _, _, err := sb.SnapshotLoad(ctx, "snaps_tx", "s1"); err == nil {
			t.Error("snapshot must be gone inside the tx that deleted it")
		}

		return nil
	})
	if err != nil {
		t.Fatalf("SnapshotDelete inside RunInTx: %v", err)
	}

	if _, _, err := sb.SnapshotLoad(context.Background(), "snaps_tx", "s1"); err == nil {
		t.Error("snapshot must be gone after the tx committed")
	}

	// A rolled-back tx must not leave the snapshot behind.
	err = tx.RunInTx(context.Background(), func(ctx context.Context) error {
		if err := sb.SnapshotSave(ctx, "snaps_tx", "s2", 1, []byte("lost")); err != nil {
			return err
		}

		return errors.New("aborted on purpose")
	})
	if err == nil {
		t.Fatal("RunInTx must propagate the callback error")
	}

	if _, _, err := sb.SnapshotLoad(context.Background(), "snaps_tx", "s2"); err == nil {
		t.Error("rolled-back snapshot save must not persist")
	}
}
