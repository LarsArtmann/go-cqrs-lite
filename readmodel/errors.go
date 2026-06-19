package readmodel

import "github.com/larsartmann/go-cqrs-lite/kv/v2"

// ErrNotFound is returned by [Store.Get] when no value exists for the given key.
//
// It is the same sentinel as [kv.ErrNotFound]: Get propagates the backend's
// error directly. Re-exported here so consumers of readmodel need not also
// import the kv package to test for it.
var ErrNotFound = kv.ErrNotFound
