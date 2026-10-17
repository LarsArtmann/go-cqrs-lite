package architecture

import (
	"time"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
)

// messageSection builds the section for one command, event or query.
func messageSection(msg catalog.Message, kind resourceKind, idx *refIndex) section {
	deprecated := ""
	if msg.Deprecated {
		deprecated = "Deprecated"
	}

	return section{
		kind:       kind,
		anchor:     idx.get(kind, string(msg.ID)).anchor,
		name:       string(msg.Name),
		version:    string(msg.Version),
		summary:    string(msg.Summary),
		badges:     msg.Badges,
		deprecated: deprecated,
		meta: metaLine([]markup{
			textMeta(messageKindLabel(msg)),
			ownerMeta(idx, msg.Owners),
			textMeta(operationLabel(msg.Operation)),
		}),
		left:  messageBlocks(msg, idx),
		right: messageRefs(msg, idx),
	}
}

// messageKindLabel describes the message's kind and direction.
func messageKindLabel(msg catalog.Message) string {
	label := string(msg.Kind)

	if msg.Direction != "" {
		label += " · " + string(msg.Direction)
	}

	return label
}

// operationLabel renders the HTTP operation a message carries, if any.
func operationLabel(op *catalog.Operation) string {
	if op == nil || op.Path == "" {
		return ""
	}

	return string(op.Method) + " " + op.Path
}

// messageBlocks builds the left-column blocks of a message section: its
// attributes, an example payload, a TypeScript type and the raw schema.
func messageBlocks(msg catalog.Message, idx *refIndex) []block {
	blocks := make([]block, 0, 6)

	if table := attributeTable(msg.Schema); table != "" {
		blocks = append(blocks, blockHTML("Attributes", table))
	}

	if payload := examplePayload(msg.Schema, msg.Examples); payload != "" {
		blocks = append(blocks, blockHTML("Example payload", codeBlock("", payload)))
	}

	if ts := typeScriptInterface(string(msg.Name), msg.Schema); ts != "" {
		blocks = append(blocks, blockHTML("TypeScript", codeBlock("", ts)))
	}

	if raw := schemaJSON(msg.Schema); raw != "" {
		blocks = append(blocks, blockHTML("JSON Schema", codeBlock("", raw)))
	}

	if changes := changeTable(msg.Changelog); changes != "" {
		blocks = append(blocks, blockHTML("Changelog", changes))
	}

	if msg.Deprecation != nil {
		blocks = appendProse(blocks, "Deprecation", msg.Deprecation.Message, idx.link)
	}

	return blocks
}

// messageRefs builds the relationship boxes of a message section.
func messageRefs(msg catalog.Message, idx *refIndex) []block {
	refs := make([]block, 0, 5)
	refs = appendRefs(refs, "Produced by", idx.messagesOf(kindService, idStrings(msg.Producers)))
	refs = appendRefs(refs, "Consumed by", idx.messagesOf(kindService, idStrings(msg.Consumers)))
	refs = appendRefs(refs, "Channels", idx.messagesOf(kindChannel, idStrings(msg.Channels)))
	refs = appendRefs(refs, "Owners", idx.owners(msg.Owners))
	refs = appendRefs(refs, "Security", plainRefs(msg.Security))
	refs = appendRefs(refs, "Labels", labelRefs(msg.Labels))

	if msg.Repository != nil && msg.Repository.URL != "" {
		refs = append(refs, blockHTML("Repository", externalLink(string(msg.Repository.URL), "Source code")))
	}

	return refs
}

// plainRefs renders identifier strings that have no in-document target.
func plainRefs(values []string) []resourceRef {
	refs := make([]resourceRef, 0, len(values))

	for _, value := range values {
		refs = append(refs, resourceRef{name: value})
	}

	return refs
}

// labelRefs renders a message's labels as "key: value" rows.
func labelRefs(labels map[string]string) []resourceRef {
	if len(labels) == 0 {
		return nil
	}

	refs := make([]resourceRef, 0, len(labels))

	for _, key := range sortedKeysOf(labels) {
		refs = append(refs, resourceRef{name: key + ": " + labels[key]})
	}

	return refs
}

// changeTable renders a message changelog as a table.
func changeTable(changes []catalog.Change) markup {
	if len(changes) == 0 {
		return ""
	}

	return renderInto(func(w *htmlWriter) {
		w.open("div", htmlClass("md-table"))
		w.open("table")
		w.open("thead")
		w.open("tr")

		for _, header := range []string{"Version", "Date", "Summary"} {
			w.element("th", header)
		}

		w.close("tr")
		w.close("thead")
		w.open("tbody")

		for _, change := range changes {
			w.open("tr")
			w.element("td", string(change.Version), htmlClass("mono"))
			w.element("td", changeDate(change.Date))
			w.element("td", string(change.Summary))
			w.close("tr")
		}

		w.close("tbody")
		w.close("table")
		w.close("div")
	})
}

// changeDate formats a changelog date, tolerating a missing one.
func changeDate(date *time.Time) string {
	if date == nil {
		return ""
	}

	return date.Format(time.DateOnly)
}
