package eventcatalog

import (
	"os"
	"path/filepath"

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

	if err := os.MkdirAll(dir, dirPerm); err != nil {
		return errorfamily.Newf(errorfamily.Infrastructure, "catalog.exporter_message.1",
			"create message dir for %s/%s: %v", serviceID, kind, err)
	}

	fm := messageFM{
		ID:         string(messageID),
		Name:       string(msg.Name),
		Version:    string(msg.Version),
		Summary:    string(msg.Summary),
		Deprecated: msg.Deprecated,
		Owners:     msg.Owners,
		Labels:     msg.Labels,
		Changelog:  toChangelog(msg.Changelog),
		Producers:  toPointers(msg.Producers),
		Consumers:  toPointers(msg.Consumers),
		Operation:  toOperation(msg.Operation),
		Badges:     toBadges(msg.Badges),
		Repository: toRepository(msg.Repository),
	}

	if msg.Schema != nil {
		fm.SchemaPath = "schemas/schema.json"
	}

	content, err := renderMDX(fm, string(msg.Name), string(msg.Summary), false)
	if err != nil {
		return errorfamily.Newf(errorfamily.Infrastructure, "catalog.exporter_message.2",
			"render message %s/%s: %v", serviceID, kind, err)
	}

	if err := e.writeMDXFile(filepath.Join(dir, indexFile), content); err != nil {
		return errorfamily.Newf(errorfamily.Infrastructure, "catalog.exporter_message.3",
			"write message file for %s/%s: %v", serviceID, kind, err)
	}

	if msg.Schema != nil {
		if err := e.writeSchema(dir, msg.Schema); err != nil {
			return errorfamily.Newf(errorfamily.Infrastructure, "catalog.exporter_message.4",
				"write schema for %s/%s: %v", serviceID, kind, err)
		}
	}

	return e.writeExamples(dir, msg.Examples)
}
