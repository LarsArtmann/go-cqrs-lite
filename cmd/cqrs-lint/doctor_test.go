package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

func TestCountSuppressions_DetectsInlineComments(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

func foo() {
	//cqrs-lint:ignore(C007) domain clock
	now := time.Now()
	_ = now
}

func bar() {
	//cqrs-lint:ignore(C007) another suppression
	_ = time.Now()
	//cqrs-lint:ignore(A001) orphaned command
}
`,
	})

	counts := countSuppressions(ctx)

	if counts["C007"] != 2 {
		t.Errorf("C007 count = %d, want 2", counts["C007"])
	}
	if counts["A001"] != 1 {
		t.Errorf("A001 count = %d, want 1", counts["A001"])
	}
}

func TestCountSuppressions_NoComments(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

func main() {}
`,
	})

	counts := countSuppressions(ctx)
	if len(counts) != 0 {
		t.Errorf("expected 0 suppressions, got %d: %v", len(counts), counts)
	}
}

func TestFindParentConfigs_NoParents(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	parents := findParentConfigs(tmpDir)
	if len(parents) != 0 {
		t.Errorf("expected 0 parent configs in temp dir, got %d: %v", len(parents), parents)
	}
}

func TestFindParentConfigs_FindsAncestorConfig(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	child := filepath.Join(root, "module-a", "sub-pkg")
	if err := os.MkdirAll(child, 0o755); err != nil {
		t.Fatal(err)
	}

	parentConfig := filepath.Join(root, ".cqrs-lint.json")
	if err := os.WriteFile(parentConfig, []byte(`{"preset":"library"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	parents := findParentConfigs(child)
	if len(parents) != 1 {
		t.Fatalf("expected 1 parent config, got %d: %v", len(parents), parents)
	}
	if parents[0] != parentConfig {
		t.Errorf("expected parent config %s, got %s", parentConfig, parents[0])
	}
}

func TestFormatConfigFeatures(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		features analyzer.ConfigFeatures
		want     string
	}{
		{
			name:     "empty",
			features: analyzer.ConfigFeatures{},
			want:     "",
		},
		{
			name:     "server only",
			features: analyzer.ConfigFeatures{Server: ptrBool(true)},
			want:     "server=true",
		},
		{
			name: "multiple features",
			features: analyzer.ConfigFeatures{
				Server:  ptrBool(true),
				Tracing: ptrTracingKind(analyzer.TracingOn),
			},
			want: "server=true, tracing=on",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := formatConfigFeatures(tt.features)
			if got != tt.want {
				t.Errorf("formatConfigFeatures() = %q, want %q", got, tt.want)
			}
		})
	}
}

func ptrBool(b bool) *bool {
	return &b
}

func ptrTracingKind(k analyzer.TracingKind) *analyzer.TracingKind {
	return &k
}
