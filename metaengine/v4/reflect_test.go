package metaengine_test

import (
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// TestTypeName: a value and its pointer share one name, and a nil value has
// none.
func TestTypeName(t *testing.T) {
	t.Parallel()

	type sample struct{ Name string }

	cases := []struct {
		name  string
		value any
		want  string
	}{
		{name: "value", value: sample{Name: "a"}, want: "sample"},
		{name: "pointer", value: &sample{Name: "a"}, want: "sample"},
		{name: "nil", value: nil, want: ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := metaengine.TypeName(tc.value); got != tc.want {
				t.Errorf("TypeName(%s) = %q, want %q", tc.name, got, tc.want)
			}
		})
	}
}
