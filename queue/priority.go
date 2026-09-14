package queue

// Priority aging: ClaimDue orders by an EFFECTIVE priority — stored
// priority plus a bounded age bonus computed inside the claim query
// (scheduling, not state; the stored priority never changes). Defined
// once here, referenced by every engine, so the semantics cannot drift;
// the conformance suite pins both constants.
const (
	// PriorityAgingDaysPerPoint is the task age — keyed on CreatedAt, the
	// only immutable timestamp in the row (UpdatedAt moves on every
	// heartbeat) — that earns one priority point.
	PriorityAgingDaysPerPoint = 3

	// PriorityAgingMaxBonus caps the age bonus. It stays small enough
	// that aging reorders tasks within a comparable importance range but
	// never lets pure age leapfrog a decisively more important task.
	PriorityAgingMaxBonus = 10
)
