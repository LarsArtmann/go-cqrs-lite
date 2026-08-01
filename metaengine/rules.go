package metaengine

import "fmt"

// PlanContext carries everything a PlanRule needs to make decisions.
// Rules run AFTER engine assignment — they enrich the PlanResult
// (diagnostics, layout plans) but do NOT override engine selection.
type PlanContext struct {
	Store  *Store
	Config planConfig
}

// PlanRule is a single composable planning decision. Rules are applied
// sequentially by RulePipeline after all engines are assigned.
//
// A rule enriches the PlanResult by appending diagnostics or layout plans.
// If a rule encounters a condition that makes the plan invalid (e.g., a
// layout conflict that cannot be resolved), it returns an error and
// Plan() aborts.
//
// Rules MUST NOT change the engine assignment for any query. That decision
// is made by the cost-based ranker in planQuery before rules run.
type PlanRule interface {
	Name() string
	Apply(result *PlanResult, ctx PlanContext) error
}

// RulePipeline applies a sequence of PlanRules in order.
type RulePipeline struct {
	rules []PlanRule
}

// NewRulePipeline creates a pipeline that applies the given rules sequentially.
func NewRulePipeline(rules ...PlanRule) *RulePipeline {
	return &RulePipeline{rules: rules}
}

// Apply runs every rule in order. If any rule returns an error, the pipeline
// stops and returns that error wrapped with the rule name.
func (p *RulePipeline) Apply(result *PlanResult, ctx PlanContext) error {
	for _, rule := range p.rules {
		if err := rule.Apply(result, ctx); err != nil {
			return fmt.Errorf("rule %q: %w", rule.Name(), err)
		}
	}

	return nil
}

// defaultRules returns the standard set of planning rules that Plan() applies.
// Order matters: schema enforcement first (fast reject), then layout (may
// create DDL), then write amplification (needs all queries assigned).
func defaultRules(cfg planConfig) []PlanRule {
	return []PlanRule{
		&schemaRule{},
		&layoutRule{dryRun: cfg.dryRun},
		&writeAmpRule{budget: cfg.writeAmplificationBudget},
	}
}
