package keycodec

import "testing"

func TestJournalSeq(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		key  []byte
		want int64
		ok   bool
	}{
		{"seq 1", JournalKey("events", 1), 1, true},
		{"seq 42", JournalKey("events", 42), 42, true},
		{"max int64-ish", JournalKey("c", 9223372036854775806), 9223372036854775806, true},
		{"not a journal key", []byte("jl\x00events"), 0, false},
		{"non-numeric tail", []byte("jl\x00events\x00aaaaaaaaaaaaaaaaaaaa"), 0, false},
		{"too short", []byte("12345"), 0, false},
	}

	for _, tc := range tests {
		if got, ok := JournalSeq(tc.key); ok != tc.ok || got != tc.want {
			t.Errorf("%s: JournalSeq(%q) = %d, %v; want %d, %v",
				tc.name, tc.key, got, ok, tc.want, tc.ok)
		}
	}
}
