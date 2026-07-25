# EventCatalog Auto-Generation: Comprehensive Design

> **Date:** 2026-05-25
> **Status:** Research Complete — Ready for Implementation
> **Scope:** `catalog/eventcatalog/` exporter + `catalog/` types

---

## 1. Executive Summary

EventCatalog is an MDX-based documentation system for event-driven architectures. Our `catalog/eventcatalog/` exporter currently generates **basic** files covering ~20% of EventCatalog's capabilities. This document maps **everything EventCatalog supports** against our current coverage, identifies the gaps, and proposes concrete implementation steps to reach near-100% coverage.

---

## 2. EventCatalog Resource Inventory (Everything It Supports)

### 2.1 Resource Types

| #   | Resource          | Directory                                  | Required Frontmatter                      | Status     |
| --- | ----------------- | ------------------------------------------ | ----------------------------------------- | ---------- |
| 1   | **Domains**       | `domains/{name}/index.mdx`                 | `id`, `name`, `version`                   | ✅ Partial |
| 2   | **Services**      | `services/{name}/index.mdx`                | `id`, `name`, `version`                   | ✅ Partial |
| 3   | **Events**        | `services/{svc}/events/{name}/index.mdx`   | `id`, `name`, `version`                   | ✅ Partial |
| 4   | **Commands**      | `services/{svc}/commands/{name}/index.mdx` | `id`, `name`, `version`                   | ✅ Partial |
| 5   | **Queries**       | `services/{svc}/queries/{name}/index.mdx`  | `id`, `name`, `version`                   | ✅ Partial |
| 6   | **Channels**      | `channels/{name}/index.mdx`                | `id`, `name`, `version`                   | ❌ Missing |
| 7   | **Flows**         | `flows/{name}/index.mdx`                   | `id`, `name`, `version`, `steps`          | ❌ Missing |
| 8   | **Data Stores**   | `data/{name}/index.mdx`                    | `id`, `name`, `version`, `container_type` | ❌ Missing |
| 9   | **Data Products** | `data-products/{name}/index.mdx`           | `id`, `name`, `version`                   | ❌ Missing |
| 10  | **Agents**        | `agents/{name}/index.mdx`                  | `id`, `name`, `version`                   | ❌ Missing |
| 11  | **Diagrams**      | `diagrams/{name}/index.mdx`                | `id`, `name`, `version`                   | ❌ Missing |
| 12  | **Teams**         | `teams/{name}.mdx`                         | `id`, `name`                              | ❌ Missing |
| 13  | **Users**         | `users/{name}.mdx`                         | `id`, `name`                              | ❌ Missing |
| 14  | **Changelogs**    | `{resource}/{name}/changelog.mdx`          | `createdAt`                               | ❌ Missing |

### 2.2 Configuration Files

| #   | File                     | Purpose               | Status   |
| --- | ------------------------ | --------------------- | -------- |
| 1   | `eventcatalog.config.js` | Catalog configuration | ✅ Basic |
| 2   | `package.json`           | NPM dependencies      | ✅ Done  |
| 3   | `llms.txt`               | LLM-readable summary  | ✅ Done  |

### 2.3 Schema Files

| #   | Format      | How Attached                             | Status     |
| --- | ----------- | ---------------------------------------- | ---------- |
| 1   | JSON Schema | `schemaPath: schemas/schema.json` + file | ✅ Done    |
| 2   | Avro        | `schemaPath: schema.avsc` + file         | ❌ Missing |
| 3   | Protobuf    | `schemaPath: schema.proto` + file        | ❌ Missing |

### 2.4 Cross-Cutting Features (per resource)

