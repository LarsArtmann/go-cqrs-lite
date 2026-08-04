# ADR-0107: Bus — Operator-Configured, Multi-Bus Support

## Status

Accepted

## Date

2026-08-04

## Context

Real deployments need both local fast-path delivery (in-process projections) and
distributed delivery (cross-service fan-out). A single bus forces a compromise.

## Decision

The System supports MULTIPLE buses simultaneously, each with a named driver.
Events fan-out to all configured buses.

## Rationale

- A single bus forces a compromise between ordering and reach.
- Multiple named buses give the operator full control.

## Consequences

- GoChannel locally + NATS for fan-out in the same System.
- The System manages fan-out and per-bus lifecycle.
