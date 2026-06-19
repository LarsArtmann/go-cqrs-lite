package readmodel

import "github.com/larsartmann/go-cqrs-lite/kv/v2"

// Backend is the untyped key-value store that a [Store] reads from and writes to.
//
// It is an alias for [kv.Store], not a new interface: [kv.MemStore] and
// [pebble.KVAdapter] already implement it. A deployer passes whichever
// implementation suits the deployment (in-memory for tests, Pebble for an
// embedded store, a future SQL adapter for managed databases) and the
// application code is identical in every case.
//
// Aliasing — rather than defining a competing interface — keeps a single
// source of truth for the key-value contract and avoids the situation where
// two interfaces happen to drift apart.
type Backend = kv.Store