| Feature                      | What It Does                 | Where Used                   | Status                         |
| ---------------------------- | ---------------------------- | ---------------------------- | ------------------------------ |
| `sends` / `receives`         | Message routing on services  | Services, Domains, Agents    | ✅ Services, ❌ Domains/Agents |
| `producers` / `consumers`    | Service routing on messages  | Events, Commands, Queries    | ❌ Missing                     |
| `owners`                     | Team/user ownership          | All resources                | ✅ Partial                     |
| `badges`                     | Visual badges                | All resources                | ❌ Missing                     |
| `specifications`             | AsyncAPI/OpenAPI files       | Services                     | ❌ Missing                     |
| `attachments`                | External links               | All resources                | ❌ Missing                     |
| `changelog` (inline)         | Change history               | Messages                     | ✅ Partial                     |
| `changelog.mdx`              | Dedicated changelog page     | Services, Domains, Messages  | ❌ Missing                     |
| `repository`                 | Code repo info               | Services, Messages, Channels | ❌ Missing                     |
| `deprecated`                 | Deprecation marker           | Messages                     | ✅ Basic                       |
| `labels`                     | Key-value tags               | Messages                     | ✅ Done                        |
| `operation`                  | HTTP method/path mapping     | Messages                     | ❌ Missing                     |
| `detailsPanel`               | UI panel visibility          | All resources                | ❌ Missing                     |
| `editUrl`                    | Edit link override           | All resources                | ❌ Missing                     |
| `visualiser`                 | Toggle visualization         | All resources                | ❌ Missing                     |
| `writesTo` / `readsFrom`     | Data store connections       | Services                     | ❌ Missing                     |
| `entities`                   | Entity ownership             | Services, Domains            | ❌ Missing                     |
| `routes`                     | Channel chaining             | Channels                     | ❌ Missing                     |
| `deliveryGuarantee`          | Delivery semantics           | Channels                     | ❌ Missing                     |
| `parameters`                 | Dynamic channel naming       | Channels                     | ❌ Missing                     |
| `sends.to` / `receives.from` | Channel routing via services | Services                     | ❌ Missing                     |
| `steps`                      | Flow definition              | Flows                        | ❌ Missing                     |
| `model`                      | AI agent model metadata      | Agents                       | ❌ Missing                     |
| `tools`                      | AI agent tooling             | Agents                       | ❌ Missing                     |
| `members`                    | Team membership              | Teams                        | ❌ Missing                     |

---

## 3. Gap Analysis: Current Code vs. EventCatalog Capabilities

### 3.1 What We Generate Now

```
output-dir/
├── eventcatalog.config.js          ✅ (basic: title, orgName, landingPage)
├── package.json                     ✅ (done)
├── llms.txt                         ✅ (done)
├── services/
│   └── {service-id}/
│       ├── index.mdx                ✅ (id, name, version, summary, owners, sends, receives, commands, queries)
│       ├── commands/{id}/
│       │   ├── index.mdx            ✅ (id, name, version, summary, deprecated, owners, labels, changelog, schemaPath)
│       │   └── schemas/schema.json  ✅ (JSON Schema)
│       ├── events/{id}/
│       │   ├── index.mdx            ✅ (same as commands)
│       │   └── schemas/schema.json  ✅
│       └── queries/{id}/
│           ├── index.mdx            ✅ (same as commands)
│           └── schemas/schema.json  ✅
└── domains/
    └── {domain-id}/
        └── index.mdx                ✅ (id, name, version, summary, owners, services)
```

### 3.2 What EventCatalog CAN Show (full capability)

```
output-dir/
├── eventcatalog.config.js           (full config: cId, sidebar, visualiser, changelog, rss, llms, etc.)
├── package.json
├── llms.txt
│
├── domains/
│   └── {domain}/
│       ├── index.mdx                (full: sends, receives, entities, badges, specifications, attachments, flows, data-products, subdomains)
│       ├── changelog.mdx            (createdAt, badges + markdown content)
│       ├── flows/{flow}/index.mdx   (steps with services, messages, actors, channels)
│       ├── services/{svc}/          (nested services inside domain)
│       ├── agents/{agent}/index.mdx
│       └── data-products/{dp}/index.mdx
│
├── services/
│   └── {service}/
│       ├── index.mdx                (full: writesTo, readsFrom, entities, badges, specifications, attachments, flows, repository)
│       ├── changelog.mdx
│       ├── commands/{cmd}/
│       │   ├── index.mdx            (full: producers, consumers, operation, badges, repository, specifications)
│       │   ├── changelog.mdx
│       │   ├── schemas/schema.json  (or .avsc, .proto)
│       │   └── examples.json
│       ├── events/{evt}/            (same structure)
│       ├── queries/{q}/             (same structure)
│       └── flows/{flow}/index.mdx
│
├── channels/
│   └── {channel}/
│       ├── index.mdx                (address, protocols, deliveryGuarantee, parameters, routes)
│       └── changelog.mdx
│
├── data/
│   └── {datastore}/
│       ├── index.mdx                (container_type, technology, classification, retention, residency)
│       └── schemas/                 (SQL DDL, etc.)
│
├── data-products/
│   └── {dp}/index.mdx               (inputs, outputs, schemaPath, repository)
│
├── agents/
│   └── {agent}/index.mdx            (model, tools, receives, sends, readsFrom, writesTo)
│
├── diagrams/
│   └── {diagram}/index.mdx          (mermaid, embedded, images)
│
├── teams/
│   └── {team}.mdx                   (members, email, slackDirectMessageUrl)
│
└── users/
    └── {user}.mdx                   (avatarUrl, role, email, slackDirectMessageUrl)
```

