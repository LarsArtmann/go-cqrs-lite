package version

// Data tables for V007 (v5-removed-api-usage). Kept separate from the
// detector so the rule logic and the curated removal surface evolve
// independently; both files stay under the 350-line limit.

// cqrsModulePrefix is the go-cqrs-lite module path prefix every consumer
// import shares.
const cqrsModulePrefix = "github.com/larsartmann/go-cqrs-lite/"

// deprecatedV5Module describes a module removed entirely at v5.
type deprecatedV5Module struct {
	fragment    string // module path relative to go-cqrs-lite/ (no /vN suffix)
	replacement string
}

var deprecatedV5Modules = []deprecatedV5Module{ //nolint:gochecknoglobals // static table
	// ADR-0123: stack presets replaced by system.New.
	{fragment: "stack/memory", replacement: "system.New with DomainConfig + DeploymentConfig"},
	{fragment: "stack/sqlite", replacement: "system.New with DomainConfig + DeploymentConfig"},
	{fragment: "stack/pebble", replacement: "system.New with DomainConfig + DeploymentConfig"},
	{fragment: "stack/bbolt", replacement: "system.New with DomainConfig + DeploymentConfig"},
	{fragment: "stack/duckdb", replacement: "system.New with DomainConfig + DeploymentConfig"},
	{fragment: "stack/postgres", replacement: "system.New with DomainConfig + DeploymentConfig"},
	{fragment: "stack/mysql", replacement: "system.New with DomainConfig + DeploymentConfig"},
	{fragment: "stack/turso", replacement: "system.New with DomainConfig + DeploymentConfig"},
	// ADR-0123: relational + view tiers absorbed into metaengine engines.
	{fragment: "storage/relational", replacement: "metaengine engines with layout planning"},
	{fragment: "storage/view", replacement: "metaengine engines with layout planning"},
}

// deprecatedV5Symbol describes one deprecated symbol inside a module that
// otherwise survives v5.
type deprecatedV5Symbol struct {
	fragment    string // module path relative to go-cqrs-lite/
	symbol      string // exported identifier
	replacement string
}

var deprecatedV5Symbols = []deprecatedV5Symbol{ //nolint:gochecknoglobals // static table
	// ADR-0123: stack root — Bundle + Materialize + projection runner.
	{fragment: "stack", symbol: "Bundle", replacement: "system.New composition"},
	{
		fragment:    "stack",
		symbol:      "New",
		replacement: "system.New with DomainConfig + DeploymentConfig",
	},
	{fragment: "stack", symbol: "Materialize", replacement: "metaengine auto-projection"},
	{fragment: "stack", symbol: "NewMaterialize", replacement: "metaengine auto-projection"},
	{fragment: "stack", symbol: "RunProjections", replacement: "projectionhost.Host"},
	{
		fragment:    "stack",
		symbol:      "TombstonePolicy",
		replacement: "event-type-driven deletion (ADR-0114)",
	},
	{
		fragment:    "stack",
		symbol:      "IncludeTombstoned",
		replacement: "event-type-driven deletion (ADR-0114)",
	},
	{
		fragment:    "stack",
		symbol:      "ExcludeTombstoned",
		replacement: "event-type-driven deletion (ADR-0114)",
	},
	{
		fragment:    "stack",
		symbol:      "OnlyTombstoned",
		replacement: "event-type-driven deletion (ADR-0114)",
	},
	// ADR-0123: graph projection tier (GraphDriver/GraphSink survive via graphadapter).
	{
		fragment:    "graph",
		symbol:      "GraphProjection",
		replacement: "metaengine auto-projection + graphadapter",
	},
	{
		fragment:    "graph",
		symbol:      "NewGraphProjection",
		replacement: "metaengine auto-projection + graphadapter",
	},
	// ADR-0126: deprecated store/journal shells.
	{
		fragment:    "schema",
		symbol:      "VersionedStore",
		replacement: "schema.UpcastSourceTransform + event.DecorateStore",
	},
	{
		fragment:    "schema",
		symbol:      "NewVersionedStore",
		replacement: "schema.UpcastSourceTransform + event.DecorateStore",
	},
	{
		fragment:    "schema",
		symbol:      "VersionedSeekableJournal",
		replacement: "schema.UpcastSourceTransform + event.DecorateJournal",
	},
	{
		fragment:    "schema",
		symbol:      "NewVersionedSeekableJournal",
		replacement: "schema.UpcastSourceTransform + event.DecorateJournal",
	},
	// ADR-0126: signing middleware shells.
	{
		fragment:    "signing",
		symbol:      "RejectingPublishMiddleware",
		replacement: "event.RejectingPublishMiddleware",
	},
	{
		fragment:    "signing",
		symbol:      "RejectingHandlerMiddleware",
		replacement: "event.RejectingHandlerMiddleware",
	},
	// ADR-0126: encryption error shells.
	{
		fragment:    "encryption",
		symbol:      "ErrInnerStoreNotJournal",
		replacement: "event.ErrInnerStoreNotJournal",
	},
	{
		fragment:    "encryption",
		symbol:      "ErrInnerStoreNotSeekable",
		replacement: "event.ErrInnerStoreNotSeekable",
	},
	{
		fragment:    "encryption",
		symbol:      "ErrInnerStoreNotBackwards",
		replacement: "event.ErrInnerStoreNotBackwards",
	},
	// ADR-0126: metadata CustomData alias.
	{fragment: "metadata", symbol: "CustomData", replacement: "metadata.Metadata[K]"},
	// ADR-0114: tombstones replaced by domain events.
	{
		fragment:    "event",
		symbol:      "DetectTombstone",
		replacement: "domain events for deletion (docs/migration/tombstone-to-domain-events.md)",
	},
	{
		fragment:    "event",
		symbol:      "MarkTombstone",
		replacement: "domain events for deletion (docs/migration/tombstone-to-domain-events.md)",
	},
	{
		fragment:    "event",
		symbol:      "MarkRebirth",
		replacement: "domain events for restore (docs/migration/tombstone-to-domain-events.md)",
	},
	{fragment: "event", symbol: "MetadataKeyTombstone", replacement: "domain events for deletion"},
	{fragment: "event", symbol: "MetadataKeyRebirth", replacement: "domain events for restore"},
	{fragment: "event", symbol: "TombstoneMark", replacement: "domain events for deletion"},
	{
		fragment:    "event",
		symbol:      "TombstoneStatus",
		replacement: "listing.StreamStatus or domain events",
	},
	{
		fragment:    "event",
		symbol:      "TombstoneActive",
		replacement: "listing.StreamStatus or domain events",
	},
	{
		fragment:    "event",
		symbol:      "TombstoneTombstoned",
		replacement: "listing.StreamStatus or domain events",
	},
	{
		fragment:    "event",
		symbol:      "TombstoneUndetermined",
		replacement: "listing.StreamStatus or domain events",
	},
	{fragment: "event", symbol: "EnsureCustom", replacement: "event.Metadata.WithCustom"},
	// ADR-0126: metadata in-place mutation.
	{fragment: "metadata", symbol: "EnsureCustom", replacement: "metadata.WithCustom"},
}
