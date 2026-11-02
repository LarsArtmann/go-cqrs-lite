package architecture

import (
	"go/token"
	"testing"
)

func TestE024IsDisplayImportPath(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		path string
		name string
		want bool
	}{
		"std log":                     {path: "log", name: "Printf", want: true},
		"slog info":                   {path: "log/slog", name: "Info", want: true},
		"slog attribute constructor":  {path: "log/slog", name: "String", want: true},
		"otel attribute":              {path: "go.opentelemetry.io/otel/attribute", name: "String", want: true},
		"stub attribute":              {path: "test.example/stub/attribute", name: "String", want: true},
		"fmt errorf":                  {path: "fmt", name: "Errorf", want: true},
		"fmt sprintf":                 {path: "fmt", name: "Sprintf", want: false},
		"strings":                     {path: "strings", name: "ToUpper", want: false},
		"id parse round-trip":         {path: "test.example/stub/go-cqrs-lite/id", name: "ParseStreamID", want: true},
		"id parse versioned module":   {path: "github.com/larsartmann/go-cqrs-lite/id/v4", name: "ParseActorID", want: true},
		"id parse strict":             {path: "github.com/larsartmann/go-cqrs-lite/id/v4", name: "ParseStreamIDStrict", want: true},
		"id non-parse constructor":    {path: "github.com/larsartmann/go-cqrs-lite/id/v4", name: "NewStreamID", want: false},
		"unrelated parse":             {path: "example.com/other", name: "ParseThing", want: false},
		"empty path":                  {path: "", name: "Info", want: false},
		"attribute-looking last word": {path: "example.com/myattribute", name: "String", want: false},
		"versioned attribute package": {path: "example.com/log/attribute/v2", name: "String", want: false},
	}

	for label, tc := range cases {
		if got := e024IsDisplayImportPath(tc.path, tc.name); got != tc.want {
			t.Errorf("e024IsDisplayImportPath(%q, %q) [%s] = %v, want %v", tc.path, tc.name, label, got, tc.want)
		}
	}
}

func TestE024DisplayQualifierFallback(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		ident string
		sel   string
		want  bool
	}{
		"slog any callee": {ident: "slog", sel: "Debug", want: true},
		"log any callee":  {ident: "log", sel: "Printf", want: true},
		"attribute":       {ident: "attribute", sel: "String", want: true},
		"fmt Errorf":      {ident: "fmt", sel: "Errorf", want: true},
		"fmt Sprintf":     {ident: "fmt", sel: "Sprintf", want: false},
		"strings":         {ident: "strings", sel: "Builder", want: false},
		"unknown":         {ident: "zeta", sel: "Info", want: false},
	}

	for label, tc := range cases {
		if got := e024DisplayQualifierFallback(tc.ident, tc.sel); got != tc.want {
			t.Errorf("e024DisplayQualifierFallback(%q, %q) [%s] = %v, want %v", tc.ident, tc.sel, label, got, tc.want)
		}
	}
}

func TestE024TokenRangeContains(t *testing.T) {
	t.Parallel()

	r := tokenRange{start: token.Pos(10), end: token.Pos(20)}

	for pos, want := range map[token.Pos]bool{
		9:  false,
		10: true,
		15: true,
		20: true,
		21: false,
	} {
		if got := r.contains(pos); got != want {
			t.Errorf("range(10,20).contains(%d) = %v, want %v", pos, got, want)
		}
	}
}
