package metaengine

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
)

// SerializablePlan is a JSON-serializable representation of a PlanResult.
// It captures the essential plan decisions — engines, queries, rules, layouts —
// without the runtime closures or reflect.Type values that make PlanResult
// non-serializable.
//
// Use cases:
//   - Persist: save a plan to disk/database for audit or rollback
//   - Diff: compare two plans to see what changed after a deploy
//   - Pin: lock a plan decision to prevent drift across restarts
//   - Debug: inspect serialized plan offline to understand planner behavior
type SerializablePlan struct {
	Engines []string             `json:"engines"`
	Queries []SerializableQuery  `json:"queries"`
	Rules   []SerializableRule   `json:"rules,omitempty"`
	Layouts []SerializableLayout `json:"layouts,omitempty"`
}

// SerializableQuery is the serializable form of a QueryAssignment.
type SerializableQuery struct {
	Name             string      `json:"name"`
	ADT              ADT         `json:"adt"`
	Engine           string      `json:"engine"`
	Complexity       string      `json:"complexity"`
	LatencyMs        float64     `json:"latency_ms"`
	Persistence      Persistence `json:"persistence,omitempty"`
	Replication      Replication `json:"replication,omitempty"`
	ReplicationLagMs int64       `json:"replication_lag_ms,omitempty"`
	NetworkRTTMs     int64       `json:"network_rtt_ms,omitempty"`
}

// SerializableRule is the serializable form of a RuleTraceEntry.
type SerializableRule struct {
	Rule   string `json:"rule"`
	Query  string `json:"query"`
	Reason string `json:"reason"`
	Layout string `json:"layout,omitempty"`
}

// SerializableLayout is the serializable form of a LayoutPlan summary.
type SerializableLayout struct {
	Table   string   `json:"table"`
	Columns []string `json:"columns"`
}

// Serialize converts a PlanResult into a SerializablePlan.
func Serialize(result *PlanResult, engines []Engine) *SerializablePlan {
	sp := &SerializablePlan{}

	engineByName := make(map[string]Engine, len(engines))
	for _, e := range engines {
		engineByName[e.Profile().Name] = e
	}

	engineNames := make([]string, len(engines))
	for i, e := range engines {
		engineNames[i] = e.Profile().Name
	}

	sp.Engines = engineNames

	for _, q := range result.Queries {
		sq := SerializableQuery{
			Name:       q.QueryName,
			ADT:        q.ADT,
			Engine:     q.EngineName,
			Complexity: string(q.Complexity),
			LatencyMs:  q.Cost.EstimatedLatencyMs,
		}

		if eng, ok := engineByName[q.EngineName]; ok {
			profile := eng.Profile()
			sq.Persistence = profile.Persistence
			sq.Replication = profile.Replication
			sq.ReplicationLagMs = profile.EffectiveReplicationLag().Milliseconds()
			sq.NetworkRTTMs = profile.EffectiveNetworkRTT().Milliseconds()
		}

		sp.Queries = append(sp.Queries, sq)
	}

	for _, rt := range result.RuleTrace {
		sp.Rules = append(sp.Rules, SerializableRule{
			Rule:   rt.Rule,
			Query:  rt.Query,
			Reason: rt.Reason,
			Layout: string(rt.Layout),
		})
	}

	for _, lp := range result.LayoutPlans {
		sp.Layouts = append(sp.Layouts, SerializableLayout{
			Table:   lp.Table,
			Columns: lp.ColumnNames(),
		})
	}

	return sp
}

// MarshalJSON serializes a SerializablePlan to JSON bytes.
func (sp *SerializablePlan) MarshalJSON() ([]byte, error) {
	type alias SerializablePlan

	data, err := json.Marshal((*alias)(sp))
	if err != nil {
		return nil, fmt.Errorf("serializable.MarshalJSON: %w", err)
	}

	return data, nil
}

// SerializeToJSON is a convenience that serializes a PlanResult directly to JSON.
func SerializeToJSON(result *PlanResult, engines []Engine) ([]byte, error) {
	sp := Serialize(result, engines)
	data, err := json.Marshal(sp, jsontext.WithIndentPrefix(""), jsontext.WithIndent("  "))
	if err != nil {
		return nil, fmt.Errorf("metaengine.SerializeToJSON: %w", err)
	}

	return data, nil
}

// DeserializePlan converts JSON bytes back into a SerializablePlan.
func DeserializePlan(data []byte) (*SerializablePlan, error) {
	type alias SerializablePlan

	var sp alias
	if err := json.Unmarshal(data, &sp); err != nil {
		return nil, fmt.Errorf("metaengine.DeserializePlan: %w", err)
	}

	return (*SerializablePlan)(&sp), nil
}
