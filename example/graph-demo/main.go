// Package main demonstrates the graph projection tier with Schema validation
// and the MemoryDriver read API.
//
// It models a simple discussion forum: users author messages, messages reply
// to other messages. The graph shape captures reply chains (variable-depth
// traversal) and authorship — the read patterns the relational tier handles
// poorly (recursive CTEs) and the KV tier cannot express.
//
// Run: go run .
// Test: go test .
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	cqrsevent "github.com/larsartmann/go-cqrs-lite/event/v3"
	cqrsgraph "github.com/larsartmann/go-cqrs-lite/graph/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
)

type messagePosted struct {
	ID       string `json:"id"`
	AuthorID string `json:"author_id"`
	Content  string `json:"content"`
	ReplyTo  string `json:"reply_to,omitempty"`
}

func forumSchema() *cqrsgraph.Schema {
	return &cqrsgraph.Schema{
		Nodes: []cqrsgraph.NodeType{
			{Label: "User", KeyProp: "id", Properties: []cqrsgraph.PropertyType{
				{Name: "name"},
			}},
			{Label: "Message", KeyProp: "id", Properties: []cqrsgraph.PropertyType{
				{Name: "content"}, {Name: "created_at"},
			}},
		},
		Edges: []cqrsgraph.EdgeType{
			{Type: "AUTHORED_BY", FromLabel: "Message", ToLabel: "User"},
			{
				Type:      "REPLY_TO",
				FromLabel: "Message",
				ToLabel:   "Message",
				Properties: []cqrsgraph.PropertyType{
					{Name: "at"},
				},
			},
		},
	}
}

func projectEvent(_ context.Context, evt cqrsevent.Event, sink cqrsgraph.GraphSink) error {
	var p messagePosted
	if err := json.Unmarshal(evt.Payload(), &p); err != nil {
		return fmt.Errorf("decode: %w", err)
	}

	msgRef := cqrsgraph.NodeRef{Label: "Message", KeyProp: "id", KeyValue: p.ID}

	if err := sink.MergeNode(msgRef, map[string]any{
		"content":    p.Content,
		"created_at": time.Now().Format(time.RFC3339),
	}); err != nil {
		return err
	}

	if err := sink.MergeEdge(cqrsgraph.EdgeRef{
		Type: "AUTHORED_BY",
		From: msgRef,
		To:   cqrsgraph.NodeRef{Label: "User", KeyProp: "id", KeyValue: p.AuthorID},
	}, nil); err != nil {
		return err
	}

	if p.ReplyTo != "" {
		if err := sink.MergeEdge(cqrsgraph.EdgeRef{
			Type: "REPLY_TO",
			From: msgRef,
			To:   cqrsgraph.NodeRef{Label: "Message", KeyProp: "id", KeyValue: p.ReplyTo},
		}, map[string]any{"at": time.Now().Format(time.RFC3339)}); err != nil {
			return err
		}
	}

	return nil
}

func main() {
	ctx := context.Background()
	driver := cqrsgraph.NewMemoryDriver()

	proj, err := cqrsgraph.NewGraphProjection(
		"forum-graph",
		driver,
		projectEvent,
		[]cqrsevent.Type{"MESSAGE_POSTED"},
		cqrsgraph.WithSchema(forumSchema()),
	)
	if err != nil {
		log.Fatalf("create projection: %v", err)
	}

	events := []messagePosted{
		{ID: "m1", AuthorID: "alice", Content: "Hello world!"},
		{ID: "m2", AuthorID: "bob", Content: "Hi Alice!", ReplyTo: "m1"},
		{ID: "m3", AuthorID: "carol", Content: "Welcome!", ReplyTo: "m1"},
		{ID: "m4", AuthorID: "bob", Content: "Thanks!", ReplyTo: "m2"},
	}

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
			log.Fatalf("project event: %v", err)
		}
	}

	fmt.Println("=== All Messages ===")

	for _, nv := range driver.Query(cqrsgraph.Pattern{Label: "Message"}) {
		fmt.Printf("  %s: %s\n", nv.Ref.KeyValue, nv.Props["content"])
	}

	fmt.Println("\n=== Reply Chain from m4 (REPLY_TO traversal) ===")
	fmt.Println("Following REPLY_TO edges from m4 back to its ancestors:")

	for _, nv := range driver.Traverse(
		cqrsgraph.NodeRef{Label: "Message", KeyProp: "id", KeyValue: "m4"},
		"REPLY_TO", -1,
	) {
		fmt.Printf("  → %s: %s\n", nv.Ref.KeyValue, nv.Props["content"])
	}

	fmt.Println("\n=== Shortest Path: m4 → m1 (reply chain) ===")

	path, err := driver.ShortestPath(
		cqrsgraph.NodeRef{Label: "Message", KeyProp: "id", KeyValue: "m4"},
		cqrsgraph.NodeRef{Label: "Message", KeyProp: "id", KeyValue: "m1"},
	)
	if err != nil {
		log.Fatalf("shortest path: %v", err)
	}

	for i, ref := range path {
		if i > 0 {
			fmt.Print(" → ")
		}

		fmt.Print(ref.KeyValue)
	}

	fmt.Println()

	// ── Schema rejection demo ──────────────────────────────────────────
	// The Schema catches typos before they create phantom nodes. Try
	// changing "User" to "Usr" in the handler above — the projection
	// returns ErrUnknownNodeType instead of silently writing garbage.
	// This is the value proposition of boundary-typed graph projections.
	fmt.Println("=== Schema Catches Typos ===")

	err = driver.RunInTx(func(sink cqrsgraph.GraphSink) error {
		return sink.MergeNode(
			cqrsgraph.NodeRef{Label: "Phantom", KeyProp: "id", KeyValue: "x"},
			nil,
		)
	})
	if err != nil {
		fmt.Printf("  Rejected unknown label: %v\n", err)
	}
}
