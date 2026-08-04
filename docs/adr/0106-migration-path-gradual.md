# ADR-0106: Migration Path — Gradual (New system/ Module)

## Status

Accepted

## Date

2026-08-04

## Context

The System needs a home module and rollout strategy. Options: dual presets (A),
gradual (B), or System in stack/ (C).

## Decision

New `system/` module. Start with sqlite + memory. Add engines mechanically via
driver registry. Bundle stays in `stack/` until System is mature.

## Rationale

- Gradual is fastest to a working System, avoids duplication.
- Option C risks old structural debt contaminating the new design.
- Driver registry makes adding engines mechanical.

## Consequences

- Non-SQLite operators wait for their driver to be ported.
- Lessons from sqlite+memory inform remaining integrations.
