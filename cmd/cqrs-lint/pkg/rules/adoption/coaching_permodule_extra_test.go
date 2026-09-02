package adoption

import (
	"context"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
	"github.com/larsartmann/go-finding"
)

// multiModuleCtx builds a standard two-module workspace: a library module
// (no server) and an example app module (has server). FeatureProfiles are
// set per-module. The primary profile matches the example so server-only
// rules would falsely fire for the library under old workspace-global behavior.
func multiModuleCtx(t *testing.T, libSrc, exampleSrc string) *analyzer.AnalysisContext {
	t.Helper()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"/repo/lib/events.go":        libSrc,
		"/repo/examples/app/main.go": exampleSrc,
	})

	ctx.FeatureProfiles = map[string]analyzer.FeatureProfile{
		"/repo/lib":          {HasServer: false},
		"/repo/examples/app": {HasServer: true},
	}
	ctx.FeatureProfile = analyzer.FeatureProfile{HasServer: true}

	return ctx
}

func assertFindings(
	t *testing.T,
	ruleID string,
	findings []finding.Finding,
	err error,
	want int,
) {
	t.Helper()

	if err != nil {
		t.Fatalf("%s detect: %v", ruleID, err)
	}

	if len(findings) != want {
		t.Errorf("%s: expected %d finding(s), got %d", ruleID, want, len(findings))

		for _, f := range findings {
			t.Logf("  finding: %s @ %s", f.Message, f.Position.File)
		}
	}
}

// --- Server-suppressed rules ---

func TestF004_PerModuleSuppressesLibraryModule(t *testing.T) {
	t.Parallel()

	libSrc := `package lib
import _ "github.com/larsartmann/go-cqrs-lite/event/v4"
type UserCreated struct{ Name string }
`
	exampleSrc := `package main
import "net/http"
func main() { _ = http.ListenAndServe(":8080", nil) }
`
	ctx := multiModuleCtx(t, libSrc, exampleSrc)
	det := NewF004Detector(ctx)
	findings, err := det.Detect(context.Background())
	assertFindings(t, "F004", findings, err, 1)
}

func TestF009_PerModuleSuppressesLibraryModule(t *testing.T) {
	t.Parallel()

	libSrc := `package lib
func DoStuff() {}
`
	exampleSrc := `package main
import (
	"net/http"
	"time"
)
func main() {
	_ = time.AfterFunc(30*time.Second, func() {})
	_ = http.ListenAndServe(":8080", nil)
}
`
	ctx := multiModuleCtx(t, libSrc, exampleSrc)
	det := NewF009Detector(ctx)
	findings, err := det.Detect(context.Background())
	assertFindings(t, "F009", findings, err, 1)
}

func TestF027_PerModuleSuppressesLibraryModule(t *testing.T) {
	t.Parallel()

	libSrc := `package lib
import _ "github.com/larsartmann/go-cqrs-lite/otel/v4"
`
	exampleSrc := `package main
import (
	"net/http"
	_ "github.com/larsartmann/go-cqrs-lite/otel/v4"
)
func main() { _ = http.ListenAndServe(":8080", nil) }
`
	ctx := multiModuleCtx(t, libSrc, exampleSrc)
	det := NewF027Detector(ctx)
	findings, err := det.Detect(context.Background())
	assertFindings(t, "F027", findings, err, 1)
}

func TestF028_PerModuleSuppressesLibraryModule(t *testing.T) {
	t.Parallel()

	libSrc := `package lib
`
	exampleSrc := `package main
import (
	"net/http"
	"log/slog"
)
func main() {
	_ = slog.Info("test")
	_ = http.ListenAndServe(":8080", nil)
}
`
	ctx := multiModuleCtx(t, libSrc, exampleSrc)
	det := NewF028Detector(ctx)
	findings, err := det.Detect(context.Background())
	assertFindings(t, "F028", findings, err, 1)
}