### 3.3 Priority Assessment

Not everything is equally valuable for a CQRS library. Here's the prioritization:

#### Tier 1: Core CQRS (MUST HAVE — directly maps to our domain)

| Resource                              | Why                                                                           | Effort |
| ------------------------------------- | ----------------------------------------------------------------------------- | ------ |
| Services with full frontmatter        | `writesTo`, `readsFrom`, `entities`, `specifications`, `repository`, `badges` | Small  |
| Messages with `producers`/`consumers` | Auto-link messages to owning services                                         | Small  |
| Channels                              | Core CQRS transport concept                                                   | Medium |
| Flows                                 | Visualize command → handler → event flow                                      | Medium |
| `sends.to` / `receives.from` routing  | Connect messages through channels                                             | Small  |
| Data stores (`data/`)                 | Event stores, read stores                                                     | Medium |
| Changelog files (`changelog.mdx`)     | Version history pages                                                         | Small  |

#### Tier 2: Enrichment (NICE TO HAVE — adds polish)

| Resource                          | Why                                           | Effort |
| --------------------------------- | --------------------------------------------- | ------ |
| Teams & Users                     | Ownership tracking                            | Small  |
| Badges                            | Visual categorization                         | Small  |
| `operation` on messages           | HTTP mapping (GET/POST/PUT/DELETE)            | Small  |
| Specifications on services        | Embed AsyncAPI/OpenAPI generated docs         | Medium |
| Attachments                       | Link to ADRs, runbooks                        | Small  |
| `repository` metadata             | Code links                                    | Small  |
| Enhanced `eventcatalog.config.js` | Enable features (changelogs, RSS, visualizer) | Small  |

#### Tier 3: Stretch (LOW PRIORITY — future consideration)

| Resource              | Why                                               | Effort |
| --------------------- | ------------------------------------------------- | ------ |
| Data products         | More analytics/data-mesh oriented                 | Medium |
| Agents                | AI agent documentation (new EventCatalog feature) | Medium |
| Diagrams              | Custom architecture diagrams                      | Medium |
| Avro/Protobuf schemas | Alternative schema formats                        | Medium |

---

## 4. Implementation Plan

### Phase 1: Enrich Existing Types (Small — Foundation for Everything)

**Goal:** Add missing fields to `catalog/` types so all exporters benefit.

#### Step 1.1: Extend `Service` struct

```go
// catalog/types.go — Service additions
type Service struct {
    ID       ServiceID  `json:"id"`
    Name     string     `json:"name"`
    Version  string     `json:"version"`
    Summary  string     `json:"summary,omitempty"`
    Owners   []string   `json:"owners,omitempty"`
    Commands []Message  `json:"commands,omitempty"`
    Events   []Message  `json:"events,omitempty"`
    Queries  []Message  `json:"queries,omitempty"`
    // NEW:
    WritesTo     []string        `json:"writesTo,omitempty"`     // data store IDs
    ReadsFrom    []string        `json:"readsFrom,omitempty"`    // data store IDs
    Entities     []string        `json:"entities,omitempty"`     // entity names
    Flows        []string        `json:"flows,omitempty"`        // flow IDs
    Repository   *Repository     `json:"repository,omitempty"`   // code repo info
    Badges       []Badge         `json:"badges,omitempty"`       // visual badges
    Specifications []Specification `json:"specifications,omitempty"` // AsyncAPI/OpenAPI
    Attachments  []Attachment    `json:"attachments,omitempty"`  // external links
}
```

#### Step 1.2: Extend `Message` struct

```go
// catalog/types.go — Message additions
type Message struct {
    // ... existing fields ...
    // NEW:
    Producers   []string  `json:"producers,omitempty"`   // service IDs that produce
    Consumers   []string  `json:"consumers,omitempty"`   // service IDs that consume
    Operation   *Operation `json:"operation,omitempty"`  // HTTP mapping
    Badges      []Badge    `json:"badges,omitempty"`      // visual badges
    Repository  *Repository `json:"repository,omitempty"` // code repo
}
```

#### Step 1.3: Add new shared types

