package search_test

import (
	"encoding/json"
	"errors"
	"flag"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/search/v4"
)

var flagUpdate = flag.Bool("update", false, "rewrite golden files")

func TestBuildStaticIndex(t *testing.T) {
	t.Parallel()

	spec := testSpec()
	spec.Fields = append(spec.Fields, search.Field[Note]{
		Name: "secret", Weight: 9, Private: true, Value: func(n Note) string { return n.Status },
	})
	index, err := search.BuildStaticIndex(spec, []Note{
		{ID: "n1", Title: "Big invoice", Body: "The March invoice", Status: "open"},
		{ID: "n2", Title: "Holiday", Status: "done"},
	}, func(n Note) string { return n.ID })
	if err != nil {
		t.Fatal(err)
	}
	if index.Version != search.StaticIndexVersion {
		t.Fatalf("version %d", index.Version)
	}
	if got := index.Fields; len(got) != 2 || got[0].Name != "title" || got[1].Weight != 0.2 {
		t.Fatalf("fields %+v; private fields must be left out", got)
	}
	if _, leaked := index.Items[0].Fields["secret"]; leaked {
		t.Fatal("a private field left the machine")
	}
	if _, empty := index.Items[1].Fields["body"]; empty {
		t.Fatal("an empty value was emitted")
	}
	encoded, _ := json.Marshal(index)
	golden := filepath.Join("testdata", "static_index.golden")
	if *flagUpdate {
		if err := os.WriteFile(golden, append(encoded, '\n'), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("golden missing; run go test ./search/v4 -update: %v", err)
	}
	if string(encoded) != string(trimNewline(want)) {
		t.Fatalf("static index JSON changed:\n got %s\nwant %s", encoded, want)
	}
}

func trimNewline(b []byte) []byte {
	if len(b) > 0 && b[len(b)-1] == '\n' {
		return b[:len(b)-1]
	}
	return b
}

func TestStaticIndexRoundTrip(t *testing.T) {
	spec := testSpec()
	index, err := search.BuildStaticIndex(spec, notes().Items, func(n Note) string { return n.ID })
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "index.json")
	if err := index.WriteFile(path); err != nil {
		t.Fatal(err)
	}
	loaded, err := search.LoadStaticIndex(path)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(loaded, index) {
		t.Fatalf("round trip changed the index:\n got %+v\nwant %+v", loaded, index)
	}
	if _, err := search.LoadStaticIndex(filepath.Join(t.TempDir(), "missing.json")); err == nil {
		t.Fatal("loading a missing file succeeded")
	}
	bad := filepath.Join(t.TempDir(), "bad.json")
	if err := os.WriteFile(bad, []byte(`{"version":99}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := search.LoadStaticIndex(bad); !errors.Is(err, os.ErrNotExist) && err == nil {
		t.Fatal("a foreign version loaded without an error")
	}
}

func TestBuildStaticIndexRejectsBadInput(t *testing.T) {
	t.Parallel()

	if _, err := search.BuildStaticIndex(
		search.Spec[Note]{},
		nil,
		func(n Note) string { return n.ID },
	); !errors.Is(
		err,
		search.ErrNoFields,
	) {
		t.Fatalf("no fields = %v", err)
	}
	if _, err := search.BuildStaticIndex(testSpec(), nil, nil); err == nil {
		t.Fatal("a nil identifier function was accepted")
	}
	var nilIndex *search.StaticIndex
	if err := nilIndex.WriteFile("x"); err == nil {
		t.Fatal("writing a nil index succeeded")
	}
}