func TestF029_PerModuleSuppressesLibraryModule(t *testing.T) {
	t.Parallel()

	libSrc := `package lib
import _ "github.com/larsartmann/go-cqrs-lite/otel/v4"
`
	exampleSrc := `package main
import (
	"net/http"
	_ "github.com/larsartmann/go-cqrs-lite/otel/v4"
)
func main() { _ = http.ListenAndServe(":8080", nil) }
`
	ctx := multiModuleCtx(t, libSrc, exampleSrc)
	det := NewF029Detector(ctx)
	findings, err := det.Detect(context.Background())
	assertFindings(t, "F029", findings, err, 1)
}

// --- CommandFlow-suppressed rules ---

func TestF007_PerModuleSuppressesLibraryModule(t *testing.T) {
	t.Parallel()

	libSrc := `package lib
func NewDispatcher() {}
`
	exampleSrc := `package main
func main() {
	d := NewDispatcher()
	d.Use()
}
`
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"/repo/lib/dispatcher.go":    libSrc,
		"/repo/examples/app/main.go": exampleSrc,
	})
	ctx.FeatureProfiles = map[string]analyzer.FeatureProfile{
		"/repo/lib":          {CommandFlow: analyzer.CommandFlowReadOnly},
		"/repo/examples/app": {CommandFlow: analyzer.CommandFlowCommands},
	}
	ctx.FeatureProfile = analyzer.FeatureProfile{CommandFlow: analyzer.CommandFlowCommands}

	det := NewF007Detector(ctx)
	findings, err := det.Detect(context.Background())
	assertFindings(t, "F007", findings, err, 1)
}

func TestF012_PerModuleSuppressesLibraryModule(t *testing.T) {
	t.Parallel()

	libSrc := `package lib
`
	exampleSrc := `package main
func main() {
	bus.SubscribeAll(func() {})
}
`
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"/repo/lib/events.go":        libSrc,
		"/repo/examples/app/main.go": exampleSrc,
	})
	ctx.FeatureProfiles = map[string]analyzer.FeatureProfile{
		"/repo/lib":          {CommandFlow: analyzer.CommandFlowReadOnly},
		"/repo/examples/app": {CommandFlow: analyzer.CommandFlowCommands},
	}
	ctx.FeatureProfile = analyzer.FeatureProfile{CommandFlow: analyzer.CommandFlowCommands}

	det := NewF012Detector(ctx)
	findings, err := det.Detect(context.Background())
	assertFindings(t, "F012", findings, err, 1)
}

// --- Bus-suppressed rules ---

func TestF017_PerModuleSuppressesLibraryModule(t *testing.T) {
	t.Parallel()

	libSrc := `package lib
`
	exampleSrc := `package main
func main() {
	bus.Subscribe(func() {})
}
`
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"/repo/lib/events.go":        libSrc,
		"/repo/examples/app/main.go": exampleSrc,
	})
	ctx.FeatureProfiles = map[string]analyzer.FeatureProfile{
		"/repo/lib":          {HasAsyncBus: false},
		"/repo/examples/app": {HasAsyncBus: true},
	}
	ctx.FeatureProfile = analyzer.FeatureProfile{HasAsyncBus: true}

	det := NewF017Detector(ctx)
	findings, err := det.Detect(context.Background())
	assertFindings(t, "F017", findings, err, 1)
}

// --- Store-isolated rules (F023, F024, F025) ---

func TestF023_PerModuleStoreIsolation(t *testing.T) {
	t.Parallel()

	libSrc := `package lib
type Item struct{ Name string }
func filterItems(items []Item) []Item {
	var out []Item
	for _, i := range items {
		if i.Name != "" { out = append(out, i) }
	}
	return out
}
`
	exampleSrc := `package main
func main() {
	items := []string{"b", "a"}
	var out []string
	for _, i := range items {
		if i != "" { out = append(out, i) }
	}
	_ = out
}
`
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"/repo/lib/filter.go":        libSrc,
		"/repo/examples/app/main.go": exampleSrc,
	})
	ctx.FeatureProfiles = map[string]analyzer.FeatureProfile{
		"/repo/lib":          {Store: analyzer.StoreNone},
		"/repo/examples/app": {Store: analyzer.StoreSQLite},
	}
	ctx.FeatureProfile = analyzer.FeatureProfile{Store: analyzer.StoreNone}

	det := NewF023Detector(ctx)
	findings, err := det.Detect(context.Background())
	assertFindings(t, "F023", findings, err, 1)
}

