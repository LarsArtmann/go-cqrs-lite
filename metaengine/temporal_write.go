package metaengine

import (
	"context"
	"reflect"
	"time"

	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

// VersionedWriter is the optional engine capability for BigTable-style
// timestamped cell writes: (collection, key, timestamp) → value. It is the
// write-side counterpart of [VersionedStorage] — engines that version cells
// natively implement both, and the Store's fold path prefers it so cell
// timestamps carry EVENT time (via [CellTimestamp]), not write time.
//
// Semantics (ADR-0141):
//   - A nil/empty value is never written; deletion is [MapDeleteAt] (a
//     timestamped tombstone), never a hard erase.
//   - Writing the same (collection, key, timestamp) twice is last-writer-wins
//     (mirrors BigTable cell dedup).
//   - Out-of-order timestamps are legal; reads resolve by timestamp.
type VersionedWriter interface {
	// MapSetAt writes value as the version of (collection, key) at timestamp
	// ts, preserving earlier versions for as-of reads.
	MapSetAt(ctx context.Context, collection, key string, value any, ts time.Time) error

	// MapDeleteAt records a tombstone for (collection, key) at timestamp ts:
	// as-of reads at t >= ts report the key as absent.
	MapDeleteAt(ctx context.Context, collection, key string, ts time.Time) error
}

// CellVersion is one surviving version of a cell, as returned by
// [CellHistoryReader.MapHistory]. A nil Value marks a tombstone (the key was
// deleted at Timestamp).
type CellVersion struct {
	Timestamp time.Time
	Value     any
}

// CellHistoryReader is the optional engine capability for BigTable-style
// range reads: the version history of one cell within [from, to]. Engines
// that version cells implement it alongside [VersionedStorage]; entries are
// returned newest-first, tombstones included. Retention-pruned versions are
// simply absent.
type CellHistoryReader interface {
	MapHistory(
		ctx context.Context,
		collection, key string,
		from, to time.Time,
	) ([]CellVersion, error)
}

// RetentionPolicy is the retention knob for versioned cells — the analog of
// BigTable column-family GC rules. The zero value keeps everything.
//
// MaxVersions keeps at most the newest N versions per cell (0 = unlimited,
// 1 = latest-only: the versioned engine collapses to a plain mutable KV).
// MaxAge drops versions older than the cutoff relative to each write's
// timestamp. Neither ever prunes the newest version of a live cell.
type RetentionPolicy struct {
	MaxVersions int
	MaxAge      time.Duration
}

// CellTimestamp derives a cell timestamp from a Record's stamps, in trust
// order: the database's Stored acknowledgment, then the server's Received
// time, then the client's Created time, falling back to wall-clock now.
//
// Replay correctness (ADR-0141 §3): when a projection host replays
// historical events, their original stamps are preserved, so versioned cells
// reflect the true temporal order rather than replay order.
func CellTimestamp(rec record.Record) time.Time {
	for _, stamp := range []record.Stamp{rec.MetaData.Stored, rec.MetaData.Received, rec.MetaData.Created} {
		if !stamp.IsZero() {
			return stamp.Time()
		}
	}

	return time.Now()
}

// asOfField is the query-input meta field declaring temporal intent. A
// non-zero AsOf on a point-lookup input routes the read through
// [VersionedStorage]; a zero value means "latest" (the normal path).
const asOfField = "AsOf"

// detectAsOfInput reports whether the input struct declares an
// `AsOf time.Time` field.
func detectAsOfInput(input any) bool {
	t := derefType(input)
	if t == nil || t.Kind() != reflect.Struct {
		return false
	}

	timeType := reflect.TypeFor[time.Time]()

	for field := range t.Fields() {
		if field.Name == asOfField && field.Type == timeType {
			return true
		}
	}

	return false
}

// extractAsOfFromInput returns the input's AsOf value when the field exists
// and carries a non-zero time (zero means "latest").
func extractAsOfFromInput(input any) (time.Time, bool) {
	v, ok := structValue(input)
	if !ok {
		return time.Time{}, false
	}

	f := v.FieldByName(asOfField)
	if !f.IsValid() || f.Type() != reflect.TypeFor[time.Time]() {
		return time.Time{}, false
	}

	ts := f.Interface().(time.Time)
	if ts.IsZero() {
		return time.Time{}, false
	}

	return ts, true
}
