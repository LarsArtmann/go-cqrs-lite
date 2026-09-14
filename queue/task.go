package queue

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

// ID identifies a task. Opaque, unique, roughly time-sortable.
type ID string

// NewID returns a new unique task ID: a millisecond timestamp prefix plus
// a crypto-random suffix. The timestamp prefix makes IDs sort by creation
// time, which the claim order (oldest first within a priority) relies on
// for stable tie-breaking.
func NewID() ID {
	var b [10]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(fmt.Sprintf("queue: crypto/rand failed: %v", err))
	}

	return ID(fmt.Sprintf("%016x", time.Now().UnixMilli()) + hex.EncodeToString(b[:]))
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
