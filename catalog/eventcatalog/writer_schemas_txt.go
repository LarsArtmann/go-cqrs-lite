package eventcatalog

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/larsartmann/go-cqrs-lite/catalog/v3"
)

func (e *Exporter) writeSchemasTxt(cat *catalog.Catalog) error {
	var buf strings.Builder

	buf.WriteString("# Schemas\n\n")
	buf.WriteString("> All message and entity schemas in this catalog.\n\n")

	for _, svc := range cat.Services {
		writeSchemasTxtMessages(&buf, string(svc.Name), svc.Commands)
		writeSchemasTxtMessages(&buf, string(svc.Name), svc.Events)
		writeSchemasTxtMessages(&buf, string(svc.Name), svc.Queries)
	}

	for _, entity := range cat.Entities {
		writeSchemasTxtEntry(
			&buf,
			"Entity",
			string(entity.Name),
			string(entity.Version),
			entity.Schema,
		)
	}

	return os.WriteFile( //nolint:wrapcheck // direct passthrough
		filepath.Join(e.outputDir, "schemas.txt"),
		[]byte(buf.String()),
		filePerm,
	)
}

func writeSchemasTxtMessages(buf *strings.Builder, owner string, msgs []catalog.Message) {
	for _, msg := range msgs {
		writeSchemasTxtEntry(buf, string(msg.Name)+" @ "+owner, string(msg.Name),
			string(msg.Version), msg.Schema)
	}
}

func writeSchemasTxtEntry(
	buf *strings.Builder,
	context, name, version string,
	schema *catalog.Schema,
) {
	if schema == nil {
		return
	}

	fmt.Fprintf(buf, "## %s (v%s)\n", name, version)
	fmt.Fprintf(buf, "Context: %s\n", context)

	data, err := json.Marshal(schema, jsontext.WithIndentPrefix(""), jsontext.WithIndent("  "))
	if err != nil {
		fmt.Fprintf(buf, "Schema: <error: %v>\n\n", err)

		return
	}

	fmt.Fprintf(buf, "```json\n%s\n```\n\n", string(data))
}
