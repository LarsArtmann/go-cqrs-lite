package search_test

import (
	"fmt"
	"math"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/larsartmann/go-cqrs-lite/search/v4"
)

func TestScoreRankingRule(t *testing.T) {
	t.Parallel()

	spec := testSpec()
	cases := []struct {
		name  string
		query string
		note  Note
		want  float64
		ok    bool
	}{
		{"exact title", "invoice", Note{Title: "Invoice"}, 3, true},
		{"prefix title", "inv", Note{Title: "Invoice"}, 2, true},
		{"substring title", "voice", Note{Title: "Invoice"}, 1, true},
		{"exact body weighs less", "invoice", Note{Body: "invoice"}, 0.2 * 3, true},
		{"prefix body", "inv", Note{Body: "Invoice"}, 0.4, true},
		{"word prefix inside a field", "inv", Note{Body: "the Invoice paid"}, 0.4, true},
		{"inside a word is a substring", "inv", Note{Body: "reinvented"}, 0.2, true},
		{"substring body", "voice", Note{Body: "Invoice"}, 0.2, true},
		{"title outranks body", "invoice", Note{Title: "Invoice", Body: "invoice"}, 3, true},
		{"case-folded both ways", "INVOICE", Note{Title: "invoice"}, 3, true},
		{"case-folded non-ascii", "über", Note{Title: "Über"}, 3, true},
		{"every term must match", "big invoice", Note{Title: "Big"}, 0, false},
		{"sums the terms", "big invoice", Note{Title: "Big Invoice"}, 4, true},
		{"a term scores on its best field only", "inv", Note{Title: "the invoice", Body: "inv"}, 2, true},
		{"unknown operator token stays text", "amount:10", Note{Body: "amount:10 due"}, 0.4, true},
		{"no match", "zzz", Note{Title: "Invoice"}, 0, false},
		{"empty record", "invoice", Note{}, 0, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, ok := search.Score(spec, search.Request{Query: tc.query}, tc.note)
			if ok != tc.ok || (ok && math.Abs(got-tc.want) > 1e-9) {
				t.Fatalf("Score(%q) = %v, %v; want %v, %v", tc.query, got, ok, tc.want, tc.ok)
			}
		})
	}
}

func TestScoreSumsAcrossFields(t *testing.T) {
	t.Parallel()

	got, ok := search.Score(
		testSpec(),
		search.Request{Query: "standup invoice"},
		Note{Title: "Standup", Body: "about invoices"},
	)
	if !ok || math.Abs(got-3.4) > 1e-9 {
		t.Fatalf("Score = %v, %v; want 3.4 (exact title 3 + word-prefix body 0.4)", got, ok)
	}
}

func TestScoreEmptyTermsIsNoMatch(t *testing.T) {
	t.Parallel()

	if _, ok := search.Score(testSpec(), search.Request{}, Note{Title: "x"}); ok {
		t.Fatal("an empty query scored a record; the unfiltered list is not a match")
	}
}

func TestFoldKeepsRuneIndices(t *testing.T) {
	t.Parallel()

	for _, text := range []string{"Hello", "İstanbul", "Straße", "Ωmega", "", "🙂 x"} {
		folded := search.Fold(text)
		if utf8.RuneCountInString(folded) != utf8.RuneCountInString(text) {
			t.Errorf("Fold(%q) = %q changed the rune count", text, folded)
		}
		if folded != strings.ToLower(text) && strings.Contains(text, "İ") {
			t.Logf("Fold differs from ToLower for %q: %q vs %q", text, folded, strings.ToLower(text))
		}
	}
}

func TestIsIdentifierLike(t *testing.T) {
	t.Parallel()

	cases := map[string]bool{
		"+49 160 1234567": true,
		"KW1/2024/0042":   true,
		"2024-0042":       true,
		"12345":           true,
		"invoice":         false,
		"Invoice 2024":    false,
		"abc123":          false,
		"big invoice":     false,
		"":                false,
		"   ":             false,
		"a1":              false,
		"a1 b2":           false,
		"#1234":           true,
		"DE-123456789":    true,
	}
	for text, want := range cases {
		if got := search.IsIdentifierLike(text); got != want {
			t.Errorf("IsIdentifierLike(%q) = %v, want %v", text, got, want)
		}
	}
}

func TestRankHitsIsStable(t *testing.T) {
	t.Parallel()

	hits := []search.Hit[Note]{
		{ID: "c", Score: 1}, {ID: "a", Score: 1}, {ID: "b", Score: 2}, {ID: "a0", Score: 2},
	}
	got := search.RankHits(hits)
	order := make([]string, len(got))
	for i, hit := range got {
		order[i] = hit.ID
	}
	if fmt.Sprint(order) != "[a0 b a c]" {
		t.Fatalf("order = %v, want [a0 b a c] (score first, then identifier)", order)
	}
}

func TestScoreFuzzyNeverPanics(t *testing.T) {
	fuzz := func(query, title, body string) {
		spec := testSpec()
		normalized, err := search.Request{Query: query}.Normalize(0)
		if err != nil {
			return
		}
		first, _ := search.Score(spec, normalized, Note{Title: title, Body: body})
		second, _ := search.Score(spec, normalized, Note{Title: title, Body: body})
		if first != second {
			t.Fatalf("Score is not deterministic for %q", query)
		}
		search.Snippet(body, query, 5)
		search.FTS5Query(query, false)
	}
	fuzz("\x00", "\xff", "a\xfd")
	fuzz("\"\x80", "😀", "\xc3")
	fuzz(strings.Repeat("ü", 500), strings.Repeat("İ", 300), "")
}

func FuzzScore(f *testing.F) {
	f.Add("big invoice", "Big Invoice", "the big one")
	f.Add("", "", "")
	f.Add("\"", "😀", "\xff")
	f.Fuzz(func(t *testing.T, query, title, body string) {
		spec := testSpec()
		normalized, err := search.Request{Query: query}.Normalize(0)
		if err != nil {
			t.Skip()
		}
		first, _ := search.Score(spec, normalized, Note{Title: title, Body: body})
		second, _ := search.Score(spec, normalized, Note{Title: title, Body: body})
		if first != second {
			t.Fatalf("Score is not deterministic for %q", query)
		}
		search.SnippetTerms(body, strings.Fields(normalized.Query), 3)
	})
}

func BenchmarkScore(b *testing.B) {
	spec := testSpec()
	normalized, _ := search.Request{Query: "big invoice march"}.Normalize(0)
	notes := make([]Note, 1000)
	for i := range notes {
		notes[i] = Note{
			ID:    fmt.Sprintf("n%d", i),
			Title: fmt.Sprintf("Note %d about the big invoice", i),
			Body:  "The March invoice for Acme needs review before the standup",
		}
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		matches := 0
		for _, note := range notes {
			if _, ok := search.Score(spec, normalized, note); ok {
				matches++
			}
		}
	}
	b.ReportMetric(float64(b.N)*float64(len(notes))/b.Elapsed().Seconds(), "score/s")
}
