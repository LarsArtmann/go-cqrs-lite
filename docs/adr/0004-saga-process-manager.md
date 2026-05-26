# ADR-0004: Saga / Process Manager Module

**Status:** Accepted  
**Date:** 2026-05-26

## Context

Real-world CQRS systems need to coordinate long-running business processes that span multiple aggregates. Examples include order fulfillment (reserve inventory → charge payment → ship order), user onboarding (create account → send welcome email → provision trial), and payment reconciliation.

Without a saga/process manager, consumers must implement this coordination manually:

- Orchestrating step sequencing across multiple command handlers
- Implementing compensation (rollback) logic when a step fails
- Managing timeouts and retry policies
- Tracking saga state and progress

This leads to duplicated boilerplate and fragile ad-hoc implementations.

## Decision

Introduce a standalone `saga/` module that provides:

- `Definition` interface — describes a saga type with typed `Steps()`
- `Step` struct — `Action` (forward command), `Compensate` (rollback command), `Timeout`
- `Instance` struct — persistent state: ID, status, current step, error
- `Runner` — registers definitions, starts instances, executes steps, triggers compensation
- `Store` interface — pluggable persistence (in-memory provided, SQL planned)
- Automatic compensation — on step failure, completed steps are rolled back in reverse order
- Built-in retry with exponential backoff using existing error classification (`event.IsRetryable`)

The saga module depends only on `core/` (commands, events, IDs) and is independently importable.

## Consequences

**Positive:**

- Long-running processes become declarative — define steps and compensation, the runner handles execution
- Compensation is automatic and ordered — no manual rollback orchestration
- Retry semantics integrate with existing error taxonomy (Rejection, Conflict, Transient)
- Standalone module — consumers only import if they need saga coordination
- Testable in isolation — `MemoryStore` + `nopDispatcher` enable unit testing without infrastructure

**Negative:**

- Saga state is separate from aggregate state — consumers must design which state lives where
- Compensation commands must be idempotent — the runner may retry compensation steps
- Timeout handling requires careful context propagation

**Neutral:**

- The `saga/` module is intentionally minimal — advanced patterns (parallel steps, sagas calling sagas) are left to consumer extensions
