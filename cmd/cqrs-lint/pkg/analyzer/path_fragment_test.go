package analyzer

import "testing"

// Import-path fragment matching has to survive Go major-version elements. The
// linter addresses packages by version-less fragment ("…/catalog/simple") while
// consumers import "…/catalog/v4/simple"; a plain substring test silently
// missed every versioned module, so the catalog/simple scanner case and F002's
// builder detection were dead in real projects while version-less fixtures
// passed (issue #79).
func TestPathContainsFragment_VersionTolerant(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		path     string
		fragment string
		want     bool
	}{
		{
			name:     "versionless fragment against versioned subpackage",
			path:     "github.com/larsartmann/go-cqrs-lite/catalog/v4/simple",
			fragment: "go-cqrs-lite/catalog/simple",
			want:     true,
		},
		{
			name:     "versionless fragment against versioned module root",
			path:     "github.com/larsartmann/go-cqrs-lite/event/v4",
			fragment: "go-cqrs-lite/event",
			want:     true,
		},
		{
			name:     "versionless fragment against versionless path",
			path:     "github.com/larsartmann/go-cqrs-lite/catalog/simple",
			fragment: "go-cqrs-lite/catalog/simple",
			want:     true,
		},
		{
			name:     "unrelated package stays unmatched",
			path:     "github.com/larsartmann/go-cqrs-lite/catalog/v4",
			fragment: "go-cqrs-lite/schema",
			want:     false,
		},
		{
			name:     "sibling subpackage of the same module stays unmatched",
			path:     "github.com/larsartmann/go-cqrs-lite/catalog/v4/docserver",
			fragment: "go-cqrs-lite/catalog/simple",
			want:     false,
		},
		{
			name:     "v-prefixed non-numeric segment is preserved",
			path:     "example.com/vendor/main",
			fragment: "example.com/vendor/main",
			want:     true,
		},
		{
			name:     "v-prefixed non-numeric segment is not stripped",
			path:     "example.com/vendor/main",
			fragment: "example.com/main",
			want:     false,
		},
		{
			name:     "multi-digit major version is stripped",
			path:     "example.com/lib/v12/inner",
			fragment: "example.com/lib/inner",
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := pathContainsFragment(tt.path, tt.fragment); got != tt.want {
				t.Errorf("pathContainsFragment(%q, %q) = %v, want %v",
					tt.path, tt.fragment, got, tt.want)
			}
		})
	}
}
