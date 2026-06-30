package indexing

import (
	"errors"
	"fmt"
	"testing"
)

func TestIsIndexAlreadyExists(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"already exists lowercase", errors.New("index idx_users already exists"), true},
		{"already exists mixed case", errors.New("Index ALREADY EXISTS"), true},
		{"unrelated", errors.New("syntax error near CREATE"), false},
		{"wrapped", fmt.Errorf("create index: %w", errors.New("already exists")), true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := isIndexAlreadyExists(tc.err); got != tc.want {
				t.Fatalf("isIndexAlreadyExists(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}
