package eventcatalog

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
)

func (e *Exporter) writeMDXFile(path, content string) error {
	return os.WriteFile( //nolint:wrapcheck // direct passthrough
		path,
		[]byte(content),
		filePerm,
	)
}

// writeResourceMDX renders frontmatter to MDX and writes it to targetPath.
// On render failure, returns an Infrastructure error tagged with errorCode
// and identifying resourceKind/resourceID for diagnosis.
func (e *Exporter) writeResourceMDX(
	fm any,
	name, summary, targetPath, errorCode, resourceKind, resourceID string,
	includeGraph bool,
) error {
	content, err := renderMDX(fm, name, summary, includeGraph)
	if err != nil {
		return errorfamily.Newf(errorfamily.Infrastructure, errorCode,
			"render %s %s: %v", resourceKind, resourceID, err)
	}

	return e.writeMDXFile(targetPath, content)
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

// writeBuilderFile builds a string with fn and writes it to filename in the
// exporter's output directory. It centralizes the strings.Builder + os.WriteFile
// pattern used by every text-file writer.
func (e *Exporter) writeBuilderFile(filename string, fn func(*strings.Builder)) error {
	var b strings.Builder
	fn(&b)

	return os.WriteFile( //nolint:wrapcheck // direct passthrough
		filepath.Join(e.outputDir, filename),
		[]byte(b.String()),
		filePerm,
	)
}

func (e *Exporter) writeConfig(cat *catalog.Catalog) error {
	if err := e.writeBuilderFile("eventcatalog.config.js", func(cfg *strings.Builder) {
		cfg.WriteString("/** @type {import('@eventcatalog/core/bin/eventcatalog.config').Config} */\n")
		cfg.WriteString("export default {\n")
		fmt.Fprintf(cfg, "  title: %q,\n", cat.Title)
		fmt.Fprintf(cfg, "  organizationName: %q,\n", cat.Title)
		cfg.WriteString("  landingPage: '',\n")
		cfg.WriteString("};\n")
	}); err != nil {
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
