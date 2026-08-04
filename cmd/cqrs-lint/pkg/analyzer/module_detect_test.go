package analyzer

import (
	"go/parser"
	"go/token"
	"strings"
	"testing"

	"golang.org/x/tools/go/packages"
)

// parseTestFile creates a minimal GoFile for detection testing.
func parseTestFile(t *testing.T, fset *token.FileSet, filename, content string) *GoFile {
	t.Helper()
	file, err := parser.ParseFile(fset, filename, content, parser.ParseComments)
	if err != nil {
		t.Fatalf("parse %s: %v", filename, err)
	}
	return &GoFile{
		Path: filename,
		AST:  file,
		Pkg: &packages.Package{
			PkgPath: "test.example/" + strings.TrimSuffix(filename, ".go"),
		},
	}
}

func TestDetectUsedModules_SomeImported(t *testing.T) {
	t.Parallel()

	src := `package main

import (
	"github.com/larsartmann/go-cqrs-lite/otel"
	"github.com/larsartmann/go-cqrs-lite/scheduling"
	"github.com/larsartmann/go-cqrs-lite/encryption"
)

func main() {}
`
	fset := token.NewFileSet()
	gf := parseTestFile(t, fset, "main.go", src)

	usage := DetectUsedModules(nil, []*GoFile{gf}, DefaultCatalog)

	checkStatus(t, usage, "otel", UsageImported)
	checkStatus(t, usage, "scheduling", UsageImported)
	checkStatus(t, usage, "encryption", UsageImported)
	checkStatus(t, usage, "signing", UsageAbsent)
	checkStatus(t, usage, "kv", UsageAbsent)
}

func TestDetectUsedModules_NoneImported(t *testing.T) {
	t.Parallel()

	src := `package main

import (
	"fmt"
)

func main() { fmt.Println("hello") }
`
	fset := token.NewFileSet()
	gf := parseTestFile(t, fset, "main.go", src)

	usage := DetectUsedModules(nil, []*GoFile{gf}, DefaultCatalog)

	// All scored modules should be absent.
	for _, e := range DefaultCatalog.Scored() {
		checkStatus(t, usage, string(e.Key), UsageAbsent)
	}
}

func TestDetectUsedModules_PathBoundaryIDvsIdempotency(t *testing.T) {
	t.Parallel()

	// Import idempotency — must NOT trigger the "id" module match.
	src := `package main

import (
	"github.com/larsartmann/go-cqrs-lite/idempotency"
)

func main() {}
`
	fset := token.NewFileSet()
	gf := parseTestFile(t, fset, "main.go", src)

	usage := DetectUsedModules(nil, []*GoFile{gf}, DefaultCatalog)

	checkStatus(t, usage, "idempotency", UsageImported)
	// "id" is a core module so not in the scored set — it won't appear in usage.
	// But let's verify idempotency didn't accidentally also match something else.
	for key, u := range usage {
		if key != "idempotency" && u.Status != UsageAbsent {
			t.Errorf("idempotency import should not match module %s (got status %s)",
				key, u.Status)
		}
	}
}

func TestDetectUsedModules_StackPaths(t *testing.T) {
	t.Parallel()

	src := `package main

import (
	sqlite "github.com/larsartmann/go-cqrs-lite/stack/sqlite"
)

func main() {}
`
	fset := token.NewFileSet()
	gf := parseTestFile(t, fset, "main.go", src)

	usage := DetectUsedModules(nil, []*GoFile{gf}, DefaultCatalog)

	checkStatus(t, usage, "stack/sqlite", UsageImported)
	checkStatus(t, usage, "stack/postgres", UsageAbsent)
	checkStatus(t, usage, "stack/pebble", UsageAbsent)
}

func TestDetectUsedModules_TransportPaths(t *testing.T) {
	t.Parallel()

	src := `package main

import (
	"github.com/larsartmann/go-cqrs-lite/transport/http"
	"github.com/larsartmann/go-cqrs-lite/transport/grpc"
)

func main() {}
`
	fset := token.NewFileSet()
	gf := parseTestFile(t, fset, "main.go", src)

	usage := DetectUsedModules(nil, []*GoFile{gf}, DefaultCatalog)

	checkStatus(t, usage, "transport/http", UsageImported)
	checkStatus(t, usage, "transport/grpc", UsageImported)
}

func TestDetectUsedModules_MultiFileUnion(t *testing.T) {
	t.Parallel()

	src1 := `package main
import "github.com/larsartmann/go-cqrs-lite/otel"
func init() {}
`
	src2 := `package main
import "github.com/larsartmann/go-cqrs-lite/signing"
func init() {}
`
	fset := token.NewFileSet()
	gf1 := parseTestFile(t, fset, "a.go", src1)
	gf2 := parseTestFile(t, fset, "b.go", src2)

	usage := DetectUsedModules(nil, []*GoFile{gf1, gf2}, DefaultCatalog)

	checkStatus(t, usage, "otel", UsageImported)
	checkStatus(t, usage, "signing", UsageImported)
}

func TestPathBoundaryMatch(t *testing.T) {
	t.Parallel()

	tests := []struct {
		path, hint string
		want       bool
	}{
		{"github.com/larsartmann/go-cqrs-lite/otel", "go-cqrs-lite/otel", true},
		{"github.com/larsartmann/go-cqrs-lite/otel/v4", "go-cqrs-lite/otel", true},
		{"github.com/larsartmann/go-cqrs-lite/idempotency", "go-cqrs-lite/id", false},
		{"github.com/larsartmann/go-cqrs-lite/id", "go-cqrs-lite/id", true},
		{"github.com/larsartmann/go-cqrs-lite/id/v4", "go-cqrs-lite/id", true},
		{"github.com/larsartmann/go-cqrs-lite/stack/sqlite", "go-cqrs-lite/stack/sqlite", true},
		{"github.com/larsartmann/go-cqrs-lite/stack", "go-cqrs-lite/stack/sqlite", false},
		{"some-other-lib/go-cqrs-lite/otel", "go-cqrs-lite/otel", true},
		{"no-match-here", "go-cqrs-lite/otel", false},
	}

	for _, tt := range tests {
		got := pathBoundaryMatch(tt.path, tt.hint)
		if got != tt.want {
			t.Errorf("pathBoundaryMatch(%q, %q) = %v, want %v", tt.path, tt.hint, got, tt.want)
		}
	}
}

func TestUsageStatusString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		status UsageStatus
		want   string
	}{
		{UsageAbsent, "missing"},
		{UsageImported, "used"},
		{UsageActive, "active"},
	}

	for _, tt := range tests {
		if got := tt.status.String(); got != tt.want {
			t.Errorf("UsageStatus(%d).String() = %q, want %q", tt.status, got, tt.want)
		}
	}
}

// checkStatus asserts that the usage map has the expected status for a module.
func checkStatus(t *testing.T, usage map[ModuleKey]ModuleUsage, key string, want UsageStatus) {
	t.Helper()
	u, ok := usage[ModuleKey(key)]
	if !ok {
		t.Fatalf("module %q not in usage map", key)
	}
	if u.Status != want {
		t.Errorf("module %q: expected status %s, got %s (evidence: %q)",
			key, want, u.Status, u.Evidence)
	}
}
