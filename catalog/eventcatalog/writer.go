package eventcatalog

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/larsartmann/go-cqrs-lite/catalog/v2"
)

func (e *Exporter) writeMDXFile(path, content string) error {
	return os.WriteFile(path, []byte(content), filePerm) //nolint:wrapcheck
}

func (e *Exporter) writeSchema(dir string, schema *catalog.Schema) error {
	schemaDir := filepath.Join(dir, "schemas")

	err := os.MkdirAll(schemaDir, dirPerm)
	if err != nil {
		return fmt.Errorf("create schema dir %s in %s: %w", schemaDir, dir, err)
	}

	data, err := catalog.SchemaToJSON(schema)
	if err != nil {
		return fmt.Errorf("marshal schema: %w", err)
	}

	return os.WriteFile(filepath.Join(schemaDir, "schema.json"), data, filePerm) //nolint:wrapcheck
}

func (e *Exporter) writeExamples(dir string, examples []json.RawMessage) error {
	if len(examples) == 0 {
		return nil
	}

	data, err := json.MarshalIndent(examples, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal examples for dir %s: %w", dir, err)
	}

	return os.WriteFile(filepath.Join(dir, "examples.json"), data, filePerm) //nolint:wrapcheck
}

func (e *Exporter) writeConfig(cat *catalog.Catalog) error {
	var cfg strings.Builder
	cfg.WriteString("/** @type {import('@eventcatalog/core/bin/eventcatalog.config').Config} */\n")
	cfg.WriteString("export default {\n")
	fmt.Fprintf(&cfg, "  title: %q,\n", cat.Title)
	fmt.Fprintf(&cfg, "  organizationName: %q,\n", cat.Title)
	cfg.WriteString("  landingPage: '',\n")
	cfg.WriteString("};\n")

	err := os.WriteFile(
		filepath.Join(e.outputDir, "eventcatalog.config.js"),
		[]byte(cfg.String()),
		filePerm,
	)
	if err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	return e.writePackageJSON(cat)
}

func (e *Exporter) writePackageJSON(cat *catalog.Catalog) error {
	pkg := map[string]any{
		"type":        "module",
		"name":        strings.ToLower(strings.ReplaceAll(cat.Title, " ", "-")),
		"version":     cat.Version,
		"private":     true,
		"description": cat.Title + " event catalog",
		"dependencies": map[string]string{
			"@eventcatalog/core": "latest",
		},
	}

	data, err := json.MarshalIndent(pkg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal package.json: %w", err)
	}

	return os.WriteFile( //nolint:wrapcheck
		filepath.Join(e.outputDir, "package.json"),
		data,
		filePerm,
	)
}
