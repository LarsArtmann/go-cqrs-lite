package sync

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testItem struct {
	Name      string    `json:"name"`
	UpdatedAt time.Time `json:"updated_at"`
}

// sharedTestVCs holds common vector clocks used across tests.
var sharedTestVCs = struct {
	ConcurrentClock1 VectorClock
	ConcurrentClock2 VectorClock
	TiedClock        VectorClock
}{
	ConcurrentClock1: VectorClock{"a": 1, "b": 2},
	ConcurrentClock2: VectorClock{"a": 2, "b": 1},
	TiedClock:        VectorClock{"a": 1},
}

func itemTimestamp(item testItem) time.Time {
	return item.UpdatedAt
}

// marshalJSON marshals v to JSON, failing the test on error.
func marshalJSON[T any](t *testing.T, v T) []byte {
	data, err := json.Marshal(v)
	require.NoError(t, err)

	return data
}

// unmarshalJSON unmarshals data to v, failing the test on error.
func unmarshalJSON[T any](t *testing.T, data []byte, v *T) {
	err := json.Unmarshal(data, v)
	require.NoError(t, err)
}

func TestLWWResolver_WinsByVectorClock(t *testing.T) {
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
			VectorClock{"node-a": 1},
			VectorClock{"node-a": 3},
			"remote",
		},
		{
			"local wins with higher clock",
			VectorClock{"node-a": 5},
			VectorClock{"node-a": 2},
			"local",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testVCConflict(t, local, remote, tt.localVC, tt.remoteVC, tt.expectWinner)
		})
	}
}

func testVCConflict(
	t *testing.T,
	local, remote testItem,
	localVC, remoteVC VectorClock,
	expectWinner string,
) {
	resolver := NewLWWResolver[testItem](itemTimestamp)

	winner := resolveConflict(t, resolver, local, remote, localVC, remoteVC)
	if winner.Name != expectWinner {
		t.Errorf("expected %s to win, got %q", expectWinner, winner.Name)
	}
}

func resolveConflict(
	t *testing.T,
	resolver *LWWResolver[testItem],
	local, remote testItem,
	localVC, remoteVC VectorClock,
) testItem {
	conflict := &Conflict[testItem]{
		Local:    local,
		Remote:   remote,
		LocalVC:  localVC,
		RemoteVC: remoteVC,
	}

	winner, err := resolver.Resolve(conflict)
	if err != nil {
		t.Fatalf("Resolve() error: %v", err)
	}

	return winner
}

func TestLWWResolver_LocalWinsByTimestamp(t *testing.T) {
	resolver := NewLWWResolver[testItem](itemTimestamp)
	now := time.Now()
	local := testItem{Name: "local", UpdatedAt: now.Add(2 * time.Hour)}
	remote := testItem{Name: "remote", UpdatedAt: now}

	winner := resolveConflict(t, resolver, local, remote,
		sharedTestVCs.ConcurrentClock1, sharedTestVCs.ConcurrentClock2)
	if winner.Name != "local" {
		t.Errorf("expected local to win (later timestamp), got %q", winner.Name)
	}
}

func TestLWWResolver_RemoteWinsByTimestamp(t *testing.T) {
	resolver := NewLWWResolver[testItem](itemTimestamp)
	now := time.Now()
	local := testItem{Name: "local", UpdatedAt: now}
	remote := testItem{Name: "remote", UpdatedAt: now.Add(2 * time.Hour)}

	winner := resolveConflict(t, resolver, local, remote,
		sharedTestVCs.ConcurrentClock1, sharedTestVCs.ConcurrentClock2)
	if winner.Name != "remote" {
		t.Errorf("expected remote to win (later timestamp), got %q", winner.Name)
	}
}

func TestLWWResolver_RemoteWinsOnTie_NoTiebreaker(t *testing.T) {
	resolver := NewLWWResolver[testItem](itemTimestamp)
	now := time.Now()
	local := testItem{Name: "local", UpdatedAt: now}
	remote := testItem{Name: "remote", UpdatedAt: now}

	winner := resolveConflict(t, resolver, local, remote,
		sharedTestVCs.TiedClock, sharedTestVCs.TiedClock)
	if winner.Name != "remote" {
		t.Errorf("expected remote to win on tie (no tiebreaker), got %q", winner.Name)
	}
}

