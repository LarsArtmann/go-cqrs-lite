package metaengine

import (
	"fmt"
	"reflect"
)

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

// declaredQuery resolves the query declaration behind a planned assignment as
// a snapshot (see snapshotForRule).
// The bool is false when the plan references a query the store no longer
// knows; advisory rules skip those silently (re-planning owns the error).
func declaredQuery(ctx PlanContext, q QueryAssignment) (queryMeta, bool) {
	if ctx.Store == nil {
		return nil, false
	}

	// Rules run without the store lock; RegisterQuery may add entries and a
	// concurrent Replan may re-assign engines.
	ctx.Store.mu.RLock()
	defer ctx.Store.mu.RUnlock()

	meta, ok := ctx.Store.queries[q.QueryName]
	if !ok {
		return nil, false
	}

	return ctx.Store.snapshotForRule(meta), true
}

// eachDeclaredQuery iterates the plan's queries that resolve to a declared
// query, yielding the assignment and its declared-query metadata. Undeclared
// queries carry no engine metadata and are skipped silently (re-planning owns
// the dangling-query error).
func eachDeclaredQuery(
	result *PlanResult,
	ctx PlanContext,
	fn func(q QueryAssignment, meta queryMeta),
) {
	for _, q := range result.Queries {
		meta, ok := declaredQuery(ctx, q)
		if !ok {
			continue
		}

		fn(q, meta)
	}
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
// create DDL), then write amplification (needs all queries assigned), then
// materialize-vs-replay (needs stats, runs last as an advisory diagnostic).
func defaultRules(cfg planConfig) []PlanRule {
	rules := []PlanRule{
		&schemaRule{},
		&layoutRule{dryRun: cfg.dryRun},
		&writeAmpRule{budget: cfg.writeAmplificationBudget},
		&replicationRule{},
		&degradedADTRule{},
		&mapUpdateReplicationRule{},
		&temporalAsOfRule{},
		&durabilityRule{},
		&liveLatencyRule{},
	}

	if len(cfg.stats) > 0 {
		rules = append(rules, &materializeRule{stats: cfg.stats})
	}

	if len(cfg.sharedCollections) > 0 {
		rules = append(rules, &sharedCollectionRule{shared: cfg.sharedCollections})
	}

	return rules
}

// snapshotForRule returns an independent copy of a registered query with its
// planner assignment, for a plan rule to read. Rules run without the store
// lock while a concurrent Replan rewrites the assignment, and even the
// value-receiver readers of a query copy the assigned engine along, so a rule
// must never touch the registered object itself.
//
// Caller MUST hold s.mu (read lock suffices). A query meta that is not a
// pointer is already an independent value and is returned as is.
func (s *Store) snapshotForRule(meta queryMeta) queryMeta {
	rv := reflect.ValueOf(meta)
	if rv.Kind() != reflect.Pointer {
		return meta
	}

	snapshot := reflect.New(rv.Elem().Type())
	snapshot.Elem().Set(rv.Elem())

	return snapshot.Interface().(queryMeta)
}
