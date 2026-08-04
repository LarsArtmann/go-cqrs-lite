# ADR-0102: Admin Web Interface — Introspection API Only

## Status

Accepted

## Date

2026-08-04

## Context

Goal G9 requires APIs so cqrs-htmx/dashboardui can render what happens in the
metaengine. go-cqrs-lite is a library, not an app — embedding HTML/handlers
violates the library-first principle.

## Decision

go-cqrs-lite ships structured introspection types (Topology, Plan, Health, Stats)
and JSON snapshot methods. All UI code lives in cqrs-htmx/dashboardui.

## Rationale

- Separation of concerns: the library provides data, the consumer renders.
- Keeps the core library free of HTTP/template dependencies.

## Consequences

- A single JSON system snapshot endpoint becomes the UI contract.
- New panels are a cqrs-htmx concern.
