# ADR-0104: System Scope — Layered-Full (Owns All Infrastructure)

## Status

Accepted

## Date

2026-08-04

## Context

Goal G8 requires app code to be deployment-agnostic, but today consumers open
DBs, build engines, and wire projection hosts manually.

## Decision

The System owns ALL infrastructure: storage instances, bus(es), projectionhost,
dispatchers. The consumer provides ONLY domain types, folds, and middleware.

## Rationale

- If the consumer wires projectionhost manually, app code still cares about
  deployment (violates G8).
- Bus type is a deployment decision = operator config.

## Consequences

- Consumer code shrinks to domain declarations.
- The System gains a large construction surface (operator-configured).
