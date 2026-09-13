package metaengine

import "time"

// CurrentHooks returns the Store's configured hooks (the zero value when
// none are set). Combine with [Hooks.Merge] before replacing via
// [WithHooks] so observability layers compose instead of clobbering each
// other:

//	obs, _ := otelobserver.Attach(store, meter) // merges internally
//
// The same read contract as WithHooks applies: configure hooks at
// construction, before concurrent use.
func (s *Store) CurrentHooks() Hooks {
	if s.hooks == nil {
		return Hooks{}
	}

	return *s.hooks
}

// Merge returns hooks that run both h and other for every callback field
// (h first); scalar fields keep h's value when set and fall back to other's.
// Use to stack observability layers — a metrics recorder plus a health
// observer, for example — without one WithHooks call erasing the other.
func (h Hooks) Merge(other Hooks) Hooks {
	out := h

	if other.OnFold != nil {
		out.OnFold = chainFold(h.OnFold, other.OnFold)
	}

	if other.OnApply != nil {
		out.OnApply = chainApply(h.OnApply, other.OnApply)
	}

	if other.OnExecute != nil {
		out.OnExecute = chainExecute(h.OnExecute, other.OnExecute)
	}

	if other.OnQuarantined != nil {
		out.OnQuarantined = chainQuarantined(h.OnQuarantined, other.OnQuarantined)
	}

	if other.OnReactivated != nil {
		out.OnReactivated = chainReactivated(h.OnReactivated, other.OnReactivated)
	}

	if other.OnProbe != nil {
		out.OnProbe = chainProbe(h.OnProbe, other.OnProbe)
	}

	if other.OnCatchUp != nil {
		out.OnCatchUp = chainCatchUp(h.OnCatchUp, other.OnCatchUp)
	}

	if out.Logger == nil {
		out.Logger = other.Logger
	}

	if out.SlowQueryThreshold == 0 {
		out.SlowQueryThreshold = other.SlowQueryThreshold
	}

	return out
}

func chainFold(
	a, b func(string, string, FoldKind, time.Duration, error),
) func(string, string, FoldKind, time.Duration, error) {
	switch {
	case a == nil:
		return b
	case b == nil:
		return a
	default:
		return func(collection, eventType string, kind FoldKind, d time.Duration, err error) {
			a(collection, eventType, kind, d, err)
			b(collection, eventType, kind, d, err)
		}
	}
}

func chainApply(a, b func(string, time.Duration, error)) func(string, time.Duration, error) {
	switch {
	case a == nil:
		return b
	case b == nil:
		return a
	default:
		return func(eventType string, d time.Duration, err error) {
			a(eventType, d, err)
			b(eventType, d, err)
		}
	}
}

func chainExecute(
	a, b func(string, ReadPattern, time.Duration, error),
) func(string, ReadPattern, time.Duration, error) {
	switch {
	case a == nil:
		return b
	case b == nil:
		return a
	default:
		return func(collection string, pattern ReadPattern, d time.Duration, err error) {
			a(collection, pattern, d, err)
			b(collection, pattern, d, err)
		}
	}
}

func chainQuarantined(a, b func(string, int, string)) func(string, int, string) {
	switch {
	case a == nil:
		return b
	case b == nil:
		return a
	default:
		return func(engine string, failures int, lastErr string) {
			a(engine, failures, lastErr)
			b(engine, failures, lastErr)
		}
	}
}

func chainReactivated(a, b func(string, string)) func(string, string) {
	switch {
	case a == nil:
		return b
	case b == nil:
		return a
	default:
		return func(engine, reason string) {
			a(engine, reason)
			b(engine, reason)
		}
	}
}

func chainProbe(a, b func(string, error)) func(string, error) {
	switch {
	case a == nil:
		return b
	case b == nil:
		return a
	default:
		return func(engine string, err error) {
			a(engine, err)
			b(engine, err)
		}
	}
}

func chainCatchUp(a, b func(string, int, error)) func(string, int, error) {
	switch {
	case a == nil:
		return b
	case b == nil:
		return a
	default:
		return func(engine string, replayed int, err error) {
			a(engine, replayed, err)
			b(engine, replayed, err)
		}
	}
}
