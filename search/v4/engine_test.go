package search_test

import (
	"context"
	"errors"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/search/v4"
)

// fakeEngine is an Engine with scripted capabilities.
type fakeEngine struct {
	caps     search.Capabilities
	embedder search.Embedder
}

func (e fakeEngine) Capabilities() search.Capabilities { return e.caps }
func (e fakeEngine) Embedder() search.Embedder         { return e.embedder }

// hashEmbedder is a deterministic embedder for tests: a text's vector is a
// function of its bytes only.
type hashEmbedder struct{}

func (hashEmbedder) Name() string  { return "test" }
func (hashEmbedder) Model() string { return "hash-1" }
func (hashEmbedder) Embed(_ context.Context, texts []string) ([][]float32, error) {
	out := make([][]float32, len(texts))
	for i, text := range texts {
		var sum float32
		for _, r := range text {
			sum += float32(r)
		}
		out[i] = []float32{sum}
	}

	return out, nil
}

var (
	nothing   = search.Capabilities{}
	fullText  = search.Capabilities{FullText: true}
	vector    = search.Capabilities{Vector: true}
	everythin = search.Capabilities{FullText: true, Vector: true}
)

func TestTierAuto(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		terms       []string
		caps        search.Capabilities
		hasEmbedder bool
		want        search.Via
	}{
		{"rule 1: one character", []string{"a"}, everythin, true, search.ViaNone},
		{"rule 1: no terms", nil, everythin, true, search.ViaNone},
		{"rule 1: two characters start a search", []string{"ab"}, nothing, false, search.ViaScan},
		{"rule 2: identifier-like", []string{"+49", "160", "1234567"}, everythin, true, search.ViaFullText},
		{"rule 2: deed number", []string{"KW1/2024/0042"}, everythin, true, search.ViaFullText},
		{"rule 3: single short token", []string{"acme"}, everythin, true, search.ViaFullText},
		{"rule 3: single token without full-text index", []string{"acme"}, nothing, true, search.ViaScan},
		{"rule 4: prose with everything", []string{"unpaid", "invoices"}, everythin, true, search.ViaHybrid},
		{"rule 4: prose without full-text index", []string{"unpaid", "invoices"}, vector, true, search.ViaVector},
		{
			"rule 4: prose, embedder but no vector index",
			[]string{"unpaid", "invoices"},
			fullText,
			true,
			search.ViaFullText,
		},
		{"rule 4: prose, no embedder", []string{"unpaid", "invoices"}, everythin, false, search.ViaFullText},
		{"prose without anything scans", []string{"unpaid", "invoices"}, nothing, false, search.ViaScan},
		{"single long word is prose", []string{"invoices"}, everythin, true, search.ViaHybrid},
		{"single long identifier stays lexical", []string{"2024041234567"}, everythin, true, search.ViaFullText},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := search.Tier(search.ModeAuto, tc.terms, tc.caps, tc.hasEmbedder)
			if err != nil || got != tc.want {
				t.Fatalf("Tier() = %v, %v; want %v, nil", got, err, tc.want)
			}
		})
	}
}

func TestTierExplicitModesAreStrict(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		mode        search.Mode
		caps        search.Capabilities
		hasEmbedder bool
		want        search.Via
		wantErr     error
	}{
		{"lexical scans without an index", search.ModeLexical, nothing, false, search.ViaScan, nil},
		{"lexical uses full-text when offered", search.ModeLexical, fullText, false, search.ViaFullText, nil},
		{"fulltext strict", search.ModeFullText, fullText, false, search.ViaFullText, nil},
		{"fulltext without an index", search.ModeFullText, nothing, false, search.ViaNone, search.ErrModeUnavailable},
		{"vector strict", search.ModeVector, vector, true, search.ViaVector, nil},
		{"vector without an embedder", search.ModeVector, vector, false, search.ViaNone, search.ErrModeUnavailable},
		{"vector without an index", search.ModeVector, fullText, true, search.ViaNone, search.ErrModeUnavailable},
		{"hybrid strict", search.ModeHybrid, everythin, true, search.ViaHybrid, nil},
		{"hybrid without vector index", search.ModeHybrid, fullText, true, search.ViaNone, search.ErrModeUnavailable},
		{"hybrid without full-text", search.ModeHybrid, vector, true, search.ViaNone, search.ErrModeUnavailable},
		{"hybrid without embedder", search.ModeHybrid, everythin, false, search.ViaNone, search.ErrModeUnavailable},
		{"unknown mode", search.Mode(8), everythin, true, search.ViaNone, search.ErrModeUnavailable},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := search.Tier(tc.mode, []string{"unpaid", "invoices"}, tc.caps, tc.hasEmbedder)
			if !errors.Is(err, tc.wantErr) || got != tc.want {
				t.Fatalf("Tier() = %v, %v; want %v, %v", got, err, tc.want, tc.wantErr)
			}
		})
	}
}

func TestNoEngine(t *testing.T) {
	t.Parallel()

	if search.NoEngine().Capabilities() != nothing || search.NoEngine().Embedder() != nil {
		t.Fatal("NoEngine must offer nothing and embed nothing")
	}
}
