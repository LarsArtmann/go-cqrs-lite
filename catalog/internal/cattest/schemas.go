package cattest

import (
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
)

// NewTestCreateOrderFlow returns a flow with a single CreateOrder step.
func NewTestCreateOrderFlow(title string) catalog.Flow {
	return catalog.Flow{
		ID: "create-order", Name: "Create Order", Version: "1.0.0",
		Summary: "",
		Steps: []catalog.FlowStep{
			{
				ID:    "1",
				Title: catalog.Title(title),
				Message: &catalog.FlowStepRef{
					ID:      testCreateOrderMsgID,
					Version: "",
				},
				Summary:   "",
				Service:   nil,
				Channel:   nil,
				Actor:     nil,
				External:  nil,
				Custom:    nil,
				NextStep:  nil,
				NextSteps: nil,
			},
		},
		Badges: nil,
	}
}

func CreateItemSchema() *catalog.Schema {
	return &catalog.Schema{
		Type: "object",
		Properties: map[string]catalog.Property{
			"name": {Type: "string", Description: "Item name"},
		},
		Required: []string{"name"},
	}
}

func StringSchema(props ...string) (*catalog.Schema, error) {
	if len(props)%2 != 0 {
		//cqrs-lint:ignore(C025) test helper — errors consumed by test code only
		return nil, fmt.Errorf(
			"cattest.StringSchema: props must be key-value pairs, got %d",
			len(props),
		)
	}

	//nolint:mnd // key-value pairs = half the input length
	m := make(map[string]catalog.Property, len(props)/2)
	for i := 0; i < len(props); i += 2 {
		m[props[i]] = catalog.Property{Type: catalog.TypeString}
	}

	return &catalog.Schema{Type: catalog.TypeObject, Properties: m}, nil
}
