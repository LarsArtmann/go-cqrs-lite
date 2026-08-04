# ADR-0105: Config Format — Go Struct + YAML + Env via koanf

## Status

Accepted

## Date

2026-08-04

## Context

The operator needs to configure engines, buses, and durability without writing
Go code (G2). A config mechanism that supports file + env + flags with
predictable merge ordering is needed.

## Decision

Go struct is canonical. koanf handles multi-source merging: YAML + env → struct.

## Rationale

- koanf is lightweight, extensible, supports multiple backends and parsers.
- The Go struct remains the single source of truth.

## Consequences

- Operators configure entirely via YAML/env.
- Config schema changes are typed Go changes (versioned, reviewed).
