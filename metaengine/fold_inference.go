package metaengine

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
)

// inferenceRequest is a marker type that tells Query() to defer fold
// generation to Plan() time (ADR-0116 Layer 1). The planner inspects each
// sample's field names and type relationships against the query input (Q)
// and result (R) types to synthesize folds automatically.
type inferenceRequest struct {
	samples []any
}

// Infer requests planner-time fold inference from event struct shapes.
// The planner inspects each sample's field names, struct composition, and
// naming conventions (Created/Updated/Deleted suffixes) to synthesize folds
// automatically. The consumer declares zero folds for the 80% case.
//
// IMPORTANT: Not recommended for production domain models. Explicit folds
// (OnRecord, On, AutoInsert, etc.) are strongly preferred because they make
// the projection semantics visible, testable, and auditable at the call site.
// Infer trades that clarity for brevity — the projection logic is hidden
// behind naming conventions and reflection. Use Infer only for quick
// prototypes, throwaway demos, or genuinely trivial CRUD views where the
// field mapping is so obvious that an explicit fold would add zero
// information. For any projection with business meaning, write the fold
// explicitly so the reader can see how events become read models.
//
// Key field is auto-detected from the query input type Q: if Q has exactly
// one non-pagination field, its Go type identifies the key. The matching
// field in the Created event struct becomes the key extractor.
//
// Filters are auto-detected: query input fields beyond the key that share a
// name with a result field become FilterOnField declarations.
//
// Nested structs are flattened for field matching: a Created event with an
// embedded Address{City, Zip} maps to a result with top-level City and Zip.
//
// Example:
//
//	type GetUser struct{ ID UserID }
//	type UserView struct{ ID UserID; Name string; Email string }
//	type UserCreated struct{ ID UserID; Name string; Email string }
//	type UserDeleted struct{ ID UserID }
//
//	q := metaengine.Query[GetUser, UserView]("users",
//	    metaengine.Infer(UserCreated{}, UserDeleted{}),
//	)
//
// The planner infers: insert fold (UserCreated → UserView by field name),
// delete fold (UserDeleted.ID → remove), key field "ID" (from GetUser.ID),
// and read pattern ReadPointLookup (GetUser has one input field).
func Infer(samples ...any) inferenceRequest {
	if len(samples) == 0 {
		panic("metaengine.Infer: at least one event sample required")
	}

	for i, s := range samples {
		t := reflect.TypeOf(s)
		if t == nil {
			panic(fmt.Sprintf("metaengine.Infer: sample[%d] is a nil interface", i))
		}

		if t.Kind() == reflect.Pointer {
			t = t.Elem()
		}

		if t.Kind() != reflect.Struct {
			panic(fmt.Sprintf(
				"metaengine.Infer: sample[%d] (%s) must be a struct, got %s",
				i, t.Name(), t.Kind(),
			))
		}
	}

	return inferenceRequest{samples: samples}
}

// conventionClassification holds the event types classified by naming suffix.
type conventionClassification struct {
	created reflect.Type
	updated reflect.Type
	deleted reflect.Type
}

// classifyByConvention inspects sample Go type names for Created/Updated/Deleted
// suffixes (ADR-0116 Layer 1). Requires at least one *Created sample.
func classifyByConvention(samples []any) (conventionClassification, error) {
	var c conventionClassification

	for _, s := range samples {
		t := reflect.TypeOf(s)
		if t.Kind() == reflect.Pointer {
			t = t.Elem()
		}

		name := t.Name()

		switch {
		case strings.HasSuffix(name, "Created"):
			if c.created != nil {
				return c, fmt.Errorf("infer: multiple *Created types: %s and %s",
					c.created.Name(), name)
			}

			c.created = t

		case strings.HasSuffix(name, "Updated"):
			if c.updated != nil {
				return c, fmt.Errorf("infer: multiple *Updated types: %s and %s",
					c.updated.Name(), name)
			}

			c.updated = t

		case strings.HasSuffix(name, "Deleted"):
			if c.deleted != nil {
				return c, fmt.Errorf("infer: multiple *Deleted types: %s and %s",
					c.deleted.Name(), name)
			}

			c.deleted = t

		default:
			return c, fmt.Errorf(
				"infer: type %s does not match *Created/*Updated/*Deleted suffix",
				name,
			)
		}
	}

	if c.created == nil {
		return c, errors.New("infer: no *Created sample provided (at least one is required)")
	}

	return c, nil
}

