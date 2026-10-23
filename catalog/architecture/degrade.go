package architecture

// degrade resolves a typed identifier the index cannot link. Redacted
// resources (exclude/neverExport) drop in every mode; name-only mode
// renders the display name as plain text; without a selection the
// identifier degrades to plain text as it always has.
func (x *refIndex) degrade(kind resourceKind, id string) (resourceRef, bool) {
	key := selKey{kind: kind, id: id}

	if x.banned[key] {
		return droppedRef(), false
	}

	if x.references == ReferencesNameOnly {
		if name := x.names[key]; name != "" {
			return plainRef(name), true
		}

		if x.kept != nil {
			return droppedRef(), false
		}
	}

	if x.kept == nil {
		return plainRef(id), true
	}

	return droppedRef(), false
}

// degradeAny is degrade for identifiers the model stores without a kind.
// The no-selection fallback appends a zero reference, which writeRef skips.
func (x *refIndex) degradeAny(id string) (resourceRef, bool) {
	if x.bannedIDs[id] {
		return droppedRef(), false
	}

	if x.references == ReferencesNameOnly {
		if name := x.namesByID[id]; name != "" {
			return plainRef(name), true
		}

		if x.kept != nil {
			return droppedRef(), false
		}
	}

	if x.kept == nil {
		return droppedRef(), true
	}

	return droppedRef(), false
}
