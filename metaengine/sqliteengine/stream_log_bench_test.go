package sqliteengine

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"testing"

	_ "modernc.org/sqlite"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// Paged journal-drain benchmarks (docs/BENCHMARKS.md, design
// docs/planning/SEQ-CARRYING-JOURNAL-READS.md §5.5): a catch-up subscriber
// draining a 100k-entry journal in pages of 500, position-based (OFFSET) vs
// token-based (index seek). The shared AUTOINCREMENT counter interleaves a
// noise collection so seq gaps are real, as in production.
const (
	drainEntries  = 100_000
	drainPageSize = 500
)

type drainFixture struct {
	engine metaengine.Engine
	log    metaengine.StreamLogBackend
	seqLog metaengine.SeqSeekableStreamLog
}

var (
	drainOnce sync.Once
	drainFix  drainFixture
	drainErr  error
)

func journalDrainFixture(b *testing.B) *drainFixture {
	drainOnce.Do(func() {
		db, err := sql.Open("sqlite", ":memory:")
		if err != nil {
			drainErr = fmt.Errorf("open: %w", err)
			return
		}

		db.SetMaxOpenConns(1)

		eng, err := NewSQLiteEngine(db)
		if err != nil {
			drainErr = fmt.Errorf("NewSQLiteEngine: %w", err)
			return
		}

		log, ok := eng.(metaengine.StreamLogBackend)
		if !ok {
			drainErr = fmt.Errorf("engine does not implement StreamLogBackend")
			return
		}

		seqLog, ok := eng.(metaengine.SeqSeekableStreamLog)
		if !ok {
			drainErr = fmt.Errorf("engine does not implement SeqSeekableStreamLog")
			return
		}

		drainFix = drainFixture{engine: eng, log: log, seqLog: seqLog}

		if err := seedDrainJournal(log); err != nil {
			drainErr = err
		}
	})

	if drainErr != nil {
		b.Fatal(drainErr)
	}

	return &drainFix
}

func appendJournalBatch(
	log metaengine.StreamLogBackend,
	col, sid, prefix string,
	start, n int,
) error {
	vals := make([]any, n)
	for j := range vals {
		vals[j] = fmt.Sprintf("%s-%06d", prefix, start+j)
	}

	if err := log.StreamAppend(context.Background(), col, sid, vals); err != nil {
		return fmt.Errorf("append %s: %w", col, err)
	}

	return nil
}

func seedDrainJournal(log metaengine.StreamLogBackend) error {
	const noiseEvery = drainEntries / 10

	for written, batch := 0, 0; written < drainEntries; written += drainPageSize {
		if err := appendJournalBatch(log, "events",
			fmt.Sprintf("stream-%04d", batch), "entry", written, drainPageSize); err != nil {
			return err
		}

		batch++

		if written%noiseEvery == 0 {
			if err := appendJournalBatch(log, "noise",
				"noise-stream", "noise", written, drainPageSize/5); err != nil {
				return err
			}
		}
	}

	return nil
}

// BenchmarkJournalPagedDrain_Position is the PRE-fix path: every page re-skips
// all previously read rows via OFFSET (O(N²/P) row visits for a full drain).
func BenchmarkJournalPagedDrain_Position(b *testing.B) {
	f := journalDrainFixture(b)
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		pos, total := int64(0), 0

		for {
			page, err := f.log.JournalReadFrom(ctx, "events", pos, drainPageSize)
			if err != nil {
				b.Fatal(err)
			}

			if len(page) == 0 {
				break
			}

			total += len(page)
			pos += int64(len(page))
		}

		if total != drainEntries {
			b.Fatalf("drained %d entries, want %d", total, drainEntries)
		}
	}
}

// BenchmarkJournalPagedDrain_Token is the seq-token path: every page is a pure
// `WHERE collection = ? AND seq > ?` index seek (O(N + (N/P)·log N)).
func BenchmarkJournalPagedDrain_Token(b *testing.B) {
	f := journalDrainFixture(b)
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		cursor, total := int64(0), 0

		for {
			page, err := f.seqLog.JournalReadFromSeq(ctx, "events", cursor, drainPageSize)
			if err != nil {
				b.Fatal(err)
			}

			if len(page) == 0 {
				break
			}

			total += len(page)
			cursor = page[len(page)-1].Seq
		}

		if total != drainEntries {
			b.Fatalf("drained %d entries, want %d", total, drainEntries)
		}
	}
}