// exportedNonMetaTypeFields returns exported struct fields that are not
// pagination metadata (Limit, After, Depth).
func exportedNonMetaTypeFields(t reflect.Type) []reflect.StructField {
	metaNames := map[string]bool{
		limitField: true,
		afterField: true,
		depthField: true,
	}

	var fields []reflect.StructField

	for i := range t.NumField() {
		f := t.Field(i)
		if f.IsExported() && !metaNames[f.Name] {
			fields = append(fields, f)
		}
	}

	return fields
}

// generateInferredFolds builds Fold values from classified event types using
// field-name matching (including nested struct flattening via matchFields).
// For collection result types, the element type is used as the fold value type.
func generateInferredFolds(
	c conventionClassification,
	resultType reflect.Type,
	keyField string,
) []Fold {
	valueType := collectionElementType(resultType)

	var folds []Fold

	folds = append(folds, autoInsertByType(c.created, valueType, keyField))

	if c.updated != nil {
		folds = append(folds, autoUpdateByType(c.updated, valueType, keyField))
	}

	if c.deleted != nil {
		folds = append(folds, autoDeleteByType(c.deleted, keyField))
	}

	return folds
}

// collectionElementType returns the element type for collection result types
// (structs with an Items []T field). For non-collection types, returns the
// input type unchanged.
func collectionElementType(resultType reflect.Type) reflect.Type {
	if itemsField, ok := resultType.FieldByName("Items"); ok &&
		itemsField.Type.Kind() == reflect.Slice {
		elemType := itemsField.Type.Elem()
		if elemType.Kind() == reflect.Pointer {
			elemType = elemType.Elem()
		}

		if elemType.Kind() == reflect.Struct {
			return elemType
		}
	}

	return resultType
}

// ensureFolds runs planner-time fold inference for queries declared with
// Infer() or InferFromNamedEvents(). For queries with explicit folds, this
// is a no-op. It classifies events by naming convention, auto-detects key
// fields from the query input type (including composite keys), generates
// folds via field-name matching (including nested struct flattening), applies
// wire event type overrides for named inference, and auto-infers filters
// and sort from query input and result types.
func (q *QueryDecl[Q, R]) ensureFolds() error {
	if !q.needsInference {
		return nil
	}

	resultType := reflect.TypeOf(q.resultSample)
	queryType := reflect.TypeOf(q.querySample)

	classified, err := classifyByConvention(samplesForClassification(q))
	if err != nil {
		return fmt.Errorf("query %q: %w", q.Name, err)
	}

	det, err := detectKeyFields(queryType, classified.created)
	if err != nil {
		return fmt.Errorf("query %q: %w", q.Name, err)
	}

	var folds []Fold

	if det.composite {
		folds = generateCompositeFolds(classified, resultType, det)
	} else {
		folds = generateInferredFolds(classified, resultType, det.fields[0])
	}

	folds = applyOverrides(folds, q.overrides)

	if len(q.namedSamples) > 0 {
		applyNamedEventTypes(folds, q.namedSamples)
	}

	q.Config = autoInferFilters(queryType, resultType, det.fields, q.Config)
	q.Config = autoInferSort(queryType, resultType, q.Config)

	adt, err := classifyADT(folds)
	if err != nil {
		return fmt.Errorf("query %q: %w", q.Name, err)
	}

	if err := deriveKeys(folds); err != nil {
		return fmt.Errorf("query %q: %w", q.Name, err)
	}

	q.Folds = folds
	q.ADT = adt
	q.infer()

	return nil
}

// samplesForClassification extracts the struct samples from the query
// declaration for convention classification. Uses namedSamples when present
// (InferFromNamedEvents), otherwise eventSamples (Infer).
func samplesForClassification[Q any, R any](q *QueryDecl[Q, R]) []any {
	if len(q.namedSamples) > 0 {
		samples := make([]any, len(q.namedSamples))
		for i, s := range q.namedSamples {
			samples[i] = s.sample
		}

		return samples
	}

	return q.eventSamples
}
