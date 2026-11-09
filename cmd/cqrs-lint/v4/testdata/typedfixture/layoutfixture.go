package main

// P014 typed-path fixtures: engine-shaped types exercising the ApplyLayout
// vs LayoutPlanApplier method-shape detection. The types are local — the
// rule detects method SHAPE (ApplyLayout call + ApplyLayoutPlan on the same
// receiver), not metaengine identity, so no engine dependency is needed.

// layoutPlan mirrors the plan value handed to ApplyLayoutPlan.
type layoutPlan struct {
	Collection string
}

// bothPathsEngine implements both the legacy and the plan write path —
// calling its legacy ApplyLayout must fire P014 exactly once.
type bothPathsEngine struct{}

func (bothPathsEngine) ApplyLayout(collection string, filterFields, sortFields []string) error {
	return nil
}

func (bothPathsEngine) ApplyLayoutPlan(plan layoutPlan) error { return nil }

// planOnlyEngine has only the plan path — there is no legacy method to
// call, so P014 cannot fire.
type planOnlyEngine struct{}

func (planOnlyEngine) ApplyLayoutPlan(plan layoutPlan) error { return nil }

// legacyOnlyEngine has only the legacy path (the pebble engine shape) —
// ApplyLayout is the only option, so P014 must stay silent.
type legacyOnlyEngine struct{}

func (legacyOnlyEngine) ApplyLayout(collection string, filterFields, sortFields []string) error {
	return nil
}

func applyLayoutCallSites() error {
	both := bothPathsEngine{}
	if err := both.ApplyLayout("tasks", []string{"done"}, []string{"created"}); err != nil {
		return err
	}

	legacy := legacyOnlyEngine{}
	if err := legacy.ApplyLayout("tasks", []string{"done"}, []string{"created"}); err != nil {
		return err
	}

	return nil
}
