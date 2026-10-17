package architecture

// ReferencesMode controls how references to resources outside the kept set
// render (ADR-0160). The zero value is [ReferencesOmit], so exports behave
// identically whether or not an option was set. Any other value behaves like
// [ReferencesOmit].
type ReferencesMode string

const (
	// ReferencesOmit drops references to resources outside the kept set.
	// It is the default: a redacted export carries no trace of what it omits.
	ReferencesOmit ReferencesMode = ""
	// ReferencesNameOnly renders out-of-scope references as the resource's
	// display name in plain text, so a slice can still say what it talks to
	// without linking to it. Resources dropped by exclude or neverExport
	// stay fully hidden in this mode too: redaction beats readability.
	ReferencesNameOnly ReferencesMode = "name-only"
	// ReferencesInclude pulls one level of the resources a kept section
	// points at into the export (a service brings its data stores, entities
	// and message channels), skipping excluded and never-exported resources.
	ReferencesInclude ReferencesMode = "include"
)
