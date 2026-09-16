package main

import (
	"strings"
	"testing"
)

// arityRepo builds fixture packages mirroring real shapes: system.New takes
// exactly 3 required args (the shape of the audit's critical doc lie), and
// catalog.NewRegistry takes 2.
func arityRepo(t *testing.T) string {
	t.Helper()

	root := t.TempDir()

	writePkg(t, root+"/system", "system",
		"type DomainConfig struct{}\n\n"+
			"type DeploymentConfig struct{}\n\n"+
			"type System struct{}\n\n"+
			"func New(ctx context, domain DomainConfig, deployment DeploymentConfig) (*System, error)\n")
	writePkg(t, root+"/catalog", "catalog",
		"type Registry struct{}\n\n"+
			"func NewRegistry(title, version string) *Registry\n")

	return root
}

func TestCheckArity_MismatchShapes(t *testing.T) {
	t.Parallel()

	root := arityRepo(t)
	res := newResolver(root)

	cases := []struct {
		name    string
		src     string
		wantSub string // empty = expect no issue
	}{
		{
			name:    "the audit's critical lie: too few args",
			src:     "sys, err := system.New(ctx, system.DeploymentConfig{})",
			wantSub: "arity: system.New called with 2 arg(s)",
		},
		{
			name:    "too many args",
			src:     "r := catalog.NewRegistry(\"t\", \"v\", \"extra\")",
			wantSub: "arity: catalog.NewRegistry called with 3 arg(s)",
		},
		{
			name:    "placeholder ellipsis counts as one arg",
			src:     "sys, err := system.New(ctx, ...)",
			wantSub: "arity: system.New called with 2 arg(s)",
		},
		{
			name:    "spread call on non-variadic",
			src:     "sys, err := system.New(ctx, d, cfg...)",
			wantSub: "arity: system.New(...)",
		},
		{
			name:    "correct call passes",
			src:     "sys, err := system.New(ctx, domain, deployment)",
			wantSub: "",
		},
		{
			name:    "wrong-example marker is skipped",
			src:     "// Wrong\nsys, err := system.New(ctx)",
			wantSub: "",
		},
		{
			name:    "comment-only args are skipped",
			src:     "sys, err := system.New(/* ctx, domain, deployment */)",
			wantSub: "",
		},
		{
			name:    "ignore directive opts the block out",
			src:     "// doc-check:ignore-arity\nsys, err := system.New()",
			wantSub: "",
		},
		{
			name:    "generic instantiation is unwrapped",
			src:     "r := catalog.NewRegistry[string](\"t\", \"v\")",
			wantSub: "",
		},
		{
			name:    "parse-garbage fragment is skipped",
			src:     "func broken( {{{ system.New(",
			wantSub: "",
		},
	}

	for _, tc := range cases {
		blocks := []block{
			{file: "doc.md", line: 7, src: tc.src},
		} //nolint:exhaustruct_v5 // single-purpose fixture

		issues := checkArity(blocks, res)

		if tc.wantSub == "" {
			if len(issues) != 0 {
				t.Errorf("%s: unexpected issues %+v", tc.name, issues)
			}

			continue
		}

		if len(issues) != 1 || !strings.Contains(issues[0].Msg, tc.wantSub) {
			t.Errorf("%s: want issue containing %q, got %+v", tc.name, tc.wantSub, issues)
		}

		if len(issues) == 1 && issues[0].Line != 7 {
			t.Errorf("%s: line mapping broken: got %d, want 7", tc.name, issues[0].Line)
		}
	}
}

func TestCheckArity_MethodsAndAmbiguousPackagesSkipped(t *testing.T) {
	t.Parallel()

	root := arityRepo(t)

	writePkg(t, root+"/a/sqlstore", "sqlstore", "type A struct{}\n\nfunc NewA() *A\n")
	writePkg(t, root+"/b/sqlstore", "sqlstore", "type B struct{}\n\nfunc NewB(x int) *B\n")

	res := newResolver(root)

	// NewB exists with arity 1 in ONE of the two sqlstore packages — an
	// ambiguous alias must never be checked (wrong-package hits impossible).
	src := "b := sqlstore.NewB()\n" +
		"sys := system.New\n" +
		"deployment{}.Payload()\n"

	blocks := []block{
		{file: "doc.md", line: 1, src: src},
	} //nolint:exhaustruct_v5 // single-purpose fixture

	if issues := checkArity(blocks, res); len(issues) != 0 {
		t.Fatalf("ambiguous/unqualified/method shapes must be skipped, got %+v", issues)
	}
}

func TestCheckArity_BlockImportScopesTheCheck(t *testing.T) {
	t.Parallel()

	root := arityRepo(t)
	res := newResolver(root)

	imp := "github.com/larsartmann/go-cqrs-lite/system/v4"

	if warns := res.warm(imp); len(warns) != 0 {
		t.Fatalf("warm warnings: %v", warns)
	}

	b := block{
		file:    "doc.md",
		line:    3,
		src:     "import \"github.com/larsartmann/go-cqrs-lite/system/v4\"\n\nsys, err := system.New(ctx)\n",
		imports: []string{imp},
	}

	issues := checkArity([]block{b}, res)
	if len(issues) != 1 || !strings.Contains(issues[0].Msg, "system.New called with 1 arg(s)") {
		t.Fatalf("block-scoped import must scope the arity check, got %+v", issues)
	}
}

func TestNormalizePlaceholders(t *testing.T) {
	t.Parallel()

	cases := []struct{ in, want string }{
		{"f(a, ...)", "f(a, " + docPlaceholder + ")"},
		{"f(...)", "f(" + docPlaceholder + ")"},
		{"f(a, …)", "f(a, " + docPlaceholder + ")"},
		{"User{Name: \"x\", ...}", "User{Name: \"x\", " + docPlaceholder + "}"},
		{"slice(args...)", "slice(args...)"},
		{"f(a)", "f(a)"},
	}

	for _, tc := range cases {
		if got := normalizePlaceholders(tc.in); got != tc.want {
			t.Errorf("normalizePlaceholders(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