```go
// catalog/types.go — New types

type Badge struct {
    Content         string `json:"content"`
    BackgroundColor string `json:"backgroundColor,omitempty"`
    TextColor       string `json:"textColor,omitempty"`
    Icon            string `json:"icon,omitempty"`
    URL             string `json:"url,omitempty"`
}

type Repository struct {
    Language string `json:"language,omitempty"`
    URL      string `json:"url,omitempty"`
}

type Specification struct {
    Type string `json:"type"`            // "asyncapi" | "openapi"
    Path string `json:"path"`            // file path
    Name string `json:"name,omitempty"`  // display name
}

type Attachment struct {
    URL         string `json:"url"`
    Title       string `json:"title,omitempty"`
    Description string `json:"description,omitempty"`
    Type        string `json:"type,omitempty"`   // grouping key
    Icon        string `json:"icon,omitempty"`
}

type Operation struct {
    Method      string   `json:"method"`                // GET, POST, PUT, DELETE, PATCH
    Path        string   `json:"path"`                  // URL path template
    StatusCodes []string `json:"statusCodes,omitempty"` // ["200", "404"]
}
```

#### Step 1.4: Extend `Domain` struct

```go
// catalog/types.go — Domain additions
type Domain struct {
    // ... existing fields ...
    // NEW:
    Sends       []MessagePointer `json:"sends,omitempty"`
    Receives    []MessagePointer `json:"receives,omitempty"`
    Entities    []string         `json:"entities,omitempty"`
    Flows       []string         `json:"flows,omitempty"`
    Badges      []Badge          `json:"badges,omitempty"`
    Attachments []Attachment     `json:"attachments,omitempty"`
}

type MessagePointer struct {
    ID      string `json:"id"`
    Version string `json:"version,omitempty"`
}
```

#### Step 1.5: Add `DataStore`, `Channel` extensions, `Flow`, `Team`, `User`

```go
// catalog/types.go — New resource types

type DataStore struct {
    ID             string   `json:"id"`
    Name           string   `json:"name"`
    Version        string   `json:"version"`
    Summary        string   `json:"summary,omitempty"`
    ContainerType  string   `json:"containerType"` // database, cache, objectStore, etc.
    Technology     string   `json:"technology,omitempty"` // "postgres@14"
    Classification string   `json:"classification,omitempty"` // internal, external, etc.
    Retention      string   `json:"retention,omitempty"`
    Residency      string   `json:"residency,omitempty"`
    Owners         []string `json:"owners,omitempty"`
    Badges         []Badge  `json:"badges,omitempty"`
}

type Flow struct {
    ID      string     `json:"id"`
    Name    string     `json:"name"`
    Version string     `json:"version"`
    Summary string     `json:"summary,omitempty"`
    Steps   []FlowStep `json:"steps"`
    Badges  []Badge    `json:"badges,omitempty"`
}

type FlowStep struct {
    ID       string `json:"id"`
    Title    string `json:"title"`
    Summary  string `json:"summary,omitempty"`
    // Exactly one of these should be set:
    Service        *FlowStepRef `json:"service,omitempty"`
    Message        *FlowStepRef `json:"message,omitempty"`
    Channel        *FlowStepRef `json:"channel,omitempty"`
    Actor          *FlowActor   `json:"actor,omitempty"`
    ExternalSystem *FlowActor   `json:"externalSystem,omitempty"`
    Custom         *FlowCustom  `json:"custom,omitempty"`
    // Connection:
    NextStep  *FlowConnection  `json:"next_step,omitempty"`
    NextSteps []FlowConnection `json:"next_steps,omitempty"`
}

type FlowStepRef struct {
    ID      string `json:"id"`
    Version string `json:"version,omitempty"`
}

type FlowActor struct {
    Name    string `json:"name"`
    Summary string `json:"summary,omitempty"`
}

type FlowCustom struct {
    Title   string `json:"title"`
    Icon    string `json:"icon,omitempty"`
    Type    string `json:"type,omitempty"`
    Summary string `json:"summary,omitempty"`
    URL     string `json:"url,omitempty"`
    Color   string `json:"color,omitempty"`
}

type FlowConnection struct {
    ID    string `json:"id"`
    Label string `json:"label,omitempty"`
}

type Team struct {
    ID                   string   `json:"id"`
    Name                 string   `json:"name"`
    Summary              string   `json:"summary,omitempty"`
    Members              []string `json:"members,omitempty"`
    Email                string   `json:"email,omitempty"`
    SlackDirectMessageURL string  `json:"slackDirectMessageUrl,omitempty"`
}

type User struct {
    ID                   string `json:"id"`
    Name                 string `json:"name"`
    AvatarURL            string `json:"avatarUrl,omitempty"`
    Role                 string `json:"role,omitempty"`
    Email                string `json:"email,omitempty"`
    SlackDirectMessageURL string `json:"slackDirectMessageUrl,omitempty"`
}
```

#### Step 1.6: Add to `Channel` struct

