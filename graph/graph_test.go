package graph

import (
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"testing"
	"time"

	cqrsevent "github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// DiscordSync-shaped payload: a message that references its author, channel,
// guild, an optional reply target, and the reactions it received. Every one of
// these is naturally a graph edge, not a relational FK column.
//
//nolint:tagliatelle // struct mirrors Discord's snake_case JSON API
type messageCreated struct {
	ID             string         `json:"id"`
	ChannelID      string         `json:"channel_id"`
	GuildID        string         `json:"guild_id"`
	AuthorID       string         `json:"author_id"`
	ReplyToMessage string         `json:"reply_to_message_id,omitempty"`
	Reactions      []reactionEdge `json:"reactions"`
}

//nolint:tagliatelle // mirrors Discord's snake_case JSON API
type reactionEdge struct {
	UserID string `json:"user_id"`
	Emoji  string `json:"emoji"`
}

// nodeRef is a tiny constructor for readable test setup.
func nodeRef(label, key string) NodeRef {
	return NodeRef{Label: label, KeyProp: "id", KeyValue: key}
}

// handleMessageCreated merges the full event subgraph: message node, author/
// channel/guild nodes, the POSTED_IN / AUTHORED_BY edges, an optional REPLY_TO
// edge (recursive — the thing relational handles with WITH RECURSIVE CTEs and
// graph handles with a single traversal), and reaction edges. All atomic.
func handleMessageCreated(_ context.Context, evt cqrsevent.Event, sink GraphSink) error {
	var p messageCreated
	if err := json.Unmarshal(evt.Payload(), &p); err != nil {
		return fmt.Errorf("decode payload: %w", err)
	}

	msgRef := nodeRef("Message", p.ID)

	if err := sink.MergeNode(msgRef, map[string]any{
		"id": p.ID, "created_at": time.Now().Format(time.RFC3339),
	}); err != nil {
		return err
	}

	// Auto-creates endpoint nodes via MergeEdge — handlers need not pre-merge.
	if err := sink.MergeEdge(EdgeRef{
		Type: "POSTED_IN", From: msgRef, To: nodeRef("Channel", p.ChannelID),
	}, nil); err != nil {
		return err
	}

	if err := sink.MergeEdge(EdgeRef{
		Type: "AUTHORED_BY", From: msgRef, To: nodeRef("User", p.AuthorID),
	}, nil); err != nil {
		return err
	}

	if p.GuildID != "" {
		if err := sink.MergeEdge(EdgeRef{
			Type: "IN_GUILD",
			From: nodeRef("Channel", p.ChannelID),
			To:   nodeRef("Guild", p.GuildID),
		}, nil); err != nil {
			return err
		}
	}

	// The recursive edge — reply chains. Relational tier needs WITH RECURSIVE;
	// graph tier stores it as a plain edge and traverses natively.
	if p.ReplyToMessage != "" {
		if err := sink.MergeEdge(EdgeRef{
			Type: "REPLY_TO", From: msgRef, To: nodeRef("Message", p.ReplyToMessage),
		}, map[string]any{"at": time.Now().Format(time.RFC3339)}); err != nil {
			return err
		}
	}

	// Reaction network: who reacted to what, with which emoji. A bipartite
	// edge set that is painful in SQL (junction table + emoji-as-column or
	// emoji-as-row) and trivial in a graph.
	for _, r := range p.Reactions {
		if err := sink.MergeEdge(EdgeRef{
			Type: "REACTED", From: nodeRef("User", r.UserID), To: msgRef,
		}, map[string]any{"emoji": r.Emoji}); err != nil {
			return err
		}
	}

	return nil
}

func newEvent(t *testing.T, eventType string, payload any) cqrsevent.Event {
	t.Helper()

	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	aggID, _ := id.ParseAggregateID("msg-agg")

	evt, err := cqrsevent.NewEvent(
		cqrsevent.Type(eventType),
		aggID,
		"Message",
		cqrsevent.Version(1),
		raw,
	)
	if err != nil {
		t.Fatalf("new event: %v", err)
	}

	return evt
}

func TestGraphProjection_MergesFullSubgraph(t *testing.T) {
	t.Parallel()

	driver := NewMemoryDriver()

	proj, err := NewGraphProjection("discord-graph", driver, handleMessageCreated,
		[]cqrsevent.Type{"MESSAGE_CREATED"})
	if err != nil {
		t.Fatalf("new projection: %v", err)
	}

	ctx := context.Background()

	err = proj.Handle(ctx, newEvent(t, "MESSAGE_CREATED", messageCreated{
		ID: "m1", ChannelID: "c1", GuildID: "g1", AuthorID: "u1",
		ReplyToMessage: "m0",
		Reactions: []reactionEdge{
			{UserID: "u2", Emoji: "👍"},
			{UserID: "u3", Emoji: "❤️"},
		},
	}))
	if err != nil {
		t.Fatalf("handle: %v", err)
	}

	g := driver.Snapshot()

	// 5 explicit/implicit nodes: m1, m0 (reply target), c1, g1, u1, u2, u3 = 7
	if got, want := len(g.nodes), 7; got != want {
		t.Fatalf("nodes = %d, want %d", got, want)
	}

	// Edges: POSTED_IN, AUTHORED_BY, IN_GUILD, REPLY_TO, 2× REACTED = 6
	if got, want := len(g.edges), 6; got != want {
		t.Fatalf("edges = %d, want %d", got, want)
	}

	// Verify the recursive edge landed.
	replyKey := EdgeRef{
		Type: "REPLY_TO",
		From: nodeRef("Message", "m1"),
		To:   nodeRef("Message", "m0"),
	}.keyOrFatal(
		t,
	)
	if _, ok := g.edges[replyKey]; !ok {
		t.Fatalf("REPLY_TO edge missing")
	}

	// Verify reaction edge carries the emoji property.
	reactKey := EdgeRef{
		Type: "REACTED",
		From: nodeRef("User", "u2"),
		To:   nodeRef("Message", "m1"),
	}.keyOrFatal(
		t,
	)
	reactEdge, ok := g.edges[reactKey]
	if !ok {
		t.Fatalf("REACTED edge missing")
	}

	if reactEdge.props["emoji"] != "👍" {
		t.Fatalf("emoji prop = %v, want 👍", reactEdge.props["emoji"])
	}

	// event.Projection conformance.
	if proj.Name() != "discord-graph" {
		t.Fatalf("name = %q", proj.Name())
	}

	if len(proj.EventTypes()) != 1 || string(proj.EventTypes()[0]) != "MESSAGE_CREATED" {
		t.Fatalf("types = %v", proj.EventTypes())
	}
}

func TestGraphProjection_AtomicRollbackOnError(t *testing.T) {
	t.Parallel()

	driver := NewMemoryDriver()

	wantErr := errors.New("boom mid-merge")

	handler := func(_ context.Context, _ cqrsevent.Event, sink GraphSink) error {
		if err := sink.MergeNode(
			nodeRef("User", "u1"),
			map[string]any{"name": "alice"},
		); err != nil {
			return err
		}

		if err := sink.MergeNode(nodeRef("User", "u2"), map[string]any{"name": "bob"}); err != nil {
			return err
		}

		return wantErr // fail AFTER writes — both must roll back.
	}

	proj, err := NewGraphProjection("rb", driver, handler, nil)
	if err != nil {
		t.Fatalf("new projection: %v", err)
	}

	err = proj.Handle(context.Background(), newEvent(t, "X", nil))
	if !errors.Is(err, wantErr) {
		t.Fatalf("err = %v, want %v", err, wantErr)
	}

	g := driver.Snapshot()
	if len(g.nodes) != 0 {
		t.Fatalf("nodes after rollback = %d, want 0", len(g.nodes))
	}
}

func TestGraphProjection_IdempotentReplay(t *testing.T) {
	t.Parallel()

	driver := NewMemoryDriver()

	proj, err := NewGraphProjection("idem", driver, handleMessageCreated, nil)
	if err != nil {
		t.Fatalf("new projection: %v", err)
	}

	evt := newEvent(t, "MESSAGE_CREATED", messageCreated{
		ID: "m1", ChannelID: "c1", AuthorID: "u1",
	})

	ctx := context.Background()

	for range 3 {
		if err := proj.Handle(ctx, evt); err != nil {
			t.Fatalf("handle: %v", err)
		}
	}

	g := driver.Snapshot()
	// Still exactly the nodes/edges from one projection — MERGE semantics.
	if got, want := len(g.nodes), 3; got != want { // m1, c1, u1
		t.Fatalf("nodes = %d, want %d (idempotency broken)", got, want)
	}

	if got, want := len(g.edges), 2; got != want { // POSTED_IN, AUTHORED_BY
		t.Fatalf("edges = %d, want %d (idempotency broken)", got, want)
	}
}

func TestGraphProjection_RemoveNodeDeletesIncidentEdges(t *testing.T) {
	t.Parallel()

	driver := NewMemoryDriver()

	// Seed: u1 authored m1, m1 posted in c1.
	seed, _ := NewGraphProjection("seed", driver, handleMessageCreated, nil)
	_ = seed.Handle(context.Background(), newEvent(t, "MESSAGE_CREATED", messageCreated{
		ID: "m1", ChannelID: "c1", AuthorID: "u1",
	}))

	// Delete the message node: POSTED_IN and AUTHORED_BY edges must go too.
	del := func(_ context.Context, _ cqrsevent.Event, sink GraphSink) error {
		return sink.RemoveNode(nodeRef("Message", "m1"))
	}

	proj, _ := NewGraphProjection("del", driver, del, nil)

	if err := proj.Handle(context.Background(), newEvent(t, "DELETE", nil)); err != nil {
		t.Fatalf("handle: %v", err)
	}

	g := driver.Snapshot()

	if _, ok := g.nodes[nodeRef("Message", "m1").keyOrFatal(t)]; ok {
		t.Fatalf("message node still present after RemoveNode")
	}

	// No edge remains incident to m1.
	m1Key := nodeRef("Message", "m1").keyOrFatal(t)

	for ek := range g.edges {
		if ek.from == m1Key || ek.to == m1Key {
			t.Fatalf("incident edge %s still present after RemoveNode", ek.typ)
		}
	}
}

func TestGraphProjection_RemoveEdgeLeavesEndpoints(t *testing.T) {
	t.Parallel()

	driver := NewMemoryDriver()

	seed, _ := NewGraphProjection("seed", driver, handleMessageCreated, nil)
	_ = seed.Handle(context.Background(), newEvent(t, "MESSAGE_CREATED", messageCreated{
		ID: "m1", ChannelID: "c1", AuthorID: "u1",
	}))

	before := driver.Snapshot()
	if len(before.edges) != 2 { // POSTED_IN, AUTHORED_BY
		t.Fatalf("seed edges = %d, want 2", len(before.edges))
	}

	// Remove only the AUTHORED_BY edge; both endpoint nodes must remain.
	del := func(_ context.Context, _ cqrsevent.Event, sink GraphSink) error {
		return sink.RemoveEdge(EdgeRef{
			Type: "AUTHORED_BY", From: nodeRef("Message", "m1"), To: nodeRef("User", "u1"),
		})
	}

	proj, _ := NewGraphProjection("del-edge", driver, del, nil)

	if err := proj.Handle(context.Background(), newEvent(t, "R", nil)); err != nil {
		t.Fatalf("handle: %v", err)
	}

	g := driver.Snapshot()

	if len(g.edges) != 1 { // only POSTED_IN remains
		t.Fatalf("edges after RemoveEdge = %d, want 1", len(g.edges))
	}

	// Both endpoints still exist.
	for _, ref := range []NodeRef{nodeRef("Message", "m1"), nodeRef("User", "u1")} {
		if _, ok := g.nodes[ref.keyOrFatal(t)]; !ok {
			t.Fatalf("endpoint node %v removed by RemoveEdge", ref)
		}
	}
}

func TestGraphProjection_CloseIdempotentAndNilSafe(t *testing.T) {
	t.Parallel()

	driver := NewMemoryDriver()
	proj, _ := NewGraphProjection("c", driver, handleMessageCreated, nil)

	if err := proj.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	// Close on a projection whose driver is nil (defensive) must not panic.
	proj.driver = nil
	if err := proj.Close(); err != nil {
		t.Fatalf("nil-driver close: %v", err)
	}
}

func TestGraphProjection_SetNodePropertySparseUpdate(t *testing.T) {
	t.Parallel()

	driver := NewMemoryDriver()

	// Seed a node with two props.
	seed := func(_ context.Context, _ cqrsevent.Event, sink GraphSink) error {
		return sink.MergeNode(nodeRef("User", "u1"), map[string]any{
			"name": "alice", "avatar": "pic.png",
		})
	}

	proj, _ := NewGraphProjection("seed", driver, seed, nil)
	_ = proj.Handle(context.Background(), newEvent(t, "S", nil))

	// Sparse update: change only "name", preserve "avatar".
	update := func(_ context.Context, _ cqrsevent.Event, sink GraphSink) error {
		return sink.SetNodeProperty(nodeRef("User", "u1"), "name", "alice2")
	}

	up, _ := NewGraphProjection("up", driver, update, nil)
	_ = up.Handle(context.Background(), newEvent(t, "U", nil))

	g := driver.Snapshot()
	n := g.nodes[nodeRef("User", "u1").keyOrFatal(t)]

	if n.props["name"] != "alice2" {
		t.Fatalf("name = %v, want alice2", n.props["name"])
	}

	if n.props["avatar"] != "pic.png" {
		t.Fatalf("avatar = %v, want pic.png (sparse update clobbered it)", n.props["avatar"])
	}
}

func TestNewGraphProjection_RejectsBadInputs(t *testing.T) {
	t.Parallel()

	handler := func(context.Context, cqrsevent.Event, GraphSink) error { return nil }

	if _, err := NewGraphProjection(
		"",
		NewMemoryDriver(),
		handler,
		nil,
	); !errors.Is(
		err,
		errNoName,
	) {
		t.Fatalf("empty name err = %v", err)
	}

	if _, err := NewGraphProjection("x", nil, handler, nil); !errors.Is(err, errNilDriver) {
		t.Fatalf("nil driver err = %v", err)
	}

	if _, err := NewGraphProjection(
		"x",
		NewMemoryDriver(),
		nil,
		nil,
	); !errors.Is(
		err,
		errNilHandler,
	) {
		t.Fatalf("nil handler err = %v", err)
	}
}

func TestNodeRefValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		ref     NodeRef
		wantErr error
	}{
		{"empty label", NodeRef{KeyProp: "id", KeyValue: "x"}, errEmptyLabel},
		{"empty keyprop", NodeRef{Label: "User", KeyValue: "x"}, errEmptyKeyProp},
		{"valid", NodeRef{Label: "User", KeyProp: "id", KeyValue: "x"}, nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := tc.ref.validate()
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("err = %v, want %v", err, tc.wantErr)
			}
		})
	}
}

// keyOrFatal is a test helper that computes an edge/node key or fails the test.
func (r EdgeRef) keyOrFatal(t *testing.T) edgeKey {
	t.Helper()

	k, err := r.key()
	if err != nil {
		t.Fatalf("edge key: %v", err)
	}

	return k
}

func (r NodeRef) keyOrFatal(t *testing.T) nodeKey {
	t.Helper()

	k, err := r.key()
	if err != nil {
		t.Fatalf("node key: %v", err)
	}

	return k
}
