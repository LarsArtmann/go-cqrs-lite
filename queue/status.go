package queue

// Status is the task lifecycle state. Transitions are validated by
// CanTransitionTo; the conformance suite pins the full legal/illegal
// matrix so no engine can invent a different lifecycle.
type Status string

const (
	// Pending means the task is waiting to be claimed (possibly delayed
	// via NotBefore).
	Pending Status = "pending"
	// Running means a worker holds the lease.
	Running Status = "running"
	// Completed means the task succeeded (terminal).
	Completed Status = "completed"
	// Dead means attempts were exhausted and the task is parked in the
	// dead-letter queue (terminal until rescued or dismissed).
	Dead Status = "dead"
	// Cancelled means the task was withdrawn before completion (terminal).
	Cancelled Status = "cancelled"
)

// AllStatuses returns every Status in lifecycle order (the single list
// other packages range over for enum-complete iteration). A function, not
// an exported slice var, so no caller can mutate the enum list for the
// whole process.
func AllStatuses() []Status {
	return []Status{Pending, Running, Completed, Dead, Cancelled}
}

// transitions lists every legal from→to pair. Anything else is invalid.
var transitions = map[Status]map[Status]bool{
	Pending:   {Running: true, Cancelled: true},
	Running:   {Pending: true, Completed: true, Dead: true, Cancelled: true},
	Completed: {},
	Dead:      {Pending: true, Cancelled: true}, // rescue re-queues; dismiss withdraws
	Cancelled: {},
}

// CanTransitionTo reports whether from → to is a legal lifecycle
// transition.
func CanTransitionTo(from, to Status) bool {
	return transitions[from][to]
}

// Terminal reports whether s is a terminal status.
func Terminal(s Status) bool {
	switch s {
	case Completed, Dead, Cancelled:
		return true
	}

	return false
}

// Valid reports whether s is a known status value.
func (s Status) Valid() bool {
	switch s {
	case Pending, Running, Completed, Dead, Cancelled:
		return true
	}

	return false
}
