package main

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

const commandImport = "import \"github.com/larsartmann/go-cqrs-lite/command/v4\""

func writeFixture(t *testing.T, root, rel, content string) {
	t.Helper()

	path := filepath.Join(root, rel)

	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func mustKinds(t *testing.T, value string) map[kind]bool {
	t.Helper()

	kinds, err := parseKinds(value)
	if err != nil {
		t.Fatalf("parseKinds(%q): %v", value, err)
	}

	return kinds
}

func summarize(pkgs []pkg) []string {
	lines := make([]string, 0, 8)

	for _, p := range pkgs {
		for _, m := range p.markers {
			line := fmt.Sprintf("%s %s %s %s receives=%v const=%s",
				filepath.Base(p.dir), m.kind, m.structName, m.wire, m.receives, m.constName)
			if m.constExpr != "" {
				line += " emits=" + m.constExpr
			}

			lines = append(lines, line)
		}
	}

	return lines
}

func assertSummary(t *testing.T, pkgs []pkg, want ...string) {
	t.Helper()

	if got := summarize(pkgs); !slices.Equal(got, want) {
		t.Errorf("markers:\n got %q\nwant %q", got, want)
	}
}

func TestScan_PerPackageFromDefaultRoot(t *testing.T) {
	root := t.TempDir()

	writeFixture(t, root, "tasks/commands.go", "package tasks\n\n"+commandImport+"\n\n"+
		"//cqrs:command task.create\ntype CreateTaskCmd struct {\n\t*command.BasicCommand\n}\n")
	writeFixture(t, root, "tasks/queries.go", "package tasks\n\n"+
		"type GetTask struct {\n\t_ struct{} `cqrs:\"query:task.get\"`\n}\n")
	writeFixture(t, root, "billing/events.go", "package billing\n\n"+
		"//cqrs:event invoice.issued receives\ntype InvoiceIssued struct{ Ref string }\n")
	writeFixture(t, root, "tasks/.hidden/h.go", "package tasks\n\n"+
		"//cqrs:command hidden.cmd\ntype HiddenCmd struct{}\n")
	writeFixture(t, root, "tasks/vendor/v/v.go", "package v\n\n"+
		"//cqrs:command vendor.cmd\ntype VendorCmd struct{}\n")
	writeFixture(t, root, "tasks/extra_test.go", "package tasks\n\n"+
		"//cqrs:command test.only\ntype TestOnlyCmd struct{}\n")
	writeFixture(t, root, "tasks/tagged.go", "//go:build ignore\n\npackage tasks\n\n"+
		"//cqrs:command tagged.cmd\ntype TaggedCmd struct{}\n")
	writeFixture(t, root, "empty/README.md", "no Go in here")

	t.Chdir(root)

	pkgs, err := scan([]string{"."}, mustKinds(t, kindAll))
	if err != nil {
		t.Fatalf("scan(.): %v", err)
	}

	if len(pkgs) != 2 {
		t.Fatalf("expected the two marking packages, got %d: %v", len(pkgs), summarize(pkgs))
	}

	if pkgs[0].name != "billing" || pkgs[1].name != "tasks" {
		t.Errorf("package names = %q, %q; directories walk in sorted order", pkgs[0].name, pkgs[1].name)
	}

	assertSummary(t, pkgs,
		"billing event InvoiceIssued invoice.issued receives=true const=EventInvoiceIssued emits=\"invoice.issued\"",
		"tasks command CreateTaskCmd task.create receives=false const=CommandCreateTask emits=\"task.create\"",
		"tasks query GetTask task.get receives=false const=QueryGetTask emits=\"task.get\"",
	)
}

func TestScan_PathForms(t *testing.T) {
	root := t.TempDir()

	writeFixture(t, root, "tasks/commands.go", "package tasks\n\n"+
		"//cqrs:command task.create\ntype CreateTaskCmd struct{}\n")
	writeFixture(t, root, "billing/events.go", "package billing\n\n"+
		"//cqrs:event invoice.issued\ntype InvoiceIssued struct{}\n")

	t.Chdir(root)

	tests := []struct {
		paths []string
		want  []string
	}{
		{paths: []string{"./..."}, want: []string{"billing", "tasks"}},
		{paths: []string{"tasks/..."}, want: []string{"tasks"}},
		{paths: []string{filepath.Join("tasks", "commands.go")}, want: []string{"tasks"}},
		{paths: []string{"tasks", "billing"}, want: []string{"tasks", "billing"}},
	}

	for _, tt := range tests {
		pkgs, err := scan(tt.paths, mustKinds(t, kindAll))
		if err != nil {
			t.Fatalf("scan(%q): %v", tt.paths, err)
		}

		dirs := make([]string, 0, len(pkgs))

		for _, p := range pkgs {
			dirs = append(dirs, filepath.Base(p.dir))
		}

		if !slices.Equal(dirs, tt.want) {
			t.Errorf("scan(%q) = %q, want %q", tt.paths, dirs, tt.want)
		}
	}
}

func TestScan_WireNameResolution(t *testing.T) {
	root := t.TempDir()

	writeFixture(t, root, "names/names.go", `package names

const (
	cmdCreateTask = command.Type("task.create")
	qryOpenTasks  = "task.open"
)

//cqrs:command task.create
type CreateTaskCmd struct{}

//cqrs:query qryOpenTasks
type OpenTasks struct{}

//cqrs:query task.closed
type ClosedTasks struct{}
`)

	pkgs, err := scan([]string{root}, mustKinds(t, kindAll))
	if err != nil {
		t.Fatalf("scan: %v", err)
	}

	assertSummary(t, pkgs,
		"names command CreateTaskCmd task.create receives=false const=cmdCreateTask",
		"names query ClosedTasks task.closed receives=false const=QueryClosedTasks emits=\"task.closed\"",
		"names query OpenTasks qryOpenTasks receives=false const=qryOpenTasks",
	)
}

func TestScan_UnselectedKindsAreIgnored(t *testing.T) {
	root := t.TempDir()

	writeFixture(t, root, "domain/domain.go", "package domain\n\n"+
		"//cqrs:event\ntype TaskCreated struct{}\n")

	pkgs, err := scan([]string{root}, mustKinds(t, string(kindCommand)))
	if err != nil {
		t.Fatalf("scan with --type=command must not parse event markers: %v", err)
	}

	if len(pkgs) != 0 {
		t.Errorf("expected no package, got %v", summarize(pkgs))
	}
}

func TestScan_CommentMarkerWinsOverTag(t *testing.T) {
	root := t.TempDir()

	writeFixture(t, root, "domain/domain.go", "package domain\n\n"+
		"//cqrs:command task.create\ntype Task struct {\n\t_ struct{} `cqrs:\"query:task.get\"`\n}\n")

	pkgs, err := scan([]string{root}, mustKinds(t, kindAll))
	if err != nil {
		t.Fatalf("scan: %v", err)
	}

	assertSummary(t, pkgs,
		"domain command Task task.create receives=false const=CommandTask emits=\"task.create\"")
}

func TestScan_Rejects(t *testing.T) {
	tests := []struct {
		name    string
		sources map[string]string
		paths   []string
		wantErr string
	}{
		{
			name:    "marker without a type string",
			sources: map[string]string{"a.go": "package p\n\n//cqrs:command\ntype A struct{}"},
			wantErr: "carries no type string",
		},
		{
			name:    "marker on a non-struct type",
			sources: map[string]string{"a.go": "package p\n\n//cqrs:command a.b\ntype A int"},
			wantErr: "not a plain struct type declaration",
		},
		{
			name: "marker on a generic struct",
			sources: map[string]string{
				"a.go": "package p\n\n//cqrs:command a.b\ntype A[T any] struct{ Value T }",
			},
			wantErr: "not a plain struct type declaration",
		},
		{
			name:    "marker on a type alias",
			sources: map[string]string{"a.go": "package p\n\n//cqrs:command a.b\ntype A = struct{ Value int }"},
			wantErr: "not a plain struct type declaration",
		},
		{
			name: "two structs claim one wire name",
			sources: map[string]string{
				"a.go": "package p\n\n//cqrs:command a.b\ntype A struct{}",
				"b.go": "package p\n\n//cqrs:command a.b\ntype B struct{}",
			},
			wantErr: `A and B both claim the command type string "a.b"`,
		},
		{
			name: "generated constant collides with a function",
			sources: map[string]string{
				"a.go": "package p\n\n//cqrs:command a.b\ntype CreateTaskCmd struct{}\n\nfunc CommandCreateTask() {}",
			},
			wantErr: "generated constant CommandCreateTask collides",
		},
		{
			name: "generated function collides with an existing one",
			sources: map[string]string{
				"a.go": "package p\n\n//cqrs:command a.b\ntype CreateTaskCmd struct{}\n\nfunc RegisterCreateTaskCmd() {}",
			},
			wantErr: "generated declaration RegisterCreateTaskCmd collides",
		},
		{
			name:    "unknown event option",
			sources: map[string]string{"a.go": "package p\n\n//cqrs:event a.b maybe\ntype A struct{}"},
			wantErr: `takes "sends" or "receives", got "maybe"`,
		},
		{
			name:    "option on a command marker",
			sources: map[string]string{"a.go": "package p\n\n//cqrs:command a.b sends\ntype A struct{}"},
			wantErr: `the cqrs:command marker takes no options, got "sends"`,
		},
		{
			name: "two packages in one directory",
			sources: map[string]string{
				"a.go": "package p\n\n//cqrs:command a.b\ntype A struct{}",
				"b.go": "package q\n\ntype B struct{}",
			},
			wantErr: "mixes package \"p\" with other packages",
		},
		{
			name:    "missing scan path",
			sources: map[string]string{},
			paths:   []string{"does-not-exist"},
			wantErr: "scan path does-not-exist",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()

			for name, content := range tt.sources {
				writeFixture(t, root, "domain/"+name, content)
			}

			paths := tt.paths
			if len(paths) == 0 {
				paths = []string{root}
			}

			_, err := scan(paths, mustKinds(t, kindAll))
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("scan() error = %v, want it to contain %q", err, tt.wantErr)
			}
		})
	}
}

