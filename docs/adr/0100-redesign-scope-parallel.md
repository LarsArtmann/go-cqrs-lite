# ADR-0100: Redesign Scope — Parallel (New Alongside Old), Gradual Migration

## Status

Accepted

## Date

2026-08-04

## Context

The `stack.Bundle` has structural debt (type-erased `db any`, bolted-on
metaengine, inconsistent presets) that blocks the goal. A clean-slate v5 would
force all consumers to migrate simultaneously; incremental evolution carries
the debt forever.

## Decision

Build a new `System` type alongside the existing `Bundle`. Migrate gradually.
Deprecate `Bundle` later, once the System is proven.

## Rationale

- Parallel approach lets early adopters use System while existing consumers keep Bundle.
- `stack/` stays untouched during build-out — no risk to current users.
- Short-term duplication is acceptable; long-term structural debt is not.

## Consequences

- Two composition models coexist until deprecation.
- `stack/` code is frozen, not deleted.