```go
// catalog/types.go — Channel additions
type Channel struct {
    // ... existing fields ...
    // NEW:
    DeliveryGuarantee string            `json:"deliveryGuarantee,omitempty"` // at-most-once, at-least-once, exactly-once
    Parameters        map[string]ChannelParameter `json:"parameters,omitempty"`
    Routes            []ChannelRoute   `json:"routes,omitempty"`
    Owners            []string         `json:"owners,omitempty"`
    Badges            []Badge          `json:"badges,omitempty"`
}

type ChannelParameter struct {
    Enum        []string `json:"enum,omitempty"`
    Default     string   `json:"default,omitempty"`
    Examples    []string `json:"examples,omitempty"`
    Description string   `json:"description,omitempty"`
}

type ChannelRoute struct {
    ID string   `json:"id"`
    To []string `json:"to,omitempty"` // channel IDs
}
```

#### Step 1.7: Add to `Catalog` struct

```go
// catalog/types.go — Catalog additions
type Catalog struct {
    Title    string      `json:"title"`
    Version  string      `json:"version"`
    Services []Service   `json:"services"`
    Domains  []Domain    `json:"domains,omitempty"`
    Channels []Channel   `json:"channels,omitempty"`
    // NEW:
    DataStores   []DataStore   `json:"dataStores,omitempty"`
    Flows        []Flow        `json:"flows,omitempty"`
    Teams        []Team        `json:"teams,omitempty"`
    Users        []User        `json:"users,omitempty"`
}
```

#### Step 1.8: Update Registry

Add `AddDataStore`, `AddFlow`, `AddTeam`, `AddUser` methods. Update `Build()` to include new resources in the immutable snapshot. Update copy helpers.

**Files changed:** `catalog/types.go`, `catalog/registry.go`, `catalog/registry_helpers.go`

---

### Phase 2: Enrich Builder API (Small — Consumer-Facing)

**Goal:** Make it easy for consumers to use the new fields.

#### Step 2.1: Add `ServiceOption` functional options

```go
// catalog/service_config.go — new file

type ServiceOption func(*serviceBuilder)

type serviceBuilder struct {
    id, name, version, summary string
    owners                     []string
    writesTo, readsFrom        []string
    entities                   []string
    flows                      []string
    repository                 *Repository
    badges                     []Badge
    specifications             []Specification
    attachments                []Attachment
}

func ServiceWritesTo(storeIDs ...string) ServiceOption { ... }
func ServiceReadsFrom(storeIDs ...string) ServiceOption { ... }
func ServiceEntities(entities ...string) ServiceOption { ... }
func ServiceFlows(flowIDs ...string) ServiceOption { ... }
func ServiceRepository(lang, url string) ServiceOption { ... }
func ServiceBadges(badges ...Badge) ServiceOption { ... }
func ServiceSpecifications(specs ...Specification) ServiceOption { ... }
func ServiceAttachments(attachments ...Attachment) ServiceOption { ... }
```

Update `Builder.AddService` to accept `ServiceOption`:

```go
func (b *Builder) AddService(id, name, version, summary string, msgs ...MessageConfig) {
    // existing behavior
}
```

Or add a separate method:

```go
func (b *Registry) SetServiceOptions(serviceID ServiceID, opts ...ServiceOption) { ... }
```

#### Step 2.2: Add `MessageOption` extensions

```go
// catalog/message_config.go additions
func Producers(serviceIDs ...string) MessageOption { ... }
func Consumers(serviceIDs ...string) MessageOption { ... }
func Operation(method, path string, statusCodes ...string) MessageOption { ... }
func MessageBadges(badges ...Badge) MessageOption { ... }
func MessageRepository(lang, url string) MessageOption { ... }
```

#### Step 2.3: Add `DomainOption` functional options

```go
func DomainSends(msgs ...MessagePointer) DomainOption { ... }
func DomainReceives(msgs ...MessagePointer) DomainOption { ... }
func DomainEntities(entities ...string) DomainOption { ... }
func DomainBadges(badges ...Badge) DomainOption { ... }
```

#### Step 2.4: Add `Builder.AddFlow`, `Builder.AddDataStore`, `Builder.AddTeam`, `Builder.AddUser`

```go
func (b *Builder) AddFlow(id, name, version, summary string, steps ...FlowStep) { ... }
func (b *Builder) AddDataStore(ds DataStore) { ... }
func (b *Builder) AddTeam(team Team) { ... }
func (b *Builder) AddUser(user User) { ... }
```

**Files changed:** `catalog/message_config.go`, new `catalog/service_config.go`, `catalog/build.go`

---

### Phase 3: Update EventCatalog Exporter (Medium — The Main Deliverable)

