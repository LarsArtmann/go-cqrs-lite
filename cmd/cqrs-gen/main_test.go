package main

import (
	"bytes"
	"go/format"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScanFile_CommandMarker(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	src := filepath.Join(tmp, "commands.go")

	content := `package example

//cqrs:command CreateUser
type CreateUserCmd struct {
	Name string
}

//cqrs:command DeleteUser
type DeleteUserCmd struct {
	ID string
}

// Not marked — should be ignored
type Unmarked struct{}
`
	if err := os.WriteFile(src, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	entries, err := scanFile(src, "command")
	if err != nil {
		t.Fatalf("scan: %v", err)
	}

	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}

	assertEntry(t, entries[0], "0", "CreateUser", "CreateUserCmd")
	assertEntry(t, entries[1], "1", "DeleteUser", "DeleteUserCmd")
}

func assertEntry(t *testing.T, entry Entry, idx, cmdType, structName string) {
	t.Helper()

	if entry.TypeName != cmdType || entry.StructName != structName {
		t.Errorf("entry[%s] = %+v", idx, entry)
	}
}

func TestScanFile_QueryMarker(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	src := filepath.Join(tmp, "queries.go")

	content := `package example

//cqrs:query GetUser
type GetUserQuery struct {
	ID string
}
`
	if err := os.WriteFile(src, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	entries, err := scanFile(src, "query")
	if err != nil {
		t.Fatalf("scan: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	assertEntry(t, entries[0], "", "GetUser", "GetUserQuery")
}

func TestGenerate_Command(t *testing.T) {
	t.Parallel()

	entries := []Entry{
		{TypeName: "CreateUser", StructName: "CreateUserCmd"},
	}

	code, err := generate("handlers", "command", entries)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(code, "package handlers") {
		t.Error("missing package declaration")
	}

	if !strings.Contains(code, "func RegisterCreateUserCmdHandler") {
		t.Error("missing handler function")
	}

	if !strings.Contains(code, `"CreateUser"`) {
		t.Error("missing command type")
	}
}

func TestGenerate_Query(t *testing.T) {
	t.Parallel()

	entries := []Entry{
		{TypeName: "GetUser", StructName: "GetUserQuery"},
	}

	code, err := generate("handlers", "query", entries)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(code, "github.com/larsartmann/go-cqrs-lite/query/v4") {
		t.Error("missing query import")
	}

	if !strings.Contains(code, "func RegisterGetUserQueryHandler") {
		t.Error("missing handler function")
	}

	if !strings.Contains(code, "[R any]") {
		t.Error("missing type parameter [R any]")
	}

	if !strings.Contains(code, "*GetUserQuery") {
		t.Error("handler should accept typed query *GetUserQuery")
	}

	if !strings.Contains(code, "query.RegisterTyped[*GetUserQuery, R]") {
		t.Error("should call query.RegisterTyped[*GetUserQuery, R]")
	}
}

func TestGenerate_ValidGoSyntax(t *testing.T) {
	t.Parallel()

	for _, handlerType := range []string{genTypeCommand, genTypeQuery, genTypeEvent} {
		entries := []Entry{
			{TypeName: "CreateUser", StructName: "CreateUserCmd", PackageName: "example"},
		}
		code, err := generate("example", handlerType, entries)
		if err != nil {
			t.Errorf("generate(%q) returned error: %v", handlerType, err)
			continue
		}
		// generate already runs format.Source, but re-checking here pins the
		// contract: the emitted file must stay valid, parseable Go.
		if _, err := format.Source([]byte(code)); err != nil {
			t.Errorf("generated %q code is not valid Go: %v\n%s", handlerType, err, code)
		}
	}
}

func TestScanPath_SkipsVendorAndHiddenDirs(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()

	vendor := filepath.Join(tmp, "vendor")
	if err := os.Mkdir(vendor, 0o750); err != nil {
		t.Fatal(err)
	}
	writeTempGoFile(
		t,
		vendor,
		"v.go",
		"package vendor\n\n//cqrs:command VendorCmd\ntype VendorCmd struct{}\n",
	)

	hidden := filepath.Join(tmp, ".hidden")
	if err := os.Mkdir(hidden, 0o750); err != nil {
		t.Fatal(err)
	}
	writeTempGoFile(
		t,
		hidden,
		"h.go",
		"package hidden\n\n//cqrs:command HiddenCmd\ntype HiddenCmd struct{}\n",
	)

	// Top-level marker must still be discovered despite the pruned subtrees.
	writeTempGoFile(
		t,
		tmp,
		"top.go",
		"package example\n\n//cqrs:command TopCmd\ntype TopCmd struct{}\n",
	)

	entries, err := scanPath(tmp, "command")
	if err != nil {
		t.Fatalf("scanPath: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("expected 1 entry (top-level only), got %d: %+v", len(entries), entries)
	}
	if entries[0].StructName != "TopCmd" {
		t.Errorf("expected TopCmd, got %s", entries[0].StructName)
	}
}

func TestDedupEntries_DropsDuplicateStructNames(t *testing.T) {
	t.Parallel()

	entries := []Entry{
		{TypeName: "CreateUser", StructName: "CreateUserCmd", PackageName: "a"},
		{TypeName: "CreateUser", StructName: "CreateUserCmd", PackageName: "b"}, // duplicate name
		{TypeName: "DeleteUser", StructName: "DeleteUserCmd", PackageName: "a"},
	}

	var buf bytes.Buffer
	deduped := dedupEntries(&buf, entries)

	if len(deduped) != 2 {
		t.Fatalf("expected 2 entries after dedup, got %d", len(deduped))
	}
	if deduped[0].StructName != "CreateUserCmd" || deduped[1].StructName != "DeleteUserCmd" {
		t.Errorf("unexpected dedup order: %+v", deduped)
	}
	if !strings.Contains(buf.String(), "duplicate struct") {
		t.Errorf("expected duplicate warning, got: %s", buf.String())
	}
}

func TestScanPath_NonExistentDir(t *testing.T) {
	t.Parallel()

	_, err := scanPath("/nonexistent/path/that/does/not/exist", "command")
	if err == nil {
		t.Fatal("expected error for non-existent path")
	}
}

func TestScanPath_InvalidGoFile(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	badFile := filepath.Join(tmp, "bad.go")
	if err := os.WriteFile(badFile, []byte("this is not valid go code {{{"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := scanPath(tmp, "command")
	if err == nil {
		t.Fatal("expected error for invalid Go file")
	}
}

func writeTempGoFile(tb testing.TB, dir, filename, content string) string {
	tb.Helper()

	path := filepath.Join(dir, filename)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		tb.Fatal(err)
	}

	return path
}

func TestScanFile_EmptyResults(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		content    string
		filename   string
		markerType string
	}{
		{
			name:       "no_markers",
			content:    "package example\n\ntype PlainStruct struct {\n\tName string\n}",
			filename:   "plain.go",
			markerType: "command",
		},
		{
			name:       "wrong_marker_type",
			content:    "package example\n\n//cqrs:command CreateUser\ntype CreateUserCmd struct{}",
			filename:   "cmd.go",
			markerType: "query",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tmp := t.TempDir()
			writeTempGoFile(t, tmp, tt.filename, tt.content)

			entries, err := scanFile(filepath.Join(tmp, tt.filename), tt.markerType)
			if err != nil {
				t.Fatalf("scan: %v", err)
			}

			if len(entries) != 0 {
				t.Errorf("expected 0 entries, got %d", len(entries))
			}
		})
	}
}

func TestScanFile_MultipleFilesInDir(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()

	cmdContent := `package example

//cqrs:command CreateUser
type CreateUserCmd struct{}`
	if err := os.WriteFile(
		filepath.Join(tmp, "commands.go"),
		[]byte(cmdContent),
		0o644,
	); err != nil {
		t.Fatal(err)
	}

	queryContent := `package example

//cqrs:query GetUser
type GetUserQuery struct{}`
	if err := os.WriteFile(
		filepath.Join(tmp, "queries.go"),
		[]byte(queryContent),
		0o644,
	); err != nil {
		t.Fatal(err)
	}

	entries, err := scanPath(tmp, "command")
	if err != nil {
		t.Fatalf("scanPath: %v", err)
	}

	if len(entries) != 1 || entries[0].StructName != "CreateUserCmd" {
		t.Errorf("expected 1 command entry, got %+v", entries)
	}
}

func TestGenerate_MultipleEntries(t *testing.T) {
	t.Parallel()

	entries := []Entry{
		{TypeName: "CreateUser", StructName: "CreateUserCmd", PackageName: "example"},
		{TypeName: "DeleteUser", StructName: "DeleteUserCmd", PackageName: "example"},
	}

	code, err := generate("handlers", "command", entries)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(code, "RegisterCreateUserCmdHandler") {
		t.Error("missing CreateUserCmd handler")
	}
	if !strings.Contains(code, "RegisterDeleteUserCmdHandler") {
		t.Error("missing DeleteUserCmd handler")
	}
}

func TestGenerate_PackageName(t *testing.T) {
	t.Parallel()

	entries := []Entry{{TypeName: "Test", StructName: "TestCmd"}}
	code, err := generate("mypkg", "command", entries)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(code, "package mypkg") {
		t.Error("expected package name 'mypkg'")
	}
}

func TestGenerate_QueryImports(t *testing.T) {
	t.Parallel()

	entries := []Entry{{TypeName: "GetUser", StructName: "GetUserQuery"}}
	code, err := generate("handlers", "query", entries)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(code, "query") {
		t.Error("missing query import")
	}
	if strings.Contains(code, "command") {
		t.Error("should not import command for query generation")
	}
}

func TestScanFile_TestFilesSkipped(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	testFile := filepath.Join(tmp, "commands_test.go")

	content := `package example

//cqrs:command TestCmd
type TestCmd struct{}`
	if err := os.WriteFile(testFile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	entries, err := scanPath(tmp, "command")
	if err != nil {
		t.Fatalf("scanPath: %v", err)
	}

	if len(entries) != 0 {
		t.Errorf("test files should be skipped, got %d entries", len(entries))
	}
}

func TestScanFile_MarkerOnTypeSpec(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	src := filepath.Join(tmp, "types.go")

	content := `package example

type (
	//cqrs:command InlineCmd
	InlineCmd struct{}
)`
	if err := os.WriteFile(src, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	entries, err := scanFile(src, "command")
	if err != nil {
		t.Fatalf("scan: %v", err)
	}

	if len(entries) != 1 || entries[0].StructName != "InlineCmd" {
		t.Errorf("expected InlineCmd, got %+v", entries)
	}
}

func TestScan_MultiplePaths(t *testing.T) {
	t.Parallel()

	tmp1 := t.TempDir()
	tmp2 := t.TempDir()

	content1 := `package pkg1

//cqrs:command CreateOrder
type CreateOrderCmd struct{}`
	if err := os.WriteFile(filepath.Join(tmp1, "order.go"), []byte(content1), 0o644); err != nil {
		t.Fatal(err)
	}

	content2 := `package pkg2

//cqrs:command CreateUser
type CreateUserCmd struct{}`
	if err := os.WriteFile(filepath.Join(tmp2, "user.go"), []byte(content2), 0o644); err != nil {
		t.Fatal(err)
	}

	entries, err := scan([]string{tmp1, tmp2}, "command")
	if err != nil {
		t.Fatalf("scan: %v", err)
	}

	if len(entries) != 2 {
		t.Errorf("expected 2 entries across 2 dirs, got %d", len(entries))
	}
}

func TestScanFile_StructTagMarker(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	src := filepath.Join(tmp, "commands.go")

	content := `package example

type CreateUserCmd struct {
	_    struct{} ` + "`" + `cqrs:"command:CreateUser"` + "`" + `
	Name string
}

type DeleteUserCmd struct {
	_  struct{} ` + "`" + `cqrs:"command:DeleteUser"` + "`" + `
	ID string
}

type PlainStruct struct {
	Name string
}`
	if err := os.WriteFile(src, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	entries, err := scanFile(src, "command")
	if err != nil {
		t.Fatalf("scan: %v", err)
	}

	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}

	assertEntry(t, entries[0], "0", "CreateUser", "CreateUserCmd")
	assertEntry(t, entries[1], "1", "DeleteUser", "DeleteUserCmd")
}

func TestScanFile_StructTagQuery(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	src := filepath.Join(tmp, "queries.go")

	content := `package example

type GetUserQuery struct {
	_ struct{} ` + "`" + `cqrs:"query:GetUser"` + "`" + `
	ID string
}`
	if err := os.WriteFile(src, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	entries, err := scanFile(src, "query")
	if err != nil {
		t.Fatalf("scan: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	assertEntry(t, entries[0], "", "GetUser", "GetUserQuery")
}

func TestScanFile_CommentOverridesStructTag(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	src := filepath.Join(tmp, "commands.go")

	// When both comment and struct tag are present, the comment wins
	content := `package example

//cqrs:command FromComment
type CreateUserCmd struct {
	_ struct{} ` + "`" + `cqrs:"command:FromTag"` + "`" + `
}`
	if err := os.WriteFile(src, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	entries, err := scanFile(src, "command")
	if err != nil {
		t.Fatalf("scan: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	if entries[0].TypeName != "FromComment" {
		t.Errorf("expected comment marker 'FromComment' to win, got %q",
			entries[0].TypeName)
	}
}

func TestRun_InvalidType(t *testing.T) {
	t.Parallel()

	code := run("invalid", "out.go", "", []string{"."})
	if code == nil {
		t.Error("expected error for invalid type, got nil")
	}
}

func TestRun_NoMarkers(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	content := `package example

type PlainStruct struct{}`
	if err := os.WriteFile(filepath.Join(tmp, "plain.go"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	code := run("command", "out.go", "", []string{tmp})
	if code != nil {
		t.Errorf("expected nil error for no markers, got %v", code)
	}
}

func TestRun_SuccessfulCommand(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	content := `package example

//cqrs:command CreateUser
type CreateUserCmd struct{}`
	writeTempGoFile(t, tmp, "commands.go", content)

	outputFile := filepath.Join(tmp, "handlers_gen.go")
	code := run("command", outputFile, "", []string{tmp})
	if code != nil {
		t.Fatalf("expected nil error, got %v", code)
	}

	generated, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("output file not created: %v", err)
	}
	if !strings.Contains(string(generated), "RegisterCreateUserCmdHandler") {
		t.Errorf("generated file missing handler, got: %s", generated)
	}
}

func TestRun_SuccessfulQueryWithCustomPkg(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	content := `package myapp

//cqrs:query GetUser
type GetUserQuery struct{}`
	writeTempGoFile(t, tmp, "queries.go", content)

	outputFile := filepath.Join(tmp, "queries_gen.go")
	code := run("query", outputFile, "handlers", []string{tmp})
	if code != nil {
		t.Fatalf("expected nil error, got %v", code)
	}

	generated, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("output file not created: %v", err)
	}
	if !strings.Contains(string(generated), "package handlers") {
		t.Errorf("expected custom package name, got: %s", generated)
	}
	if !strings.Contains(string(generated), "query") {
		t.Errorf("expected query import, got: %s", generated)
	}
}

func TestRun_WriteError(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	content := `package example

//cqrs:command CreateUser
type CreateUserCmd struct{}`
	writeTempGoFile(t, tmp, "commands.go", content)

	code := run("command", "/nonexistent/dir/handlers_gen.go", "", []string{tmp})
	if code == nil {
		t.Error("expected error for write error, got nil")
	}
}

func TestRun_DefaultPath(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	content := `package example

//cqrs:command CreateItem
type CreateItemCmd struct{}`
	writeTempGoFile(t, tmp, "item.go", content)

	outputFile := filepath.Join(tmp, "gen.go")
	code := run("command", outputFile, "", []string{tmp})
	if code != nil {
		t.Fatalf("expected nil error, got %v", code)
	}

	generated, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("output file not created: %v", err)
	}
	if !strings.Contains(string(generated), "RegisterCreateItemCmdHandler") {
		t.Errorf("generated file missing handler, got: %s", generated)
	}
}

func TestGenerate_Event(t *testing.T) {
	t.Parallel()

	entries := []Entry{
		{TypeName: "UserCreated", StructName: "UserCreatedPayload"},
	}

	code, err := generate("handlers", "event", entries)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(code, "github.com/larsartmann/go-cqrs-lite/event/v4") {
		t.Error("missing event import")
	}

	if !strings.Contains(code, "github.com/larsartmann/go-cqrs-lite/codec/v4") {
		t.Error("missing codec import")
	}

	if !strings.Contains(code, "func RegisterUserCreatedPayloadHandler") {
		t.Error("missing handler function")
	}

	if !strings.Contains(code, "func(context.Context, UserCreatedPayload) error") {
		t.Error("handler should accept typed payload")
	}

	if !strings.Contains(code, "bus.Subscribe(event.Type") {
		t.Error("should call bus.Subscribe with event type")
	}

	if !strings.Contains(code, `event.Type("UserCreated")`) {
		t.Error("missing event type string")
	}
}

func TestRun_SuccessfulEvent(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	content := "package example\n\n//cqrs:event UserCreated\ntype UserCreatedPayload struct{}\n"
	writeTempGoFile(t, tmp, "event.go", content)

	outputFile := filepath.Join(tmp, "gen.go")
	code := run("event", outputFile, "", []string{tmp})
	if code != nil {
		t.Errorf("expected nil error, got %v", code)
	}

	generated, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("output file not created: %v", err)
	}

	if !strings.Contains(string(generated), "RegisterUserCreatedPayloadHandler") {
		t.Errorf("generated file missing handler, got: %s", generated)
	}
}

func TestScanFile_EventStructTag(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	content := "package example\n\ntype UserCreatedPayload struct {\n\t_ struct{} `cqrs:\"event:UserCreated\"`\n}\n"
	writeTempGoFile(t, tmp, "tag.go", content)

	entries, err := scanFile(filepath.Join(tmp, "tag.go"), "event")
	if err != nil {
		t.Fatalf("scanFile: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	if entries[0].TypeName != "UserCreated" {
		t.Errorf("TypeName = %q, want %q", entries[0].TypeName, "UserCreated")
	}
}
