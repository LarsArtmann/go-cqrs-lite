package sync

import (
	"encoding/json"
	"time"
)

// Conflict represents a synchronization conflict between local and remote versions.
// Type parameter T is the entity type being synced.
type Conflict[T any] struct {
	Local     T           `json:"local"`
	Remote    T           `json:"remote"`
	LocalVC   VectorClock `json:"localVc"`
	RemoteVC  VectorClock `json:"remoteVc"`
	Timestamp time.Time   `json:"timestamp"`
}

// ConflictResolver determines the winner when sync conflicts occur.
// Implementations can use different strategies: LWW, custom merge, CRDT, etc.
type ConflictResolver[T any] interface {
	// Resolve determines the winning value for a conflict.
	Resolve(conflict *Conflict[T]) (T, error)
}

// LWWResolver resolves conflicts using Last-Write-Wins strategy.
// It compares vector clocks first (causal ordering), then falls back to timestamps.
// If timestamps are equal, the tiebreaker function breaks the tie.
//
// Type parameter T is the entity type. The TimestampFunc extracts the timestamp
// used for LWW comparison from an entity.
type LWWResolver[T any] struct {
	// TimestampFunc extracts the comparison timestamp from an entity.
	TimestampFunc func(T) time.Time
	// Tiebreaker is called when timestamps are equal. Returns true if local wins.
	// If nil, remote wins on tie.
	Tiebreaker func(local, remote T) bool
}

// NewLWWResolver creates a Last-Write-Wins resolver with the given timestamp extractor.
// Returns ErrNilTimestampFunc if timestampFunc is nil.
func NewLWWResolver[T any](timestampFunc func(T) time.Time) (*LWWResolver[T], error) {
	if timestampFunc == nil {
		return nil, ErrNilTimestampFunc
	}

	return &LWWResolver[T]{
		TimestampFunc: timestampFunc,
		Tiebreaker:    nil,
	}, nil
}

// Resolve implements ConflictResolver using vector clock comparison with LWW fallback.
func (r *LWWResolver[T]) Resolve(conflict *Conflict[T]) (T, error) {
	comparison := conflict.LocalVC.Cmp(conflict.RemoteVC)

	switch comparison {
	case OrderBefore:
		return conflict.Remote, nil
	case OrderAfter:
		return conflict.Local, nil
	case OrderConcurrent, OrderEqual:
		localTime := r.TimestampFunc(conflict.Local)
		remoteTime := r.TimestampFunc(conflict.Remote)

		if localTime.After(remoteTime) {
			return conflict.Local, nil
		}

		if remoteTime.After(localTime) {
			return conflict.Remote, nil
		}

		if r.Tiebreaker != nil && r.Tiebreaker(conflict.Local, conflict.Remote) {
			return conflict.Local, nil
		}

		return conflict.Remote, nil
	default:
		return conflict.Remote, nil
	}
}

// MergeResult represents the outcome of a CRDT merge.
type MergeResult int

const (
	// MergeResultLocalWins means the local version was chosen.
	MergeResultLocalWins MergeResult = iota
	// MergeResultRemoteWins means the remote version was chosen.
	MergeResultRemoteWins
	// MergeResultMerged means a merged version was created.
	MergeResultMerged
	// MergeResultConflict means the conflict could not be auto-resolved.
	MergeResultConflict
)

func (r MergeResult) String() string {
	switch r {
	case MergeResultLocalWins:
		return "local_wins"
	case MergeResultRemoteWins:
		return "remote_wins"
	case MergeResultMerged:
		return "merged"
	case MergeResultConflict:
		return "conflict"
	default:
		return "unknown"
	}
}

// SyncMessage is the envelope for sync protocol messages.
type SyncMessage struct {
	Type    SyncMessageType `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

// SyncContext contains shared fields for sync request and response.
type SyncContext struct {
	NodeID NodeID      `json:"nodeId"`
	Clock  VectorClock `json:"clock"`
}

// NewSyncContext creates a new SyncContext with the given node ID and clock.
func NewSyncContext(nodeID NodeID, clock VectorClock) SyncContext {
	return SyncContext{
		NodeID: nodeID,
		Clock:  clock,
	}
}

// SyncRequest requests sync from a peer.
type SyncRequest struct {
	SyncContext

	Since time.Time `json:"since"`
}

// SyncResponse contains operations from a peer.
type SyncResponse[T any] struct {
	SyncContext

	Operations []*Operation[T] `json:"operations"`
}
