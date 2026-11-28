package search_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/query/v4"
	"github.com/larsartmann/go-cqrs-lite/search/v4"
)

// corpus is the conformance corpus: documents, a query and the order the
// in-memory ranking rule must produce. Go tests run it here; the browser
// matcher of templ-components replays the same file, so the two runtimes
// cannot drift apart silently.
type corpus struct {
	Version int          `json:"version"`
	Cases   []corpusCase `json:"cases"`
}

// corpusField and corpusDoc reuse the static index shapes.
type (
	corpusField = search.StaticField
	corpusDoc   = search.StaticItem
)

type corpusCase struct {
	Name   string        `json:"name"`
	Query  string        `json:"query"`
	Fields []corpusField `json:"fields"`
	Docs   []corpusDoc   `json:"docs"`
	Want   []string      `json:"want"`
}

// corpusCases is the corpus, written to testdata/conformance.json.
func corpusCases() []corpusCase {
	title := corpusField{Name: "title", Weight: 1}
	body := corpusField{Name: "body", Weight: 0.2}
	fields := []corpusField{title, body}
	docs := func(pairs ...[2]string) []corpusDoc {
		out := make([]corpusDoc, 0, len(pairs))
		for _, pair := range pairs {
			out = append(out, corpusDoc{ID: pair[0], Fields: map[string]string{"title": pair[1]}})
		}

		return out
	}
	withBody := func(doc corpusDoc, body string) corpusDoc {
		doc.Fields["body"] = body

		return doc
	}

	return []corpusCase{
		{
			Name: "exact beats word prefix beats substring", Query: "inv", Fields: fields,
			Docs: []corpusDoc{
				{ID: "sub", Fields: map[string]string{"title": "reinvented"}},
				{ID: "pre", Fields: map[string]string{"title": "Invoice March"}},
				{ID: "exact", Fields: map[string]string{"title": "Inv"}},
			},
			Want: []string{"exact", "pre", "sub"},
		},
		{
			Name: "a title match outranks a body match", Query: "invoice", Fields: fields,
			Docs: []corpusDoc{
				{ID: "body", Fields: map[string]string{"title": "Standup", "body": "about the invoice"}},
				{ID: "title", Fields: map[string]string{"title": "Invoice", "body": "nothing"}},
			},
			Want: []string{"title", "body"},
		},
		{
			Name: "every term must match", Query: "big invoice", Fields: fields,
			Docs: []corpusDoc{
				{ID: "both", Fields: map[string]string{"title": "Big Invoice"}},
				{ID: "big-only", Fields: map[string]string{"title": "Big"}},
				{ID: "invoice-only", Fields: map[string]string{"title": "Invoice"}},
			},
			Want: []string{"both"},
		},
		{
			Name: "a term may live in another field", Query: "big invoice", Fields: fields,
			Docs: []corpusDoc{
				{ID: "split", Fields: map[string]string{"title": "Big", "body": "the invoice"}},
				{ID: "none", Fields: map[string]string{"title": "Small note"}},
			},
			Want: []string{"split"},
		},
		{
			Name: "case folds including non-ascii, ties by identifier", Query: "über", Fields: fields,
			Docs: []corpusDoc{
				{ID: "upper", Fields: map[string]string{"title": "ÜBERSCHRIFT"}},
				{ID: "lower", Fields: map[string]string{"title": "über allem"}},
			},
			Want: []string{"lower", "upper"},
		},
		{
			Name: "ties break by identifier", Query: "note", Fields: fields,
			Docs: docs([2]string{"c", "Note"}, [2]string{"a", "Note"}, [2]string{"b", "note"}),
			Want: []string{"a", "b", "c"},
		},
		{
			Name: "the sum over terms orders partial prominence", Query: "march invoice", Fields: fields,
			Docs: []corpusDoc{
				withBody(corpusDoc{ID: "title-once", Fields: map[string]string{"title": "March"}}, "an invoice"),
				withBody(corpusDoc{ID: "both-words", Fields: map[string]string{"title": "March invoice"}}, ""),
				withBody(corpusDoc{ID: "body-twice", Fields: map[string]string{"title": "Standup"}}, "March invoice"),
			},
			Want: []string{"both-words", "title-once", "body-twice"},
		},
		{
			Name: "polish and german text match on whole words", Query: "rechnung", Fields: fields,
			Docs: []corpusDoc{
				{ID: "de", Fields: map[string]string{"title": "Rechnung März", "body": "Zażółć gęślą jaźń"}},
				{ID: "pl", Fields: map[string]string{"title": "Faktura", "body": "Rechnung"}},
			},
			Want: []string{"de", "pl"},
		},
		{
			Name: "an identifier is matched literally", Query: "KW1/2024/0042", Fields: fields,
			Docs: []corpusDoc{
				{ID: "hit", Fields: map[string]string{"title": "Deed KW1/2024/0042"}},
				{ID: "miss", Fields: map[string]string{"title": "Deed KW1/2024/0043"}},
			},
			Want: []string{"hit"},
		},
		{
			Name: "an empty query keeps the document order", Query: "", Fields: fields,
			Docs: docs([2]string{"z", "Last"}, [2]string{"a", "First"}),
			Want: []string{"z", "a"},
		},
	}
}

