package id_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/core/v5/id"
)

func TestNewStreamRef_Validating(t *testing.T) {
	t.Parallel()

	sid := id.NewStreamID()

	ref, err := id.NewStreamRef("User", sid)
	if err != nil {
		t.Fatalf("NewStreamRef(User, %q) = %v, want nil", sid, err)
	}

	if ref.Type != "User" || ref.ID != sid {
		t.Errorf("NewStreamRef = %v, want {User %v}", ref, sid)
	}

	if _, err := id.NewStreamRef("", sid); err == nil {
		t.Error("NewStreamRef with empty type must fail (ADR-0166)")
	}

	if _, err := id.NewStreamRef("User", id.StreamID{}); err == nil {
		t.Error("NewStreamRef with zero ID must fail (ADR-0166)")
	}
}

func TestNewStreamRefOrZero(t *testing.T) {
	t.Parallel()

	sid := id.NewStreamID()

	if got := id.NewStreamRefOrZero("User", sid); got != (id.StreamRef{Type: "User", ID: sid}) {
		t.Errorf("NewStreamRefOrZero(User, %q) = %v, want the typed ref", sid, got)
	}

	if got := id.NewStreamRefOrZero("", sid); !got.IsZero() {
		t.Errorf("NewStreamRefOrZero with empty type = %v, want zero", got)
	}

	if got := id.NewStreamRefOrZero("User", id.StreamID{}); !got.IsZero() {
		t.Errorf("NewStreamRefOrZero with zero ID = %v, want zero", got)
	}
}

func TestParseStreamRef(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		key     string
		want    id.StreamRef
		wantErr bool
	}{
		{
			name: "storage key round trip",
			key:  "User:01JTEST",
			want: id.StreamRef{Type: "User", ID: idtestStreamID(t, "01JTEST")},
		},
		{
			name: "ID may contain colons (domain/derived IDs)",
			key:  "lock:user1:resource2",
			want: id.StreamRef{Type: "lock", ID: idtestStreamID(t, "user1:resource2")},
		},
		{name: "no separator", key: "User01JTEST", wantErr: true},
		{name: "empty type", key: ":01JTEST", wantErr: true},
		{name: "empty ID", key: "User:", wantErr: true},
		{name: "empty key", key: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := id.ParseStreamRef(tt.key)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("ParseStreamRef(%q) = %v, want error", tt.key, got)
				}

				return
			}

			if err != nil {
				t.Fatalf("ParseStreamRef(%q) = %v, want nil", tt.key, err)
			}

			if got != tt.want {
				t.Errorf("ParseStreamRef(%q) = %v, want %v", tt.key, got, tt.want)
			}

			if key := got.StreamKey(); key != tt.key {
				t.Errorf("StreamKey() = %q, want the input key %q", key, tt.key)
			}
		})
	}
}

func idtestStreamID(t *testing.T, s string) id.StreamID {
	t.Helper()

	sid, err := id.ParseStreamID(s)
	if err != nil {
		t.Fatalf("ParseStreamID(%q): %v", s, err)
	}

	return sid
}
