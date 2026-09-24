package eventcatalog_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
	"github.com/larsartmann/go-cqrs-lite/catalog/v4/eventcatalog"
	"github.com/larsartmann/go-cqrs-lite/catalog/v4/internal/cattest"
)

// TestExporter_DefaultWritesBootstrapFiles pins the default contract: local
// export directories stay directly buildable (npm install && eventcatalog
// build), so both bootstrap files must exist.
func TestExporter_DefaultWritesBootstrapFiles(t *testing.T) {
	tmpDir := exportToTempDir(t, cattest.BuildTestCatalog())

	for _, bootstrap := range []string{"eventcatalog.config.js", "package.json"} {
		if _, err := os.Stat(filepath.Join(tmpDir, bootstrap)); err != nil {
			t.Errorf("default export should write %s: %v", bootstrap, err)
		}
	}
}

// TestExporter_SkipBootstrapFilesOmitsOnlyBootstrap pins the option's
// contract: exactly the two bootstrap files disappear; every resource, the
// manifest, and the AI-consumption files are untouched.
func TestExporter_SkipBootstrapFilesOmitsOnlyBootstrap(t *testing.T) {
	cat := cattest.BuildTestCatalog()
	tmpDir := t.TempDir()

	exp := eventcatalog.NewExporter(tmpDir, eventcatalog.WithSkipBootstrapFiles())
	if err := exp.Export(cat); err != nil {
		t.Fatalf("export: %v", err)
	}

	for _, bootstrap := range []string{"eventcatalog.config.js", "package.json"} {
		if _, err := os.Stat(filepath.Join(tmpDir, bootstrap)); !os.IsNotExist(err) {
			t.Errorf("skip-bootstrap export must not write %s (stat err: %v)", bootstrap, err)
		}
	}

	stillThere := []string{
		"services/order-svc/index.mdx",
		"events/OrderCreated/index.mdx",
		"catalog.index.json",
		"llms.txt",
		"schemas.txt",
	}
	for _, rel := range stillThere {
		if _, err := os.Stat(filepath.Join(tmpDir, filepath.FromSlash(rel))); err != nil {
			t.Errorf("skip-bootstrap export must still write %s: %v", rel, err)
		}
	}
}

// TestExporter_SkipBootstrapFilesLeavesContentIdentical proves the option
// never changes CONTENT — a hub merging a skip-bootstrap export gets exactly
// the same MDX bytes as a default export.
func TestExporter_SkipBootstrapFilesLeavesContentIdentical(t *testing.T) {
	cat := cattest.BuildTestCatalog()

	defaultDir := exportToTempDir(t, cat)
	skipDir := t.TempDir()

	exp := eventcatalog.NewExporter(skipDir, eventcatalog.WithSkipBootstrapFiles())
	if err := exp.Export(cat); err != nil {
		t.Fatalf("export: %v", err)
	}

	rel := filepath.Join("events", "OrderCreated", "index.mdx")

	defaultContent, err := os.ReadFile(filepath.Join(defaultDir, rel))
	if err != nil {
		t.Fatalf("read default export: %v", err)
	}

	skipContent, err := os.ReadFile(filepath.Join(skipDir, rel))
	if err != nil {
		t.Fatalf("read skip-bootstrap export: %v", err)
	}

	if string(defaultContent) != string(skipContent) {
		t.Errorf(
			"skip-bootstrap export changed content of %s:\ndefault:\n%s\nskip:\n%s",
			rel, defaultContent, skipContent,
		)
	}
}

// plainRefExport registers one service (version 1.2.3) whose event is
// auto-derived onto a channel, exports it with the given options, and
// returns the event and channel MDX.
func plainRefExport(t *testing.T, opts ...eventcatalog.Option) (eventMDX, channelMDX string) {
	t.Helper()

	reg := cattest.NewTestRegistry(catalog.Service{
		ID: "order-svc", Name: "Order Service", Version: "1.2.3", Summary: "orders",
	})
	reg.AddEvent("order-svc", catalog.Message{
		Kind: catalog.EventMessage, ID: "OrderPlaced", Name: "Order Placed",
		Version: "1.0.0", Summary: "placed", Direction: catalog.Sends,
		Channels: []catalog.ChannelID{"orders"},
	})
	reg.AddChannel(catalog.Channel{
		ID: "orders", Name: "Orders", Version: "1.0.0", Summary: "order messages",
		Messages: []catalog.MessageID{"OrderPlaced"},
	})

	tmpDir := t.TempDir()
	if err := eventcatalog.NewExporter(tmpDir, opts...).Export(reg.Build()); err != nil {
		t.Fatalf("export: %v", err)
	}

	read := func(rel string) string {
		t.Helper()

		data, err := os.ReadFile(filepath.Join(tmpDir, rel))
		if err != nil {
			t.Fatalf("read %s: %v", rel, err)
		}

		return string(data)
	}

	return read(filepath.Join("events", "OrderPlaced", "index.mdx")),
		read(filepath.Join("channels", "orders", "index.mdx"))
}

// TestExporter_DefaultUsesCompositeRefIDs pins the @eventcatalog/core
// contract: producers/consumers and channel message pointers carry the
// "<id>-<version>" Astro entry IDs the core content layer resolves.
func TestExporter_DefaultUsesCompositeRefIDs(t *testing.T) {
	eventMDX, channelMDX := plainRefExport(t)

	if !strings.Contains(eventMDX, "- order-svc-1.2.3") {
		t.Errorf("default producers should be composite entry IDs, got:\n%s", eventMDX)
	}

	if !strings.Contains(channelMDX, "id: OrderPlaced-1.0.0") {
		t.Errorf("default channel pointer should be composite entry ID, got:\n%s", channelMDX)
	}
}

// TestExporter_PlainRefIDsEmitsBareRefs pins the @eventcatalog/linter
// contract: refs are bare frontmatter IDs (the linter keys its index by id
// and can never resolve the composite form).
func TestExporter_PlainRefIDsEmitsBareRefs(t *testing.T) {
	eventMDX, channelMDX := plainRefExport(t, eventcatalog.WithPlainRefIDs())

	if !strings.Contains(eventMDX, "- order-svc") {
		t.Errorf("plain producers should be bare service IDs, got:\n%s", eventMDX)
	}

	if strings.Contains(eventMDX, "order-svc-1.2.3") {
		t.Errorf("plain producers must not carry the version suffix, got:\n%s", eventMDX)
	}

	if !strings.Contains(channelMDX, "id: OrderPlaced\n") {
		t.Errorf("plain channel pointer should be the bare message ID, got:\n%s", channelMDX)
	}

	if strings.Contains(channelMDX, "id: OrderPlaced-1.0.0") {
		t.Errorf("plain channel pointer must not carry the version suffix, got:\n%s", channelMDX)
	}

	if !strings.Contains(channelMDX, "version: 1.0.0") {
		t.Errorf("channel pointer must keep the separate version field, got:\n%s", channelMDX)
	}
}
