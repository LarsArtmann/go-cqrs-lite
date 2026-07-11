package eventcatalog

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
	"github.com/larsartmann/go-cqrs-lite/catalog/v4/internal/cattest"
)

func TestExporter_Export_WriteServiceError(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

	reg := catalog.NewRegistry("TestCatalog", "1.0.0")
	reg.AddService(catalog.Service{ID: "svc", Name: "Svc", Version: "1.0.0"})

	cat := reg.Build()

	svcDir := filepath.Join(tmpDir, "services")

	err := os.MkdirAll(svcDir, 0o750)
	if err != nil {
		t.Fatal(err)
	}

	err = os.Chmod(svcDir, 0o000)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(svcDir, 0o750) //nolint:errcheck // cleanup

	exp := NewExporter(tmpDir)

	err = exp.Export(cat)
	if err == nil {
		t.Error("expected error when service dir is read-only")
	}
}

func TestExporter_Export_WriteMessageError(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

	reg := catalog.NewRegistry("TestCatalog", "1.0.0")
	reg.AddService(catalog.Service{ID: "svc", Name: "Svc", Version: "1.0.0"})
	reg.AddCommand("svc", catalog.Message{
		Kind: catalog.CommandMessage, ID: "Cmd", Name: "Cmd", Version: "1.0.0",
	})

	cat := reg.Build()
	requireExportPermissionError(t, cat, tmpDir, filepath.Join(tmpDir, "services", "svc"))
}

func TestExporter_Export_WriteDomainError(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

	reg := catalog.NewRegistry("TestCatalog", "1.0.0")
	reg.AddDomain(
		catalog.Domain{
			ID: "dom", Name: "Dom",
			Version: "1.0.0", Services: []catalog.ServiceID{"svc"},
		},
	)

	cat := reg.Build()

	domDir := filepath.Join(tmpDir, "domains")

	err := os.MkdirAll(domDir, 0o750)
	if err != nil {
		t.Fatal(err)
	}

	err = os.Chmod(domDir, 0o000)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(domDir, 0o750) //nolint:errcheck // cleanup

	exp := NewExporter(tmpDir)

	err = exp.Export(cat)
	if err == nil {
		t.Error("expected error when domain dir is read-only")
	}
}

func TestExporter_Export_WriteConfigAndLLMsErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		chmod   os.FileMode
		wantErr string
	}{
		{
			name:    "config_write",
			chmod:   0o000,
			wantErr: "expected error when output dir is read-only for config write",
		},
		{name: "llms_txt_write", chmod: 0o500, wantErr: "expected error when llms.txt write fails"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tmpDir := t.TempDir()

			reg := catalog.NewRegistry("TestCatalog", "1.0.0")
			cat := reg.Build()

			err := os.Chmod(tmpDir, tt.chmod)
			if err != nil {
				t.Fatal(err)
			}
			defer os.Chmod(tmpDir, 0o750) //nolint:errcheck // cleanup

			exp := NewExporter(tmpDir)

			err = exp.Export(cat)
			if err == nil {
				t.Error(tt.wantErr)
			}
		})
	}
}

func TestExporter_Export_MessageWithSchemaWriteError(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

	reg := catalog.NewRegistry("TestCatalog", "1.0.0")
	reg.AddService(catalog.Service{ID: "svc", Name: "Svc", Version: "1.0.0"})
	cattest.AddCommandWithSchema(
		t,
		reg,
		"svc",
		"Cmd",
		"Cmd",
		"1.0.0",
		&catalog.Schema{Type: catalog.TypeObject},
	)

	cat := reg.Build()

	cmdDir := filepath.Join(tmpDir, "services", "svc", "commands", "Cmd")

	err := os.MkdirAll(cmdDir, 0o750)
	if err != nil {
		t.Fatal(err)
	}

	err = os.Chmod(cmdDir, 0o000)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(cmdDir, 0o750) //nolint:errcheck // cleanup

	exp := NewExporter(tmpDir)

	err = exp.Export(cat)
	if err == nil {
		t.Error("expected error when schema dir creation fails")
	}
}

func TestExporter_Export_ExamplesFileMarshalError(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

	reg := cattest.NewRegistry(t, "TestCatalog", "1.0.0")
	cattest.AddService(t, reg, catalog.ServiceID("svc"), "Svc", "1.0.0")
	cattest.AddCommandWithExample(
		t, reg, catalog.MessageID("Cmd"), "Cmd", "1.0.0",
		`{invalid json !!!`,
	)

	cat := reg.Build()

	exp := NewExporter(tmpDir)

	err := exp.Export(cat)
	if err == nil {
		t.Error("expected error when examples contain invalid JSON")
	}
}

