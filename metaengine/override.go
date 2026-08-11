package metaengine

// overrideFold wraps a Fold to mark it as an override for inference.
// When used with Infer(), the wrapped fold replaces any inferred fold
// for the same event type. This is the escape hatch for the 20% case
// where auto-projection gets the fold wrong.
type overrideFold struct {
	Fold
}

// Override marks a fold as an override for auto-projection inference.
// When combined with Infer(), this fold replaces any inferred fold
// for the same event type.
//
// Example:
//
//	q := metaengine.Query[GetUser, UserView]("users",
//	    metaengine.Infer(UserCreated{}, UserUpdated{}, UserDeleted{}),
//	    metaengine.Override(metaengine.OnRecord(UserUpdated{},
//	        func(_ record.Record, e UserUpdated) (UserID, UserView) {
//	            return e.ID, UserView{Name: e.Name, Version: e.Version + 1}
//	        })),
//	)
//
// The planner infers folds for UserCreated and UserDeleted automatically,
// but uses the explicit fold for UserUpdated instead of the inferred one.
func Override(f Fold) overrideFold {
	return overrideFold{Fold: f}
}

// applyOverrides replaces inferred folds whose event types match override
// folds. Override folds that don't match any inferred event type are appended
// as additional folds (the override declares a fold for an event the Infer
// samples didn't cover).
func applyOverrides(inferred []Fold, overrides []overrideFold) []Fold {
	if len(overrides) == 0 {
		return inferred
	}

	used := make(map[int]bool, len(overrides))

	result := make([]Fold, 0, len(inferred)+len(overrides))

	for _, f := range inferred {
		replaced := false

		for i, ov := range overrides {
			if ov.EventType() == f.EventType() {
				result = append(result, ov.Fold)
				used[i] = true
				replaced = true

				break
			}
		}

		if !replaced {
			result = append(result, f)
		}
	}

	for i, ov := range overrides {
		if !used[i] {
			result = append(result, ov.Fold)
		}
	}

	return result
}
