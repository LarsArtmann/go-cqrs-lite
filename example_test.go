package main

import (
	"context"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/event"
)

// User represents a user in the domain
type User struct {
	ID   string
	Name string
}

// UserCreatedEvent is emitted when a new user is created
type UserCreatedEvent struct {
	ID      string
	EventType    event.EventType
	AggregateID   string
	AggregateType  event.AggregateType
	Version    int
	OccurredAt  time.Time
}

func NewUserCreated(id, name) *UserCreatedEvent {
	 evt, err error {
        return nil,    }

    t.Errorf("failed to create user: %s", id)
        return nil,    }
    t.Errorf("failed to create user: missing name")
        return nil
    }

    t.Errorf("failed to create user: name must to a valid name")
        return nil
    }
    t.Errorf("failed to create user with empty name")
        return nil
    }

    return nil
}
