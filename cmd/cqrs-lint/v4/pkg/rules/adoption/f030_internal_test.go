package adoption

import "testing"

// TestFragmentMatchesImport pins the segment-boundary contract behind F030's
// longest-fragment matching. Plain strings.Contains would make the deleted
// parent idempotency fragment match "go-cqrs-lite/idempotencyx" or a foreign
// "example.com/.../idempotency" path.
func TestFragmentMatchesImport(t *testing.T) {
	t.Parallel()

	tests := []struct {
		path, fragment string
		want           bool
	}{
		{"github.com/larsartmann/go-cqrs-lite/idempotency/v4", "go-cqrs-lite/idempotency", true},
		{"github.com/larsartmann/go-cqrs-lite/idempotency", "go-cqrs-lite/idempotency", true},
		{
			"github.com/larsartmann/go-cqrs-lite/idempotency/kvstore/v4",
			"go-cqrs-lite/idempotency",
			true,
		},
		{"github.com/other/idempotency", "go-cqrs-lite/idempotency", false},
		{"github.com/larsartmann/go-cqrs-lite/idempotencyx", "go-cqrs-lite/idempotency", false},
		{
			"example.com/larsartmann/go-cqrs-lite/idempotency",
			"go-cqrs-lite/idempotency",
			false,
		},
	}

	for _, tt := range tests {
		if got := fragmentMatchesImport(tt.path, tt.fragment); got != tt.want {
			t.Errorf("fragmentMatchesImport(%q, %q) = %v, want %v",
				tt.path, tt.fragment, got, tt.want)
		}
	}
}

// TestLongestDeletedModuleMatch pins specific-first resolution: a path that
// matches both a deleted ancestor and a more specific deleted module resolves
// to the specific one, and a still-present submodule suppresses the ancestor.
func TestLongestDeletedModuleMatch(t *testing.T) {
	t.Parallel()

	tests := []struct {
		path     string
		wantFrag string
		wantOK   bool
	}{
		{
			"github.com/larsartmann/go-cqrs-lite/idempotency/kvstore/v4",
			"go-cqrs-lite/idempotency/kvstore",
			true,
		},
		{"github.com/larsartmann/go-cqrs-lite/idempotency/v4", "go-cqrs-lite/idempotency", true},
		{"github.com/larsartmann/go-cqrs-lite/idempotency/sqlstore/v4", "", false},
		{"github.com/larsartmann/go-idempotency", "", false},
	}

	for _, tt := range tests {
		got, ok := longestDeletedModuleMatch(tt.path)
		if ok != tt.wantOK || got.fragment != tt.wantFrag {
			t.Errorf("longestDeletedModuleMatch(%q) = (%q, %v), want (%q, %v)",
				tt.path, got.fragment, ok, tt.wantFrag, tt.wantOK)
		}
	}
}
