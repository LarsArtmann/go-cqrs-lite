package main

import (
	"slices"
	"strings"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// ──────────────────────────────────────────────────────────────────────────
// Domain Model — the core vocabulary.
//
// This file defines the Task aggregate's types: branded IDs, value objects,
// and domain errors. Everything here is pure Go — no infrastructure deps
// beyond event/ (for error taxonomy) and id/ (for branded IDs).
// ──────────────────────────────────────────────────────────────────────────

const streamType = id.StreamType("Task")

// TaskID is the identifier for a Task aggregate. It uses id.StreamID
// because the event system keys on that type. For compile-time branded
// IDs with custom markers (id.Of[TaskMarker]), see the id/ package docs.
type TaskID = id.StreamID

// Priority ranks task urgency.
type Priority string

const (
	PriorityLow    Priority = "low"
	PriorityMedium Priority = "medium"
	PriorityHigh   Priority = "high"
	PriorityUrgent Priority = "urgent"
)

// Valid reports whether p is a recognised priority level.
func (p Priority) Valid() bool {
	switch p {
	case PriorityLow, PriorityMedium, PriorityHigh, PriorityUrgent:
		return true
	default:
		return false
	}
}

// Status tracks the task lifecycle: pending → active → completed → archived.
type Status string

const (
	StatusPending   Status = "pending"
	StatusActive    Status = "active"
	StatusCompleted Status = "completed"
	StatusArchived  Status = "archived"
)

// ──────────────────────────────────────────────────────────────────────────
// State — the fold result rebuilt from events.
// ──────────────────────────────────────────────────────────────────────────

// TaskState is the event-sourced aggregate state. Every field is derived
// from events via the apply function — never mutated directly.
type TaskState struct {
	Exists      bool
	Title       string
	Description string
	Priority    Priority
	AssigneeID  string
	Status      Status
	DueDate     *time.Time
	BlockedBy   []id.StreamID
	Tombstoned  bool
}

// IsActive reports whether the task exists and is not soft-deleted.
func (s TaskState) IsActive() bool {
	return s.Exists && !s.Tombstoned
}

// CanTransitionTo checks whether a status transition is allowed.
func (s TaskState) CanTransitionTo(target Status) bool {
	if !s.Exists || s.Tombstoned {
		return false
	}

	transitions := map[Status][]Status{
		StatusPending:   {StatusActive, StatusCompleted},
		StatusActive:    {StatusCompleted, StatusPending},
		StatusCompleted: {StatusArchived, StatusActive},
		StatusArchived:  {},
	}

	allowed, ok := transitions[s.Status]
	if !ok {
		return false
	}

	return slices.Contains(allowed, target)
}

// HasDependency reports whether the task is blocked by depID.
func (s TaskState) HasDependency(depID id.StreamID) bool {
	return slices.Contains(s.BlockedBy, depID)
}

// ──────────────────────────────────────────────────────────────────────────
// Value helpers — parse-don't-validate at the boundary.
// ──────────────────────────────────────────────────────────────────────────

// ParsePriority converts a raw string to a Priority, rejecting unknown values.
func ParsePriority(s string) (Priority, error) {
	p := Priority(strings.ToLower(strings.TrimSpace(s)))
	if !p.Valid() {
		return "", errorfamily.NewRejection(
			"task.priority.invalid",
			"priority must be low, medium, high, or urgent",
		)
	}

	return p, nil
}

// normaliseTitle trims surrounding whitespace. Returns a rejection error if empty.
func normaliseTitle(raw string) (string, error) {
	t := strings.TrimSpace(raw)
	if t == "" {
		return "", errorfamily.NewRejection("task.title.empty", "title must not be empty")
	}

	if len(t) > maxTitleLength {
		return "", errorfamily.NewRejection(
			"task.title.too_long",
			"title must not exceed 500 characters",
		)
	}

	return t, nil
}
