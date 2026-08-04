# ADR-0103: Scream Store — Tiered Enforcement

## Status

Accepted

## Date

2026-08-04

## Context

Goal G10 requires a mechanism that screams and prevents failures when operators
make unsafe changes. Today all diagnostics are advisory; SwapEngine does zero
validation.

## Decision

Three tiers: SCREAM (hard block, refuse to start), WARN+OVERRIDE (loud, requires
explicit ACK), ADVISORY (dashboard yellow, non-blocking).

## Rationale

- The system screams loudly but doesn't prevent informed decisions, except for
  genuinely destructive changes.
- Matches the reality that some changes are risky-but-intentional.

## Consequences

- Operators can never silently lose data; destructive changes require force-migrate.
- The pinned manifest + plan-diff engine becomes a deployment gate.
