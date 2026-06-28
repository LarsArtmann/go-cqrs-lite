package main

import (
	"context"
	"encoding/json"
	"testing"

	cqrsevent "github.com/larsartmann/go-cqrs-lite/event/v3"
	cqrsgraph "github.com/larsartmann/go-cqrs-lite/graph/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
)

func seedGraph(t *testing.T) *cqrsgraph.MemoryDriver {
	t.Helper()

	driver := cqrsgraph.NewMemoryDriver()

	proj, err := cqrsgraph.NewGraphProjection(
		"forum-graph",
		driver,
		projectEvent,
		[]cqrsevent.Type{"MESSAGE_POSTED"},
		cqrsgraph.WithSchema(forumSchema()),
	)
	if err != nil {
		t.Fatalf("create projection: %v", err)
	}

	events := []messagePosted{
		{ID: "m1", AuthorID: "alice", Content: "Hello world!"},
		{ID: "m2", AuthorID: "bob", Content: "Hi Alice!", ReplyTo: "m1"},
		{ID: "m3", AuthorID: "carol", Content: "Welcome!", ReplyTo: "m1"},
		{ID: "m4", AuthorID: "bob", Content: "Thanks!", ReplyTo: "m2"},
	}

	ctx := context.Background()

	for _, msg := range events {
		payload, _ := json.Marshal(msg)
		aggID, _ := id.ParseAggregateID("forum")
		evt, _ := cqrsevent.NewEvent(
			"MESSAGE_POSTED",
			aggID,
			"Forum",
			cqrsevent.Version(1),
			payload,
		)

		if err := proj.Handle(ctx, evt); err != nil {
			t.Fatalf("project event: %v", err)
		}
	}

	return driver
}

func TestQuery_ReturnsAllMessages(t *testing.T) {
	t.Parallel()

	driver := seedGraph(t)

	messages := driver.Query(cqrsgraph.Pattern{Label: "Message"})
	if len(messages) != 4 {
		t.Fatalf("expected 4 messages, got %d", len(messages))
	}
}

func TestTraverse_ReplyChain(t *testing.T) {
	t.Parallel()

	driver := seedGraph(t)

	// m4 REPLY_TO m2 REPLY_TO m1 — traversal should find both ancestors.
	ancestors := driver.Traverse(
		cqrsgraph.NodeRef{Label: "Message", KeyProp: "id", KeyValue: "m4"},
		"REPLY_TO", -1,
	)
	if len(ancestors) != 2 {
		t.Fatalf("expected 2 ancestors (m2, m1), got %d", len(ancestors))
	}
}

func TestShortestPath_ReplyChain(t *testing.T) {
	t.Parallel()

	driver := seedGraph(t)

	path, err := driver.ShortestPath(
		cqrsgraph.NodeRef{Label: "Message", KeyProp: "id", KeyValue: "m4"},
		cqrsgraph.NodeRef{Label: "Message", KeyProp: "id", KeyValue: "m1"},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(path) != 3 {
		t.Fatalf("expected 3-node path (m4→m2→m1), got %d: %v", len(path), path)
	}
}

func TestSchema_RejectsInvalidLabel(t *testing.T) {
	t.Parallel()

	driver := cqrsgraph.NewMemoryDriver(cqrsgraph.WithDriverSchema(forumSchema()))

	err := driver.RunInTx(func(sink cqrsgraph.GraphSink) error {
		return sink.MergeNode(
			cqrsgraph.NodeRef{Label: "Bogus", KeyProp: "id", KeyValue: "x"},
			nil,
		)
	})
	if err == nil {
		t.Fatal("expected error for unknown label, got nil")
	}
}

func TestSchema_RejectsUnknownProperty(t *testing.T) {
	t.Parallel()

	driver := cqrsgraph.NewMemoryDriver(cqrsgraph.WithDriverSchema(forumSchema()))

	err := driver.RunInTx(func(sink cqrsgraph.GraphSink) error {
		return sink.MergeNode(
			cqrsgraph.NodeRef{Label: "User", KeyProp: "id", KeyValue: "x"},
			map[string]any{"bogus_prop": "value"},
		)
	})
	if err == nil {
		t.Fatal("expected error for unknown property, got nil")
	}
}

func TestQuery_FilterByContent(t *testing.T) {
	t.Parallel()

	driver := seedGraph(t)

	result := driver.Query(cqrsgraph.Pattern{
		Label: "Message",
		Where: func(props map[string]any) bool {
			content, _ := props["content"].(string)

			return content == "Hello world!"
		},
	})
	if len(result) != 1 {
		t.Fatalf("expected 1 message, got %d", len(result))
	}

	if result[0].Ref.KeyValue != "m1" {
		t.Fatalf("expected m1, got %s", result[0].Ref.KeyValue)
	}
}
