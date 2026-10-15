package eventcatalog_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4/eventcatalog"
	"github.com/larsartmann/go-cqrs-lite/catalog/v4/internal/cattest"
)

// errorflowFixture is a schemaVersion-1 erraudit errorflow export covering
// the columns the catalog projects: coded family errors, a wrap child, an
// uncoded sentinel, and a dead-sentinel candidate.
const errorflowFixture = `{
  "schemaVersion": 1,
  "package": "./example/demo",
  "filesScanned": 6,
  "errors": [
    {
      "name": "ErrCheckout",
      "kind": "family.new",
      "code": "checkout.failed",
      "family": "rejection",
      "message": "checkout failed",
      "declared": "errors.go:10",
      "wrappedBy": [{"position": "cart.go:42", "function": "Checkout"}],
      "checkedBy": [{"position": "api.go:7", "function": "handle"}],
      "status": "referenced"
    },
    {
      "name": "ErrPaymentDeclined",
      "kind": "family.wrap",
      "code": "CodePaymentDeclined",
      "family": "rejection",
      "declared": "errors.go:20",
      "wraps": "ErrCheckout",
      "checkedBy": [],
      "status": "referenced"
    },
    {
      "name": "ErrLegacy",
      "kind": "errors.new",
      "declared": "errors.go:30",
      "status": "unreferenced"
    }
  ]
}
`

func writeErrorflowFixture(t *testing.T) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "errorflow.json")
	if err := os.WriteFile(path, []byte(errorflowFixture), 0o600); err != nil {
		t.Fatalf("write errorflow fixture: %v", err)
	}

	return path
}

func TestErrorflow_ServiceFrontmatterCarriesXErrors(t *testing.T) {
	tmpDir := t.TempDir()

	exp := eventcatalog.NewExporter(tmpDir, eventcatalog.WithErrorflowFile(writeErrorflowFixture(t)))
	if err := exp.Export(cattest.BuildTestCatalog()); err != nil {
		t.Fatalf("export: %v", err)
	}

	svc, err := os.ReadFile(filepath.Join(tmpDir, "services", "order-svc", "index.mdx"))
	if err != nil {
		t.Fatalf("read service page: %v", err)
	}
	content := string(svc)

	for _, want := range []string{
		"x-errors:",
		"scope: module",
		"tool: erraudit",
		"name: ErrCheckout",
		"code: checkout.failed",
		"family: rejection",
		"status: unreferenced",
		"name: ErrPaymentDeclined",
		"wraps: ErrCheckout",
	} {
		if !strings.Contains(content, want) {
			t.Errorf("service frontmatter missing %q:\n%s", want, content)
		}
	}

	// Site lists stay out of frontmatter: the catalog carries the surface,
	// not call positions (those live in the errorflow JSON / insights page).
	if strings.Contains(content, "cart.go:42") {
		t.Errorf("frontmatter must not carry per-site positions:\n%s", content)
	}
}

// TestErrorflow_XKeyToleranceContract pins the EventCatalog x-key contract:
// the extension block MUST live under the x- namespace (unknown x-* keys
// pass core's schema validation; unknown bare keys do not). If this test
// forces a rename away from x-errors, the downstream core tolerance must be
// re-verified first.
func TestErrorflow_XKeyToleranceContract(t *testing.T) {
	tmpDir := t.TempDir()

	exp := eventcatalog.NewExporter(tmpDir, eventcatalog.WithErrorflowFile(writeErrorflowFixture(t)))
	if err := exp.Export(cattest.BuildTestCatalog()); err != nil {
		t.Fatalf("export: %v", err)
	}

	svc, err := os.ReadFile(filepath.Join(tmpDir, "services", "order-svc", "index.mdx"))
	if err != nil {
		t.Fatalf("read service page: %v", err)
	}
	content := string(svc)

	if !strings.Contains(content, "x-errors:") {
		t.Fatalf("error block must be emitted under the x-errors key:\n%s", content)
	}

	inFrontmatter := false
	seenOpening := false
	for line := range strings.SplitSeq(string(svc), "\n") {
		if line == "---" {
			if !seenOpening {
				seenOpening = true
				inFrontmatter = true

				continue
			}
			inFrontmatter = false

			continue
		}
		if !inFrontmatter {
			continue
		}
		if strings.HasPrefix(line, "errors:") {
			t.Errorf("bare `errors:` key in frontmatter breaks the x-key tolerance contract:\n%s", line)
		}
	}
}

