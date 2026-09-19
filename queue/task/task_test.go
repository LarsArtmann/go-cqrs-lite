package task_test

import (
	"sync"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/queue/v4/task"
)

// TestNewID_MonotonicOrder pins the ID contract the claim tie-break relies
// on: IDs minted later sort >= IDs minted earlier (`id ASC` = oldest first
// within a priority). The original crypto-random suffix made same-ms IDs a
// random permutation, so the conformance suite's status_counts pin claimed
// the wrong task ~20% of runs.
func TestNewID_MonotonicOrder(t *testing.T) {
	prev := task.ID("")

	for range 1000 {
		id := task.NewID()
		if prev != "" && id < prev {
			t.Fatalf("ID order broke: %s minted after %s", id, prev)
		}

		prev = id
	}
}

// TestNewID_ConcurrentUnique pins uniqueness under concurrent minting, and
// that every ID keeps the 36-char ms+seed+seq shape.
func TestNewID_ConcurrentUnique(t *testing.T) {
	const workers = 8

	const perWorker = 500

	ids := make([][]task.ID, workers)

	var wg sync.WaitGroup

	for w := range workers {
		wg.Add(1)

		go func() {
			defer wg.Done()

			minted := make([]task.ID, perWorker)
			for i := range perWorker {
				minted[i] = task.NewID()
			}

			ids[w] = minted
		}()
	}

	wg.Wait()

	seen := make(map[task.ID]struct{}, workers*perWorker)
	for _, batch := range ids {
		for _, id := range batch {
			if len(id) != 36 {
				t.Fatalf("malformed ID %q", id)
			}

			if _, dup := seen[id]; dup {
				t.Fatalf("duplicate ID %q", id)
			}

			seen[id] = struct{}{}
		}
	}
}

// TestNewID_WithinSameMillisecondStrictlyOrdered pins that a burst of mints
// inside one millisecond (the conformance enqueue shape) is strictly
// increasing — the exact property the random suffix violated.
func TestNewID_WithinSameMillisecondStrictlyOrdered(t *testing.T) {
	first := task.NewID()

	burst := make([]task.ID, 100)
	for i := range burst {
		burst[i] = task.NewID()
	}

	for i, id := range burst {
		if id <= first {
			t.Fatalf("burst[%d]=%s not after %s — same-ms tie-break would be random", i, id, first)
		}
	}
}
