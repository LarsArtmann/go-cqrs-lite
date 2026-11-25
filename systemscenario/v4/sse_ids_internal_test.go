package systemscenario

import (
	"testing"

	"github.com/larsartmann/go-sse/sseparse"
)

// TestSSEStream_ReceivedIDs_Internal pins ReceivedIDs against the raw events
// the read loop records: arrival order, an empty id kept as an empty string,
// and a slice the caller cannot mutate the stream through.
func TestSSEStream_ReceivedIDs_Internal(t *testing.T) {
	t.Parallel()

	s := &SSEStream[struct{ ID string }]{}

	if ids := s.ReceivedIDs(); len(ids) != 0 {
		t.Fatalf("ReceivedIDs before any event = %v, want empty", ids)
	}

	s.record(sseparse.Event{ID: "7", DataLines: []string{`{"ID":"a"}`}})
	s.record(sseparse.Event{DataLines: []string{`{"ID":"b"}`}})
	s.record(sseparse.Event{ID: "9", DataLines: []string{`{"ID":"c"}`}})

	ids := s.ReceivedIDs()

	if len(ids) != 3 || ids[0] != "7" || ids[1] != "" || ids[2] != "9" {
		t.Fatalf("ReceivedIDs = %v, want [7 <empty> 9] in arrival order", ids)
	}

	ids[0] = "mutated"

	if again := s.ReceivedIDs(); again[0] != "7" {
		t.Fatalf("ReceivedIDs aliases stream state: mutating the result changed it to %v", again)
	}
}
