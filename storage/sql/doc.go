// Package sql provides shared SQL infrastructure for the storage module.
// It contains the Dialect abstraction, base types, helpers, and error types
// used by all SQL-based store implementations: event store, command store,
// query store, snapshot store, checkpoint store, and KV store. Each Dialect
// (Postgres, SQLite) exposes the schema DDL for every store it supports.
package sql