func TestErrorflow_ManifestCarriesErrorSurface(t *testing.T) {
	tmpDir := t.TempDir()

	exp := eventcatalog.NewExporter(tmpDir, eventcatalog.WithErrorflowFile(writeErrorflowFixture(t)))
	if err := exp.Export(cattest.BuildTestCatalog()); err != nil {
		t.Fatalf("export: %v", err)
	}

	manifest, err := os.ReadFile(filepath.Join(tmpDir, "catalog.index.json"))
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	content := string(manifest)

	for _, want := range []string{
		`"schemaVersion": 2`,
		`"errorSurface"`,
		`"tool": "erraudit"`,
		`"name": "ErrCheckout"`,
		`"wrappedBy": 1`,
		`"checkedBy": 1`,
		`"status": "unreferenced"`,
	} {
		if !strings.Contains(content, want) {
			t.Errorf("manifest missing %q:\n%s", want, content)
		}
	}
}

func TestErrorflow_PlainExportUnchanged(t *testing.T) {
	tmpDir := t.TempDir()

	exp := eventcatalog.NewExporter(tmpDir)
	if err := exp.Export(cattest.BuildTestCatalog()); err != nil {
		t.Fatalf("export: %v", err)
	}

	manifest, err := os.ReadFile(filepath.Join(tmpDir, "catalog.index.json"))
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	if strings.Contains(string(manifest), "errorSurface") {
		t.Errorf("plain export must not carry errorSurface:\n%s", manifest)
	}

	svc, err := os.ReadFile(filepath.Join(tmpDir, "services", "order-svc", "index.mdx"))
	if err != nil {
		t.Fatalf("read service page: %v", err)
	}
	if strings.Contains(string(svc), "x-errors") {
		t.Errorf("plain export must not carry x-errors frontmatter:\n%s", svc)
	}
}

func TestErrorflow_SchemaMismatchRejected(t *testing.T) {
	path := filepath.Join(t.TempDir(), "errorflow.json")
	future := strings.Replace(errorflowFixture, `"schemaVersion": 1`, `"schemaVersion": 2`, 1)
	if err := os.WriteFile(path, []byte(future), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	exp := eventcatalog.NewExporter(t.TempDir(), eventcatalog.WithErrorflowFile(path))
	err := exp.Export(cattest.BuildTestCatalog())
	if err == nil {
		t.Fatal("schema-mismatched errorflow file must fail the export")
	}
	if !strings.Contains(err.Error(), "schemaVersion") {
		t.Errorf("error should name the schema mismatch: %v", err)
	}
}

func TestErrorflow_MissingFileRejected(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "does-not-exist.json")

	exp := eventcatalog.NewExporter(t.TempDir(), eventcatalog.WithErrorflowFile(missing))
	if err := exp.Export(cattest.BuildTestCatalog()); err == nil {
		t.Fatal("missing errorflow file must fail the export")
	}
}

// TestErrorflow_ErrorflowModuleScopedSurface asserts the scope metadata is
// honest: the surface is module-scope, carried by every service page of the
// export — readers must not read it as per-service attribution.
func TestErrorflow_ErrorflowModuleScopedSurface(t *testing.T) {
	tmpDir := t.TempDir()

	exp := eventcatalog.NewExporter(tmpDir, eventcatalog.WithErrorflowFile(writeErrorflowFixture(t)))
	if err := exp.Export(cattest.BuildTestCatalog()); err != nil {
		t.Fatalf("export: %v", err)
	}

	for _, svcID := range []string{"order-svc", "payment-svc"} {
		if _, err := os.Stat(filepath.Join(tmpDir, "services", svcID)); err != nil {
			continue
		}

		svc, err := os.ReadFile(filepath.Join(tmpDir, "services", svcID, "index.mdx"))
		if err != nil {
			t.Fatalf("read %s page: %v", svcID, err)
		}
		if !strings.Contains(string(svc), "scope: module") {
			t.Errorf("service %s must carry the module-scope label:\n%s", svcID, svc)
		}
	}
}
