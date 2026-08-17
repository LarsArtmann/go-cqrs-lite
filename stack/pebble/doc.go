// Package pebble provides a PebbleDB-backed preset for [stack.Bundle].
//
// It wires every capability (event store, command store, query store, snapshot
// store, checkpoint store, and read-model backend) to a single embedded
// PebbleDB database sharing one LSM tree via disjoint key prefixes.
//
//	b, err := pebble.New("/var/lib/myapp/pebble")
//	defer b.Close()
//
// The event bus uses watermill.EventBus (GoChannel, in-process) since
// PebbleDB is a storage engine, not a message broker.
//
// All data is persistent on disk. The returned Bundle owns the *pebble.DB;
// Close releases it.
//
// Deprecated: removed in v5 (ADR-0123): system.System is the single
// composition root; presets do not survive the v5 cut. Migrate to
// system.New with DeploymentConfig (the pebbleengine backend) before v5.
package pebble
