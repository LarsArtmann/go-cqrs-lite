package api

import (
	"go/ast"
	"strings"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/rules/lintutil"
)

// deprecatedAPIEntry pairs a deprecated call target with its replacement.
// Every row is a symbol whose doc comment in the named go-cqrs-lite module
// carries a `Deprecated:` marker; the table is a claim about the library,
// so a row without that marker is a false "deprecated" warning.
type deprecatedAPIEntry struct {
	pathFragment string // exact go-cqrs-lite module path owning the symbol (version-less)
	symbol       string // selector name being called (incl. type conversions)
	replacement  string // sanctioned replacement, shown in the suggestion
}

var deprecatedAPIEntries = []deprecatedAPIEntry{ //nolint:gochecknoglobals // static lookup table
	{pathFragment: "go-cqrs-lite/command", symbol: "ParseType", replacement: "record.ParseType(s, command.ErrEmptyCommandType)"},
	{pathFragment: "go-cqrs-lite/command", symbol: "ParseAggregateType", replacement: "command.ParseStreamType"},
	{pathFragment: "go-cqrs-lite/command", symbol: "NewAggregateRef", replacement: "command.NewStreamRef"},
	{pathFragment: "go-cqrs-lite/command", symbol: "AggregateType", replacement: "command.StreamType"},
	{pathFragment: "go-cqrs-lite/command", symbol: "AggregateRef", replacement: "command.StreamRef"},
	{pathFragment: "go-cqrs-lite/event", symbol: "ParseType", replacement: "record.ParseType(s, event.ErrEmptyEventType)"},
	{pathFragment: "go-cqrs-lite/event", symbol: "ParseAggregateType", replacement: "id.ParseStreamType"},
	{pathFragment: "go-cqrs-lite/event", symbol: "NewAggregateRef", replacement: "id.NewStreamRef"},
	{pathFragment: "go-cqrs-lite/event", symbol: "AggregateType", replacement: "event.StreamType"},
	{pathFragment: "go-cqrs-lite/event", symbol: "AggregateID", replacement: "event.StreamID"},
	{pathFragment: "go-cqrs-lite/event", symbol: "AggregateRef", replacement: "event.StreamRef"},
	{pathFragment: "go-cqrs-lite/event", symbol: "DetectTombstone", replacement: "domain deletion/restore events (ADR-0114)"},
	{pathFragment: "go-cqrs-lite/query", symbol: "ParseType", replacement: "record.ParseType(s, query.ErrEmptyQueryType)"},
	{pathFragment: "go-cqrs-lite/listing", symbol: "NewInMemoryAggregateReader", replacement: "listing.NewInMemoryStreamReader"},
}

// matchDeprecatedAPI reports the deprecated-API entry matching a call of
// selName through qualifier. The qualifier must resolve (via the file's
// imports) to EXACTLY the entry's module path — a substring match made
// "go-cqrs-lite/command" also own commandlifecycle.
func matchDeprecatedAPI(file *ast.File, qualifier, selName string) (deprecatedAPIEntry, bool) {
	path, ok := lintutil.QualifierToImportPath(file, qualifier)
	if !ok {
		return deprecatedAPIEntry{}, false
	}

	path = analyzer.StripVersionSegments(path)

	for _, e := range deprecatedAPIEntries {
		if e.symbol == selName && strings.HasSuffix(path, "/"+e.pathFragment) {
			return e, true
		}
	}

	return deprecatedAPIEntry{}, false
}
