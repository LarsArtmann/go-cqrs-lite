package domain

import (
	"github.com/larsartmann/go-cqrs-lite/id/v3"
)

type TodoMarker struct {
	id.AggregateMarker
}

type TodoID = id.Of[TodoMarker]

func NewTodoID() TodoID { return id.New[TodoMarker]() }

func ParseTodoID(s string) (TodoID, error) { return id.Parse[TodoMarker](s) }
