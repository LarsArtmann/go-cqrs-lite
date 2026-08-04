# ADR-0108: Decider Routing — Declarative Command→Event→Stream Relationships

## Status

Accepted

## Date

2026-08-04

## Context

Lars's core goal (G6): "Knowing ONLY the Commands + Events + Queries and their
relations we should be able to build superb Projections." Today there is NO
declarative relationship layer — every link is a hardcoded procedural closure.

## Decision

The System captures command→event→stream-type relationships as DATA at
registration time, enabling automatic routing and projection wiring.

## Rationale

- Capturing relationships as data (not erased closures) enables auto-wiring.
- Multiple commands on the same stream type share the decider automatically.

## Consequences

- The routing graph becomes a first-class introspectable artifact.
- The exact API shape needs prototyping; the invariant is "relationships as data."
