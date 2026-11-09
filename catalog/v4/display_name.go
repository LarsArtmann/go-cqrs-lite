package catalog

import "strings"

// applyDisplayNameFallbacks back-fills Message.Name from the message ID for
// messages that reached the built catalog without a display name. The
// auto-derivation path (camelCaseToHuman over the Go type name) yields an
// empty name whenever the type has no usable name (e.g. Query[struct{}]),
// which then renders a blank title next to fully-declared siblings. The ID is
// always present, so it is always available as a fallback.
func applyDisplayNameFallbacks(cat *Catalog) {
	for i := range cat.Services {
		svc := &cat.Services[i]

		for j := range svc.Commands {
			fallbackName(&svc.Commands[j])
		}

		for j := range svc.Events {
			fallbackName(&svc.Events[j])
		}

		for j := range svc.Queries {
			fallbackName(&svc.Queries[j])
		}
	}
}

// fallbackName back-fills m.Name from the message ID (see
// applyDisplayNameFallbacks). A free function, not a method, so Message keeps
// a single value-receiver method set.
func fallbackName(m *Message) {
	if m.Name != "" || m.ID == "" {
		return
	}

	m.Name = Name(displayNameFromID(m.ID))
}

// displayNameFromID derives a human-readable name from a message ID by
// splitting on word separators and capitalizing each word:
// "crm.count" → "Crm Count", "contact_created" → "Contact Created".
func displayNameFromID(id MessageID) string {
	segments := strings.FieldsFunc(string(id), func(r rune) bool {
		return r == '.' || r == '_' || r == '-'
	})
	if len(segments) == 0 {
		return ""
	}

	words := make([]string, 0, len(segments))
	for _, seg := range segments {
		words = append(words, camelCaseToHuman(seg))
	}

	return strings.Join(words, " ")
}
