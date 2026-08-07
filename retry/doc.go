// Package retry is DEPRECATED. Import github.com/larsartmann/go-retry directly.
//
// This module was a re-export alias for go-retry (ADR-0064). It is no longer
// maintained and will not receive updates. The sole internal consumer
// (middleware/) has been migrated to import go-retry directly.
//
// To migrate, change your import from:
//
//	import "github.com/larsartmann/go-cqrs-lite/retry/v4"
//
// to:
//
//	import "github.com/larsartmann/go-retry"
//
// All types, functions, and sentinels are identical (type aliases).
package retry
