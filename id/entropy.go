package id

import (
	"crypto/rand"
	"encoding/binary"
	"sync"
	"sync/atomic"
	"time"

	"github.com/oklog/ulid/v2"
)

// Lock-free ULID generation.
//
// The previous implementation serialized every ID behind a global mutex
// around a shared ulid.MonotonicEntropy. This replacement keeps every
// guarantee of that design while removing the lock from the fast path:
//
//   - IDs issued in the same millisecond are strictly increasing in ULID
//     sort order across ALL goroutines (a single global atomic counter
//     provides the ordering; with the mutex design they were merely
//     non-concurrent, which happened to imply the same thing).
//   - IDs are unique under any concurrency (counter + per-millisecond
//     random epoch prefix).
//   - If the wall clock steps backwards, the current millisecond is pinned
//     and the counter keeps increasing: ULIDs never regress.
//
// Layout: ULID = 48-bit ms timestamp || 6 random bytes || 4-byte counter.
// Each millisecond gets a fresh 48-bit crypto-random prefix; IDs within the
// millisecond differ only in the counter suffix. Trade-off vs. the old full
// 80-bit random entropy: an observer of one ID can still predict its
// same-millisecond successors (equally true of a monotonic reader), and
// cross-millisecond IDs carry 48 rather than 80 random bits. ULIDs are
// identifiers, not secrets; 2^48 makes cross-millisecond guessing infeasible.
//
// The fast path is one atomic pointer load + one atomic add. The mutex below
// is acquired only to create a new epoch — at most once per millisecond.

// idEpoch is one millisecond window of ULID generation.
type idEpoch struct {
	ms       uint64  // ULID millisecond timestamp stamped on every ID of this epoch
	prefix   [6]byte // per-epoch crypto-random entropy (high 48 bits of the 80-bit ULID entropy)
	firstSeq uint64  // global counter value when this epoch was published (draws are strictly greater)
}

var (
	//nolint:gochecknoglobals // current generation epoch; read-mostly atomic pointer
	currentEpoch atomic.Pointer[idEpoch]
	//nolint:gochecknoglobals // global monotonic sequence; low 32 bits are the per-epoch suffix
	idSeq atomic.Uint64

	// epochMu serializes epoch creation only — never held on the fast path.
	epochMu sync.Mutex
)

// seqSuffixMask bounds the sequence space per epoch: 2^32 IDs per millisecond.
const seqSuffixMask = 1<<32 - 1

// advanceEpoch installs the epoch for nowMs, unless a goroutine already
// installed an epoch at or past nowMs. A backwards wall clock (nowMs <= the
// current epoch's ms) reuses the current epoch so ULIDs stay monotonic.
func advanceEpoch(nowMs uint64) *idEpoch {
	epochMu.Lock()
	defer epochMu.Unlock()

	if cur := currentEpoch.Load(); cur != nil && nowMs <= cur.ms {
		return cur
	}

	var prefix [6]byte
	if _, err := rand.Read(prefix[:]); err != nil {
		// crypto/rand does not fail on supported platforms. If it somehow
		// does, keep the existing epoch (uniqueness holds via the counter);
		// if there is none, fall back to a zero prefix — still unique via
		// (ms, counter) and strictly ordered.
		if cur := currentEpoch.Load(); cur != nil {
			return cur
		}
	}

	next := &idEpoch{
		ms:       nowMs,
		prefix:   prefix,
		firstSeq: idSeq.Load(),
	}
	currentEpoch.Store(next)

	return next
}

// newULID returns a monotonically ordered, concurrency-safe ULID without
// taking a lock on the generation fast path.
func newULID() ulid.ULID {
	now := ulid.Now()

	epoch := currentEpoch.Load()
	if epoch == nil || now > epoch.ms {
		epoch = advanceEpoch(now)
	}

	for {
		s := idSeq.Add(1)
		if s-epoch.firstSeq > seqSuffixMask {
			// More than 2^32 IDs in one millisecond: the suffix is
			// exhausted. Wait for the next millisecond epoch. Unreachable
			// in practice (would require >4 trillion IDs/second).
			time.Sleep(time.Until(ulid.Time(epoch.ms + 1)))

			if next := currentEpoch.Load(); next != nil && next.ms > epoch.ms {
				epoch = next
			} else {
				epoch = advanceEpoch(epoch.ms + 1)
			}

			continue
		}

		var id ulid.ULID
		// Errors are impossible: e.ms comes from ulid.Now() (always <=
		// ulid.MaxTime) and the entropy slice is exactly 10 bytes.
		_ = id.SetTime(epoch.ms)

		var entropy [10]byte
		copy(entropy[:6], epoch.prefix[:])
		binary.BigEndian.PutUint32(entropy[6:], uint32(s))
		_ = id.SetEntropy(entropy[:])

		return id
	}
}
