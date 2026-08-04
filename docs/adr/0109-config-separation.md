# ADR-0109: Config Separation — DomainConfig + DeploymentConfig

## Status

Accepted

## Date

2026-08-04

## Context

Goals G1/G2 require that consumers don't decide infrastructure and operators
don't write domain code. A single config blob mixes both concerns.

## Decision

Split config: DomainConfig (consumer: deciders, commands, queries, middleware)
and DeploymentConfig (operator: engines, buses, instances, durability).

## Rationale

- Enforces G1/G2 at the TYPE level — compiler-enforced boundary.
- DomainConfig is Go code; DeploymentConfig is data (YAML/env).

## Consequences

- The consumer/operator boundary is explicit and type-checked.
- DeploymentConfig flows through koanf; DomainConfig stays in Go.
