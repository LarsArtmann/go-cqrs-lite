package eventcatalog

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/catalog/v3"
)

func (e *Exporter) writeMDXFile(path, content string) error {
	return os.WriteFile( //nolint:wrapcheck // direct passthrough
		path,
		[]byte(content),
		filePerm,
	)
}

func (e *Exporter) writeSchema(dir string, schema *catalog.Schema) error {
	schemaDir := filepath.Join(dir, "schemas")

	err := os.MkdirAll(schemaDir, dirPerm)
	if err != nil {
		return errorfamily.Newf(
			errorfamily.Infrastructure,
			"catalog.writer.1",
			"create schema dir %s in %s: %v",
			schemaDir,
			dir,
			err,
		)
	}

	data, err := catalog.SchemaToJSON(schema)
	if err != nil {
		return errorfamily.Newf(
			errorfamily.Infrastructure,
			"catalog.writer.2",
			"marshal schema: %v",
			err,
		)
	}

	return os.WriteFile( //nolint:wrapcheck // os.WriteFile returns direct error
		filepath.Join(schemaDir, "schema.json"),
		data,
		filePerm,
	)
}

func (e *Exporter) writeExamples(dir string, examples []jsontext.Value) error {
	if len(examples) == 0 {
		return nil
	}

	data, err := json.Marshal(
		examples,
		json.Deterministic(true),
		jsontext.WithIndentPrefix(""),
		jsontext.WithIndent("  "),
	)
	if err != nil {
		return errorfamily.Newf(
			errorfamily.Infrastructure,
			"catalog.writer.3",
			"marshal examples for dir %s: %v",
			dir,
			err,
		)
	}

	return os.WriteFile( //nolint:wrapcheck // os.WriteFile returns direct error
		filepath.Join(dir, "examples.json"),
		data,
		filePerm,
	)
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
		return errorfamily.Newf(
			errorfamily.Infrastructure,
			"catalog.writer.4",
			"write config: %v",
			err,
		)
	}

	return e.writePackageJSON(cat)
}

func (e *Exporter) writePackageJSON(cat *catalog.Catalog) error {
	pkg := map[string]any{
		"type":        "module",
		"name":        strings.ToLower(strings.ReplaceAll(string(cat.Title), " ", "-")),
		"version":     string(cat.Version),
		"private":     true,
		"description": string(cat.Title) + " event catalog",
		"dependencies": map[string]string{
			"@eventcatalog/core": "latest",
		},
	}

	data, err := json.Marshal(
		pkg,
		json.Deterministic(true),
		jsontext.WithIndentPrefix(""),
		jsontext.WithIndent("  "),
	)
	if err != nil {
		return errorfamily.Newf(
			errorfamily.Infrastructure,
			"catalog.writer.5",
			"marshal package.json: %v",
			err,
		)
	}

	return os.WriteFile( //nolint:wrapcheck // os.WriteFile returns direct error
		filepath.Join(e.outputDir, "package.json"),
		data,
		filePerm,
	)
}
