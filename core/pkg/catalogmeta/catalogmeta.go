// Package catalogmeta provides shared documentation metadata types used by
// core packages for auto-catalog generation. It exists to prevent duplication
// of the common CatalogMeta fields across command, query, and event packages.
package catalogmeta

// Meta contains the common documentation metadata fields shared by all
// message kinds (commands, queries, events).
type Meta struct {
	Name    string
	Version string
	Summary string
}
