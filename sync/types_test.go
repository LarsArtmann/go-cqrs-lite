package sync

import (
	"testing"
)

func TestParseOperationID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    OperationID
		wantErr bool
	}{
		{"valid ID", "op-123", OperationID("op-123"), false},
		{
			"UUID-like",
			"550e8400-e29b-41d4-a716-446655440000",
			OperationID("550e8400-e29b-41d4-a716-446655440000"),
			false,
		},
		{"empty string", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := ParseOperationID(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestMustParseOperationID(t *testing.T) {
	t.Parallel()

	t.Run("valid ID returns ID", func(t *testing.T) {
		t.Parallel()

		id := MustParseOperationID("op-1")
		if id != OperationID("op-1") {
			t.Errorf("got %q, want %q", id, "op-1")
		}
	})

	t.Run("empty string panics", func(t *testing.T) {
		t.Parallel()

		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic")
			}
		}()

		MustParseOperationID("")
	})
}

func TestOperationID_String(t *testing.T) {
	t.Parallel()

	if got := OperationID("op-42").String(); got != "op-42" {
		t.Errorf("got %q, want %q", got, "op-42")
	}
}

func TestOperationID_IsZero(t *testing.T) {
	t.Parallel()

	t.Run("non-zero ID", func(t *testing.T) {
		t.Parallel()
		if OperationID("op-1").IsZero() {
			t.Error("expected false for non-zero ID")
		}
	})

	t.Run("zero ID", func(t *testing.T) {
		t.Parallel()
		if !OperationID("").IsZero() {
			t.Error("expected true for zero ID")
		}
	})
}

func TestParseNodeID(t *testing.T) {
	t.Parallel()

	t.Run("valid node ID", func(t *testing.T) {
		t.Parallel()

		got, err := ParseNodeID("node-a")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if got != NodeID("node-a") {
			t.Errorf("got %q, want %q", got, "node-a")
		}
	})

	t.Run("empty string", func(t *testing.T) {
		t.Parallel()

		_, err := ParseNodeID("")
		if err == nil {
			t.Fatal("expected error for empty node ID")
		}
	})
}
