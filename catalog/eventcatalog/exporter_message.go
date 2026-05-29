package eventcatalog

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/larsartmann/go-cqrs-lite/catalog"
)

func (e *Exporter) writeMessage(
	serviceID catalog.ServiceID,
	kind string,
	msg catalog.Message,
) error {
	messageID := catalog.GetID(msg)
	dir := filepath.Join(e.outputDir, "services", string(serviceID), kind, string(messageID))

	err := os.MkdirAll(dir, dirPerm)
	if err != nil {
		return fmt.Errorf("create message dir for %s/%s: %w", serviceID, kind, err)
	}

	md := buildMessageFrontmatter(messageID, msg)

	err = e.writeMDXFile(filepath.Join(dir, indexFile), md.String())
	if err != nil {
		return fmt.Errorf("write message file for %s/%s: %w", serviceID, kind, err)
	}

	if msg.Schema != nil {
		err = e.writeSchema(dir, msg.Schema)
		if err != nil {
			return fmt.Errorf("write schema for %s/%s: %w", serviceID, kind, err)
		}
	}

	return e.writeExamples(dir, msg.Examples)
}

func buildMessageFrontmatter(messageID catalog.MessageID, msg catalog.Message) *frontmatterWriter {
	md := newFrontmatterWriter()
	md.addField("id", string(messageID))
	md.addField("name", msg.Name)
	md.addField("version", msg.Version)

	if msg.Summary != "" {
		md.addQuotedField("summary", msg.Summary)
	}

	if msg.Deprecated {
		md.addField("deprecated", "true")
	}

	md.addListField("owners", msg.Owners)
	writeLabels(md, msg.Labels)
	writeChangelog(md, msg.Changelog)
	writeIDListField(md, "producers", msg.Producers)
	writeIDListField(md, "consumers", msg.Consumers)
	writeOperation(md, msg.Operation)
	writeBadges(md, msg.Badges)
	writeRepository(md, msg.Repository)

	if msg.Schema != nil {
		_, _ = md.WriteString("schemaPath: schemas/schema.json\n")
	}

	md.finish(msg.Name, msg.Summary)

	return md
}

func writeLabels(md *frontmatterWriter, labels map[string]string) {
	if len(labels) == 0 {
		return
	}

	_, _ = md.WriteString("labels:\n")

	for k, v := range labels {
		_, _ = fmt.Fprintf(md, "  %s: %q\n", k, v)
	}
}

func writeChangelog(md *frontmatterWriter, changelog []catalog.Change) {
	if len(changelog) == 0 {
		return
	}

	_, _ = md.WriteString("changelog:\n")

	for _, c := range changelog {
		_, _ = fmt.Fprintf(md, "  - version: %q\n    summary: %q", c.Version, c.Summary)

		if c.Date != nil {
			_, _ = fmt.Fprintf(md, "\n    date: %q", c.Date.Format(time.DateOnly))
		}

		_, _ = md.WriteString("\n")
	}
}
