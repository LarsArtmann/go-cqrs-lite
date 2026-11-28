package search_test

import (
	"errors"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/query/v4"
	"github.com/larsartmann/go-cqrs-lite/search/v4"
)

func TestRequestNormalize(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		request search.Request
		limit   int
		want    string
		wantErr error
	}{
		{"trims", search.Request{Query: "  big invoice  "}, 0, "big invoice", nil},
		{"collapses whitespace", search.Request{Query: "big\t\n  invoice"}, 0, "big invoice", nil},
		{"keeps quoted spaces", search.Request{Query: ` status:"in progress" `}, 0, `status:"in progress"`, nil},
		{"empty", search.Request{}, 0, "", nil},
		{"at the limit", search.Request{Query: "ab"}, 2, "ab", nil},
		{"over the limit", search.Request{Query: "abc"}, 2, "", search.ErrQueryTooLong},
		{"counts runes not bytes", search.Request{Query: "ää"}, 2, "ää", nil},
		{"control character", search.Request{Query: "a\x00b"}, 0, "", search.ErrQueryInvalid},
		{"newline is collapsed, not rejected", search.Request{Query: "a\nb"}, 0, "a b", nil},
		{"negative limit", search.Request{Query: "a"}, -1, "", search.ErrInvalidLimit},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := tc.request.Normalize(tc.limit)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("Normalize() error = %v, want %v", err, tc.wantErr)
			}
			if got.Query != tc.want {
				t.Fatalf("Query = %q, want %q", got.Query, tc.want)
			}
		})
	}
}

func TestRequestNormalizeKeepsTheRest(t *testing.T) {
	t.Parallel()

	filters := url.Values{"status": {"open"}}
	page := query.Pagination{Page: 3, PageSize: 10}
	got, err := search.Request{Query: " x ", Filters: filters, Sort: "title", Page: page}.Normalize(0)
	if err != nil {
		t.Fatal(err)
	}
	if got.Filters.Get("status") != "open" || got.Sort != "title" || got.Page != page {
		t.Fatalf("Normalize changed the rest of the request: %+v", got)
	}
}

func TestTokenize(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		query     string
		keys      []string
		wantTerms []string
		wantTok   map[string][]string
	}{
		{"plain words", "big invoice", nil, []string{"big", "invoice"}, nil},
		{
			"known key",
			"from:x big invoice",
			[]string{"from"},
			[]string{"big", "invoice"},
			map[string][]string{"from": {"x"}},
		},
		{"key is case-insensitive", "FROM:x", []string{"from"}, nil, map[string][]string{"from": {"x"}}},
		{"unknown key stays text", "amount:10 invoice", []string{"from"}, []string{"amount:10", "invoice"}, nil},
		{"no keys declared", "from:x", nil, []string{"from:x"}, nil},
		{
			"several values keep their order",
			"tag:a tag:b",
			[]string{"tag"},
			nil,
			map[string][]string{"tag": {"a", "b"}},
		},
		{
			"quoted value",
			`status:"in progress" x`,
			[]string{"status"},
			[]string{"x"},
			map[string][]string{"status": {"in progress"}},
		},
		{"quoted phrase stays terms", `"big invoice" x`, nil, []string{"big invoice", "x"}, nil},
		{"empty value demotes to term", "from: x", []string{"from"}, []string{"from:", "x"}, nil},
		{"malformed key demotes to term", "a=b:1", []string{"a=b"}, []string{"a=b:1"}, nil},
		{"empty key demotes", ":x", []string{""}, []string{":x"}, nil},
		{"non-ascii terms", "Über Straße", nil, []string{"Über", "Straße"}, nil},
		{
			"key with dash and underscore",
			"due-by_y:1",
			[]string{"due-by_y"},
			nil,
			map[string][]string{"due-by_y": {"1"}},
		},
		{"empty query", "", []string{"from"}, nil, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			terms, tokens := search.Tokenize(tc.query, tc.keys...)
			if !reflect.DeepEqual(terms, tc.wantTerms) {
				t.Errorf("terms = %q, want %q", terms, tc.wantTerms)
			}
			if len(tokens) != len(tc.wantTok) || (len(tc.wantTok) > 0 && !reflect.DeepEqual(tokens, tc.wantTok)) {
				t.Errorf("tokens = %v, want %v", tokens, tc.wantTok)
			}
		})
	}
}

func TestSpecTokensUsesDeclaredKeys(t *testing.T) {
	t.Parallel()

	terms, tokens := testSpec().Tokens("status:open invoice")
	if len(tokens) != 0 {
		t.Fatalf("a spec without keys split an operator: %v", tokens)
	}
	spec := testSpec()
	spec.Keys = []string{"status"}
	terms, tokens = spec.Tokens("status:open invoice")
	if !reflect.DeepEqual(terms, []string{"invoice"}) ||
		!reflect.DeepEqual(tokens, map[string][]string{"status": {"open"}}) {
		t.Fatalf("terms %q tokens %v", terms, tokens)
	}
}

func TestTokensNeverSplitOperatorAlone(t *testing.T) {
	t.Parallel()

	// Auto rule 1: only operator tokens means no text search.
	terms, _ := search.Tokenize("status:open tag:x", "status", "tag")
	if len(terms) != 0 {
		t.Fatalf("terms = %q, want none", terms)
	}
}

func TestCollapseWhitespaceIsIdempotent(t *testing.T) {
	t.Parallel()

	got, err := search.Request{Query: " a   b "}.Normalize(0)
	if err != nil {
		t.Fatal(err)
	}
	again, _ := got.Normalize(0)
	if again.Query != got.Query || strings.Contains(got.Query, "  ") {
		t.Fatalf("not idempotent: %q then %q", got.Query, again.Query)
	}
}
