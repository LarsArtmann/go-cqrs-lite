package sync

import (
	"encoding/json"
	"testing"
	"time"
)

type testItem struct {
	Name      string    `json:"name"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func itemTimestamp(item testItem) time.Time {
	return item.UpdatedAt
}

func TestLWWResolver_WinsByVectorClock(t *testing.T) {
	t.Parallel()

	now := time.Now()
	local := testItem{Name: "local", UpdatedAt: now}
	remote := testItem{Name: "remote", UpdatedAt: now}

	tests := []struct {
		name         string
		localVC      VectorClock
		remoteVC     VectorClock
		expectWinner string
	}{
		{
			"remote wins with higher clock",
			VectorClock{NodeID("node-a"): 1},
			VectorClock{NodeID("node-a"): 3},
			"remote",
		},
		{
			"local wins with higher clock",
			VectorClock{NodeID("node-a"): 5},
			VectorClock{NodeID("node-a"): 2},
			"local",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			resolver, _ := NewLWWResolver(itemTimestamp)

			conflict := &Conflict[testItem]{
				Local:    local,
				Remote:   remote,
				LocalVC:  tt.localVC,
				RemoteVC: tt.remoteVC,
			}

			winner, err := resolver.Resolve(conflict)
			if err != nil {
				t.Fatalf("Resolve() error: %v", err)
			}

			if winner.Name != tt.expectWinner {
				t.Errorf("expected %s to win, got %q", tt.expectWinner, winner.Name)
			}
		})
	}
}

func TestLWWResolver_LocalWinsByTimestamp(t *testing.T) {
	t.Parallel()

	resolver, _ := NewLWWResolver(itemTimestamp)
	now := time.Now()
	local := testItem{Name: "local", UpdatedAt: now.Add(2 * time.Hour)}
	remote := testItem{Name: "remote", UpdatedAt: now}

	conflict := &Conflict[testItem]{
		Local:    local,
		Remote:   remote,
		LocalVC:  VectorClock{NodeID("a"): 1, NodeID("b"): 2},
		RemoteVC: VectorClock{NodeID("a"): 2, NodeID("b"): 1},
	}

	winner, err := resolver.Resolve(conflict)
	if err != nil {
		t.Fatalf("Resolve() error: %v", err)
	}

	if winner.Name != "local" {
		t.Errorf("expected local to win (later timestamp), got %q", winner.Name)
	}
}

func TestLWWResolver_RemoteWinsByTimestamp(t *testing.T) {
	t.Parallel()

	resolver, _ := NewLWWResolver(itemTimestamp)
	now := time.Now()
	local := testItem{Name: "local", UpdatedAt: now}
	remote := testItem{Name: "remote", UpdatedAt: now.Add(2 * time.Hour)}

	conflict := &Conflict[testItem]{
		Local:    local,
		Remote:   remote,
		LocalVC:  VectorClock{NodeID("a"): 1, NodeID("b"): 2},
		RemoteVC: VectorClock{NodeID("a"): 2, NodeID("b"): 1},
	}

	winner, err := resolver.Resolve(conflict)
	if err != nil {
		t.Fatalf("Resolve() error: %v", err)
	}

	if winner.Name != "remote" {
		t.Errorf("expected remote to win (later timestamp), got %q", winner.Name)
	}
}

func TestLWWResolver_RemoteWinsOnTie_NoTiebreaker(t *testing.T) {
	t.Parallel()

	resolver, _ := NewLWWResolver(itemTimestamp)
	now := time.Now()
	local := testItem{Name: "local", UpdatedAt: now}
	remote := testItem{Name: "remote", UpdatedAt: now}

	conflict := &Conflict[testItem]{
		Local:    local,
		Remote:   remote,
		LocalVC:  VectorClock{NodeID("a"): 1},
		RemoteVC: VectorClock{NodeID("a"): 1},
	}

	winner, err := resolver.Resolve(conflict)
	if err != nil {
		t.Fatalf("Resolve() error: %v", err)
	}

	if winner.Name != "remote" {
		t.Errorf("expected remote to win on tie (no tiebreaker), got %q", winner.Name)
	}
}

func TestLWWResolver_Tiebreaker(t *testing.T) {
	t.Parallel()

	resolver, _ := NewLWWResolver(itemTimestamp)
	resolver.Tiebreaker = func(local, remote testItem) bool {
		return local.Name < remote.Name
	}

	now := time.Now()

	tests := []struct {
		name         string
		localName    string
		remoteName   string
		expectWinner string
	}{
		{"local wins with smaller name", "aaa", "zzz", "aaa"},
		{"remote wins with smaller name", "zzz", "aaa", "aaa"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			local := testItem{Name: tt.localName, UpdatedAt: now}
			remote := testItem{Name: tt.remoteName, UpdatedAt: now}

			conflict := &Conflict[testItem]{
				Local:    local,
				Remote:   remote,
				LocalVC:  VectorClock{NodeID("a"): 1},
				RemoteVC: VectorClock{NodeID("a"): 1},
			}

			winner, err := resolver.Resolve(conflict)
			if err != nil {
				t.Fatalf("Resolve() error: %v", err)
			}

			if winner.Name != tt.expectWinner {
				t.Errorf("expected %s to win via tiebreaker, got %q", tt.expectWinner, winner.Name)
			}
		})
	}
}

func TestLWWResolver_ImplementsInterface(t *testing.T) {
	t.Parallel()

	var resolver ConflictResolver[testItem] = &LWWResolver[testItem]{
		TimestampFunc: itemTimestamp,
	}
	_ = resolver
}

func TestConflict_JSON_RoundTrip(t *testing.T) {
	t.Parallel()

	now := time.Now()
	conflict := Conflict[testItem]{
		Local:     testItem{Name: "local", UpdatedAt: now},
		Remote:    testItem{Name: "remote", UpdatedAt: now.Add(time.Hour)},
		LocalVC:   VectorClock{NodeID("a"): 1},
		RemoteVC:  VectorClock{NodeID("b"): 2},
		Timestamp: now,
	}

	data, err := json.Marshal(conflict)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var decoded Conflict[testItem]
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if decoded.Local.Name != "local" {
		t.Errorf("local name: got %q, want %q", decoded.Local.Name, "local")
	}

	if decoded.Remote.Name != "remote" {
		t.Errorf("remote name: got %q, want %q", decoded.Remote.Name, "remote")
	}

	if decoded.LocalVC.Get(NodeID("a")) != 1 {
		t.Errorf("local VC: got %d, want 1", decoded.LocalVC.Get(NodeID("a")))
	}

	if decoded.RemoteVC.Get(NodeID("b")) != 2 {
		t.Errorf("remote VC: got %d, want 2", decoded.RemoteVC.Get(NodeID("b")))
	}
}

func TestMergeResult_Values(t *testing.T) {
	t.Parallel()

	values := []MergeResult{
		MergeResultLocalWins,
		MergeResultRemoteWins,
		MergeResultMerged,
		MergeResultConflict,
	}

	for i, v := range values {
		if int(v) != i {
			t.Errorf("MergeResult value %d has unexpected ordinal %d", i, v)
		}
	}
}

func TestSyncMessage_JSON(t *testing.T) {
	t.Parallel()

	msg := SyncMessage{
		Type:    SyncMessageTypeRequest,
		Payload: json.RawMessage(`{"test":true}`),
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var decoded SyncMessage
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if decoded.Type != SyncMessageTypeRequest {
		t.Errorf("type: got %q, want %q", decoded.Type, SyncMessageTypeRequest)
	}
}

func TestSyncRequest_JSON(t *testing.T) {
	t.Parallel()

	since := time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)
	req := SyncRequest{
		SyncContext: SyncContext{
			NodeID: MustParseNodeID("node-1"),
			Clock:  VectorClock{NodeID("node-1"): 5},
		},
		Since: since,
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var decoded SyncRequest
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if decoded.NodeID.String() != "node-1" {
		t.Errorf("node ID: got %q, want %q", decoded.NodeID.String(), "node-1")
	}

	if decoded.Clock.Get(NodeID("node-1")) != 5 {
		t.Errorf("clock: got %d, want 5", decoded.Clock.Get(NodeID("node-1")))
	}
}

func TestSyncResponse_JSON(t *testing.T) {
	t.Parallel()

	resp := SyncResponse[testItem]{
		SyncContext: SyncContext{
			NodeID: MustParseNodeID("node-2"),
			Clock:  VectorClock{NodeID("node-1"): 5, NodeID("node-2"): 3},
		},
		Operations: []*Operation[testItem]{
			NewOperation(
				OperationID("op-1"),
				OpCreate,
				MustParseNodeID("node-2"),
				testItem{Name: "item1", UpdatedAt: time.Now()},
			),
		},
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var decoded SyncResponse[testItem]
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if decoded.NodeID.String() != "node-2" {
		t.Errorf("node ID: got %q, want %q", decoded.NodeID.String(), "node-2")
	}

	if len(decoded.Operations) != 1 {
		t.Fatalf("operations: got %d, want 1", len(decoded.Operations))
	}

	if decoded.Operations[0].Payload.Name != "item1" {
		t.Errorf("payload name: got %q, want %q", decoded.Operations[0].Payload.Name, "item1")
	}
}

func TestNewLWWResolver_NilTimestampFunc_ReturnsError(t *testing.T) {
	t.Parallel()

	_, err := NewLWWResolver[testItem](nil)
	if err == nil {
		t.Fatal("expected error when TimestampFunc is nil")
	}
}

func TestNewSyncContext(t *testing.T) {
	t.Parallel()

	nodeID := MustParseNodeID("node-1")
	clock := VectorClock{nodeID: 5}

	ctx := NewSyncContext(nodeID, clock)

	if ctx.NodeID != nodeID {
		t.Errorf("NodeID = %q, want %q", ctx.NodeID, nodeID)
	}

	if ctx.Clock.Get(nodeID) != 5 {
		t.Errorf("Clock[nodeID] = %d, want 5", ctx.Clock.Get(nodeID))
	}
}
