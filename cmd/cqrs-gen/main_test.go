package main

import (
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

	if entries[0].CommandType != "CreateUser" || entries[0].StructName != "CreateUserCmd" {
		t.Errorf("entry[0] = %+v", entries[0])
	}

	if entries[1].CommandType != "DeleteUser" || entries[1].StructName != "DeleteUserCmd" {
		t.Errorf("entry[1] = %+v", entries[1])
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

	if entries[0].CommandType != "GetUser" || entries[0].StructName != "GetUserQuery" {
		t.Errorf("entry = %+v", entries[0])
	}
}

func TestGenerate_Command(t *testing.T) {
	t.Parallel()

	entries := []Entry{
		{CommandType: "CreateUser", StructName: "CreateUserCmd"},
	}

	code := generate("handlers", "command", entries)

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
		{CommandType: "GetUser", StructName: "GetUserQuery"},
	}

	code := generate("handlers", "query", entries)

	if !strings.Contains(code, "github.com/larsartmann/go-cqrs-lite/core/query") {
		t.Error("missing query import")
	}

	if !strings.Contains(code, "func RegisterGetUserQueryHandler") {
		t.Error("missing handler function")
	}
}

func TestExtractMarker(t *testing.T) {
	t.Parallel()

	// Cannot test extractMarker directly without parsing a real AST,
	// but the scanFile tests cover it end-to-end.
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

func TestScanFile_NoMarkers(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	src := filepath.Join(tmp, "plain.go")

	content := `package example

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

	if len(entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(entries))
	}
}

func TestScanFile_WrongMarkerType(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	src := filepath.Join(tmp, "cmd.go")

	content := `package example

//cqrs:command CreateUser
type CreateUserCmd struct{}`
	if err := os.WriteFile(src, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	entries, err := scanFile(src, "query")
	if err != nil {
		t.Fatalf("scan: %v", err)
	}

	if len(entries) != 0 {
		t.Errorf("expected 0 entries when scanning for queries in command file, got %d", len(entries))
	}
}

func TestScanFile_MultipleFilesInDir(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()

	cmdContent := `package example

//cqrs:command CreateUser
type CreateUserCmd struct{}`
	if err := os.WriteFile(filepath.Join(tmp, "commands.go"), []byte(cmdContent), 0o644); err != nil {
		t.Fatal(err)
	}

	queryContent := `package example

//cqrs:query GetUser
type GetUserQuery struct{}`
	if err := os.WriteFile(filepath.Join(tmp, "queries.go"), []byte(queryContent), 0o644); err != nil {
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
		{CommandType: "CreateUser", StructName: "CreateUserCmd", PackagePath: "example"},
		{CommandType: "DeleteUser", StructName: "DeleteUserCmd", PackagePath: "example"},
	}

	code := generate("handlers", "command", entries)

	if !strings.Contains(code, "RegisterCreateUserCmdHandler") {
		t.Error("missing CreateUserCmd handler")
	}
	if !strings.Contains(code, "RegisterDeleteUserCmdHandler") {
		t.Error("missing DeleteUserCmd handler")
	}
}

func TestGenerate_PackageName(t *testing.T) {
	t.Parallel()

	entries := []Entry{{CommandType: "Test", StructName: "TestCmd"}}
	code := generate("mypkg", "command", entries)

	if !strings.Contains(code, "package mypkg") {
		t.Error("expected package name 'mypkg'")
	}
}

func TestGenerate_QueryImports(t *testing.T) {
	t.Parallel()

	entries := []Entry{{CommandType: "GetUser", StructName: "GetUserQuery"}}
	code := generate("handlers", "query", entries)

	if !strings.Contains(code, "core/query") {
		t.Error("missing query import")
	}
	if strings.Contains(code, "core/command") {
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

func TestMustAbs(t *testing.T) {
	t.Parallel()

	result := mustAbs(".")
	if !filepath.IsAbs(result) {
		t.Errorf("expected absolute path, got %s", result)
	}
}

func TestRun_InvalidType(t *testing.T) {
	t.Parallel()

	code := run("invalid", "out.go", "", []string{"."})
	if code != 1 {
		t.Errorf("expected exit 1 for invalid type, got %d", code)
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
	if code != 0 {
		t.Errorf("expected exit 0 for no markers, got %d", code)
	}
}

func TestRun_SuccessfulCommand(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	content := `package example

//cqrs:command CreateUser
type CreateUserCmd struct{}`
	if err := os.WriteFile(filepath.Join(tmp, "commands.go"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	outputFile := filepath.Join(tmp, "handlers_gen.go")
	code := run("command", outputFile, "", []string{tmp})
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
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
	if err := os.WriteFile(filepath.Join(tmp, "queries.go"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	outputFile := filepath.Join(tmp, "queries_gen.go")
	code := run("query", outputFile, "handlers", []string{tmp})
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}

	generated, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("output file not created: %v", err)
	}
	if !strings.Contains(string(generated), "package handlers") {
		t.Errorf("expected custom package name, got: %s", generated)
	}
	if !strings.Contains(string(generated), "core/query") {
		t.Errorf("expected query import, got: %s", generated)
	}
}

func TestRun_WriteError(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	content := `package example

//cqrs:command CreateUser
type CreateUserCmd struct{}`
	if err := os.WriteFile(filepath.Join(tmp, "commands.go"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	code := run("command", "/nonexistent/dir/handlers_gen.go", "", []string{tmp})
	if code != 1 {
		t.Errorf("expected exit 1 for write error, got %d", code)
	}
}

func TestRun_DefaultPath(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	content := `package example

//cqrs:command CreateItem
type CreateItemCmd struct{}`
	if err := os.WriteFile(filepath.Join(tmp, "item.go"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	outputFile := filepath.Join(tmp, "gen.go")
	code := run("command", outputFile, "", []string{tmp})
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}

	generated, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("output file not created: %v", err)
	}
	if !strings.Contains(string(generated), "RegisterCreateItemCmdHandler") {
		t.Errorf("generated file missing handler, got: %s", generated)
	}
}
