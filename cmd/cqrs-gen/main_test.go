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