func TestExporter_Export_SchemaDirPermissionError(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

	reg := catalog.NewRegistry("TestCatalog", "1.0.0")
	reg.AddService(catalog.Service{ID: "svc", Name: "Svc", Version: "1.0.0"})
	reg.AddCommand("svc", catalog.Message{
		Kind:    catalog.CommandMessage,
		ID:      "Cmd",
		Name:    "Cmd",
		Version: "1.0.0",
		Schema: &catalog.Schema{
			Type:       "object",
			Properties: map[string]catalog.Property{"x": {Type: "string"}},
		},
	})

	cat := reg.Build()

	cmdDir := filepath.Join(tmpDir, "services", "svc", "commands", "Cmd")

	err := os.MkdirAll(cmdDir, 0o750)
	if err != nil {
		t.Fatal(err)
	}

	err = os.Chmod(cmdDir, 0o000)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(cmdDir, 0o750) //nolint:errcheck // cleanup

	exp := NewExporter(tmpDir)

	err = exp.Export(cat)
	if err == nil {
		t.Error("expected error when schema dir creation fails")
	}
}

func TestExporter_Export_EventWriteError(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

	reg := catalog.NewRegistry("TestCatalog", "1.0.0")
	reg.AddService(catalog.Service{ID: "svc", Name: "Svc", Version: "1.0.0"})
	reg.AddEvent("svc", catalog.Message{
		Kind: catalog.EventMessage, ID: "Evt", Name: "Evt", Version: "1.0.0",
	})

	cat := reg.Build()
	requireExportPermissionError(t, cat, tmpDir, filepath.Join(tmpDir, "services", "svc"))
}

func TestExporter_Export_QueryWriteError(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

	reg := catalog.NewRegistry("TestCatalog", "1.0.0")
	reg.AddService(catalog.Service{ID: "svc", Name: "Svc", Version: "1.0.0"})
	reg.AddQuery("svc", catalog.Message{
		Kind: catalog.QueryMessage, ID: "Qry", Name: "Qry", Version: "1.0.0",
	})

	cat := reg.Build()
	requireExportPermissionError(t, cat, tmpDir, filepath.Join(tmpDir, "services", "svc"))
}

func TestExporter_Export_MessageSummaryWithSchema(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

	reg := catalog.NewRegistry("TestCatalog", "1.0.0")
	reg.AddService(catalog.Service{ID: "svc", Name: "Svc", Version: "1.0.0"})
	reg.AddCommand("svc", catalog.Message{
		Kind:    catalog.CommandMessage,
		ID:      "Cmd",
		Name:    "Cmd",
		Version: "1.0.0",
		Summary: "A command with summary and schema",
		Schema:  &catalog.Schema{Type: catalog.TypeObject},
	})

	cat := reg.Build()

	exp := NewExporter(tmpDir)

	err := exp.Export(cat)
	if err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(
		filepath.Join(tmpDir, "services", "svc", "commands", "Cmd", "index.mdx"),
	)
	if err != nil {
		t.Fatal(err)
	}

	content := string(data)
	if !strings.Contains(content, "summary:") {
		t.Errorf("message frontmatter missing summary: %s", content)
	}
}

func TestExporter_Export_PackageJSON(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

	reg := catalog.NewRegistry("My Catalog", "2.0.0")
	cat := reg.Build()

	exp := NewExporter(tmpDir)

	err := exp.Export(cat)
	if err != nil {
		t.Fatal(err)
	}

	pkgPath := filepath.Join(tmpDir, "package.json")

	data, err := os.ReadFile(pkgPath)
	if err != nil {
		t.Fatalf("read package.json: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, `"name": "my-catalog"`) {
		t.Errorf("package.json missing name: %s", content)
	}

	if !strings.Contains(content, `"version": "2.0.0"`) {
		t.Errorf("package.json missing version: %s", content)
	}

	if !strings.Contains(content, `"private": true`) {
		t.Errorf("package.json missing private: %s", content)
	}

	if !strings.Contains(content, "@eventcatalog/core") {
		t.Errorf("package.json missing eventcatalog dependency: %s", content)
	}
}
