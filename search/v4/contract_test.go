package search_test

import (
	"errors"
	"testing"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/search/v4"
)

func TestModeStringAndValidity(t *testing.T) {
	t.Parallel()

	cases := map[search.Mode]string{
		search.ModeAuto:     "auto",
		search.ModeLexical:  "lexical",
		search.ModeFullText: "fulltext",
		search.ModeVector:   "vector",
		search.ModeHybrid:   "hybrid",
		search.Mode(9):      "mode(9)",
	}
	for mode, want := range cases {
		if got := mode.String(); got != want {
			t.Errorf("Mode(%d).String() = %q, want %q", mode, got, want)
		}
		if valid := mode.IsValid(); valid != (mode <= search.ModeHybrid) {
			t.Errorf("Mode(%d).IsValid() = %v", mode, valid)
		}
	}
}

func TestViaStringAndPredicates(t *testing.T) {
	t.Parallel()

	cases := map[search.Via]string{
		search.ViaNone:     "none",
		search.ViaScan:     "scan",
		search.ViaFullText: "fulltext",
		search.ViaVector:   "vector",
		search.ViaHybrid:   "hybrid",
		search.Via(7):      "via(7)",
	}
	for via, want := range cases {
		if got := via.String(); got != want {
			t.Errorf("Via(%d).String() = %q, want %q", via, got, want)
		}
		if valid := via.IsValid(); valid != (via <= search.ViaHybrid) {
			t.Errorf("Via(%d).IsValid() = %v", via, valid)
		}
	}
	text := map[search.Via]bool{
		search.ViaNone: false, search.ViaScan: true, search.ViaFullText: true,
		search.ViaVector: true, search.ViaHybrid: true,
	}
	semantic := map[search.Via]bool{
		search.ViaNone: false, search.ViaScan: false, search.ViaFullText: false,
		search.ViaVector: true, search.ViaHybrid: true,
	}
	for via := range text {
		if via.IsText() != text[via] || via.IsSemantic() != semantic[via] {
			t.Errorf("Via(%s): IsText %v, IsSemantic %v", via, via.IsText(), via.IsSemantic())
		}
	}
}

// testSpec is the spec every test searches with: title first, body last.
func testSpec() search.Spec[Note] {
	return search.Spec[Note]{
		Fields: []search.Field[Note]{
			{Name: "title", Weight: 1, Value: func(n Note) string { return n.Title }},
			{Name: "body", Weight: 0.2, Value: func(n Note) string { return n.Body }},
		},
	}
}

func TestSpecValidate(t *testing.T) {
	t.Parallel()

	title := search.Field[Note]{Name: "title", Value: func(n Note) string { return n.Title }}
	body := search.Field[Note]{Name: "body", Weight: 0.2, Value: func(n Note) string { return n.Body }}
	cases := []struct {
		name    string
		spec    search.Spec[Note]
		wantErr error
	}{
		{"valid", testSpec(), nil},
		{"zero weight is the default", search.Spec[Note]{Fields: []search.Field[Note]{title, body}}, nil},
		{"no fields", search.Spec[Note]{}, search.ErrNoFields},
		{"duplicate name", search.Spec[Note]{Fields: []search.Field[Note]{title, title}}, search.ErrDuplicateField},
		{
			"negative weight",
			search.Spec[Note]{Fields: []search.Field[Note]{{Name: "x", Weight: -1, Value: title.Value}}},
			search.ErrInvalidWeight,
		},
		{"no name", search.Spec[Note]{Fields: []search.Field[Note]{{Value: title.Value}}}, search.ErrInvalidField},
		{
			"blank name",
			search.Spec[Note]{Fields: []search.Field[Note]{{Name: "  ", Value: title.Value}}},
			search.ErrInvalidField,
		},
		{"no value function", search.Spec[Note]{Fields: []search.Field[Note]{{Name: "x"}}}, search.ErrInvalidField},
		{
			"unknown mode",
			search.Spec[Note]{Fields: []search.Field[Note]{title}, Mode: search.Mode(8)},
			search.ErrModeUnavailable,
		},
		{
			"negative query limit",
			search.Spec[Note]{Fields: []search.Field[Note]{title}, MaxQueryRunes: -1},
			search.ErrInvalidLimit,
		},
		{
			"negative scan limit",
			search.Spec[Note]{Fields: []search.Field[Note]{title}, ScanLimit: -5},
			search.ErrInvalidLimit,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := tc.spec.Validate()
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("Validate() = %v, want %v", err, tc.wantErr)
			}
		})
	}
}

func TestSpecCloneSharesNothing(t *testing.T) {
	t.Parallel()

	less := func(a, b Note) int { return 0 }
	index, err := search.New(search.Spec[Note]{
		Fields: testSpec().Fields,
		Keys:   []string{"status"},
		Sorts:  map[string]func(a, b Note) int{"title": less},
	}, search.NoEngine(), search.Slice[Note]{})
	if err != nil {
		t.Fatal(err)
	}
	spec := index.Spec()
	copied := index.Spec()

	copied.Fields[0].Name = "changed"
	copied.Keys[0] = "changed"
	delete(copied.Sorts, "title")

	if spec.Fields[0].Name != "title" || spec.Keys[0] != "status" || len(spec.Sorts) != 1 {
		t.Fatalf("Spec() shares state with the index: %+v %v %v", spec.Fields[0].Name, spec.Keys, spec.Sorts)
	}
}

func TestRejectionsKeepTheirCode(t *testing.T) {
	t.Parallel()

	_, err := search.Request{Query: "abc"}.Normalize(2)
	if !errors.Is(err, search.ErrQueryTooLong) {
		t.Fatalf("Normalize() = %v, want ErrQueryTooLong", err)
	}
	if got := errorfamily.Code(err); got != "search.query_too_long" {
		t.Fatalf("code = %q, the detail message lost the sentinel's code", got)
	}
}
