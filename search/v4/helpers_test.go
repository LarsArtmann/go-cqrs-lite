package search_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/search/v4"
)

func TestLikePattern(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"invoice":     "%invoice%",
		"50%":         `%50\%%`,
		"under_score": `%under\_score%`,
		`back\slash`:  `%back\\slash%`,
		`%_\`:         `%\%\_\\%`,
	}
	for in, want := range cases {
		if got := search.LikePattern(in); got != want {
			t.Errorf("LikePattern(%q) = %q, want %q", in, got, want)
		}
	}
	if got := search.EscapeLike("plain"); got != "plain" {
		t.Errorf("EscapeLike changed a literal without wildcards: %q", got)
	}
}

func TestFTS5Query(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		query   string
		phrase  bool
		want    string
		wantErr error
	}{
		{"words are quoted and joined", `big invoice`, false, `"big" AND "invoice"`, nil},
		{"embedded quote doubled", `say "hi"`, false, `"say" AND """hi"""`, nil},
		{"phrase", `big invoice`, true, `"big invoice"`, nil},
		{"injection stays a word", `a" OR 1=1 --`, false, `"a""" AND "OR" AND "1=1" AND "--"`, nil},
		{"operators are not grammar", `a OR b NOT c`, false, `"a" AND "OR" AND "b" AND "NOT" AND "c"`, nil},
		{"collapses whitespace", "  big \n invoice ", false, `"big" AND "invoice"`, nil},
		{"empty", "   ", false, "", nil},
		{"control character", "a\x00b", false, "", search.ErrQueryInvalid},
		{"too long by runes", strings.Repeat("ü", 257), false, "", search.ErrQueryTooLong},
		{"too long", strings.Repeat("a", 300), false, "", search.ErrQueryTooLong},
		{"non-ascii counts runes", "ü", false, `"ü"`, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := search.FTS5Query(tc.query, tc.phrase)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("FTS5Query() error = %v, want %v", err, tc.wantErr)
			}
			if got != tc.want {
				t.Fatalf("FTS5Query() = %q, want %q", got, tc.want)
			}
		})
	}
}
