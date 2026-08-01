package metaengine

import (
	"fmt"
	"reflect"
)

// schemaRule validates that fold value types match the declared query result
// type. Mismatches would surface at runtime as decode errors; catching them
// at Plan time gives early feedback.
type schemaRule struct{}

func (*schemaRule) Name() string { return "schema-enforcement" }

func (*schemaRule) Apply(result *PlanResult, ctx PlanContext) error {
	for _, q := range result.Queries {
		rt, ok := ctx.Store.queries[q.QueryName]
		if !ok {
			continue
		}

		if rt.QueryResultType() == nil {
			continue
		}

		for _, fold := range rt.QueryFolds() {
			var vt reflect.Type
			switch f := fold.(type) {
			case *insertFold:
				vt = f.valueType
			case *updateFold:
				vt = f.valueType
			case *removeFold:
				vt = f.valueType
			}

			if vt != nil && vt != rt.QueryResultType() {
				result.Diagnostics = append(result.Diagnostics, Diagnostic{
					Level: DiagLevelWarn,
					Query: rt.QueryName(),
					Message: fmt.Sprintf(
						"fold for %s returns %s but query result type is %s — "+
							"runtime decode may fail",
						fold.EventType(),
						vt.String(),
						rt.QueryResultType().String(),
					),
				})
			}
		}
	}

	return nil
}
