package commands

import (
	"encoding/json"
	"fmt"
)

type CommandType string

const (
	CommandTypeCreate       CommandType = "create"
	CommandTypeUpdate       CommandType = "update"
	CommandTypeDelete       CommandType = "delete"
	CommandTypeChangeStatus CommandType = "change_status"
)

func MarshalCommandJSON(cmd any, cmdType CommandType) ([]byte, error) {
	data, err := json.Marshal(cmd)
	if err != nil {
		return nil, fmt.Errorf("marshal command %s: %w", cmdType, err)
	}
	var base map[string]any
	if err := json.Unmarshal(data, &base); err != nil {
		return nil, fmt.Errorf("unmarshal command %s: %w", cmdType, err)
	}
	base["type"] = string(cmdType)
	return json.Marshal(base)
}
