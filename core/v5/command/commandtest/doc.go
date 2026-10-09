// Package commandtest provides reusable conformance tests for [command.Store]
// implementations. Backends (pebble, bbolt, memory, sql) import this package
// in their _test.go files and call [RunStoreSuite] to get the full suite for
// free, mirroring the [eventtest] pattern.
//
//	func TestCommandStoreSuite(t *testing.T) {
//	    commandtest.RunStoreSuite(t, func(t *testing.T) commandtest.StoreSuite {
//	        return newCommandStore(t)
//	    })
//	}
package commandtest
