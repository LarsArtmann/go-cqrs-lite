# ADR-0099: Backend Selection — Hybrid (Registry + Config), Leaning Runtime

## Status

Accepted

## Date

2026-08-04

## Context

The current stack has no backend registry or plugin system (§2.3 of the design
document). Every backend is a hardcoded constructor in its own Go module; backend
selection = import path = recompile. An operator who isn't also the developer
cannot swap backends, which directly violates goal G2 (deployer decides
infrastructure).

## Decision

Adopt a hybrid model: drivers are registered at compile time (determining which
are available), but operator config picks which to use and how — the
`database/sql` model.

## Rationale

- Pure compile-time (current) violates G2; pure runtime (full plugin loading)
  is unnecessary and adds Go plugin overhead.
- Compile-time safety (only vetted drivers linked) + runtime flexibility
  (operator picks which to activate).
- Zero runtime overhead: registration is a map insert at init.

## Consequences

- Operators can switch engines via YAML/env without recompiling.
- Adding a backend = register a driver + write an adapter (mechanical).
