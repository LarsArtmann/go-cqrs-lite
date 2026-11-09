package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildDemoCatalog(t *testing.T) {
	t.Parallel()

	cat := buildDemoCatalog("Demo API", "1.0.0")

	if cat == nil {
		t.Fatal("expected non-nil catalog")
	}

	if string(cat.Title) != "Demo API" {
		t.Errorf("title = %q, want %q", cat.Title, "Demo API")
	}

	if string(cat.Version) != "1.0.0" {
		t.Errorf("version = %q, want %q", cat.Version, "1.0.0")
	}

	if len(cat.Services) != 1 {
		t.Fatalf("expected 1 service, got %d", len(cat.Services))
	}

	svc := cat.Services[0]
	if string(svc.ID) != "demo-svc" {
		t.Errorf("service ID = %q, want demo-svc", svc.ID)
	}

	if len(svc.Commands) != 1 {
		t.Fatalf("expected 1 command, got %d", len(svc.Commands))
	}

	if len(svc.Queries) != 1 {
		t.Fatalf("expected 1 query, got %d", len(svc.Queries))
	}

	cmd := svc.Commands[0]
	if cmd.Operation == nil {
		t.Fatal("expected command to have an operation")
	}

	if cmd.Operation.Method != "POST" || cmd.Operation.Path != "/api/items" {
		t.Errorf(
			"command operation = %s %s, want POST /api/items",
			cmd.Operation.Method,
			cmd.Operation.Path,
		)
	}

	qry := svc.Queries[0]
	if qry.Operation == nil {
		t.Fatal("expected query to have an operation")
	}

	if qry.Operation.Method != "GET" || qry.Operation.Path != "/api/items" {
		t.Errorf(
			"query operation = %s %s, want GET /api/items",
			qry.Operation.Method,
			qry.Operation.Path,
		)
	}
}

func TestBuildDemoCatalog_Validate(t *testing.T) {
	t.Parallel()

	cat := buildDemoCatalog("Demo API", "1.0.0")

	if violations := cat.Validate(); len(violations) > 0 {
		t.Fatalf(
			"demo catalog should validate cleanly, got %d violations: %v",
			len(violations),
			violations,
		)
	}
}

func TestGenerate_OpenAPI(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cat := buildDemoCatalog("Demo", "1.0.0")

	if err := generate(cat, "openapi", dir, "Demo", "1.0.0"); err != nil {
		t.Fatalf("generate openapi: %v", err)
	}

	path := filepath.Join(dir, "openapi.json")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("openapi.json not created: %v", err)
	}
}

func TestGenerate_AsyncAPI(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cat := buildDemoCatalog("Demo", "1.0.0")

	if err := generate(cat, "asyncapi", dir, "Demo", "1.0.0"); err != nil {
		t.Fatalf("generate asyncapi: %v", err)
	}

	path := filepath.Join(dir, "asyncapi.json")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("asyncapi.json not created: %v", err)
	}
}

func TestGenerate_D2(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cat := buildDemoCatalog("Demo", "1.0.0")

	if err := generate(cat, "d2", dir, "Demo", "1.0.0"); err != nil {
		t.Fatalf("generate d2: %v", err)
	}

	path := filepath.Join(dir, "diagram.d2")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("diagram.d2 not created: %v", err)
	}
}

func TestGenerate_LLMs(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cat := buildDemoCatalog("Demo", "1.0.0")

	if err := generate(cat, "llms", dir, "Demo", "1.0.0"); err != nil {
		t.Fatalf("generate llms: %v", err)
	}

	path := filepath.Join(dir, "llms.txt")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("llms.txt not created: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read llms.txt: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "Demo") {
		t.Error("llms.txt missing catalog title")
	}

	if !strings.Contains(content, "Demo Service") {
		t.Error("llms.txt missing service name")
	}

	if !strings.Contains(content, "CreateItem") {
		t.Error("llms.txt missing command name")
	}

	if !strings.Contains(content, "ListItems") {
		t.Error("llms.txt missing query name")
	}
}

func TestGenerate_UnknownFormat(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cat := buildDemoCatalog("Demo", "1.0.0")

	err := generate(cat, "xml", dir, "Demo", "1.0.0")
	if err == nil {
		t.Fatal("expected error for unknown format")
	}
}
