package metaengine

import "time"

// FilterBindValue coerces a declarative filter value into the wire
// representation the SQL engines compare against.
//
// Views serialize as JSON — a time.Time column is stored as RFC3339 text with
// a "T" separator — and planned-table time columns are TEXT holding the same
// ISO-8601 form (LayoutPlan's type mapping: time.Time → TEXT). Binding a raw
// time.Time instead lets the driver render its default Go layout (space
// separator), and lexicographic TEXT comparison against RFC3339 text then
// corrupts same-day range bounds: Le/Lt leak the whole day, Ge/Gt drop it
// (issue #62, observed on sqliteengine v4.5.2 in pbx-lens).
//
// time.Time renders via RFC3339Nano — byte-identical to encoding/json's
// marshaling of the same instant, so filter binds compare exactly against
// stored view JSON. Offset-preserving: a stored "+02:00" document matches a
// "+02:00" filter bind; mixed-offset columns are outside the TEXT-compare
// contract. []any lists (FilterIn) coerce element-wise; every other value
// passes through untouched, and a list without time values returns the
// original slice unallocated.
func FilterBindValue(v any) any {
	switch t := v.(type) {
	case time.Time:
		return t.Format(time.RFC3339Nano)
	case []any:
		hasTime := false

		for _, e := range t {
			if _, ok := e.(time.Time); ok {
				hasTime = true

				break
			}
		}

		if !hasTime {
			return v
		}

		out := make([]any, len(t))
		for i, e := range t {
			out[i] = FilterBindValue(e)
		}

		return out
	default:
		return v
	}
}
