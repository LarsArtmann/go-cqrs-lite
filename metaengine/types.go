package metaengine

import (
	"fmt"
	"sort"
	"strings"
)

// ADT is the Abstract Data Type the planner infers from fold return types.
type ADT string

const (
	ADTMap       ADT = "map"
	ADTSet       ADT = "set"
	ADTCounter   ADT = "counter"
	ADTGraph     ADT = "graph"
	ADTLog       ADT = "log"
	ADTStreamLog ADT = "stream_log"
	ADTSortedMap ADT = "sorted_map"
	ADTMultimap  ADT = "multimap"

	// ADTDueClaim is the lease-fenced due-claim capability (ADR-0142):
	// timers and task queues. Not a fold-inferred read ADT — it is a
	// write-side capability asserted via [SupportsDueClaims].
	ADTDueClaim ADT = "due_claim"

	// ADTDedup is the TTL check-and-set dedup window (ADR-0142). Write-side
	// capability asserted via [SupportsDedup].
	ADTDedup ADT = "dedup"
)

// ReadPattern describes how a query reads its projection.
type ReadPattern string

const (
	ReadPointLookup    ReadPattern = "point_lookup"
	ReadMembership     ReadPattern = "membership"
	ReadFilteredScan   ReadPattern = "filtered_scan"
	ReadAggregate      ReadPattern = "aggregate"
	ReadTraversal      ReadPattern = "traversal"
	ReadScan           ReadPattern = "scan"
	ReadMultiLookup    ReadPattern = "multi_lookup"
	ReadLogTail        ReadPattern = "log_tail"
	ReadVectorSearch   ReadPattern = "vector_search"
	ReadFullTextSearch ReadPattern = "full_text_search"
	ReadSpatialRange   ReadPattern = "spatial_range"
)

// Delta is a counter update: key to delta.
type Delta map[string]int64

// Edge is a graph edge returned by On folds with Edge return type.
//
// From/To are deliberately untyped: a single collection can mix node types
// at runtime (follows: User→User alongside assigned: Task→User), which Go
// generics cannot express (no existential types, and engine interface
// methods cannot be generic over per-value node types). Domain type safety
// lives one layer up in the typed fold handler; Edge is the storage
// boundary, same contract as MapSet's (key, value any) and MultiEntry.
type Edge struct {
	From any
	To   any
}

// EdgeRemoval is a sentinel return type for tombstone-driven edge deletion
// (ADR-0114 style): the event that added Edge{From, To} is retracted, so the
// edge is removed. Return it from OnRecord to classify the fold as an edge
// removal:
//
//	metaengine.OnRecord(Unfollowed{}, func(_ record.Record, e Unfollowed) metaengine.EdgeRemoval {
//	    return metaengine.EdgeRemoval{From: e.Follower, To: e.Followee}
//	})
type EdgeRemoval struct {
	From any
	To   any
}

// MultiEntry is a sentinel return type for multimap folds: one key maps to many values.
// Return it from OnRecord to classify the fold as a multimap insert:
//
//	metaengine.OnRecord(TaskAssigned{}, func(_ record.Record, e TaskAssigned) metaengine.MultiEntry {
//	    return metaengine.MultiEntry{Key: e.Assignee, Value: e.TaskID}
//	})
type MultiEntry struct {
	Key   any
	Value any
}

// Append is a sentinel return type for log folds: append a value to an ordered log.
// Return it from OnRecord to classify the fold as a log append:
//
//	metaengine.OnRecord(TaskCreated{}, func(_ record.Record, e TaskCreated) metaengine.Append {
//	    return metaengine.Append{Value: e}
//	})
type Append struct {
	Value any
}

// Skip is a sentinel return type signaling that an event does not apply
// to this projection (no-op). Return it from an OnRecord fold handler:
//
//	metaengine.OnRecord(SomeEvent{}, func(_ record.Record, e SomeEvent) metaengine.Skip { return metaengine.Skip{} })
type Skip struct{}

// Cursor marks a position in a paginated stream for continuation.
// In a query input struct, a field of type *Cursor named "After" signals
// keyset pagination continuation.
type Cursor struct {
	Value any
}

// Complexity is a Big-O class for cost estimation.
type Complexity string

const (
	ComplexityO1      Complexity = "O(1)"
	ComplexityOLogN   Complexity = "O(logN)"
	ComplexityON      Complexity = "O(N)"
	ComplexityONLogN  Complexity = "O(NlogN)"
	ComplexityODegree Complexity = "O(degree^depth)"
)

// RefusesADT reports whether the engine has recorded an explicit capability
// refusal for the ADT (ADR-0142), returning the architectural reason.
func (p EngineProfile) RefusesADT(adt ADT) (string, bool) {
	reason, ok := p.RefusedADTs[adt]
	return reason, ok
}

// fallback rather than a native backend. Returns false for ADTs not in Supports.
func (p EngineProfile) IsDegraded(adt ADT) bool {
	return p.DegradedADTs[adt]
}

func (p EngineProfile) String() string {
	parts := make([]string, 0, len(p.Supports))
	for adt, c := range p.Supports {
		parts = append(parts, fmt.Sprintf("%s@%s", adt, c))
	}

	sort.Strings(parts)

	extras := make([]string, 0, 4)
	if p.IsReplicated() {
		extras = append(extras, fmt.Sprintf("replication=%s", p.Replication))
		if p.ReplicationLag > 0 {
			extras = append(extras, fmt.Sprintf("lag=%s", p.ReplicationLag))
		}
	}
	if p.NetworkRTT > 0 {
		extras = append(extras, fmt.Sprintf("rtt=%s", p.NetworkRTT))
	}
	if p.IsVolatile() {
		extras = append(extras, "volatile")
	}

	suffix := ""
	if len(extras) > 0 {
		suffix = " (" + strings.Join(extras, ", ") + ")"
	}

	return fmt.Sprintf("%s: %s%s", p.Name, strings.Join(parts, " "), suffix)
}
