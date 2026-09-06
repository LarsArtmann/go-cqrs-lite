package snapshot

import (
	"context"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

// Snapshot is a materialized aggregate state captured at a stream version.
// Construct one with [NewSnapshot] (which enforces the invariants below) or
// check an externally-built value with [Validate].
//
// Invariants:
//
//   - Stream identity is non-zero (StreamType non-empty, StreamID non-zero).
//   - Version >= 1: version 0 means "no events applied", which has nothing
//     worth snapshotting.
//   - State is non-nil and non-empty.
//   - Encoding declares the codec the State bytes were written with.
//     [record.EncodingUnknown] (the zero value) marks legacy snapshots
//     written before the stamp existed; readers fall back to envelope or
//     configured-codec detection for those.
//
// The JSON tags carry the honest v5 stream vocabulary. Writers emit only
// stream_id/stream_type; readers additionally accept the pre-v5
// aggregateId/aggregateType spellings via the decode-only fallback in
// wire.go (deleted at v6). See docs/planning/v5-deprecation-sweep.md.
type Snapshot struct {
	StreamID   id.StreamID     `json:"stream_id"`
	StreamType id.StreamType   `json:"stream_type"`
	Version    event.Version   `json:"version"`
	State      []byte          `json:"state"`
	Encoding   record.Encoding `json:"encoding,omitempty"`
	CreatedAt  time.Time       `json:"createdAt"`
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