**Goal:** Generate complete EventCatalog files from enriched types.

#### Step 3.1: Enhance service MDX output

**Current output:**

```yaml
id: order-svc
name: Order Service
version: 1.0.0
summary: "Manages orders"
sends:
  - id: OrderCreated
commands:
  - id: CreateOrder
queries:
  - id: GetOrder
```

**Enhanced output:**

```yaml
---
id: order-svc
name: Order Service
version: 1.0.0
summary: "Manages orders"
owners:
  - order-team
sends:
  - id: OrderCreated
receives:
  - id: OrderCancelled
commands:
  - id: CreateOrder
queries:
  - id: GetOrder
writesTo:
  - id: orders-db
    version: 1.0.0
readsFrom:
  - id: products-cache
    version: 1.0.0
entities:
  - id: Order
  - id: OrderItem
badges:
  - content: Production
    backgroundColor: green
    textColor: green
repository:
  language: Go
  url: https://github.com/example/order-service
---
# Order Service

Manages orders

<NodeGraph />
```

#### Step 3.2: Enhance message MDX output

**Enhanced output for events:**

```yaml
---
id: OrderCreated
name: Order Created
version: 1.0.0
summary: "Fired when a new order is placed"
schemaPath: schemas/schema.json
producers:
  - id: order-svc
    version: 1.0.0
consumers:
  - id: notification-svc
  - id: analytics-svc
owners:
  - order-team
badges:
  - content: Domain Event
    backgroundColor: orange
    textColor: orange
---
# Order Created

Fired when a new order is placed
```

#### Step 3.3: Add channel MDX generation

New `writeChannel` method:

```yaml
---
id: order-events
name: Order Events Channel
version: 1.0.0
summary: "Central event stream for order-related events"
address: orders.{env}.events
protocols:
  - kafka
deliveryGuarantee: at-least-once
parameters:
  env:
    enum:
      - dev
      - stg
      - prod
    default: dev
owners:
  - platform-team
---
# Order Events Channel
```

#### Step 3.4: Add data store MDX generation

```yaml
---
id: orders-db
name: Orders Database
version: 1.0.0
container_type: database
technology: postgres@16
classification: internal
summary: "Primary data store for the orders domain"
owners:
  - order-team
---
# Orders Database
```

#### Step 3.5: Add flow MDX generation

```yaml
---
id: create-order-flow
name: Create Order Flow
version: 1.0.0
summary: "Complete flow for creating a new order"
steps:
  - id: 1
    title: Create Order Command
    message:
      id: CreateOrder
      version: 1.0.0
    next_step:
      id: 2
      label: submit
  - id: 2
    title: Order Service
    service:
      id: order-svc
      version: 1.0.0
    next_step:
      id: 3
      label: publishes
  - id: 3
    title: Order Created Event
    message:
      id: OrderCreated
      version: 1.0.0
---
# Create Order Flow

Complete flow for creating a new order

<NodeGraph />
```

#### Step 3.6: Add team/user MDX generation

```yaml
# teams/order-team.mdx
---
id: order-team
name: Order Team
summary: "Team responsible for order management"
members:
  - alice
  - bob
email: orders@example.com
---
# Order Team
```

```yaml
# users/alice.mdx
---
id: alice
name: Alice Smith
role: Senior Engineer
email: alice@example.com
---
# Alice Smith
```

#### Step 3.7: Add changelog MDX generation

```yaml
# services/order-svc/changelog.mdx
---
createdAt: 2026-05-25
badges:
  - content: "Added query"
    backgroundColor: green
    textColor: green
---
### Added GetOrder query

Service now supports querying orders by ID.
```

#### Step 3.8: Auto-generate `producers`/`consumers` from service topology

**Key insight:** When a service has `sends: [OrderCreated]`, we can automatically infer that:

- The service is a **producer** of `OrderCreated`
- The event's `producers` array should include this service

This cross-referencing should happen in `Export()` based on the catalog data.

#### Step 3.9: Enhanced `eventcatalog.config.js`

```js
/** @type {import('@eventcatalog/core/bin/eventcatalog.config').Config} */
export default {
	title: "E-Commerce",
	organizationName: "E-Commerce",
	landingPage: "/docs",
	cId: "auto-generated-unique-id",
	changelog: {
		enabled: true,
	},
	llms: {
		enabled: true,
	},
	docs: {
		sidebar: {
			type: "TREE_VIEW",
		},
	},
};
```

**Files changed:** `catalog/eventcatalog/exporter.go`, `catalog/eventcatalog/writer.go`

---

### Phase 4: Auto-Derivation Intelligence (Medium — The Differentiator)

