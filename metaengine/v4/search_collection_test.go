package metaengine

import (
	"context"
	"testing"
)

func TestMemorySearchIndex_CollectionsAreIsolated(t *testing.T) {
	t.Parallel()

	index := NewMemorySearchIndex()
	ctx := context.Background()

	steps := []struct {
		collection string
		doc        IndexedText
	}{
		{"articles", IndexedText{ID: "doc1", Content: "the quick brown fox"}},
		{"notes", IndexedText{ID: "doc1", Content: "a completely different text about storage"}},
		{"notes", IndexedText{ID: "doc2", Content: "foxes live in forests"}},
	}

	for _, s := range steps {
		if err := index.Insert(ctx, s.collection, s.doc); err != nil {
			t.Fatalf("Insert(%s): %v", s.collection, err)
		}
	}

	articles, err := index.Query(ctx, "articles", "quick", 10)
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(articles) != 1 || articles[0].ID != "doc1" {
		t.Errorf("articles query returned %v, want exactly doc1", articles)
	}

	notes, err := index.Query(ctx, "notes", "foxes", 10)
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(notes) != 1 || notes[0].ID != "doc2" {
		t.Errorf("notes query returned %v, want exactly doc2", notes)
	}

	unknown, err := index.Query(ctx, "missing", "quick", 10)
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(unknown) != 0 {
		t.Errorf("unknown collection returned %v, want nothing", unknown)
	}
}