func TestConformanceCorpusGolden(t *testing.T) {
	corpus := corpus{Version: 1, Cases: corpusCases()}
	encoded, err := json.MarshalIndent(corpus, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join("testdata", "conformance.json")
	if *flagUpdate {
		if err := os.WriteFile(path, append(encoded, '\n'), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("corpus missing; run go test ./search/v4 -update: %v", err)
	}
	if string(encoded) != string(trimNewline(want)) {
		t.Fatalf(
			"conformance.json is out of date; run go test ./search/v4 -update (the browser matcher replays this file)",
		)
	}
}

// searchStaticItem searches documents in the static-index shape.
func searchStaticItem(t *testing.T, tc corpusCase, query string) []string {
	t.Helper()

	fields := make([]search.Field[corpusDoc], 0, len(tc.Fields))
	for _, field := range tc.Fields {
		name := field.Name
		weight := field.Weight
		fields = append(fields, search.Field[corpusDoc]{
			Name: name, Weight: weight, Value: func(doc corpusDoc) string { return doc.Fields[name] },
		})
	}
	source := search.Slice[corpusDoc]{Items: tc.Docs, Identify: func(doc corpusDoc) string { return doc.ID }}
	index, err := search.New(search.Spec[corpusDoc]{Fields: fields}, search.NoEngine(), source)
	if err != nil {
		t.Fatal(err)
	}
	result, err := index.Search(context.Background(), search.Request{Query: query, Page: pageOf(100)})
	if err != nil {
		t.Fatal(err)
	}
	got := make([]string, len(result.Data))
	for i, hit := range result.Data {
		got[i] = hit.ID
	}

	return got
}

func TestConformanceCorpus(t *testing.T) {
	for _, tc := range corpusCases() {
		t.Run(tc.Name, func(t *testing.T) {
			got := searchStaticItem(t, tc, tc.Query)
			if !reflect.DeepEqual(got, tc.Want) {
				t.Fatalf("query %q: got %v, want %v", tc.Query, got, tc.Want)
			}
		})
	}
}

func pageOf(size int) query.Pagination {
	return query.Pagination{Page: 1, PageSize: uint(size)}
}

// TestConformanceFileMatchesSource reads the written file back, so the file
// the browser replays is the file Go tested.
func TestConformanceFileMatchesSource(t *testing.T) {
	encoded, err := os.ReadFile(filepath.Join("testdata", "conformance.json"))
	if err != nil {
		t.Fatal(err)
	}
	var corpus corpus
	if err := json.Unmarshal(encoded, &corpus); err != nil {
		t.Fatal(err)
	}
	if corpus.Version != 1 || len(corpus.Cases) != len(corpusCases()) {
		t.Fatalf("file has version %d and %d cases", corpus.Version, len(corpus.Cases))
	}
	for i, tc := range corpus.Cases {
		if tc.Name != corpusCases()[i].Name || strings.TrimSpace(tc.Query) != corpusCases()[i].Query {
			t.Fatalf("case %d differs from its source: %q vs %q", i, tc.Name, corpusCases()[i].Name)
		}
	}
}

func TestConformanceCorpusFiltersNothing(t *testing.T) {
	for _, tc := range corpusCases() {
		for _, doc := range tc.Docs {
			if len(doc.Fields) == 0 {
				t.Fatalf("case %q has a document without fields: %s", tc.Name, doc.ID)
			}
		}
	}
}
