package snapshot

import (
	"context"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
)

type Snapshot struct {
	AggregateID   id.AggregateID   `json:"aggregateId"`
	AggregateType id.AggregateType `json:"aggregateType"`
	Version       event.Version    `json:"version"`
	State         []byte           `json:"state"`
	CreatedAt     time.Time        `json:"createdAt"`
}

type SnapshotSink interface {
	Save(ctx context.Context, snapshot Snapshot) error

	Delete(ctx context.Context, ref id.AggregateRef) error
}

type SnapshotSource interface {
	Load(
		ctx context.Context,
		ref id.AggregateRef,
	) (*Snapshot, error)

	LoadAtVersion(
		ctx context.Context,
		ref id.AggregateRef,
		version event.Version,
	) (*Snapshot, error)
}

type SnapshotStore interface {
	SnapshotSink
	SnapshotSource
}
