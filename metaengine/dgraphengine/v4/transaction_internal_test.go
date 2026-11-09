package dgraphengine

import (
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// Compile-time capability pins (internal: the concrete type is unexported).
// If a capability is ever dropped, the build fails here instead of a
// consumer's code failing at runtime.
var (
	_ metaengine.Transactional    = (*dgraphEngine)(nil)
	_ metaengine.HealthChecker    = (*dgraphEngine)(nil)
	_ metaengine.Calibratable     = (*dgraphEngine)(nil)
	_ metaengine.Prober           = (*dgraphEngine)(nil)
	_ metaengine.TransactMeasurer = (*dgraphEngine)(nil)
)
