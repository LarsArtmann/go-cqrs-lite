package eventcatalog

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/catalog/v3"
)

func (e *Exporter) writeMessage(
	serviceID catalog.ServiceID,
	kind string,
	msg catalog.Message,
) error {
	messageID := catalog.Key(msg)
	dir := filepath.Join(e.outputDir, "services", string(serviceID), kind, string(messageID))

	err := os.MkdirAll(dir, dirPerm)
	if err != nil {
		return errorfamily.Newf(
			errorfamily.Infrastructure,
			"catalog.exporter_message.1",
			"create message dir for %s/%s: %v",
			serviceID,
			kind,
			err,
		)
	}

	md := buildMessageFrontmatter(messageID, msg)

	err = e.writeMDXFile(filepath.Join(dir, indexFile), md.String())
	if err != nil {
		return errorfamily.Newf(
			errorfamily.Infrastructure,
			"catalog.exporter_message.2",
			"write message file for %s/%s: %v",
			serviceID,
			kind,
			err,
		)
	}

	if msg.Schema != nil {
		err = e.writeSchema(dir, msg.Schema)
		if err != nil {
			return errorfamily.Newf(
				errorfamily.Infrastructure,
				"catalog.exporter_message.3",
				"write schema for %s/%s: %v",
				serviceID,
				kind,
				err,
			)
		}
	}

	return e.writeExamples(dir, msg.Examples)
}

func buildMessageFrontmatter(messageID catalog.MessageID, msg catalog.Message) *frontmatterWriter {
	md := newFrontmatterWriter()
	md.addField("id", string(messageID))
	md.addField("name", string(msg.Name))
	md.addField("version", string(msg.Version))

	if msg.Summary != "" {
		md.addQuotedField("summary", string(msg.Summary))
	}

	if msg.Deprecated {
		_, _ = md.WriteString("deprecated: true\n")
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

	md.finish(string(msg.Name), string(msg.Summary))

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
