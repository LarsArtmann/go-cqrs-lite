package snapshot_test

import (
	"context"
	"errors"
	"testing"

	"github.com/larsartmann/go-codec"
	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4/idtest"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
	"github.com/larsartmann/go-cqrs-lite/snapshot/v4"
	"github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
)

func TestNewSnapshot_SetsFieldsAndStampsTime(t *testing.T) {
	t.Parallel()

	ref := id.NewStreamRef("User", idtest.ParseStreamID(t, "01HK1540X0841Y0A6BSX1VKR95"))
	state := []byte(`{"name":"John"}`)

	snap, err := snapshot.NewSnapshot(ref, event.Version(5), state, record.EncodingCBOR)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if snap.StreamType != "User" || snap.StreamID != ref.ID {
		t.Errorf("identity = %s/%s, want User/%s", snap.StreamType, snap.StreamID, ref.ID)
	}

	if snap.Version != 5 {
		t.Errorf("Version = %d, want 5", snap.Version)
	}

	if snap.Encoding != record.EncodingCBOR {
		t.Errorf("Encoding = %s, want %s", snap.Encoding, record.EncodingCBOR)
	}

	if snap.CreatedAt.IsZero() {
		t.Error("CreatedAt not stamped")
	}

	if snap.Ref() != ref {
		t.Errorf("Ref() = %s, want %s", snap.Ref(), ref)
	}
}

func TestNewSnapshot_ClonesState(t *testing.T) {
	t.Parallel()

	ref := id.NewStreamRef("User", id.NewStreamID())
	state := []byte(`{"v":1}`)

	snap, err := snapshot.NewSnapshot(ref, event.Version(2), state, record.EncodingJSON)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	state[0] = 'X'

	if snap.State[0] != '{' {
		t.Error("NewSnapshot did not clone state; caller mutation leaked in")
	}
}

func TestNewSnapshot_RejectsInvalidInputs(t *testing.T) {
	t.Parallel()

	validID := idtest.ParseStreamID(t, "01HK1540X0841Y0A6BSX1VKR95")
	state := []byte(`{}`)

	tests := []struct {
		name    string
		ref     id.StreamRef
		version event.Version
		state   []byte
		wantErr error
	}{
		{
			name:    "zero ref",
			ref:     id.StreamRef{},
			version: 1,
			state:   state,
			wantErr: id.ErrEmptyStreamType,
		},
		{
			name:    "zero version",
			ref:     id.NewStreamRef("User", validID),
			version: 0,
			state:   state,
			wantErr: snapshot.ErrInvalidSnapshot,
		},
		{
			name:    "nil state",
			ref:     id.NewStreamRef("User", validID),
			version: 1,
			state:   nil,
			wantErr: snapshot.ErrInvalidSnapshot,
		},
		{
			name:    "empty state",
			ref:     id.NewStreamRef("User", validID),
			version: 1,
			state:   []byte{},
			wantErr: snapshot.ErrInvalidSnapshot,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := snapshot.NewSnapshot(tt.ref, tt.version, tt.state, record.EncodingCBOR)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("NewSnapshot error = %v, want Is(%v)", err, tt.wantErr)
			}

			if family := errorfamily.Classify(err); family != errorfamily.Rejection {
				t.Errorf("family = %s, want Rejection", family)
			}
		})
	}
}

func TestSnapshot_Validate(t *testing.T) {
	t.Parallel()

	validID := idtest.ParseStreamID(t, "01HK1540X0841Y0A6BSX1VKR95")

	tests := []struct {
		name    string
		snap    snapshot.Snapshot
		wantErr bool
	}{
		{
			name: "valid",
			snap: snapshot.Snapshot{
				StreamID:   validID,
				StreamType: "User",
				Version:    3,
				State:      []byte(`{}`),
				Encoding:   record.EncodingUnknown,
			},
			wantErr: false,
		},
		{
			name: "zero version",
			snap: snapshot.Snapshot{
				StreamID:   validID,
				StreamType: "User",
				Version:    0,
				State:      []byte(`{}`),
			},
			wantErr: true,
		},
		{
			name: "nil state",
			snap: snapshot.Snapshot{
				StreamID:   validID,
				StreamType: "User",
				Version:    3,
				State:      nil,
			},
			wantErr: true,
		},
		{
			name: "missing type",
			snap: snapshot.Snapshot{
				StreamID: validID,
				Version:  3,
				State:    []byte(`{}`),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.snap.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSaveSnapshot_RejectsInvalidInput(t *testing.T) {
	t.Parallel()

	store := memory.NewMemorySnapshotStore()
	t.Cleanup(func() { _ = store.Close() })

	err := snapshot.SaveSnapshot(
		context.Background(),
		store,
		"User",
		id.NewStreamID(),
		event.Version(0),
		[]byte(`{}`),
	)
	if !errors.Is(err, snapshot.ErrInvalidSnapshot) {
		t.Fatalf("SaveSnapshot(version=0) error = %v, want Is(ErrInvalidSnapshot)", err)
	}
}

func TestTypedStore_SaveStampsEncoding(t *testing.T) {
	t.Parallel()

	store := memory.NewMemorySnapshotStore()
	t.Cleanup(func() { _ = store.Close() })

	typed := snapshot.NewTypedStore[map[string]any](store, codec.CBORCodec{})
	ref := id.NewStreamRef("User", id.NewStreamID())

	err := typed.Save(context.Background(), snapshot.TypedSnapshot[map[string]any]{
		StreamID:   ref.ID,
		StreamType: ref.Type,
		Version:    4,
		State:      map[string]any{"hp": float64(42)},
	})
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	raw, err := store.Load(context.Background(), ref)
	if err != nil {
		t.Fatalf("Load raw: %v", err)
	}

	if raw.Encoding != record.EncodingCBOR {
		t.Errorf("raw.Encoding = %s, want %s", raw.Encoding, record.EncodingCBOR)
	}

	if err := raw.Validate(); err != nil {
		t.Errorf("saved snapshot failed Validate: %v", err)
	}
}
