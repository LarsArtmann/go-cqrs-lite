package indexing

import (
	"errors"
	"testing"
)

func TestIsUnsupportedPragma(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"nil error", nil, false},
		{"not a valid pragma", errors.New("Parse error: Not a valid pragma name"), true},
		{"unknown pragma", errors.New("unknown pragma foo"), true},
		{"unrecognized pragma", errors.New("unrecognized pragma: bar"), true},
		{"unrelated error", errors.New("connection refused"), false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := isUnsupportedPragma(tc.err); got != tc.want {
				t.Errorf("isUnsupportedPragma(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}
