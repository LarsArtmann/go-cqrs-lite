package eventcatalog

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	yaml "github.com/go-faster/yaml"
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

// writeExamples writes each example payload as its own file under
// <message>/examples/. EventCatalog's example loader reads individual files
// from that folder (any text format); a single examples.json array at the
// message root was never read and the data was silently lost.
func (e *Exporter) writeExamples(dir string, examples []jsontext.Value) error {
	if len(examples) == 0 {
		return nil
	}

	examplesDir := filepath.Join(dir, "examples")

	err := os.MkdirAll(examplesDir, dirPerm)
	if err != nil {
		return errorfamily.Newf(
			errorfamily.Infrastructure,
			"catalog.writer.3",
			"create examples dir in %s: %v",
			dir,
			err,
		)
	}

	for i, ex := range examples {
		data, err := json.Marshal(
			ex,
			jsontext.WithIndentPrefix(""),
			jsontext.WithIndent("  "),
		)
		if err != nil {
			return errorfamily.Newf(
				errorfamily.Infrastructure,
				"catalog.writer.3b",
				"marshal example %d for dir %s: %v",
				i+1,
				dir,
				err,
			)
		}

		err = os.WriteFile(
			filepath.Join(examplesDir, fmt.Sprintf("example-%d.json", i+1)),
			data,
			filePerm,
		)
		if err != nil {
			return errorfamily.Newf(
				errorfamily.Infrastructure,
				"catalog.writer.3c",
				"write example %d for dir %s: %v",
				i+1,
				dir,
				err,
			)
		}
	}

	return nil
}

// writeChangelogFile writes the message changelog as EventCatalog's
// changelog.mdx sidecar file (collection pattern "**/changelog.(md|mdx)").
// EventCatalog has no changelog frontmatter field on messages — inline lists
// are rejected as unknown properties and fail the downstream build.
func (e *Exporter) writeChangelogFile(dir string, changes []catalog.Change) error {
	if len(changes) == 0 {
		return nil
	}

	content := "---\n---\n\n" + changelogBody(changes)

	return e.writeMDXFile(filepath.Join(dir, "changelog.mdx"), content)
}

// writeUbiquitousLanguageFile writes a domain's ubiquitous language as
// EventCatalog's ubiquitous-language.mdx sidecar file with a dictionary
// list. The old frontmatter field was an unknown property that failed the
// downstream build.
func (e *Exporter) writeUbiquitousLanguageFile(
	dir string,
	terms []catalog.UbiquitousLanguageTerm,
) error {
	if len(terms) == 0 {
		return nil
	}

	dictionary := make([]dictionaryTermFM, len(terms))
	for i, t := range terms {
		name := string(t.Name)
		dictionary[i] = dictionaryTermFM{ID: name, Name: name, Description: t.Description}
	}

	data, err := yaml.Marshal(struct {
		Dictionary []dictionaryTermFM `yaml:"dictionary"`
	}{Dictionary: dictionary})
	if err != nil {
		return errorfamily.WrapCorruption(err, "catalog.marshal_ubiquitous_language",
			"marshal ubiquitous language dictionary")
	}

	content := "---\n" + string(data) + "---\n\n# Ubiquitous Language\n"

	return e.writeMDXFile(filepath.Join(dir, "ubiquitous-language.mdx"), content)
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
	// WithSkipBootstrapFiles: the hub (or whoever merges this tree) owns its
	// own eventcatalog.config.js/package.json — emitting per-source copies
	// would force first-wins merge semantics on them.
	if e.skipBootstrapFiles {
		return nil
	}

	if err := e.writeBuilderFile("eventcatalog.config.js", func(cfg *strings.Builder) {
		cfg.WriteString(
			"/** @type {import('@eventcatalog/core/bin/eventcatalog.config').Config} */\n",
		)
		cfg.WriteString(
			"// Auto-generated by go-cqrs-lite. Regenerate by re-running the exporter.\n",
		)
		cfg.WriteString("export default {\n")
		fmt.Fprintf(cfg, "  cId: '%s',\n", stableCatalogID(string(cat.Title)))
		fmt.Fprintf(cfg, "  title: %q,\n", cat.Title)
		fmt.Fprintf(cfg, "  tagline: %q,\n",
			"Event-driven architecture documentation, auto-generated from Go types.")
		fmt.Fprintf(cfg, "  organizationName: %q,\n", cat.Title)
		cfg.WriteString("  llmsTxt: { enabled: true },\n")
		// Changelog pages are opt-in in EventCatalog
		// (config.changelog.enabled defaults to false) — without this flag the
		// exported changelog.mdx sidecar files never render. They stay disabled
		// when the catalog contains agents: @eventcatalog/core 4.6.3 crashes
		// rendering /docs/agents/<id>/<version>/changelog (getBadgeHref on an
		// undefined badge) for ANY agent, with or without a changelog file.
		// Drop this guard when bumping eventCatalogCoreVersion past a fix.
		if shouldEnableChangelog(cat) {
			cfg.WriteString("  changelog: { enabled: true },\n")
		}
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

func shouldEnableChangelog(cat *catalog.Catalog) bool {
	if len(cat.Agents) > 0 {
		return false
	}

	return catalogHasChangelogs(cat)
}

func catalogHasChangelogs(cat *catalog.Catalog) bool {
	hasChangelog := func(msgs []catalog.Message) bool {
		return slices.ContainsFunc(
			msgs,
			func(m catalog.Message) bool { return len(m.Changelog) > 0 },
		)
	}

	for _, svc := range cat.Services {
		if hasChangelog(svc.Commands) || hasChangelog(svc.Events) || hasChangelog(svc.Queries) {
			return true
		}
	}

	return false
}

func (e *Exporter) writePackageJSON(cat *catalog.Catalog) error {
	pkg := map[string]any{
		"type":        "module",
		"name":        strings.ToLower(strings.ReplaceAll(string(cat.Title), " ", "-")),
		"version":     string(cat.Version),
		"private":     true,
		"description": string(cat.Title) + " event catalog",
		"dependencies": map[string]string{
			"@eventcatalog/core": eventCatalogCoreVersion,
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
