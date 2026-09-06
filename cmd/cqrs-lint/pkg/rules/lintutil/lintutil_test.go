package lintutil_test

import (
	"go/parser"
	"go/token"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/rules/lintutil"
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

	// A dot import binds no qualifier: it must never resolve, for any
	// qualifier name. The old behavior returned the path for anything,
	// letting rules attribute other packages' symbols to the dot import.
	if path, ok := lintutil.QualifierToImportPath(file, "anything"); ok {
		t.Errorf("dot import resolved for arbitrary qualifier: %q", path)
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

func TestQualifierToImportPath_DotImportBindsNothing(t *testing.T) {
	t.Parallel()

	src := `package main

import (
	. "github.com/larsartmann/go-codec"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
)
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}

	// A dot import binds no qualifier, so it must never resolve — for any
	// qualifier name — or rules would attribute other packages' symbols to it.
	if path, ok := lintutil.QualifierToImportPath(file, "go-codec"); ok {
		t.Errorf("dot import resolved for arbitrary qualifier: %q, %v", path, ok)
	}
}

func TestLastSegment_StripsTwoDigitMajorVersions(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"github.com/foo/event/v4":  "event",
		"github.com/foo/event/v10": "event",
		"github.com/foo/event/v99": "event",
		"github.com/foo/event":     "event",
		"net/http":                 "http",
	}

	for path, want := range cases {
		if got := lintutil.LastSegmentForTest(path); got != want {
			t.Errorf("lastSegment(%q) = %q, want %q", path, got, want)
		}
	}
}
