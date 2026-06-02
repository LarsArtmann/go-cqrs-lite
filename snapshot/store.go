package snapshot

import (
	"context"
	"io"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
)

var (
	ErrSnapshotNotFound    = event.NewRejection("snapshot.not_found", "snapshot not found")
	ErrSnapshotStoreClosed = event.NewInfrastructure(
		"snapshot.store_closed",
		"snapshot store is closed",
	)
)

type Snapshot struct {
	AggregateID   id.AggregateID
	AggregateType event.AggregateType
	Version       event.Version
	State         []byte
	CreatedAt     time.Time
}

type SnapshotSink interface {
	io.Closer

	Save(ctx context.Context, snapshot Snapshot) error

	Delete(ctx context.Context, ref event.AggregateRef) error
}

type SnapshotSource interface {
	io.Closer

	Load(
		ctx context.Context,
		ref event.AggregateRef,
	) (*Snapshot, error)

	LoadAtVersion(
		ctx context.Context,
		ref event.AggregateRef,
		version event.Version,
	) (*Snapshot, error)
}

type SnapshotStore interface {
	SnapshotSink
	SnapshotSource
}
