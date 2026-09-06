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
	// ADR-0126: metadata EnsureCustom is a method (undetectable via
	// qualifier matching) — tracked in the drift method allowlist instead.
	// ADR-0111: per-module ParseType shims replaced by record.ParseType.
	{
		fragment:    "command",
		symbol:      "ParseType",
		replacement: "record.ParseType(s, ErrEmptyCommandType)",
	},
	{fragment: "event", symbol: "ParseType", replacement: "record.ParseType(s, ErrEmptyEventType)"},
	{fragment: "query", symbol: "ParseType", replacement: "record.ParseType(s, ErrEmptyQueryType)"},
	// v5: manual snapshot helper replaced by encoding-aware construction.
	{
		fragment:    "snapshot",
		symbol:      "SaveSnapshot",
		replacement: "NewSnapshot + SnapshotSink.Save (carries record.Encoding)",
	},
	// ADR-0123: graph projection surface absorbed by metaengine + graphadapter.
	{
		fragment:    "graph",
		symbol:      "Handler",
		replacement: "metaengine auto-projection + graphadapter",
	},
	{
		fragment:    "graph",
		symbol:      "ProjectionOption",
		replacement: "metaengine auto-projection + graphadapter",
	},
	{
		fragment:    "graph",
		symbol:      "WithSchema",
		replacement: "metaengine auto-projection + graphadapter",
	},
	// ADR-0114: metadata tombstone-marking middleware dies with the marks.
	{
		fragment:    "listing",
		symbol:      "StatusMiddleware",
		replacement: "event-type-driven deletion (ADR-0114)",
	},
	// v5: silent-validation helper superseded by the checked form.
	{
		fragment:    "storage/sql",
		symbol:      "KeysetPositionQuery",
		replacement: "storage/sql.KeysetPositionQueryChecked",
	},
	{
		fragment:    "storage",
		symbol:      "WithoutRelationalAutoMigrate",
		replacement: "metaengine engines with layout planning",
	},
	// ADR-0123: storage-root re-exports of the removed view tier.
	{
		fragment:    "storage",
		symbol:      "ViewColumn",
		replacement: "metaengine engines with layout planning",
	},
	{
		fragment:    "storage",
		symbol:      "ViewMapper",
		replacement: "metaengine engines with layout planning",
	},
	{
		fragment:    "storage",
		symbol:      "IndexSpec",
		replacement: "metaengine engines with layout planning",
	},
	{
		fragment:    "storage",
		symbol:      "SQLViewStore",
		replacement: "metaengine engines with layout planning",
	},
	{
		fragment:    "storage",
		symbol:      "ViewStoreOption",
		replacement: "metaengine engines with layout planning",
	},
	{
		fragment:    "storage",
		symbol:      "NewSQLiteViewStore",
		replacement: "metaengine engines with layout planning",
	},
	{
		fragment:    "storage",
		symbol:      "NewSQLViewStore",
		replacement: "metaengine engines with layout planning",
	},
	{
		fragment:    "storage",
		symbol:      "NewViewStoreWithDialect",
		replacement: "metaengine engines with layout planning",
	},
	{
		fragment:    "storage",
		symbol:      "AutoMapper",
		replacement: "metaengine engines with layout planning",
	},
	{
		fragment:    "storage",
		symbol:      "AutoMapperWithTombstone",
		replacement: "metaengine engines with layout planning",
	},
	// ADR-0123: storage-root re-exports of the removed relational tier.
	{
		fragment:    "storage",
		symbol:      "NewRelationalProjection",
		replacement: "metaengine engines with layout planning",
	},
	{
		fragment:    "storage",
		symbol:      "NewRelationalStore",
		replacement: "metaengine engines with layout planning",
	},
	{
		fragment:    "storage",
		symbol:      "RelationalProjection",
		replacement: "metaengine engines with layout planning",
	},
	{
		fragment:    "storage",
		symbol:      "RelationalProjectionOption",
		replacement: "metaengine engines with layout planning",
	},
	{
		fragment:    "storage",
		symbol:      "RelationalSchema",
		replacement: "metaengine engines with layout planning",
	},
	{
		fragment:    "storage",
		symbol:      "RelationalTable",
		replacement: "metaengine engines with layout planning",
	},
	{
		fragment:    "storage",
		symbol:      "RelationalColumn",
		replacement: "metaengine engines with layout planning",
	},
	{
		fragment:    "storage",
		symbol:      "RelationalStore",
		replacement: "metaengine engines with layout planning",
	},
	{
		fragment:    "storage",
		symbol:      "RelationalHandler",
		replacement: "metaengine engines with layout planning",
	},
	{fragment: "storage", symbol: "Row", replacement: "metaengine engines with layout planning"},
	{
		fragment:    "storage",
		symbol:      "ProjectionSink",
		replacement: "metaengine engines with layout planning",
	},
}
