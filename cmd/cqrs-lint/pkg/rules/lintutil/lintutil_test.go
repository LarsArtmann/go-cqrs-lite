package lintutil_test

import (
	"go/parser"
	"go/token"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/lintutil"
)

func TestQualifierToImportPath_NoAlias(t *testing.T) {
	t.Parallel()

	src := `package main
import "github.com/larsartmann/go-cqrs-lite/event/v4"
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}

	path, ok := lintutil.QualifierToImportPath(file, "event")
	if !ok {
		t.Fatal("expected qualifier 'event' to resolve")
	}
	if path != "github.com/larsartmann/go-cqrs-lite/event/v4" {
		t.Errorf("got %q, want the event import path", path)
	}
}

func TestQualifierToImportPath_WithAlias(t *testing.T) {
	t.Parallel()

	src := `package main
import cqrs "github.com/larsartmann/go-cqrs-lite/event/v4"
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}

	path, ok := lintutil.QualifierToImportPath(file, "cqrs")
	if !ok {
		t.Fatal("expected aliased qualifier 'cqrs' to resolve")
	}
	if path != "github.com/larsartmann/go-cqrs-lite/event/v4" {
		t.Errorf("got %q, want the event import path", path)
	}

	_, ok = lintutil.QualifierToImportPath(file, "event")
	if ok {
		t.Error("expected 'event' to NOT resolve when aliased to 'cqrs'")
	}
}

func TestQualifierToImportPath_DotImport(t *testing.T) {
	t.Parallel()

	src := `package main
import . "github.com/larsartmann/go-cqrs-lite/event/v4"
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}

	path, ok := lintutil.QualifierToImportPath(file, "anything")
	if !ok {
		t.Fatal("expected dot import to resolve any qualifier")
	}
	if path != "github.com/larsartmann/go-cqrs-lite/event/v4" {
		t.Errorf("got %q, want the event import path", path)
	}
}

func TestQualifierToImportPath_BlankImport(t *testing.T) {
	t.Parallel()

	src := `package main
import _ "github.com/larsartmann/go-cqrs-lite/event/v4"
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}

	_, ok := lintutil.QualifierToImportPath(file, "event")
	if ok {
		t.Error("expected blank import to NOT resolve")
	}
}

func TestQualifierResolvesTo(t *testing.T) {
	t.Parallel()

	src := `package main
import ev "github.com/larsartmann/go-cqrs-lite/event/v4"
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}

	if !lintutil.QualifierResolvesTo(file, "ev", "go-cqrs-lite/event") {
		t.Error("expected 'ev' to resolve to go-cqrs-lite/event")
	}

	if lintutil.QualifierResolvesTo(file, "ev", "go-cqrs-lite/command") {
		t.Error("expected 'ev' to NOT resolve to go-cqrs-lite/command")
	}
}

func TestImportQualifierMap(t *testing.T) {
	t.Parallel()

	src := `package main

import (
	cqrs "github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/command/v4"
	. "github.com/larsartmann/go-cqrs-lite/codec/v4"
	_ "github.com/larsartmann/go-cqrs-lite/id/v4"
)
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}

	m := lintutil.ImportQualifierMap(file)

	if m["cqrs"] != "github.com/larsartmann/go-cqrs-lite/event/v4" {
		t.Errorf("cqrs alias: got %q", m["cqrs"])
	}

	if m["command"] != "github.com/larsartmann/go-cqrs-lite/command/v4" {
		t.Errorf("command: got %q", m["command"])
	}

	if m["."] != "github.com/larsartmann/go-cqrs-lite/codec/v4" {
		t.Errorf("dot import: got %q", m["."])
	}

	if _, ok := m["id"]; ok {
		t.Error("blank import should not appear in the map")
	}
}
