package eventcatalog

import (
	"os"
	"path/filepath"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

func (e *Exporter) writeMessage(
	kind string,
	msg catalog.Message,
	serviceVersions map[catalog.ServiceID]catalog.Version,
) error {
	messageID := catalog.Key(msg)
	dir := filepath.Join(e.outputDir, kind, string(messageID))

	if err := os.MkdirAll(dir, dirPerm); err != nil {
		return errorfamily.Newf(errorfamily.Infrastructure, "catalog.exporter_message.1",
			"create message dir for %s: %v", messageID, err)
	}

	fm := messageFM{
		ID:         string(messageID),
		Name:       string(msg.Name),
		Version:    string(msg.Version),
		Summary:    string(msg.Summary),
		Deprecated: toDeprecated(msg.Deprecated, msg.Deprecation),
		Owners:     msg.Owners,
		Labels:     msg.Labels,
		Channels:   stringIDsToStrings(msg.Channels),
		Schemas:    toSchemas(msg.Schemas),
		Changelog:  toChangelog(msg.Changelog),
		Producers:  toServiceRefs(msg.Producers, serviceVersions),
		Consumers:  toServiceRefs(msg.Consumers, serviceVersions),
		Operation:  toOperation(msg.Operation),
		Responses:  toResponses(msg.Responses),
		Badges:     toBadges(msg.Badges),
		Repository: toRepository(msg.Repository),
	}

	if msg.Schema != nil {
		fm.SchemaPath = schemaPathKey
	}

	content, err := renderMDX(fm, string(msg.Name), string(msg.Summary), false)
	if err != nil {
		return errorfamily.Newf(errorfamily.Infrastructure, "catalog.exporter_message.2",
			"render message %s: %v", messageID, err)
	}

	if err := e.writeMDXFile(filepath.Join(dir, indexFile), content); err != nil {
		return errorfamily.Newf(errorfamily.Infrastructure, "catalog.exporter_message.3",
			"write message file for %s: %v", messageID, err)
	}

	if msg.Schema != nil {
		if err := e.writeSchema(dir, msg.Schema); err != nil {
			return errorfamily.Newf(errorfamily.Infrastructure, "catalog.exporter_message.4",
				"write schema for %s: %v", messageID, err)
		}
	}

	return e.writeExamples(dir, msg.Examples)
}
