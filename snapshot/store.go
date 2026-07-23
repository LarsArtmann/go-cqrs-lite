package snapshot

import (
	"context"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

type Snapshot struct {
	StreamID   id.StreamID   `json:"aggregateId"`
	StreamType id.StreamType `json:"streamType"`
	Version       event.Version    `json:"version"`
	State         []byte           `json:"state"`
	CreatedAt     time.Time        `json:"createdAt"`
}

type SnapshotSink interface {
	Save(ctx context.Context, snapshot Snapshot) error

	Delete(ctx context.Context, ref id.StreamRef) error
}

type SnapshotSource interface {
	Load(
		ctx context.Context,
		ref id.StreamRef,
	) (*Snapshot, error)

	LoadAtVersion(
		ctx context.Context,
		ref id.StreamRef,
		version event.Version,
	) (*Snapshot, error)
}

type SnapshotStore interface {
	SnapshotSink
	SnapshotSource
}