**Goal:** Automatically derive relationships from existing catalog data, minimizing what consumers need to specify manually.

#### Step 4.1: Auto-derive `producers`/`consumers` from `sends`/`receives`

When `Service.sends` includes `OrderCreated`, automatically add `order-svc` to `OrderCreated.producers`.

When `Service.receives` includes `OrderPlaced`, automatically add `order-svc` to `OrderPlaced.consumers`.

This eliminates manual cross-referencing.

#### Step 4.2: Auto-generate Flows from Decider topology

A `Decider[State]` naturally defines a flow:

1. Command arrives (message step)
2. Service processes it (service step)
3. Events emitted (message steps)

If we can access decider metadata, we can auto-generate a flow per command showing the full CQRS pipeline:

```
[CreateOrder command] → [Order Service] → [OrderCreated event]
```

#### Step 4.3: Auto-generate Channel from message patterns

If all events from a service share a naming convention (e.g., `order.*`), auto-suggest a channel with `address: orders.events`.

**Files changed:** new `catalog/eventcatalog/auto_derive.go`

---

### Phase 5: Tests & Golden Files (Medium — Quality Assurance)

#### Step 5.1: Update golden test files

Add golden files for:

- `eventcatalog-service-full.mdx` — service with all new fields
- `eventcatalog-event-full.mdx` — message with producers/consumers/badges
- `eventcatalog-channel.mdx` — new channel type
- `eventcatalog-datastore.mdx` — new data store type
- `eventcatalog-flow.mdx` — new flow type
- `eventcatalog-team.mdx` — new team type
- `eventcatalog-user.mdx` — new user type
- `eventcatalog-changelog.mdx` — new changelog type

#### Step 5.2: Update existing tests

All existing tests must continue passing (backward-compatible additions only).

#### Step 5.3: Integration test

Test the full pipeline: `Builder → Registry → Build → Export → verify all files`.

**Files changed:** `catalog/eventcatalog/exporter_test.go`, `catalog/eventcatalog/golden_test.go`, golden testdata files

---

## 5. Backward Compatibility

All changes are **additive**:

- New struct fields are `omitempty` — existing code compiles without changes
- New registry methods are additions — existing API unchanged
- New exporter output adds files — existing files unchanged
- `Builder.AddService` signature unchanged (variadic `...MessageConfig`)
- Existing golden tests keep passing

---

## 6. File Size Budget

Per project convention (max 250 lines/file):

| File                               | Current Lines | After Changes | Action                                                                                           |
| ---------------------------------- | ------------- | ------------- | ------------------------------------------------------------------------------------------------ |
| `catalog/types.go`                 | 175           | ~300          | Split: extract `types_resources.go` for DataStore/Flow/Team/User (~120 lines)                    |
| `catalog/eventcatalog/exporter.go` | 252           | ~350          | Split: extract `exporter_resources.go` for channel/flow/datastore/team/user writing (~120 lines) |
| `catalog/eventcatalog/writer.go`   | 198           | ~280          | Split: extract `writer_frontmatter.go` for new frontmatter helpers (~100 lines)                  |
| `catalog/registry.go`              | 229           | ~300          | Split: extract `registry_resources.go` for new Add methods (~80 lines)                           |
| `catalog/registry_helpers.go`      | 90            | ~150          | Manageable in one file                                                                           |

---

## 7. Success Metrics

| Metric                         | Current                      | Target                    |
| ------------------------------ | ---------------------------- | ------------------------- |
| EventCatalog features covered  | ~20%                         | ~85%                      |
| Resource types generated       | 3 (service, message, domain) | 10+ (all Tier 1 + Tier 2) |
| Frontmatter fields per service | 9                            | 20+                       |
| Frontmatter fields per message | 10                           | 15+                       |
| Cross-reference accuracy       | Manual                       | Auto-derived              |
| Test coverage (eventcatalog/)  | ~91%                         | >90%                      |

---

## 8. Execution Order

