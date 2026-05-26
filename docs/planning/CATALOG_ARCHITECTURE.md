## Catalog System Architecture

The `catalog` module provides automatic documentation generation from Go CQRS types to AsyncAPI 3.0 and EventCatalog formats.

### Three-Layer Design

```
┌──────────────────────────────────────────────────────┐
│                   catalog (core)                      │
│  types.go — Message, Service, Domain, Channel, Schema │
│  schema.go — SchemaFromType[T]() via reflect          │
│  registry.go — Thread-safe Registry, Build() → Catalog│
└──────────────────────┬───────────────────────────────┘
                       │ Catalog (immutable IR)
           ┌───────────┴───────────┐
           ▼                       ▼
┌─────────────────────┐  ┌─────────────────────────┐
│ catalog/asyncapi/   │  │ catalog/d2/            │  │ catalog/eventcatalog/   │
│ AsyncAPI 3.0 YAML   │  │ MDX files on disk       │
│ Document.MarshalYAML│  │ services/{id}/index.mdx │
│ Document.MarshalJSON│  │ schemas/schema.json     │
└─────────────────────┘  └─────────────────────────┘
```

### Key Design Decisions

1. **go-faster/yaml** — Replaced custom YAML marshaler (`catalog/yaml/`, deleted). Well-maintained, zero-transitive-dep YAML library.

2. **Reflection-based schema generation** — `SchemaFromType[T any]() *Schema` uses `reflect.TypeOf` to inspect struct fields. Reads `json` (name + omitempty), `doc`/`description` (description), and `format` (format) struct tags. **Anonymous (embedded) fields are automatically skipped**.

3. **Type alias for MarshalJSON** — AsyncAPI `Document.MarshalJSON()` uses `type alias Document` to break infinite recursion when calling `json.MarshalIndent`.

4. **Registry pattern** — Thread-safe with `sync.RWMutex`. `AddService` merges messages into existing services. `Build()` produces an immutable `*Catalog`.

5. **AsyncAPI mapping** — Commands → `receive`, Events with `Sends` → `send`, Events with `Receives` → `receive`, Queries → `receive`. Channel addresses via `toDotAddress` (CamelCase → dot.separated).

6. **EventCatalog structure** — MDX files with YAML frontmatter (`---` delimited). `schema.json` only created when schema is non-nil. Service frontmatter includes `sends`, `receives`, `commands`, and `queries` lists.

7. **D2 diagram export** (`catalog/d2`) — Generates D2 text from `*catalog.Catalog`. Services become containers, commands/events/queries become color-coded nodes (command=blue, event=red queue, query=purple), domains become grouping labels. Wire via `catalog.Builder` directly. Follows same `Exporter` pattern as `asyncapi` and `eventcatalog`.
