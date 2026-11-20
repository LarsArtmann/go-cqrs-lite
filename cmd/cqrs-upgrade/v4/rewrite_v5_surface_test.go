package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeSurfaceFixture writes a consumer-shaped file and returns its path.
func writeSurfaceFixture(t *testing.T, name, src string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}

	return path
}

func TestRewriteV5Surface_SingleValueCtorRenamed(t *testing.T) {
	t.Parallel()

	path := writeSurfaceFixture(t, "a.go", `package consumer

import "github.com/larsartmann/go-cqrs-lite/core/v5/id"

func mk() id.StreamRef {
	ref := id.NewStreamRef("User", id.NewStreamID())
	return ref
}
`)

	changes, manual, changed, err := rewriteV5SurfaceInFile(path, false)
	if err != nil {
		t.Fatalf("surface pass: %v", err)
	}
	if !changed {
		t.Fatal("expected the file to be rewritten")
	}
	if len(changes) != 1 || changes[0].Kind != "ctor-single-value" {
		t.Fatalf("changes = %+v, want one ctor-single-value", changes)
	}

	out, _ := os.ReadFile(path)
	if !strings.Contains(string(out), "ref := id.NewStreamRefOrZero(\"User\", id.NewStreamID())") {
		t.Errorf("output missing the OrZero rename:\n%s", out)
	}
	if strings.Contains(string(out), "id.NewStreamRef(") {
		t.Errorf("output still contains the bare ctor:\n%s", out)
	}
	_ = manual
}

func TestRewriteV5Surface_TwoValueCtorUntouched(t *testing.T) {
	t.Parallel()

	src := `package consumer

import "github.com/larsartmann/go-cqrs-lite/core/v5/id"

func mk() (id.StreamRef, error) {
	ref, err := id.NewStreamRef("User", id.NewStreamID())
	return ref, err
}
`
	path := writeSurfaceFixture(t, "a.go", src)

	_, _, changed, err := rewriteV5SurfaceInFile(path, false)
	if err != nil {
		t.Fatalf("surface pass: %v", err)
	}
	if changed {
		out, _ := os.ReadFile(path)
		t.Fatalf("two-value context must stay untouched, got:\n%s", out)
	}
}

func TestRewriteV5Surface_RecordStringCtorGetsHelper(t *testing.T) {
	t.Parallel()

	path := writeSurfaceFixture(t, "a.go", `package consumer

import (
	"github.com/larsartmann/go-cqrs-lite/core/v5/record"
)

var refs = []record.Record{{
	StreamID:   record.NewStreamRef("User", "01JTEST"),
	StreamType: "User",
	Type:       "user.created",
}}
`)

	changes, manual, _, err := rewriteV5SurfaceInFile(path, false)
	if err != nil {
		t.Fatalf("surface pass: %v", err)
	}

	var sawCtor, sawHelper bool
	for _, c := range changes {
		if c.Kind == "record-string-ctor" {
			sawCtor = true
		}
	}
	for _, m := range manual {
		if strings.Contains(m, "collapsed into Stream") {
			sawHelper = true
		}
	}
	if !sawCtor {
		t.Errorf("changes = %+v, want a record-string-ctor rewrite", changes)
	}
	if !sawHelper {
		t.Errorf("manual = %v, want the collapsed-field hint", manual)
	}

	out, _ := os.ReadFile(path)
	s := string(out)
	if !strings.Contains(s, "cqrsV5StreamRef(\"User\", \"01JTEST\")") {
		t.Errorf("output missing the helper call:\n%s", s)
	}
	if !strings.Contains(s, "func cqrsV5StreamRef(streamType, entityID string) record.StreamRef") {
		t.Errorf("output missing the injected helper:\n%s", s)
	}
	if !strings.Contains(s, `"github.com/larsartmann/go-cqrs-lite/core/v5/id"`) {
		t.Errorf("output missing the id import:\n%s", s)
	}
}

func TestRewriteV5Surface_Idempotent(t *testing.T) {
	t.Parallel()

	path := writeSurfaceFixture(t, "a.go", `package consumer

import "github.com/larsartmann/go-cqrs-lite/core/v5/id"

func mk() id.StreamRef {
	return id.NewStreamRefOrZero("User", id.NewStreamID())
}
`)

	_, _, changed, err := rewriteV5SurfaceInFile(path, false)
	if err != nil {
		t.Fatalf("surface pass: %v", err)
	}
	if changed {
		t.Error("a second run over already-migrated code must be a no-op")
	}
}

// TestRewriteV5Surface_ApplySmoke is the M14·e end-to-end apply-mode run:
// one fixture consumer repo directory, the full pass (imports + surface),
// and a parse-level verification of the result.
func TestRewriteV5Surface_ApplySmoke(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	src := `package consumer

import (
	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/core/v5/id"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

func handle(cmd *command.BasicCommand) record.Record {
	ref := id.NewStreamRef("User", cmd.StreamID())
	_ = event.AsRecord
	return record.Record{Type: "user.created", StreamID: record.NewStreamRef("User", "01J"), StreamType: "User"}
}
`
	if err := os.WriteFile(filepath.Join(dir, "consumer.go"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}

	rep := runV5Pass(dir, false)
	if rep.Error != "" {
		t.Fatalf("runV5Pass errored: %s", rep.Error)
	}
	if len(rep.Rewrites) == 0 {
		t.Fatal("expected import rewrites")
	}
	if rep.SurfaceChanges == 0 {
		t.Fatal("expected surface rewrites")
	}
	if len(rep.ManualSites) == 0 {
		t.Fatal("expected manual-review hints for the collapsed literal fields")
	}

	out, err := os.ReadFile(filepath.Join(dir, "consumer.go"))
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	for _, want := range []string{
		"core/v5/command",
		"core/v5/event",
		"id.NewStreamRefOrZero(\"User\", cmd.StreamID())",
		"cqrsV5StreamRef(\"User\", \"01J\")",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("output missing %q:\n%s", want, s)
		}
	}
	if strings.Contains(s, "command/v4") || strings.Contains(s, "event/v4") || strings.Contains(s, "record/v4") {
		t.Errorf("output still carries v4 imports:\n%s", s)
	}
}