func TestLWWResolver_Tiebreaker(t *testing.T) {
	resolver := NewLWWResolver[testItem](itemTimestamp)
	resolver.Tiebreaker = nameTiebreaker
	now := time.Now()

	tests := []struct {
		name         string
		localName    string
		remoteName   string
		expectWinner string
	}{
		{"local wins with lexicographically smaller name", "aaa", "zzz", "aaa"},
		{"remote wins with lexicographically smaller name", "zzz", "aaa", "aaa"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			local := testItem{Name: tt.localName, UpdatedAt: now}
			remote := testItem{Name: tt.remoteName, UpdatedAt: now}

			winner := resolveConflict(t, resolver, local, remote,
				sharedTestVCs.TiedClock, sharedTestVCs.TiedClock)
			if winner.Name != tt.expectWinner {
				t.Errorf("expected %s to win via tiebreaker, got %q", tt.expectWinner, winner.Name)
			}
		})
	}
}

var nameTiebreaker = func(local, remote testItem) bool {
	return local.Name < remote.Name
}

func TestLWWResolver_ImplementsInterface(t *testing.T) {
	var _ ConflictResolver[testItem] = NewLWWResolver[testItem](itemTimestamp)
}

func TestConflict_JSON_RoundTrip(t *testing.T) {
	now := time.Now()
	conflict := Conflict[testItem]{
		Local:     testItem{Name: "local", UpdatedAt: now},
		Remote:    testItem{Name: "remote", UpdatedAt: now.Add(time.Hour)},
		LocalVC:   VectorClock{"a": 1},
		RemoteVC:  VectorClock{"b": 2},
		Timestamp: now,
	}

	data := marshalJSON(t, conflict)

	var decoded Conflict[testItem]
	unmarshalJSON(t, data, &decoded)

	assert.Equal(t, "local", decoded.Local.Name)
	assert.Equal(t, "remote", decoded.Remote.Name)
	assert.Equal(t, int64(1), decoded.LocalVC.Get("a"))
	assert.Equal(t, int64(2), decoded.RemoteVC.Get("b"))
}

func TestMergeResult_Values(t *testing.T) {
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
	msg := SyncMessage{
		Type:    SyncMessageTypeRequest,
		Payload: json.RawMessage(`{"test":true}`),
	}

	data := marshalJSON(t, msg)

	var decoded SyncMessage
	unmarshalJSON(t, data, &decoded)

	assert.Equal(t, SyncMessageTypeRequest, decoded.Type)
}

func TestSyncRequest_JSON(t *testing.T) {
	since := time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)
	req := SyncRequest{
		SyncContextMixin: SyncContextMixin{
			NodeID: MustParseNodeID("node-1"),
			Clock:  VectorClock{"node-1": 5},
		},
		Since: since,
	}

	data := marshalJSON(t, req)

	var decoded SyncRequest
	unmarshalJSON(t, data, &decoded)

	assert.Equal(t, "node-1", decoded.NodeID.String())
	assert.Equal(t, int64(5), decoded.Clock.Get("node-1"))
}

func TestSyncResponse_JSON(t *testing.T) {
	resp := SyncResponse[testItem]{
		SyncContextMixin: SyncContextMixin{
			NodeID: MustParseNodeID("node-2"),
			Clock:  VectorClock{"node-1": 5, "node-2": 3},
		},
		Operations: []*Operation[testItem]{
			NewOperation(
				"op-1",
				OpCreate,
				MustParseNodeID("node-2"),
				testItem{Name: "item1", UpdatedAt: time.Now()},
			),
		},
	}

	data := marshalJSON(t, resp)

	var decoded SyncResponse[testItem]
	unmarshalJSON(t, data, &decoded)

	assert.Equal(t, "node-2", decoded.NodeID.String())
	assert.Len(t, decoded.Operations, 1)
	assert.Equal(t, "item1", decoded.Operations[0].Payload.Name)
}
