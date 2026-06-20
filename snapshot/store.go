package snapshot

import (
	"context"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
)

type Snapshot struct {
	AggregateID   id.AggregateID      `json:"aggregateId"`
	AggregateType event.AggregateType `json:"aggregateType"`
	Version       event.Version       `json:"version"`
	State         []byte              `json:"state"`
	CreatedAt     time.Time           `json:"createdAt"`
}

type SnapshotSink interface {
	Save(ctx context.Context, snapshot Snapshot) error

	Delete(ctx context.Context, ref event.AggregateRef) error
}

type SnapshotSource interface {
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
