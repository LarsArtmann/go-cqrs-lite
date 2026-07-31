// Package mysql provides a production-ready MySQL/MariaDB preset for go-cqrs-lite.
//
// It wires all CQRS stores (event store, command store, query store, snapshot store,
// checkpoint store, KV read-model store) to a single MySQL database via the
// [storage.SQLBackend] facade, using the MySQL-specific dialect.
//
// # Quick Start
//
//	import mysql "github.com/larsartmann/go-cqrs-lite/stack/mysql/v4"
//
//	bundle, err := mysql.New("user:password@tcp(localhost:3306)/mydb")
//	if err != nil { ... }
//	defer bundle.Close()
//
// The preset auto-migrates the schema on construction (all tables use
// CREATE TABLE IF NOT EXISTS). To disable auto-migration, pass
// [mysql.WithDSN]([sqlopt.WithoutAutoMigrate]()).
//
// # Multi-Database Topology
//
// MySQL supports separate databases on the same server for event/query/view
// isolation, configured via [mysql.WithDSN]:
//
//	bundle, err := mysql.New(primaryDSN,
//	    mysql.WithDSN(
//	        sqlopt.WithEventDB("user:pass@tcp(host:3306)/events_db"),
//	        sqlopt.WithQueryDB("user:pass@tcp(host:3306)/queries_db"),
//	        sqlopt.WithViewDB("user:pass@tcp(host:3306)/views_db"),
//	    ),
//	)
//
// # MariaDB Compatibility
//
// This preset is fully compatible with MariaDB 10.2+ (the first version with
// JSON type alias). MariaDB's JSON type is an alias for LONGTEXT, but all
// schema and query operations work identically.
//
// # Event Bus
//
// MySQL has no native pub/sub mechanism. This preset uses an in-process
// Watermill event bus ([cqrswatermill.NewEventBus]) for single-process
// deployments. For multi-process event delivery, use a message broker
// (NATS, Redis, Kafka) via the [watermill] module.
package mysql