| Step      | Phase | Description                                              | Effort   | Dependency |
| --------- | ----- | -------------------------------------------------------- | -------- | ---------- |
| 1         | P1    | Add shared types (Badge, Repository, etc.) to `types.go` | 1h       | None       |
| 2         | P1    | Add DataStore, Flow, Team, User types                    | 1h       | Step 1     |
| 3         | P1    | Extend Service, Message, Domain, Channel structs         | 30min    | Step 1     |
| 4         | P1    | Add to Catalog struct                                    | 15min    | Step 2     |
| 5         | P1    | Update Registry with new Add methods                     | 1h       | Steps 2-4  |
| 6         | P1    | Update copy helpers                                      | 30min    | Steps 2-4  |
| 7         | P2    | Add ServiceOption + MessageOption extensions             | 1h       | Steps 1-3  |
| 8         | P2    | Add Builder.AddFlow, AddDataStore, etc.                  | 30min    | Step 5     |
| 9         | P3    | Update eventcatalog exporter for new fields              | 2h       | Steps 5-8  |
| 10        | P3    | Add channel/datastore/flow/team/user MDX generation      | 2h       | Step 9     |
| 11        | P3    | Auto-derive producers/consumers                          | 1h       | Step 9     |
| 12        | P4    | Enhanced eventcatalog.config.js                          | 30min    | Step 9     |
| 13        | P5    | Update tests + golden files                              | 2h       | Steps 9-12 |
| 14        | P4    | Auto-generate flows from decider topology                | 2h       | Step 10    |
| **Total** |       |                                                          | **~15h** |            |

---

## 9. Key Design Decisions

### D1: Struct extension vs. new types

**Decision:** Extend existing structs with `omitempty` fields rather than creating parallel types.

**Rationale:** Backward compatible. Existing `Service{ID: "x"}` still compiles. No consumer breakage.

### D2: Auto-derivation in Export vs. Registry

**Decision:** Auto-derive `producers`/`consumers` in the exporter, not the registry.

**Rationale:** Registry is the source of truth (what the consumer declared). Exporter enriches for a specific target format. Keeps registry clean, exporter smart.

### D3: Flow auto-generation

**Decision:** Phase 4 (stretch). Start with manual flow registration via `Builder.AddFlow`.

**Rationale:** Auto-deriving flows from decider metadata requires tight coupling to the `core/decider` package, which `catalog` shouldn't depend on. Instead, consumers who know their decider topology can construct flows manually. Future: provide a helper function in `example/` or `integration/`.

### D4: Functional options vs. struct literals

**Decision:** Use functional options (`MessageOption`, `ServiceOption`) for everything.

**Rationale:** Follows existing pattern (`catalog.Name()`, `catalog.Summary()`). Backward compatible. Discoverable via godoc.

### D5: Schema format support

**Decision:** JSON Schema only for now. Avro/Protobuf deferred.

**Rationale:** JSON Schema is what `SchemaFromType[T]()` generates. Avro/Protobuf would need a separate conversion layer. Not blocking for CQRS use cases.

---

## 10. Risks & Mitigations

| Risk                                         | Likelihood | Mitigation                                                                          |
| -------------------------------------------- | ---------- | ----------------------------------------------------------------------------------- |
| EventCatalog frontmatter format changes      | Medium     | Pin `@eventcatalog/core` version in `package.json`. Add version note.               |
| File size limits exceeded                    | Low        | Plan splits in advance (see Section 6).                                             |
| Consumer confusion about which fields to use | Low        | Provide clear `Builder` API with defaults. Only advanced consumers need new fields. |
| Test coverage drops during refactor          | Low        | Update tests incrementally per step. Run full suite after each change.              |

---

## Appendix A: EventCatalog Frontmatter Quick Reference

### Domain

```yaml
id, name, version (required)
summary, owners, services, domains, sends, receives, entities, flows,
badges, specifications, visualiser, editUrl, detailsPanel, attachments
```

### Service

```yaml
id, name, version (required)
summary, owners, sends, receives, writesTo, readsFrom, entities, flows,
badges, specifications, visualiser, editUrl, detailsPanel, attachments, repository
```

### Message (Event/Command/Query)

```yaml
id, name, version (required)
summary, producers, consumers, schemaPath, owners, badges, deprecated,
labels, changelog, operation, repository, specifications, channels,
editUrl, detailsPanel, attachments, visualiser, sidebar, draft, hidden, styles
```

### Channel

```yaml
id, name, version (required)
summary, address, protocols, deliveryGuarantee, parameters, owners, badges,
routes, repository, editUrl, detailsPanel, attachments
```

### Data Store

```yaml
id, name, version, container_type (required)
summary, technology, classification, retention, residency, owners, badges,
detailsPanel, attachments, styles
```

### Flow

```yaml
id, name, version (required)
summary, badges
steps: [{id, title, summary, service|message|actor|channel|custom, next_step|next_steps}]
```

### Team

```yaml
id, name (required)
summary, members, email, slackDirectMessageUrl
```

### User

```yaml
id, name (required)
avatarUrl, role, email, slackDirectMessageUrl
```

### Changelog (per resource)

```yaml
createdAt (required)
badges (optional)
```

Body: markdown content describing the change.
