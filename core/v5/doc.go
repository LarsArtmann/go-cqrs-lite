// Package core is the v5 core module of go-cqrs-lite: the single module that
// merges the Tier 0-3 domain trains (and the composition surface) as packages
// (core/v5/event, core/v5/command, ...).
//
// It exists alongside the v4 trains during the dual-support transition
// (ADR-0152): v4 directories are never modified, v5 is a copy-forward, and
// fixes cherry-pick v4 to v5 until the fleet completes migration. Drivers
// (engines, storage drivers, transports, queues) stay separate modules and
// depend on this core.
//
// Versioning: one lockstep version per v5 wave; no per-train semver.
package core
