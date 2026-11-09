package metaengine

// ── Branded unit types for compile-time unit safety ──
// These prevent accidentally mixing nanoseconds with rate-per-second or item counts.

// Nanoseconds is a calibrated time cost for a single operation.
type Nanoseconds float64

// Milliseconds returns the value in milliseconds (for display/cost comparison).
func (n Nanoseconds) Milliseconds() float64 { return float64(n) / 1e6 }

// RatePerSecond is a rate measured in operations per second.
type RatePerSecond float64

// ItemCount is a count of items (volume, stream length, etc.).
type ItemCount int64
