package keycodec

import (
	"bytes"
	"sync"
	"sync/atomic"
	"testing"
)

// TestSplitGroupAndSeq_RoundTrip pins the 20-digit+NUL seq tail layout: keys
// built by the Seq* constructors split back into exactly the group and seq
// they were built from. wholeGroup distinguishes stream keys (group is
// "col\x00sid") from collection-scoped keys (group is the first segment).
func TestSplitGroupAndSeq_RoundTrip(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		key        []byte
		prefixLen  int
		wholeGroup bool
		wantGroup  string
		wantSeq    int64
	}{
		{
			name:       "stream key group is col+sid",
			key:        StreamKey("tasks", "01JTEST", 7),
			prefixLen:  len("sl\x00"),
			wholeGroup: true,
			wantGroup:  "tasks\x0001JTEST",
			wantSeq:    7,
		},
		{
			name:       "stream key wholeGroup=false keeps first segment",
			key:        StreamKey("tasks", "01JTEST", 7),
			prefixLen:  len("sl\x00"),
			wholeGroup: false,
			wantGroup:  "tasks",
			wantSeq:    7,
		},
		{
			name:       "journal key",
			key:        JournalKey("events", 42),
			prefixLen:  len("jl\x00"),
			wholeGroup: true,
			wantGroup:  "events",
			wantSeq:    42,
		},
		{
			name:       "log key",
			key:        LogKey("events", 9223372036854775806),
			prefixLen:  len("l\x00"),
			wholeGroup: true,
			wantGroup:  "events",
			wantSeq:    9223372036854775806,
		},
		{
			name:       "multimap key first-segment group",
			key:        MultimapKey("col", "mykey", 3),
			prefixLen:  len("mm\x00"),
			wholeGroup: false,
			wantGroup:  "col",
			wantSeq:    3,
		},
	}

	for _, tc := range tests {
		group, seq, ok := SplitGroupAndSeq(tc.key, tc.prefixLen, tc.wholeGroup)
		if !ok {
			t.Errorf("%s: SplitGroupAndSeq(%q) not ok", tc.name, tc.key)

			continue
		}

		if group != tc.wantGroup || seq != tc.wantSeq {
			t.Errorf(
				"%s: got (%q, %d), want (%q, %d)",
				tc.name,
				group,
				seq,
				tc.wantGroup,
				tc.wantSeq,
			)
		}
	}
}

// TestSplitGroupAndSeq_Rejects pins the malformed-key rejection paths.
func TestSplitGroupAndSeq_Rejects(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		key       []byte
		prefixLen int
	}{
		{"too short", []byte("sl\x00col"), len("sl\x00")},
		{"no NUL before seq", []byte("l\x00col" + "12345678901234567890"), len("l\x00")},
		{"non-numeric seq", []byte("l\x00col\x00aaaaaaaaaaaaaaaaaaaa"), len("l\x00")},
		{"empty key", nil, 2},
	}

	for _, tc := range tests {
		if group, seq, ok := SplitGroupAndSeq(tc.key, tc.prefixLen, true); ok {
			t.Errorf("%s: accepted malformed key %q as (%q, %d)", tc.name, tc.key, group, seq)
		}
	}
}

// TestSplitGroupAndSeq_TailLayout pins the wire layout directly: the seq is
// exactly 20 zero-padded decimal bytes preceded by one NUL.
func TestSplitGroupAndSeq_TailLayout(t *testing.T) {
	t.Parallel()

	key := LogKey("c", 42)

	if !bytes.HasSuffix(key, []byte("\x0000000000000000000042")) {
		t.Fatalf("key must end in NUL + 20 zero-padded digits, got %q", key)
	}

	if len(key[len("l\x00"):]) != len("c")+SeqTailLen {
		t.Fatalf("group+tail must be len(group)+%d, got %d", SeqTailLen, len(key))
	}
}

// TestSeedSeqMax pins the max-only CAS contract: seeding never lowers a
// counter, and unknown groups start at the seeded value.
func TestSeedSeqMax(t *testing.T) {
	t.Parallel()

	var m sync.Map

	SeedSeqMax(&m, "g", 5)

	v, _ := m.Load("g")
	if got := v.(*atomic.Int64).Load(); got != 5 {
		t.Fatalf("fresh group seeded to %d, want 5", got)
	}

	SeedSeqMax(&m, "g", 2)
	v, _ = m.Load("g")
	if got := v.(*atomic.Int64).Load(); got != 5 {
		t.Fatalf("lower seed must not lower counter, got %d want 5", got)
	}

	SeedSeqMax(&m, "g", 9)
	v, _ = m.Load("g")
	if got := v.(*atomic.Int64).Load(); got != 9 {
		t.Fatalf("higher seed must raise counter, got %d want 9", got)
	}
}
