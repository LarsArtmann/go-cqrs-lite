package sync

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
				require.Error(t, err)
				assert.Contains(t, err.Error(), "operation ID cannot be empty")

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestMustParseOperationID(t *testing.T) {
	t.Parallel()

	t.Run("valid ID returns ID", func(t *testing.T) {
		t.Parallel()

		id := MustParseOperationID("op-1")
		assert.Equal(t, OperationID("op-1"), id)
	})

	t.Run("empty string panics", func(t *testing.T) {
		t.Parallel()

		assert.Panics(t, func() {
			MustParseOperationID("")
		})
	})
}

func TestOperationID_String(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "op-42", OperationID("op-42").String())
}

func TestOperationID_IsZero(t *testing.T) {
	t.Parallel()

	t.Run("non-zero ID", func(t *testing.T) {
		t.Parallel()
		assert.False(t, OperationID("op-1").IsZero())
	})

	t.Run("zero ID", func(t *testing.T) {
		t.Parallel()
		assert.True(t, OperationID("").IsZero())
	})
}

func TestParseNodeID(t *testing.T) {
	t.Parallel()

	t.Run("valid node ID", func(t *testing.T) {
		t.Parallel()

		got, err := ParseNodeID("node-a")
		require.NoError(t, err)
		assert.Equal(t, NodeID("node-a"), got)
	})

	t.Run("empty string", func(t *testing.T) {
		t.Parallel()

		_, err := ParseNodeID("")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "node ID cannot be empty")
	})
}