func TestF024_PerModuleStoreIsolation(t *testing.T) {
	t.Parallel()

	libSrc := `package lib
func paginate(items []string, offset, limit int) []string {
	return items[offset : offset+limit]
}
`
	exampleSrc := `package main
func main() {
	items := []string{"a", "b", "c"}
	offset := 0
	limit := 10
	_ = items[offset : offset+limit]
}
`
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"/repo/lib/paginate.go":      libSrc,
		"/repo/examples/app/main.go": exampleSrc,
	})
	ctx.FeatureProfiles = map[string]analyzer.FeatureProfile{
		"/repo/lib":          {Store: analyzer.StoreNone},
		"/repo/examples/app": {Store: analyzer.StoreSQLite},
	}
	ctx.FeatureProfile = analyzer.FeatureProfile{Store: analyzer.StoreNone}

	det := NewF024Detector(ctx)
	findings, err := det.Detect(context.Background())
	assertFindings(t, "F024", findings, err, 1)
}

func TestF025_PerModuleStoreIsolation(t *testing.T) {
	t.Parallel()

	libSrc := `package lib
func countItems(items []string) int {
	count := 0
	for range items { count++ }
	return count
}
`
	exampleSrc := `package main
func main() {
	items := []string{"a", "b", "c"}
	count := 0
	for range items { count++ }
	_ = count
}
`
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"/repo/lib/count.go":         libSrc,
		"/repo/examples/app/main.go": exampleSrc,
	})
	ctx.FeatureProfiles = map[string]analyzer.FeatureProfile{
		"/repo/lib":          {Store: analyzer.StoreNone},
		"/repo/examples/app": {Store: analyzer.StoreSQLite},
	}
	ctx.FeatureProfile = analyzer.FeatureProfile{Store: analyzer.StoreNone}

	det := NewF025Detector(ctx)
	findings, err := det.Detect(context.Background())
	assertFindings(t, "F025", findings, err, 1)
}

// --- Metaengine-isolated rule ---

func TestF026_PerModuleMetaengineIsolation(t *testing.T) {
	t.Parallel()

	libSrc := `package lib
import _ "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
`
	exampleSrc := `package main
import "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
func main() {
	metaengine.NewReader[string](nil, "col")
}
`
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"/repo/lib/meta.go":          libSrc,
		"/repo/examples/app/main.go": exampleSrc,
	})
	ctx.FeatureProfiles = map[string]analyzer.FeatureProfile{
		"/repo/lib":          {HasMetaengine: true},
		"/repo/examples/app": {HasMetaengine: true},
	}
	ctx.FeatureProfile = analyzer.FeatureProfile{HasMetaengine: true}

	det := NewF026Detector(ctx)
	findings, err := det.Detect(context.Background())
	assertFindings(t, "F026", findings, err, 1)
}

// --- PII detection per-module (F006) ---

func TestF006_PerModuleEncryptionImportIsolation(t *testing.T) {
	t.Parallel()

	libSrc := `package lib
type UserCreated struct {
	Email string
	Name  string
}
`
	exampleSrc := `package main
import _ "github.com/larsartmann/go-cqrs-lite/encryption/v4"
func main() {}
`
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"/repo/lib/events.go":        libSrc,
		"/repo/examples/app/main.go": exampleSrc,
	})
	ctx.FeatureProfiles = map[string]analyzer.FeatureProfile{
		"/repo/lib":          {},
		"/repo/examples/app": {},
	}

	det := NewF006Detector(ctx)
	findings, err := det.Detect(context.Background())
	assertFindings(t, "F006", findings, err, 1)
}

// --- CBOR codec per-module (F008) ---

