package search_test

import (
	"reflect"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/larsartmann/go-cqrs-lite/search/v4"
)

func TestSnippet(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		text   string
		query  string
		radius int
		want   []search.Segment
	}{
		{
			name:  "marks every term",
			text:  "The big invoice and a small invoice",
			query: "invoice",
			want: []search.Segment{
				{Text: "The big "},
				{Text: "invoice", Marked: true},
				{Text: " and a small "},
				{Text: "invoice", Marked: true},
			},
		},
		{
			name:  "marks overlapping terms once",
			text:  "big bigger",
			query: "big bigg",
			want: []search.Segment{
				{Text: "big", Marked: true},
				{Text: " "},
				{Text: "bigg", Marked: true},
				{Text: "er"},
			},
		},
		{
			name: "case-folded match keeps the original text", text: "Die Rechnung", query: "rechnung",
			want: []search.Segment{{Text: "Die "}, {Text: "Rechnung", Marked: true}},
		},
		{
			name: "no match yields the start", text: "Nothing to see here", query: "invoice",
			want: []search.Segment{{Text: "Nothing to see here"}},
		},
		{
			name: "empty query", text: "Nothing", query: "",
			want: []search.Segment{{Text: "Nothing"}},
		},
		{
			name: "empty text", text: "", query: "invoice", want: nil,
		},
		{
			name:  "window around a late match snaps to words",
			text:  "one two three four five six seven eight nine ten eleven twelve",
			query: "eleven", radius: 8,
			want: []search.Segment{{Text: "…"}, {Text: "nine ten "}, {Text: "eleven", Marked: true}, {Text: " twelve"}},
		},
		{
			name:   "cut end carries an ellipsis",
			text:   "invoice and then a very long tail of text",
			query:  "invoice",
			radius: 6,
			want:   []search.Segment{{Text: "invoice", Marked: true}, {Text: " and then"}, {Text: "…"}},
		},
		{
			name:   "a window is never cut inside the match",
			text:   "abcdefabcdefabcdef",
			query:  "abcdefabcdef",
			radius: 1,
			want:   []search.Segment{{Text: "abcdefabcdefabcdef", Marked: true}},
		},
		{
			name: "radius zero means the default", text: "a b c", query: "c", radius: 0,
			want: []search.Segment{{Text: "a b "}, {Text: "c", Marked: true}},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := search.Snippet(tc.text, tc.query, tc.radius)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("Snippet() = %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestSnippetCutsOnRuneBoundaries(t *testing.T) {
	t.Parallel()

	texts := []string{
		"Zażółć gęślą jaźń i szukaj recherche w Świętokrzyskiem",
		"Über den Großen Ärmelkanal fliegt ein Strauß mit Rückflugticket",
		"το κείμενο έχει μια αναζήτηση εδώ Lorem ipsum dolor sit amet",
	}
	for _, text := range texts {
		for _, term := range []string{"gęślą", "ärmel", "αναζήτηση", "jaźń"} {
			for radius := 1; radius < 12; radius++ {
				got := search.Snippet(text, term, radius)
				for _, segment := range got {
					if !utf8.ValidString(segment.Text) {
						t.Fatalf("Snippet(%q, %q, %d) cut inside a rune: %+v", text, term, radius, got)
					}
				}
				if len(got) > 0 && got[0].Text != "…" && !strings.HasPrefix(text, got[0].Text) {
					t.Fatalf("Snippet(%q, %q, %d) did not start at the text start: %+v", text, term, radius, got)
				}
			}
		}
	}
}

func TestSnippetIsTheWindow(t *testing.T) {
	t.Parallel()

	text := "Prefix words before the match and a long tail of words after it that goes on and on"
	for _, term := range []string{"match", "before", "on"} {
		for radius := 0; radius < 40; radius += 3 {
			got := search.Snippet(text, term, radius)
			joined := ""
			var joinedSb129 strings.Builder
			for _, segment := range got {
				joinedSb129.WriteString(segment.Text)
			}
			joined += joinedSb129.String()
			trimmed := strings.Trim(joined, "…")
			if !strings.Contains(text, trimmed) {
				t.Fatalf("Snippet(%q, %d) = %q is not a slice of the text", term, radius, joined)
			}
			if strings.Contains(joined, "……") {
				t.Fatalf("double ellipsis: %q", joined)
			}
		}
	}
}
