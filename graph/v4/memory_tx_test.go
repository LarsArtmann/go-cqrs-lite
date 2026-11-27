package graph

import (
	"testing"
	"time"
)

// Two transactions that overlap in time must both survive: the later commit
// may not swap in a snapshot taken before the other one's writes.
func TestMemoryDriverConcurrentTransactionsKeepAllWrites(t *testing.T) {
	driver := NewMemoryDriver()

	// Closed once the first transaction has written and is still open.
	firstWritten := make(chan struct{})
	secondDone := make(chan struct{})

	first := make(chan error, 1)
	go func() {
		first <- driver.RunInTx(func(sink GraphSink) error {
			if err := sink.MergeNode(NodeRef{Label: "User", KeyProp: "id", KeyValue: "first"}, nil); err != nil {
				return err
			}

			close(firstWritten)

			// Let the second transaction commit first. With serialised
			// writers it can only run once this transaction is over.
			select {
			case <-secondDone:
			case <-time.After(500 * time.Millisecond):
			}

			return nil
		})
	}()

	second := make(chan error, 1)
	go func() {
		<-firstWritten

		second <- driver.RunInTx(func(sink GraphSink) error {
			return sink.MergeNode(NodeRef{Label: "User", KeyProp: "id", KeyValue: "second"}, nil)
		})
		close(secondDone)
	}()

	if err := <-first; err != nil {
		t.Fatalf("first transaction: %v", err)
	}

	if err := <-second; err != nil {
		t.Fatalf("second transaction: %v", err)
	}

	nodes := driver.Query(Pattern{})
	if len(nodes) != 2 {
		t.Fatalf("both transactions reported success, but the graph holds %d node(s)", len(nodes))
	}
}