func TestParseKinds(t *testing.T) {
	if kinds, err := parseKinds(kindAll); err != nil || len(kinds) != 3 {
		t.Errorf("parseKinds(all) = %v, %v; want all three kinds", kinds, err)
	}

	if kinds, err := parseKinds(string(kindQuery)); err != nil || len(kinds) != 1 || !kinds[kindQuery] {
		t.Errorf("parseKinds(query) = %v, %v; want only query", kinds, err)
	}

	if kinds, err := parseKinds("command,query"); err != nil || len(kinds) != 2 ||
		!kinds[kindCommand] || !kinds[kindQuery] {
		t.Errorf("parseKinds(command,query) = %v, %v; want exactly those two kinds", kinds, err)
	}

	if _, err := parseKinds("bogus"); err == nil {
		t.Error("parseKinds(bogus) must fail")
	}

	if _, err := parseKinds("command,bogus"); err == nil {
		t.Error("parseKinds(command,bogus) must fail")
	}

	if _, err := parseTarget("bogus"); err == nil {
		t.Error("parseTarget(bogus) must fail")
	}
}

func TestConstantName(t *testing.T) {
	tests := map[kind]map[string]string{
		kindCommand: {
			"CreateTaskCmd": "CommandCreateTask", "CreateUserCommand": "CommandCreateUser", "Cmd": "CommandCmd",
		},
		kindQuery: {"GetTask": "QueryGetTask", "GetUserQuery": "QueryGetUser"},
		kindEvent: {
			"TaskCreated": "EventTaskCreated", "UserCreatedPayload": "EventUserCreated",
			"UserCreatedEvent": "EventUserCreated",
		},
	}

	for k, cases := range tests {
		for structName, want := range cases {
			if got := constantName(k, structName); got != want {
				t.Errorf("constantName(%s, %s) = %s, want %s", k, structName, got, want)
			}
		}
	}
}
