# Design Spike: CQRS-Lite Dashboard Web UI

**Status:** Raw Idea — No Design Yet
**Module:** New `dashboard/` module

## Problem

Developers and operators need visibility into aggregates, events, projections, and tombstone states. Currently, the only way to inspect the system is through direct database queries or custom tooling.

## Vision

A self-contained web UI that:

1. Lists all aggregate types and their instances
2. Shows the event stream for any aggregate (with payload inspection)
3. Displays projection state (read models) with filtering
4. Shows tombstone status per aggregate
5. Visualizes event flow (which commands produce which events)
6. Monitors health: store connection, projection lag, checkpoint positions

## Architecture

```
dashboard/
├── server.go       # HTTP server with embedded SPA
├── handlers.go     # REST API endpoints
├── embed.go        # go:embed for static assets
└── web/            # Frontend (React/Vue/Svelte or server-rendered)
```

### API Endpoints

```
GET  /api/aggregates                    → list aggregate types
GET  /api/aggregates/:type              → list instances
GET  /api/aggregates/:type/:id/events   → event stream
GET  /api/aggregates/:type/:id/status   → tombstone status
GET  /api/projections                   → projection states
GET  /api/health                        → store/bus health
WS   /api/events/stream                 → live event feed (WebSocket/SSE)
```

### Key Design Decisions

1. **Library, not standalone service** — The dashboard is a handler you mount in your existing HTTP server, not a separate deployment.
2. **Read-only** — No mutation capability. This is an observation tool, not an admin panel.
3. **Embed assets** — Static files embedded via `//go:embed` for single-binary deployment.
4. **Backend-agnostic** — Works with any `event.Store` and `event.Journal` implementation.

## Recommendation

**Defer until consumer requests.** The existing catalog exporters (AsyncAPI, OpenAPI, D2, EventCatalog) provide schema-level visibility. Runtime inspection is a separate concern that benefits from a dedicated frontend effort.
