# Design: Storage Plugin System

> **Goal:** Storage backends are **plugins** (database/sql pattern). The deployer activates
> them via config at deployment time. The consumer's code is identical regardless of backend.

**Status:** Proposed (2026-07-23)
**Related:** [storage-domain-separation.md](storage-domain-separation.md),
[STORAGE_GUIDE.md](../STORAGE_GUIDE.md), [ADR-0033](../adr/0033-multi-db-split.md)

---

## Table of Contents

1. [Why Not Go's `plugin` Package?](#1-why-not-gos-plugin-package)
2. [The Pattern: database/sql Registration](#2-the-pattern-databasesql-registration)
3. [Concrete API](#3-concrete-api)
4. [Multi-Concern Deployment](#4-multi-concern-deployment)
5. [Bus Plugins](#5-bus-plugins)
6. [Why Not hashicorp/go-plugin?](#6-why-not-hashicorpgo-plugin)
7. [Migration Path](#7-migration-path)

---

## 1. Why Not Go's `plugin` Package?

Go has three mechanisms for "plugins." Two are traps.

| Mechanism                                  | How it works                                                                                        | Verdict                                                                                                                            |
| ------------------------------------------ | --------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------- |
| **`plugin` package (.so files)**           | `plugin.Open("foo.so")` loads compiled shared libs at runtime                                       | **Trap.** Linux/macOS only. Requires EXACT same Go toolchain + dependency versions. Fragile. Go team doesn't recommend it.         |
| **`hashicorp/go-plugin` (out-of-process)** | Separate plugin binaries communicate via gRPC                                                       | **Wrong level.** Adds gRPC serialization to every storage operation. For application frameworks (Terraform, Vault), not libraries. |
| **Registration pattern (blank import)**    | `import _ "driver/postgres"` -> `init()` registers -> `sql.Open("postgres", dsn)` activates by name | **THE Go way.** This is how `database/sql` works. It IS Go's plugin system.                                                        |

The `database/sql` package is Go's most successful plugin architecture. Every SQL driver
(pq, sqlite3, mysql) is a "plugin" loaded via blank import and activated by name string.

The key distinction:

```
"Selected at deployment time"  <-  config picks which registered plugin to activate
"Loaded at deployment time"    <-  binary doesn't contain the plugin at all (needs .so or go-plugin)
```

The registration pattern gives deployment-time **selection**. True runtime loading requires
either the fragile `plugin` package or heavyweight `go-plugin`. For a **library**, the
registration pattern is the only correct answer.

---

## 2. The Pattern: database/sql Registration

### How It Works Today vs. How It Should Work

```go
// === TODAY: consumer hardcodes the backend ===
bundle, _ := sqlite.New("events.db",
    sqlite.WithViewDB("views.db"),
    sqlite.WithMaxOpenConns(1),
)
// Consumer imports storage/sqlite. Consumer picks the engine.
// Changing engines = editing consumer code + recompiling.

// === TARGET: deployer picks via config ===
// Consumer's main.go:
import (
    _ "go-cqrs-lite/stack/sqlite/v4"  // registers "sqlite"
    _ "go-cqrs-lite/stack/pebble/v4"  // registers "pebble"
    _ "go-cqrs-lite/stack/memory/v4"  // registers "memory"
)

bundle, _ := stack.Deploy(loadConfig("deploy.yaml"))
// Consumer's code is IDENTICAL regardless of backend.
// Deployer changes deploy.yaml -> different storage. No recompilation.
```

### The Deployer's Config

```yaml
# deploy.yaml -- the deployer owns this file, not the developer
events:
  driver: sqlite
  dsn: /data/events.db
  options:
    wal: true
    maxOpenConns: 1

views:
  driver: pebble
  dsn: /data/views

commands:
  driver: sqlite
  dsn: /data/commands.db

bus:
  driver: memory
```

The developer never writes this file. The deployer does. Changing storage = editing YAML,
not editing Go code.

---

## 3. Concrete API

### The Plugin Interface and Registry

```go
// stack/plugin.go
package stack

import (
    "fmt"
    "sync"
)

// Plugin is the interface that every storage/bus backend implements.
// Each plugin is registered via init() and activated by name via Open/Deploy.
type Plugin interface {
    // Name uniquely identifies this plugin (e.g. "sqlite", "pebble", "memory", "nats").
    Name() string

    // Capabilities declares what storage concerns this plugin can serve.
    // Not every plugin provides everything -- a Pebble plugin might serve
    // events+snapshots+views but not command/query audit logs.
    Capabilities() CapabilitySet

    // Build creates the storage primitives from configuration.
    // Returns a Partial that fills specific Bundle fields.
    Build(cfg PluginConfig) (*Partial, error)
}

// CapabilitySet describes what a plugin can provide.
type CapabilitySet struct {
    EventStore  bool
    ViewStore   bool
    Snapshot    bool
    Checkpoint  bool
    CommandLog  bool
    QueryLog    bool
    Bus         bool
}

// Satisfies checks if this set includes all required capabilities.
func (c CapabilitySet) Satisfies(required CapabilitySet) bool {
    return (!required.EventStore || c.EventStore) &&
        (!required.ViewStore || c.ViewStore) &&
        (!required.Snapshot || c.Snapshot) &&
        (!required.Checkpoint || c.Checkpoint) &&
        (!required.CommandLog || c.CommandLog) &&
        (!required.QueryLog || c.QueryLog) &&
        (!required.Bus || c.Bus)
}

// Partial is a partially-filled Bundle contribution from one plugin.
type Partial struct {
    EventSink            event.EventSink
    EventSource          event.EventSource
    Journal              event.Journal
    SeekableJournal      event.SeekableJournal
    BackwardsSource      event.BackwardsSource
    Publisher            event.Publisher
    Subscriber           event.Subscriber
    CommandSink          command.CommandSink
    CommandSource        command.CommandSource
    SeekableCommandJournal command.SeekableCommandJournal
    QuerySink            query.QuerySink
    QuerySource          query.QuerySource
    SeekableQueryJournal query.SeekableQueryJournal
    SnapshotStore        snapshot.SnapshotStore
    CheckpointStore      event.CheckpointStore
    ReadModels           kv.Store
}

// PluginConfig is passed to a plugin's Build function.
type PluginConfig struct {
    DSN     string         // connection string (file path, URL, etc.)
    Options map[string]any // plugin-specific options
}

// --- Registry ---

var (
    pluginsMu sync.RWMutex
    plugins   = map[string]Plugin{}
)

// Register adds a plugin to the global registry.
// Called from init() in each plugin package. Panics on duplicate (same as database/sql).
func Register(p Plugin) {
    pluginsMu.Lock()
    defer pluginsMu.Unlock()
    if _, dup := plugins[p.Name()]; dup {
        panic(fmt.Sprintf("stack: plugin %q already registered", p.Name()))
    }
    plugins[p.Name()] = p
}

// Available returns the names of all registered plugins (for diagnostics).
func Available() []string {
    pluginsMu.RLock()
    defer pluginsMu.RUnlock()
    names := make([]string, 0, len(plugins))
    for name := range plugins {
        names = append(names, name)
    }
    return names
}

func getPlugin(name string) (Plugin, error) {
    pluginsMu.RLock()
    defer pluginsMu.RUnlock()
    p, ok := plugins[name]
    if !ok {
        return nil, fmt.Errorf(
            "stack: unknown plugin %q -- available: %v (did you forget to import the plugin package?)",
            name, Available(),
        )
    }
    return p, nil
}
```

### The Deploy Function

```go
// stack/deploy.go
package stack

// DeploymentConfig maps storage concerns to plugins.
// The deployer provides this (typically loaded from YAML/env/file).
type DeploymentConfig struct {
    Events    *ConcernConfig `yaml:"events,omitempty"`
    Views     *ConcernConfig `yaml:"views,omitempty"`
    Commands  *ConcernConfig `yaml:"commands,omitempty"`
    Snapshots *ConcernConfig `yaml:"snapshots,omitempty"`
    Bus       *ConcernConfig `yaml:"bus,omitempty"`
}

type ConcernConfig struct {
    Driver  string         `yaml:"driver"`
    DSN     string         `yaml:"dsn,omitempty"`
    Options map[string]any `yaml:"options,omitempty"`
}

// Deploy assembles a full Bundle from one or more plugins based on deployment config.
//
// This is the primary entry point for deployment-time storage selection.
// The deployer provides a config (typically from YAML); Deploy activates
// the right plugins and assembles a complete Bundle.
func Deploy(cfg DeploymentConfig) (*Bundle, error) {
    bundle := &Bundle{}

    concerns := []struct {
        name string
        cfg  *ConcernConfig
    }{
        {"events", cfg.Events},
        {"views", cfg.Views},
        {"commands", cfg.Commands},
        {"snapshots", cfg.Snapshots},
        {"bus", cfg.Bus},
    }

    for _, c := range concerns {
        if c.cfg == nil {
            continue // optional concern, skip
        }

        plugin, err := getPlugin(c.cfg.Driver)
        if err != nil {
            return nil, fmt.Errorf("concern %q: %w", c.name, err)
        }

        partial, err := plugin.Build(PluginConfig{
            DSN:     c.cfg.DSN,
            Options: c.cfg.Options,
        })
        if err != nil {
            return nil, fmt.Errorf("concern %q (driver %q): %w", c.name, c.cfg.Driver, err)
        }

        bundle.mergePartial(partial)
    }

    if err := bundle.validateComplete(); err != nil {
        return nil, err
    }

    return bundle, nil
}

// Open is the simple single-backend shortcut.
// Use Deploy for multi-concern/multi-backend setups.
func Open(driver string, cfg PluginConfig) (*Bundle, error) {
    plugin, err := getPlugin(driver)
    if err != nil {
        return nil, err
    }
    partial, err := plugin.Build(cfg)
    if err != nil {
        return nil, err
    }
    bundle := &Bundle{}
    bundle.mergePartial(partial)
    return bundle, nil
}
```

### Plugin Registration (in each preset)

```go
// stack/sqlite/plugin.go
package sqlite

import (
    "github.com/larsartmann/go-cqrs-lite/stack/v4"
)

func init() {
    stack.Register(sqlitePlugin{})
}

type sqlitePlugin struct{}

func (sqlitePlugin) Name() string { return "sqlite" }

func (sqlitePlugin) Capabilities() stack.CapabilitySet {
    return stack.CapabilitySet{
        EventStore:  true,
        ViewStore:   true,
        Snapshot:    true,
        Checkpoint:  true,
        CommandLog:  true,
        QueryLog:    true,
        Bus:         true, // in-process GoChannel
    }
}

func (sqlitePlugin) Build(cfg stack.PluginConfig) (*stack.Partial, error) {
    // Thin wrapper around the existing New() constructor.
    // Translates PluginConfig.Options into typed sqlite options.
    bundle, err := New(cfg.DSN, optsFromMap(cfg.Options)...)
    if err != nil {
        return nil, err
    }
    return bundle.ToPartial(), nil
}

// ToPartial extracts Bundle fields into a stack.Partial.
// This method would live on Bundle in the stack package.
```

```go
// stack/memory/plugin.go
package memory

func init() {
    stack.Register(memoryPlugin{})
}

type memoryPlugin struct{}

func (memoryPlugin) Name() string { return "memory" }

func (memoryPlugin) Capabilities() stack.CapabilitySet {
    return stack.CapabilitySet{
        EventStore: true,
        ViewStore:  true,
        Snapshot:   true,
        Checkpoint: true,
        CommandLog: true,
        QueryLog:   true,
        Bus:        true,
    }
}

func (memoryPlugin) Build(cfg stack.PluginConfig) (*stack.Partial, error) {
    bundle := New() // existing constructor
    return bundle.ToPartial(), nil
}
```

### The Consumer Experience (Final Form)

```go
// main.go -- CONSUMER CODE (100% engine-agnostic)
package main

import (
    "os"
    "gopkg.in/yaml.v3"

    _ "github.com/larsartmann/go-cqrs-lite/stack/sqlite/v4"  // registers "sqlite"
    _ "github.com/larsartmann/go-cqrs-lite/stack/pebble/v4"  // registers "pebble"
    _ "github.com/larsartmann/go-cqrs-lite/stack/memory/v4"  // registers "memory"

    "github.com/larsartmann/go-cqrs-lite/stack/v4"
)

func main() {
    // 1. Load deployment config (deployer owns this file)
    cfg := loadDeployConfig("deploy.yaml")

    // 2. Activate plugins -> assembled Bundle
    bundle, err := stack.Deploy(cfg)
    if err != nil {
        log.Fatal(err)
    }

    // 3. Domain wiring (zero storage-specific code)
    repo, _ := stack.Repository(bundle, TaskDecider)
    host, _ := stack.ProjectionHost(bundle)
    tasks, _ := stack.Bind(bundle, taskReadModel)

    host.Register(tasks)
    go host.Start(ctx)
    defer host.Stop()

    // 4. Serve
    http.Handle("/tasks", taskHandler(repo))
    http.ListenAndServe(":8080", nil)
}

func loadDeployConfig(path string) stack.DeploymentConfig {
    data, _ := os.ReadFile(path)
    var cfg stack.DeploymentConfig
    yaml.Unmarshal(data, &cfg)
    return cfg
}
```

### The Deployer Experience

```bash
# Development: everything in memory
cat > deploy.yaml <<EOF
events:
  driver: memory
views:
  driver: memory
bus:
  driver: memory
EOF

# Staging: single SQLite
cat > deploy.yaml <<EOF
events:
  driver: sqlite
  dsn: /data/app.db
  options:
    wal: true
views:
  driver: sqlite
  dsn: /data/app.db  # same DB, different tables
bus:
  driver: memory
EOF

# Production: multi-DB topology
cat > deploy.yaml <<EOF
events:
  driver: sqlite
  dsn: /data/events.db
  options:
    wal: true
    maxOpenConns: 1
views:
  driver: pebble
  dsn: /data/views
commands:
  driver: sqlite
  dsn: /data/commands.db
bus:
  driver: memory
EOF

# Same binary. Zero recompilation. Different topology.
./myapp
```

---

## 4. Multi-Concern Deployment

The `DeploymentConfig` allows different plugins for different storage concerns.

This directly supports the multi-DB topology from ADR-0033:

```
┌─────────────────────────────────────────────────────────────┐
│                      stack.Deploy(cfg)                       │
│                                                              │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐       │
│  │   "sqlite"   │  │   "pebble"   │  │   "memory"   │       │
│  │   plugin     │  │   plugin     │  │   plugin     │       │
│  │              │  │              │  │              │       │
│  │ Events DB    │  │ Views path   │  │ Bus          │       │
│  │ Commands DB  │  │              │  │              │       │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘       │
│         │                 │                 │                │
│         └────────┬────────┴────────┬────────┘                │
│                  ▼                 ▼                          │
│         ┌──────────────────────────────┐                     │
│         │         *Bundle              │                     │
│         │  EventSink:   sqlite store   │                     │
│         │  EventSource: sqlite store   │                     │
│         │  ReadModels:  pebble kv      │                     │
│         │  Publisher:   memory bus     │                     │
│         │  Subscriber:  memory bus     │                     │
│         └──────────────────────────────┘                     │
└──────────────────────────────────────────────────────────────┘
```

The same `DeploymentConfig` can also collapse to a single backend for simple deployments:

```yaml
# Single SQLite for everything (dev/test)
events:
  driver: sqlite
  dsn: ":memory:"
views:
  driver: sqlite
  dsn: ":memory:"
```

Or use the shortcut:

```go
bundle, _ := stack.Open("sqlite", stack.PluginConfig{DSN: ":memory:"})
```

---

## 5. Bus Plugins

Message bus backends follow the exact same registration pattern:

```go
// bus/nats/plugin.go
package nats

func init() {
    stack.Register(natsPlugin{})
}

type natsPlugin struct{}

func (natsPlugin) Name() string { return "nats" }

func (natsPlugin) Capabilities() stack.CapabilitySet {
    return stack.CapabilitySet{Bus: true} // bus only
}

func (natsPlugin) Build(cfg stack.PluginConfig) (*stack.Partial, error) {
    nc, err := nats.Connect(cfg.DSN)
    if err != nil {
        return nil, err
    }
    bus := watermill.NewEventBus(
        watermill.WithBackend(natsPublisher(nc), natsSubscriber(nc), nc),
    )
    return &stack.Partial{
        Publisher:  bus,
        Subscriber: bus,
    }, nil
}
```

```go
// bus/memory/plugin.go (already provided by stack/memory)
// bus/redis/plugin.go (future)
```

Consumer's main.go:

```go
import (
    _ "go-cqrs-lite/stack/memory/v4"
    _ "go-cqrs-lite/bus/nats/v4"     // registers "nats"
    _ "go-cqrs-lite/bus/redis/v4"    // registers "redis" (future)
)
```

Deployer's config:

```yaml
bus:
  driver: nats
  dsn: nats://prod-nats:4222
```

Switching from in-process to distributed messaging = changing one YAML line. Zero code changes.

---

## 6. Why Not hashicorp/go-plugin?

`hashicorp/go-plugin` (used by Terraform, Vault, Nomad, Packer) is the gold standard for
**true** out-of-process plugins in Go. A separate binary is installed at deploy time, the
main app discovers and launches it, and they communicate via gRPC.

| Aspect                    | go-plugin                                | Registration pattern              |
| ------------------------- | ---------------------------------------- | --------------------------------- |
| True runtime loading      | Yes (separate binaries)                  | No (compiled in, config-selected) |
| Performance               | gRPC serialization on every call         | Direct function calls             |
| Complexity                | High (plugin lifecycle, gRPC, handshake) | Low (map lookup + interface call) |
| Windows support           | Yes                                      | Yes                               |
| Language-agnostic plugins | Yes (any language with gRPC)             | No (Go only)                      |
| Appropriate for a library | **No**                                   | **Yes**                           |

**The deciding factor:** go-plugin adds gRPC serialization to EVERY storage operation.
Every `store.Save()`, every `store.Load()`, every `kv.Set()` becomes a gRPC round-trip with
protobuf serialization. For a high-throughput event store, this is unacceptable overhead.

go-plugin is appropriate for **applications** (Terraform providers, Vault secrets engines)
where operations are coarse-grained and latency-tolerant. For a **library** where the storage
adapter is called on every event append, the registration pattern is the only viable choice.

**If a consumer genuinely needs out-of-process storage plugins**, they can build a thin
gRPC wrapper around the registration pattern: their plugin binary implements `Plugin` via
gRPC, and registers itself with the same `stack.Register()` API. The library doesn't need
to impose this overhead on everyone.

---

## 7. Migration Path

### What Changes for Existing Consumers

**Nothing breaks.** The existing APIs remain:

```go
// This still works (existing API, unchanged)
bundle, _ := sqlite.New("events.db")

// This also works (new shortcut)
bundle, _ := stack.Open("sqlite", stack.PluginConfig{DSN: "events.db"})

// This also works (new multi-concern path)
bundle, _ := stack.Deploy(cfg)
```

### Implementation Steps

| Step                                           | Effort | Description                                                                              |
| ---------------------------------------------- | ------ | ---------------------------------------------------------------------------------------- |
| 1. Add `Plugin` interface + registry           | ~2h    | `stack/plugin.go` with `Plugin`, `CapabilitySet`, `Partial`, `Register()`, `Available()` |
| 2. Add `Deploy()` + `Open()`                   | ~2h    | `stack/deploy.go` with config parsing + multi-plugin assembly                            |
| 3. Add `Bundle.ToPartial()` + `mergePartial()` | ~1h    | Extract Bundle fields into Partial; merge Partials into Bundle                           |
| 4. Register sqlite preset                      | ~1h    | `stack/sqlite/plugin.go` with `init()` registration                                      |
| 5. Register memory preset                      | ~30m   | `stack/memory/plugin.go`                                                                 |
| 6. Register pebble preset                      | ~1h    | `stack/pebble/plugin.go`                                                                 |
| 7. Add bus plugins                             | ~1h    | `bus/memory/plugin.go`, optionally `bus/nats/plugin.go`                                  |
| 8. Config loading helper                       | ~30m   | `stack.LoadConfig(path)` YAML decoder for `DeploymentConfig`                             |

**Total: ~1 day**

### What the Presets Look Like After Migration

Each preset (`stack/sqlite/`, `stack/pebble/`, etc.) gains a `plugin.go` file with an `init()`
registration. The existing `New()` constructor remains as the programmatic API. The `init()`
registration is a thin wrapper that calls `New()` and packages the result as a `Partial`.

```go
// stack/sqlite/plugin.go (NEW FILE, ~30 lines)
package sqlite

func init() {
    stack.Register(sqlitePlugin{})
}

type sqlitePlugin struct{}

func (sqlitePlugin) Name() string { return "sqlite" }

func (sqlitePlugin) Capabilities() stack.CapabilitySet {
    return stack.CapabilitySet{
        EventStore: true, ViewStore: true, Snapshot: true,
        Checkpoint: true, CommandLog: true, QueryLog: true, Bus: true,
    }
}

func (p sqlitePlugin) Build(cfg stack.PluginConfig) (*stack.Partial, error) {
    bundle, err := New(cfg.DSN, optsFromMap(cfg.Options)...)
    if err != nil {
        return nil, err
    }
    return bundle.ToPartial(), nil
}
```

### Example: Full Consumer App

```go
// main.go
package main

import (
    "log"
    "os"

    _ "github.com/larsartmann/go-cqrs-lite/stack/sqlite/v4"
    _ "github.com/larsartmann/go-cqrs-lite/stack/memory/v4"

    "github.com/larsartmann/go-cqrs-lite/stack/v4"
)

// Domain logic (pure, engine-agnostic, zero storage imports)
var TaskDecider = decider.Decider[TaskState]{...}
var taskReadModel = stack.Declare[TaskView, TaskID]("tasks").
    Key(taskKey).
    OnCreate(taskOnCreate).
    FilterBy("status").
    Done()

func main() {
    cfg, _ := stack.LoadConfig("deploy.yaml")
    bundle, err := stack.Deploy(cfg)
    if err != nil {
        log.Fatal(err)
    }

    repo, _ := stack.Repository(bundle, TaskDecider)
    host, _ := stack.ProjectionHost(bundle)
    mat, _ := stack.Bind(bundle, taskReadModel)
    host.Register(mat)

    go host.Start(ctx)
    defer host.Stop()

    // ... HTTP server
}
```

```yaml
# deploy.yaml
events:
  driver: sqlite
  dsn: /data/events.db
  options:
    wal: true
views:
  driver: memory
bus:
  driver: memory
```

The consumer's Go code never imports `database/sql`, never touches `*sql.DB`, never writes
SQL DDL, and never mentions a storage backend by name. The deployer controls everything via
YAML. Changing the entire storage topology is a config edit, not a code change.