func TestF008_PerModuleEventCountIsolation(t *testing.T) {
	t.Parallel()

	libEvents := ""
	var libEventsSb404 strings.Builder
	for i := 0; i < 5; i++ {
		libEventsSb404.WriteString("event.New(\"agg.event" + itoa(i) + "\", nil)\n")
	}
	libEvents += libEventsSb404.String()

	libSrc := "package lib\n" +
		"import \"github.com/larsartmann/go-cqrs-lite/event/v4\"\n" +
		"import \"github.com/larsartmann/go-codec\"\n" +
		"func init() {\n" +
		"\t_ = codec.JSONCodec{}\n" +
		libEvents +
		"}\n"

	exampleSrc := `package main
import "github.com/larsartmann/go-codec"
func main() { _ = codec.CBORCodec{} }
`
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"/repo/lib/events.go":        libSrc,
		"/repo/examples/app/main.go": exampleSrc,
	})
	ctx.FeatureProfiles = map[string]analyzer.FeatureProfile{
		"/repo/lib":          {},
		"/repo/examples/app": {},
	}

	det := NewF008Detector(ctx)
	findings, err := det.Detect(context.Background())
	assertFindings(t, "F008", findings, err, 1)
}

// --- Graph traversal per-module (F010) ---

func TestF010_PerModuleGraphImportIsolation(t *testing.T) {
	t.Parallel()

	libSrc := `package lib
func FindAncestors(id string) []string { return nil }
`
	exampleSrc := `package main
import _ "github.com/larsartmann/go-cqrs-lite/graph/v4"
func main() {}
`
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"/repo/lib/traverse.go":      libSrc,
		"/repo/examples/app/main.go": exampleSrc,
	})
	ctx.FeatureProfiles = map[string]analyzer.FeatureProfile{
		"/repo/lib":          {},
		"/repo/examples/app": {},
	}

	det := NewF010Detector(ctx)
	findings, err := det.Detect(context.Background())
	assertFindings(t, "F010", findings, err, 1)
}

// --- F015 query count per-module ---

func TestF015_PerModuleQueryCountIsolation(t *testing.T) {
	t.Parallel()

	libSrc := `package lib
func init() {
	query.Register("q1", nil)
	query.Register("q2", nil)
	query.Register("q3", nil)
}
`
	exampleSrc := `package main
import _ "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
func main() {}
`
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"/repo/lib/queries.go":       libSrc,
		"/repo/examples/app/main.go": exampleSrc,
	})
	ctx.FeatureProfiles = map[string]analyzer.FeatureProfile{
		"/repo/lib":          {},
		"/repo/examples/app": {},
	}

	det := NewF015Detector(ctx)
	findings, err := det.Detect(context.Background())
	assertFindings(t, "F015", findings, err, 1)
}

// --- F016 aggregate count per-module ---

func TestF016_PerModuleAggregateCountIsolation(t *testing.T) {
	t.Parallel()

	libEvents := ""
	var libEventsSb496 strings.Builder
	for _, agg := range []string{"user", "order", "product", "payment", "shipping"} {
		libEventsSb496.WriteString("event.New(\"" + agg + ".created\", nil)\n")
	}
	libEvents += libEventsSb496.String()

	libSrc := "package lib\n" +
		"import \"github.com/larsartmann/go-cqrs-lite/event/v4\"\n" +
		"func init() {\n" + libEvents + "}\n"

	exampleSrc := `package main
import _ "github.com/larsartmann/go-cqrs-lite/listing/v4"
func main() {}
`
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"/repo/lib/events.go":        libSrc,
		"/repo/examples/app/main.go": exampleSrc,
	})
	ctx.FeatureProfiles = map[string]analyzer.FeatureProfile{
		"/repo/lib":          {},
		"/repo/examples/app": {},
	}

	det := NewF016Detector(ctx)
	findings, err := det.Detect(context.Background())
	assertFindings(t, "F016", findings, err, 1)
}

// --- F018 FilterOn per-module ---

func TestF018_PerModuleMetaengineImportIsolation(t *testing.T) {
	t.Parallel()

	libSrc := `package lib
import "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
func init() {
	metaengine.FilterOn(func() {})
}
`
	exampleSrc := `package main
import "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
func main() {
	metaengine.FilterOnField("col", nil)
}
`
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"/repo/lib/meta.go":          libSrc,
		"/repo/examples/app/main.go": exampleSrc,
	})
	ctx.FeatureProfiles = map[string]analyzer.FeatureProfile{
		"/repo/lib":          {},
		"/repo/examples/app": {},
	}

	det := NewF018Detector(ctx)
	findings, err := det.Detect(context.Background())
	assertFindings(t, "F018", findings, err, 1)
}
