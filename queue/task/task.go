// Package task defines the core Task record and its lifecycle, exactly as
// in the donor (go-taskqueue internal/task): the Status state machine, ID
// minting, and the enqueue template with its defaults.
package task

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

// ID identifies a task. Opaque, unique, roughly time-sortable.
type ID string

// idMintState makes NewID monotonic within a millisecond. The claim order
// (oldest first within a priority) breaks created_at ties with `id ASC`,
// so same-millisecond IDs must sort by mint order — a purely random
// suffix made that tie-break a random permutation (observed as the
// status_counts conformance flake: the wrong task got claimed ~20% of
// runs). seed is re-rolled every millisecond so cross-process uniqueness
// survives; seq orders same-millisecond mints inside this process.
var idMintState = struct {
	sync.Mutex
	lastMS int64
	seq    uint64
	seed   [6]byte
}{}

// NewID returns a new unique task ID: a millisecond timestamp prefix plus
// a per-millisecond random seed and a monotonic sequence. The timestamp
// prefix and monotonic sequence make IDs sort by creation time, which the
// claim order (oldest first within a priority) relies on for stable
// tie-breaking.
func NewID() ID {
	idMintState.Lock()

	now := time.Now().UnixMilli()
	if now > idMintState.lastMS {
		idMintState.lastMS = now
		idMintState.seq = 0

		if _, err := rand.Read(idMintState.seed[:]); err != nil {
			idMintState.Unlock()

			panic(fmt.Sprintf("queue: crypto/rand failed: %v", err))
		}
	}

	seq := idMintState.seq
	idMintState.seq++
	ms := idMintState.lastMS
	seed := idMintState.seed

	idMintState.Unlock()

	return ID(fmt.Sprintf("%016x%s%08x", ms, hex.EncodeToString(seed[:]), seq))
}

// String returns the raw ID.
func (id ID) String() string { return string(id) }

// Task is the unit of work: what to run, for which project, under which
// constraints. Payload is the consumer's domain type, serialized by the
// engine's [Codec].
type Task[T any] struct {
	ID           ID         `json:"id"`
	Project      string     `json:"project,omitempty"`
	Type         string     `json:"type"`
	Payload      T          `json:"payload,omitempty"`
	Deps         []ID       `json:"deps,omitempty"`
	Priority     int        `json:"priority,omitempty"`
	Attempts     int        `json:"attempts"`
	MaxAttempts  int        `json:"maxAttempts"`
	NotBefore    time.Time  `json:"notBefore"`
	Status       Status     `json:"status"`
	LeaseOwner   string     `json:"leaseOwner,omitempty"`
	LeaseExpires *time.Time `json:"leaseExpires,omitempty"`
	LastError    string     `json:"lastError,omitempty"`
	CreatedAt    time.Time  `json:"createdAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`
	CompletedAt  *time.Time `json:"completedAt,omitempty"`
}

// New is a task template for enqueueing. ID, Attempts, Status and
// timestamps are assigned by the store; everything else is
// caller-supplied.
type New[T any] struct {
	Project     string
	Type        string
	Payload     T
	Deps        []ID
	Priority    int
	MaxAttempts int
	NotBefore   time.Time
	// DedupKey, when set, makes Enqueue idempotent: if a task with the
	// same key already exists, that task is returned unchanged and no
	// duplicate is created. Use a stable derivation (e.g. hash of project
	// + source + title) so repeated producers converge instead of
	// re-enqueueing.
	DedupKey string
}

// DefaultMaxAttempts is used when New.MaxAttempts is zero.
const DefaultMaxAttempts = 3

// Normalize applies defaults to a template: the attempt budget when unset,
// which every engine applies identically at Enqueue.
func (n New[T]) Normalize() New[T] {
	if n.MaxAttempts <= 0 {
		n.MaxAttempts = DefaultMaxAttempts
	}

	return n
}
