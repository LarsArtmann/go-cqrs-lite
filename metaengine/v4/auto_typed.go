package metaengine

import "reflect"

// The Typed fold family is the wire-name counterpart of the Auto family
// (ADR-0116 Layer 2): identical mechanical field-copy folds, but the fold
// binds to an EXPLICIT event.Type() string (e.g. "pet.adopted") instead of
// the payload struct's Go type name. This is the same bridge [OnTyped]
// provides for hand-written handlers, applied to the auto constructors —
// for CQRS journals whose wire names (user.created) deliberately differ
// from payload struct names (UserCreated) without suffix constraints.
//
// Fold classification is DECLARED here (insert/update/remove), never
// inferred from the payload name: a wire "pet.adopted" with payload
// PetAdopted folds as an insert exactly because the caller said so.

// AutoInsertTyped is [AutoInsert] with an explicit wire event type. The fold
// inserts a new row keyed by the event's keyField, copying every event field
// that matches a result field (same field-copy semantics as AutoInsert).
//
//	metaengine.AutoInsertTyped[PetAdopted, PetView]("pet.adopted", "ID")
func AutoInsertTyped[E any, R any](eventType string, keyField string) Fold {
	return retypeFold(
		autoInsertByType(reflect.TypeFor[E](), reflect.TypeFor[R](), keyField),
		eventType,
	)
}

// AutoUpdateTyped is [AutoUpdate] with an explicit wire event type. The fold
// merges non-zero event fields into the existing row (partial update
// semantics), keyed by the event's keyField.
//
//	metaengine.AutoUpdateTyped[PetRenamed, PetView]("pet.renamed", "ID")
func AutoUpdateTyped[E any, R any](eventType string, keyField string) Fold {
	return retypeFold(
		autoUpdateByType(reflect.TypeFor[E](), reflect.TypeFor[R](), keyField),
		eventType,
	)
}

// AutoDeleteTyped is [AutoDelete] with an explicit wire event type. The fold
// removes the row keyed by the event's keyField.
//
//	metaengine.AutoDeleteTyped[PetReleased]("pet.released", "ID")
func AutoDeleteTyped[E any](eventType string, keyField string) Fold {
	return retypeFold(
		autoDeleteByType(reflect.TypeFor[E](), keyField),
		eventType,
	)
}

// AutoCRUDTyped is [AutoCRUD] with explicit wire event types for all three
// folds. Insert, update, and remove semantics are declared per event, so no
// payload name suffix convention applies.
//
//	folds := metaengine.AutoCRUDTyped[PetAdopted, PetRenamed, PetReleased, PetView](
//	    "pet.adopted", "pet.renamed", "pet.released", "ID",
//	)
//	q := metaengine.Query[ScanInput, PetView]("pets", folds...)
func AutoCRUDTyped[C, U, D, R any](
	eventTypeC string,
	eventTypeU string,
	eventTypeD string,
	keyField string,
) []Fold {
	return []Fold{
		AutoInsertTyped[C, R](eventTypeC, keyField),
		AutoUpdateTyped[U, R](eventTypeU, keyField),
		AutoDeleteTyped[D](eventTypeD, keyField),
	}
}

// retypeFold rebinds a constructed fold to an explicit wire event type. The
// concrete fold kinds are package-internal, so the switch stays exhaustive;
// unknown kinds return unchanged (EventTypeName-derived) rather than panic —
// future fold kinds must opt in explicitly.
func retypeFold(f Fold, eventType string) Fold {
	switch t := f.(type) {
	case *insertFold:
		t.eventType = eventType
	case *updateFold:
		t.eventType = eventType
	case *removeFold:
		t.eventType = eventType
	}

	return f
}
